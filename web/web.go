package web

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"gitlab.com/sorenmat/seneferu/builder"
	"gitlab.com/sorenmat/seneferu/model"
	"gitlab.com/sorenmat/seneferu/storage"
	"golang.org/x/net/websocket"
	"gopkg.in/go-playground/webhooks.v3"
	"gopkg.in/go-playground/webhooks.v3/github"
	"k8s.io/client-go/kubernetes"
)

// HandleRelease handles GitHub release events
func HandleRelease(payload interface{}, header webhooks.Header) {
	log.Println("Not Handling Release right now")

	pl := payload.(github.ReleasePayload)

	// only want to compile on full releases
	if pl.Release.Draft || pl.Release.Prerelease || pl.Release.TargetCommitish != "master" {
		return
	}
}

// HandlePullRequest handles GitHub pull_request events
func HandlePullRequest(service storage.Service, kubectl *kubernetes.Clientset, token string, targetURL string, dockerRegHost string, sshkey string) webhooks.ProcessPayloadFunc {
	log.Println("Handling Pull Request right now")
	return func(payload interface{}, header webhooks.Header) {
		pl := payload.(github.PullRequestPayload)

		repo, err := service.LoadByOrgAndName(pl.Repository.Owner.Login, pl.Repository.Name)
		if err != nil {
			log.Println("got an error, assuming we couldn't find the repository")
			repo = &model.Repo{
				Org:  pl.Repository.Owner.Login,
				Name: pl.Repository.Name,
			}
			log.Printf("Trying to save %v\n", repo)
			// this is odd move to a save function
			err = service.SaveRepo(repo)
			if err != nil {
				log.Println(err)
			}
		}

		build := &model.Build{
			Org:        pl.PullRequest.Head.Repo.Owner.Login,
			Name:       pl.PullRequest.Head.Repo.Name,
			Commit:     pl.PullRequest.Head.Sha,
			Ref:        pl.PullRequest.Head.Ref,
			Committers: []string{pl.PullRequest.Head.User.Login},
			Status:     "Created",
			Timestamp:  time.Now(),
			TreesURL:   pl.PullRequest.Head.Repo.TreesURL,
			StatusURL:  pl.PullRequest.StatusesURL,
		}
		fmt.Println("Build: ", build)
		err = builder.ExecuteBuild(kubectl, service, build, repo, token, targetURL, dockerRegHost, sshkey)
		if err != nil {
			log.Printf("Build failure %v\n", err)
		}
	}
}

// HandlePush receives and handles the push event from github
func HandlePush(service storage.Service, kubectl *kubernetes.Clientset, token string, targetURL string, dockerRegHost string, sshkey string) webhooks.ProcessPayloadFunc {
	return func(payload interface{}, header webhooks.Header) {
		log.Println("Handling Push Request")

		pl := payload.(github.PushPayload)

		repo, err := service.LoadByOrgAndName(pl.Repository.Owner.Name, pl.Repository.Name)
		if err != nil {
			log.Println("got an error, assuming we couldn't find the repository")
			repo = &model.Repo{
				Org:  pl.Repository.Owner.Name,
				Name: pl.Repository.Name,
			}
			log.Printf("Trying to save %v\n", repo)
			// this is odd move to a save function
			err = service.SaveRepo(repo)
			if err != nil {
				log.Println(err)
			}
		}

		build := &model.Build{
			Org:        pl.Repository.Owner.Name,
			Name:       pl.Repository.Name,
			Commit:     pl.HeadCommit.ID,
			Ref:        pl.Ref,
			Committers: []string{pl.Pusher.Name},
			Status:     "Created",
			Timestamp:  time.Now(),
			TreesURL:   pl.Repository.TreesURL,
			StatusURL:  pl.Repository.StatusesURL,
		}

		err = builder.ExecuteBuild(kubectl, service, build, repo, token, targetURL, dockerRegHost, sshkey)
		if err != nil {
			log.Printf("Build failure %v\n", err)
		}

	}
}

func HandlePing() webhooks.ProcessPayloadFunc {
	return func(payload interface{}, header webhooks.Header) {
		pl := payload.(github.PingPayload)
		log.Println("Got Ping Request", pl)
	}
}

func HandleStatus() webhooks.ProcessPayloadFunc {
	return func(payload interface{}, header webhooks.Header) {
		log.Println("Got Status Request")
	}
}
func StartWebServer(db storage.Service, kubectl *kubernetes.Clientset, secret string, targetURL string, token string, dockerRegHost string, sshkey string) {

	// Github hook
	hook := github.New(&github.Config{Secret: secret})
	hook.RegisterEvents(HandleRelease, github.ReleaseEvent)
	hook.RegisterEvents(HandleStatus(), github.StatusEvent)
	hook.RegisterEvents(HandlePullRequest(db, kubectl, token, targetURL, dockerRegHost, sshkey), github.PullRequestEvent)
	hook.RegisterEvents(HandlePing(), github.PingEvent)
	hook.RegisterEvents(HandlePush(db, kubectl, token, targetURL, dockerRegHost, sshkey), github.PushEvent)

	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
	}))
	e.Use(middleware.Logger())

	e.Static("/static", "static")
	e.File("/", "index.html")
	e.GET("/status", handleStatus())
	e.GET("/builds", handleFetchAllBuilds(db))
	e.GET("/repos", handleFetchRepos(db))
	e.GET("/repo/:org/:id", handleFetchRepoData(db))
	e.GET("/repo/:org/:id/builds", handleFetchBuilds(db))
	e.GET("/repo/:org/:id/build/:buildid", handleFetchBuild(db))
	e.GET("/repo/:org/:id/build/:buildid/step/:step", handleFetchStep(db))

	// handle github web hook
	e.Any("/webhook", func(c echo.Context) (err error) {
		req := c.Request()
		res := c.Response()
		ct := req.Header.Get("Content-Type")
		if ct != "application/json" {
			log.Printf("Received payload on /webhook with unsupported mediatype, the request was %v", ct)
			return fmt.Errorf(http.StatusText(415))
		}
		webhooks.Handler(hook).ServeHTTP(res, req)
		return nil
	})
	log.Println("Starting server....")
	e.Start(":8080")
}

var sockets = NewSockets()

type Sockets struct {
	Add     chan *websocket.Conn
	Remove  chan *websocket.Conn
	sockets []*websocket.Conn
}

func (s *Sockets) GetSockets() []*websocket.Conn {
	return s.sockets
}

func (s *Sockets) handle() {
	for {
		select {
		case add := <-s.Add:
			s.sockets = append(s.sockets, add)
		case remove := <-s.Remove:
			for i, c := range s.sockets {
				if c == remove {
					s.sockets = append(s.sockets[:i], s.sockets[i+1:]...)
				}
			}
		}
	}
}

func NewSockets() *Sockets {
	s := Sockets{}
	s.Add = make(chan *websocket.Conn)
	s.Remove = make(chan *websocket.Conn)

	go s.handle()
	return &s
}

func logStream(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		sockets.Add <- ws
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func handleFetchBuilds(db storage.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		org := c.Param("org")
		if org == "" {
			return fmt.Errorf("org can't be empty")
		}
		id := c.Param("id")
		if id == "" {
			return fmt.Errorf("id can't be empty")
		}
		log.Printf("Fetching builds for repo id: %v", id)

		builds, err := db.LoadBuilds(org, id)
		if err != nil {
			return err
		}
		log.Printf("Found %v builds", len(builds))
		return c.JSON(200, builds)
	}
}

func handleFetchRepoData(db storage.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		org := c.Param("org")
		id := c.Param("id")
		log.Printf("Id: %v\tOrg: %v\n", id, org)
		repo, err := db.LoadByOrgAndName(org, id)
		if err != nil {
			return err
		}

		log.Println("Found repo ", repo)
		err = c.JSON(200, repo)
		if err != nil {
			log.Println("unable to marshal json")
		}
		return err
	}
}

func handleFetchBuild(db storage.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		org := c.Param("org")
		id := c.Param("id")
		buildidStr := c.Param("buildid")
		log.Printf("Id: %v\tOrg: %v\tBuildId: %v\n", id, org, buildidStr)

		buildid, err := strconv.Atoi(buildidStr)
		if err != nil {
			return err
		}

		b, err := db.LoadSteps(org, id, buildid)
		if err != nil {
			return err
		}
		return c.JSON(200, b)
	}
}

func handleFetchStep(db storage.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		org := c.Param("org")
		id := c.Param("id")
		buildidStr := c.Param("buildid")
		step := c.Param("step")

		log.Printf("Id: %v\tOrg: %v\tBuildId: %v\tStep: %v\n", id, org, buildidStr, step)

		buildid, err := strconv.Atoi(buildidStr)
		if err != nil {
			return err
		}

		b, err := db.LoadStep(org, id, buildid, step)
		if err != nil {
			return err
		}
		return c.JSON(200, b)
	}
}

func handleFetchRepos(db storage.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		repos, err := db.All()
		if err != nil {
			log.Println(err)
			return err
		}
		return c.JSON(200, repos)
	}
}

func handleFetchAllBuilds(db storage.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		max := 0
		maxStr := c.QueryParam("max")
		if maxStr != "" {
			max, _ = strconv.Atoi(maxStr)
		}
		repos, err := db.LoadAllBuilds(max)
		if err != nil {
			log.Println(err)
			return err
		}
		return c.JSON(200, repos)
	}
}

func handleStatus() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(200, "ok")
	}
}
