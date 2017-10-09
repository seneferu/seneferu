package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/asdine/storm"
	"github.com/boltdb/bolt"
	"github.com/labstack/echo"
	"golang.org/x/net/websocket"
	"gopkg.in/go-playground/webhooks.v3"
	"gopkg.in/go-playground/webhooks.v3/github"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/helm"
)

func (r *Repo) Save(b *Build) error {
	added := false
	for _, v := range r.Build {
		if v.Number == b.Number {
			v = b
			added = true
		}
	}
	if !added {
		// new build
		r.Build = append(r.Build, b)
	}
	return r.db.Save(r)
}

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

func HandlePush(db *storm.DB, kubectl *kubernetes.Clientset) webhooks.ProcessPayloadFunc {
	return func(payload interface{}, header webhooks.Header) {
		fmt.Println("Handling Push Request")

		pl := payload.(github.PushPayload)

		// Do whatever you want from here...
		fmt.Printf("%+v", pl)

		name := fmt.Sprintf("%v.%v", pl.Repository.Owner.Name, pl.Repository.Name)
		repo := getRepo(db, name)
		if repo == nil {
			panic("unable to find repo") //TODO reconsider this
		}
		build := &Build{
			Repo:       pl.Repository.Name,
			Owner:      pl.Repository.Owner.Name,
			Commit:     pl.HeadCommit.ID,
			Committers: []string{pl.Pusher.Name},
			Status:     "Created",
			Timestamp:  time.Now(),
		}
		err := executeBuild(kubectl, build, repo)
		if err != nil {
			panic(err) //TODO reconsider this
		}

	}
}

func startWebServer(db *storm.DB, kubectl *kubernetes.Clientset, secret string, helmHost string) {

	// Github hook
	hook := github.New(&github.Config{Secret: secret})
	hook.RegisterEvents(HandleRelease, github.ReleaseEvent)
	hook.RegisterEvents(HandlePullRequest, github.PullRequestEvent)
	hook.RegisterEvents(HandlePush(db, kubectl), github.PushEvent)

	e := echo.New()

	e.Static("/styles", "styles")
	e.Static("/scripts", "scripts")
	e.Static("/images", "images")
	e.File("/", "index.html")
	e.File("/v2", "index_new.html")
	e.GET("/status", handleStatus())
	e.GET("/repos", handleFetchRepos(db))
	e.GET("/repo/:id", handleFetchRepoData(db))
	e.GET("/repo/:id/builds", handleFetchBuilds(db))
	e.GET("/repo/:id/build/:buildid", handleFetchBuild(db))
	e.GET("/helm/:release", handleHelm(helmHost))
	//e.GET("/ws/:repo/build/:buildid/:step", logStream)
	e.GET("/ws", logStream)
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

func handleFetchBuilds(db *storm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		log.Printf("Fetching builds for repo id: %v", id)
		var repo Repo

		err := db.One("Id", id, &repo)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(repo)
		return c.JSON(200, repo.Build)
	}
}

func handleFetchRepoData(db *storm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		err := c.JSON(200, getRepo(db, id))
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
			c.Error(fmt.Errorf("release can't be empty"))
		}
		client := helm.NewClient(helm.Host(host))

		_, err := client.GetVersion()
		if err != nil {
			c.Error(err)
		}

		list, err := client.ReleaseHistory(release, helm.WithMaxHistory(10))
		if err != nil {
			c.Error(err)
		}

		var deployments []Deployment
		for _, v := range list.GetReleases() {
			d := Deployment{Version: strconv.Itoa(int(v.Version)), Name: v.Name, Status: v.GetInfo().GetStatus().Code.String(), Description: v.GetInfo().GetDescription()}
			deployments = append(deployments, d)
		}
		c.JSON(200, deployments)

		return nil
	}
}

func handleFetchBuild(db *storm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {

		id := c.Param("id")
		buildidStr := c.Param("buildid")
		buildid, err := strconv.Atoi(buildidStr)
		if err != nil {
			c.Error(err)
		}

		var repo Repo
		err = db.One("Id", id, &repo)
		if err != nil {
			log.Fatal(err)
		}
		for _, b := range repo.Build {
			if b.Number == buildid {
				return c.JSON(200, b)
			}
		}
		return c.JSON(500, nil)
	}
}

func handleFetchRepos(db *storm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		var dbrepos []Repo
		err := db.All(&dbrepos)
		if err != nil {
			log.Fatal(err)
		}

		return c.JSON(200, dbrepos)
	}
}

func handleStatus() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(200, "ok")
	}
}

func getRepo(db *storm.DB, name string) *Repo {

	updateFunc := func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(name))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	}
	err := db.Bolt.Update(updateFunc)
	if err != nil {
		fmt.Println("Bucket error:", err)
	}
	repo := &Repo{db: db}

	err = db.One("Id", name, repo)
	if err != nil {
		if err == storm.ErrNotFound {
			repo.Name = name
			repo.Id = name
			err = db.Save(repo)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
	return repo
}
