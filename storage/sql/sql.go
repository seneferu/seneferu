package sql

import (
	"fmt"
	"log"

	"database/sql"
	"github.com/DavidHuie/gomigrate"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"gitlab.com/sorenmat/seneferu/model"
	"gitlab.com/sorenmat/seneferu/storage"
	"os"
	"path/filepath"
	"strings"
)

// BoltDB is the service implementation for BoltDB
type SQLDB struct {
	db *sql.DB
}

// New Create a new BoltDB compatiable service
func New() (storage.Service, error) {
	host := "localhost"
	if os.Getenv("POSTGRES_HOST") != "" {
		host = os.Getenv("POSTGRES_HOST")
	}

	db, err := sql.Open("postgres", "host="+host+" user=postgres dbname=seneferu sslmode=disable")
	if err != nil {
		return nil, errors.Wrap(err, "unable to load db")
	}

	b := SQLDB{db: db}
	b.syncSchema()
	return &b, nil
}

func guessMigrationPath() string {
	path, err := filepath.Abs("./migrations")
	if err != nil {
		log.Println("unable to find migration folder", path)
	}
	_, err = os.Stat(path)
	if err != nil {
		log.Println("unable to find migration folder")
		path, _ = filepath.Abs("../migrations")
	}
	_, err = os.Stat(path)
	if err != nil {
		log.Printf("unable to find migration folder last attempt was %v", path)
		path, _ = filepath.Abs("../../migrations")
	}
	_, err = os.Stat(path)
	if err != nil {
		log.Fatalf("unable to find migration path, last attempt was %v", path)
	}

	return path
}

func (r *SQLDB) syncSchema() {
	// Keep schemas in sync
	path := guessMigrationPath()
	migrator, err := gomigrate.NewMigratorWithLogger(r.db, gomigrate.Postgres{}, path, log.New(os.Stdout, "", log.LstdFlags))
	if err != nil {
		log.Fatal("NewMigratorWithLogger ", err)
	}

	err = migrator.Migrate()
	if err != nil {
		log.Fatal("Migrate: ", err)
	}

}

// All returns all the ids of all the repos
func (b *SQLDB) All() ([]*model.Repo, error) {
	var result []*model.Repo

	rows, err := b.db.Query("SELECT id, name, url FROM repositories")
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var name string
		var url string
		err = rows.Scan(&id, &name, &url)
		if err != nil {
			return result, err
		}
		result = append(result, &model.Repo{Id: id, Name: name, Url: url})
	}
	return result, nil
}

// LoadById loads a repo from the database given an id
func (b *SQLDB) LoadById(id string) (*model.Repo, error) {
	var repo model.Repo

	rows, err := b.db.Query("SELECT id,url,name FROM repositories WHERE ID=$1", id)
	if err != nil {
		return &repo, err
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		err = rows.Scan(&repo.Id, &repo.Url, &repo.Name)
		if err != nil {
			return nil, err
		}
		found = true
	}
	if !found {
		return nil, fmt.Errorf("no repository found with name %v", id)
	}
	return &repo, nil
}

// LoadBuild loads a given build from the repository
func (b *SQLDB) LoadBuild(repo string, build int) (*model.Build, error) {
	bb := &model.Build{}

	rows, err := b.db.Query("SELECT repo, number, comitters, created, success, status, owner, commit, coverage, duration FROM builds WHERE REPO=$1 AND NUMBER=$2", repo, build)
	if err != nil {
		return bb, err
	}

	defer rows.Close()

	for rows.Next() {
		var commiters string
		err = rows.Scan(&bb.Repo, &bb.Number, &commiters, &bb.Timestamp, &bb.Success, &bb.Status, &bb.Owner, &bb.Commit, &bb.Coverage, &bb.Took)
		bb.Committers = strings.Split(commiters, ",")
		if err != nil {
			return nil, err
		}
	}
	return bb, nil

}

// LoadBuilds loads a all builds for a given repository
func (b *SQLDB) LoadBuilds(repo string) ([]*model.Build, error) {
	var bb []*model.Build

	rows, err := b.db.Query("SELECT repo,number,comitters,status,owner,commit,coverage,duration FROM builds WHERE REPO=$1", repo)
	if err != nil {
		return bb, err
	}
	defer rows.Close()

	for rows.Next() {
		b := &model.Build{}
		var c string
		err = rows.Scan(&b.Repo, &b.Number, &c, &b.Status, &b.Owner, &b.Commit, &b.Coverage, &b.Took)
		b.Committers = strings.Split(c, ",")
		bb = append(bb, b)
		if err != nil {
			return nil, err
		}
	}
	return bb, nil
}

// LoadStep loads a given step in a repo based on the repo name, build id and step name
func (b *SQLDB) LoadStep(reponame string, build int, stepname string) (*model.Step, error) {
	result := &model.Step{}
	rows, err := b.db.Query("SELECT buildnumber,repo,name,log,status,exitcode FROM steps WHERE REPO=$1 AND buildnumber=$2 AND NAME=$3", reponame, build, stepname)
	if err != nil {
		return result, err
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&result.BuildNumber, &result.Repo, &result.Name, &result.Log, &result.Status, &result.ExitCode)
		if err != nil {
			return nil, err
		}
	}
	return result, nil

}

// SaveRepo saves the repository in the database
func (b *SQLDB) SaveRepo(repo *model.Repo) error {
	stmt, err := b.db.Prepare("INSERT INTO repositories(id, name, url) VALUES($1, $2, $3)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(repo.Id, repo.Name, repo.Url)
	if err != nil {
		return err
	}
	return nil
}

// SaveBuild saves a build in the database with the repo id as the key
func (b *SQLDB) SaveBuild(build *model.Build) error {
	if build.Repo == "" {
		return fmt.Errorf("repo is required for the build struct")
	}
	stmt, err := b.db.Prepare("INSERT INTO builds(repo,number,comitters,status,success,owner,commit,coverage,	duration) " +
		"VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)" +
		"ON CONFLICT (repo, number) DO UPDATE SET " +
		"comitters=$3,status=$4,success=$5,owner=$6,commit=$7,coverage=$8, duration=$9 WHERE builds.repo=$1 AND builds.number=$2")
	if err != nil {
		log.Println(err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(build.Repo, build.Number, fmt.Sprintf("%v", build.Committers), build.Status, build.Success, build.Owner, build.Commit, build.Coverage, build.Took)
	if err != nil {
		return err
	}
	return nil
}

// SaveStep saves a step in the database using the repo-buildnumber-stepname as key
func (b *SQLDB) SaveStep(step *model.Step) error {
	if step.Repo == "" {
		return fmt.Errorf("repo is required for the build struct")
	}
	if step.BuildNumber <= 0 {
		return fmt.Errorf("a build number larger then 0 is required")
	}
	if step.BuildNumber <= 0 {
		return fmt.Errorf("the step must have a build number larger then 0")
	}

	stmt, err := b.db.Prepare("INSERT INTO steps(buildnumber, repo,name,log,status,exitcode) " +
		"VALUES($1, $2, $3, $4, $5, $6) ON CONFLICT (buildnumber, repo, name) DO UPDATE SET " +
		"buildnumber=$1, repo=$2,name=$3,log=$4,status=$5,exitcode=$6 WHERE steps.buildnumber=$1 AND steps.repo=$2 AND steps.name=$3")

	if err != nil {
		log.Println(err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(step.BuildNumber, step.Repo, step.Name, step.Log, step.Status, step.ExitCode)
	if err != nil {
		return err
	}
	return nil
}

// GetNextBuildNumber returns the next available build number, this is currently globally unique
func (b *SQLDB) GetNextBuildNumber() (int, error) {
	buildNumber := 1
	return buildNumber, nil

}

// Close the connection to the database
func (b SQLDB) Close() {
	b.db.Close()
}
