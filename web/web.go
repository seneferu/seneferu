package web

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/labstack/echo"
	"gitlab.com/sorenmat/seneferu/builder"
	"gitlab.com/sorenmat/seneferu/model"
	"gitlab.com/sorenmat/seneferu/storage"
	"gitlab.com/sorenmat/seneferu/webstream"
	"golang.org/x/net/websocket"
	"gopkg.in/go-playground/webhooks.v3"
	"gopkg.in/go-playground/webhooks.v3/github"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"
)

// HandleRelease handles GitHub release events
func HandleRelease(payload interface{}, header webhooks.Header) {
	fmt.Println("Handling Release")

	pl := payload.(github.ReleasePayload)

	// only want to compile on full releases
	if pl.Release.Draft || pl.Release.Prerelease || pl.Release.TargetCommitish != "master" {
		return
	}

	// Do whatever you want from here...
	fmt.Printf("%+v", pl)
}

// HandlePullRequest handles GitHub pull_request events
func HandlePullRequest(payload interface{}, header webhooks.Header) {

	fmt.Println("Handling Pull Request")

	pl := payload.(github.PullRequestPayload)

	// Do whatever you want from here...
	fmt.Printf("%+v", pl)
}

func HandlePush(service storage.Service, kubectl *kubernetes.Clientset) webhooks.ProcessPayloadFunc {
	return func(payload interface{}, header webhooks.Header) {
		fmt.Println("Handling Push Request")

		pl := payload.(github.PushPayload)

		repo, err := service.LoadByOrgAndName(pl.Repository.Owner.Name, pl.Repository.Name)
		if err != nil {
			repo = &model.Repo{
				Org:  pl.Repository.Owner.Name,
				Name: pl.Repository.Name,
			}
			// this is odd move to a save function
			err = service.SaveRepo(repo)
			if err != nil {
				fmt.Println(err)
			}
		}
		build := &model.Build{
			Org:        pl.Repository.Owner.Name,
			Name:       pl.Repository.Name,
			Commit:     pl.HeadCommit.ID,
			Committers: []string{pl.Pusher.Name},
			Status:     "Created",
			Timestamp:  time.Now(),
		}
		err = builder.ExecuteBuild(kubectl, service, build, repo)
		if err != nil {
			panic(err) //TODO reconsider this
		}

	}
}

func StartWebServer(db storage.Service, kubectl *kubernetes.Clientset, secret string, helmHost string) {

	// Github hook
	hook := github.New(&github.Config{Secret: secret})
	hook.RegisterEvents(HandleRelease, github.ReleaseEvent)
	hook.RegisterEvents(HandlePullRequest, github.PullRequestEvent)
	hook.RegisterEvents(HandlePush(db, kubectl), github.PushEvent)

	broker := webstream.NewServer()

	e := echo.New()

	e.Static("/styles", "styles")
	e.Static("/scripts", "scripts")
	e.Static("/images", "images")
	e.File("/", "index.html")
	e.GET("/status", handleStatus())
	e.GET("/repos", handleFetchRepos(db))
	e.GET("/repo/:org/:id", handleFetchRepoData(db))
	e.GET("/repo/:org/:id/builds", handleFetchBuilds(db))
	e.GET("/repo/:org/:id/build/:buildid", handleFetchBuild(db))
	e.GET("/helm/:release", handleHelm(helmHost))
	//e.GET("/ws/:repo/build/:buildid/:step", logStream)
	e.GET("/ws", func(c echo.Context) (err error) {
		req := c.Request()
		res := c.Response()

		broker.ServeHTTP(res, req)
		return nil
	})

	// handle github web hook
	e.Any("/webhook", func(c echo.Context) (err error) {
		req := c.Request()
		res := c.Response()
		webhooks.Handler(hook).ServeHTTP(res, req)
		//server.ServeHTTP(res, req)
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
		fmt.Printf("Id: %v,Org: %v\n", id, org)
		repo, err := db.LoadByOrgAndName(org, id)
		if err != nil {
			c.Error(err)
			return err
		}

		fmt.Println("Found repo ", repo)
		err = c.JSON(200, repo)
		if err != nil {
			log.Println("unable to marshal json")
		}
		return err
	}
}

func handleHelm(host string) echo.HandlerFunc {
	return func(c echo.Context) error {
		release := c.Param("release")
		if release == "" {
			return fmt.Errorf("release can't be empty")
		}
		client := helm.NewClient(helm.Host(host))

		_, err := client.GetVersion()
		if err != nil {
			return err
		}

		list, err := client.ReleaseHistory(release, helm.WithMaxHistory(10))
		if err != nil {
			return err
		}

		var deployments []model.Deployment
		for _, v := range list.GetReleases() {
			d := model.Deployment{Version: strconv.Itoa(int(v.Version)), Name: v.Name, Status: v.GetInfo().GetStatus().Code.String(), Description: v.GetInfo().GetDescription()}
			deployments = append(deployments, d)
		}
		c.JSON(200, deployments)

		return nil
	}
}

func handleFetchBuild(db storage.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		org := c.Param("org")
		id := c.Param("id")
		fmt.Printf("Id: %v,Org: %v\n", id, org)
		buildidStr := c.Param("buildid")

		buildid, err := strconv.Atoi(buildidStr)
		if err != nil {
			return err
		}

		b, err := db.LoadStepInfos(org, id, buildid)
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

func handleStatus() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(200, "ok")
	}
}
