package sql

import (
	"fmt"
	"log"

	"database/sql"
	"os"
	"path/filepath"
	"strings"

	"github.com/DavidHuie/gomigrate"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"gitlab.com/sorenmat/seneferu/model"
	"gitlab.com/sorenmat/seneferu/storage"
)

// Postgresql is the service implementation for Postgresql
type SQLDB struct {
	db *sql.DB
}

// New Create a new Postgresql compatible service
func New() (storage.Service, error) {
	host := "localhost"
	if os.Getenv("POSTGRES_HOST") != "" {
		host = os.Getenv("POSTGRES_HOST")
	}
	user := "postgres"
	if os.Getenv("POSTGRES_USER") != "" {
		user = os.Getenv("POSTGRES_USER")
	}
	password := "postgres"
	if os.Getenv("POSTGRES_PASSWD") != "" {
		password = os.Getenv("POSTGRES_PASSWD")
	}

	db, err := sql.Open("postgres", "host="+host+" user="+user+" password="+password+" dbname=seneferu sslmode=disable")
	if err != nil {
		return nil, errors.Wrap(err, "unable to open connection to database")
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
func (r *SQLDB) All() ([]*model.Repo, error) {
	result := make([]*model.Repo, 0)

	rows, err := r.db.Query("SELECT org, name, url FROM repositories")
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var org string
		var name string
		var url string
		err = rows.Scan(&org, &name, &url)
		if err != nil {
			return result, err
		}
		result = append(result, &model.Repo{Org: org, Name: name, URL: url})
	}
	return result, nil
}

// LoadByOrgAndName loads a repo from the database given an org and name
func (r *SQLDB) LoadByOrgAndName(org, name string) (*model.Repo, error) {
	var repo model.Repo

	rows, err := r.db.Query("SELECT org,url,name,org FROM repositories WHERE ORG=$1 AND NAME=$2", org, name)
	if err != nil {
		return &repo, err
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		err = rows.Scan(&repo.Org, &repo.URL, &repo.Name, &repo.Org)
		if err != nil {
			return nil, err
		}
		found = true
	}
	if !found {
		return nil, fmt.Errorf("no repository found with name %v/%v", org, name)
	}
	return &repo, nil
}

// LoadBuild loads a given build from the repository
func (r *SQLDB) LoadBuild(org, name string, build int) (*model.Build, error) {
	bb := &model.Build{}

	rows, err := r.db.Query("SELECT org, name, number, comitters, created, success, status, commit, coverage, duration FROM builds WHERE ORG=$1 AND NAME=$2 AND NUMBER=$3", org, name, build)
	if err != nil {
		return bb, err
	}

	defer rows.Close()

	for rows.Next() {
		var commiters string
		err = rows.Scan(&bb.Org, &bb.Name, &bb.Number, &commiters, &bb.Timestamp, &bb.Success, &bb.Status, &bb.Commit, &bb.Coverage, &bb.Duration)
		bb.Committers = strings.Split(commiters, ",")
		if err != nil {
			return nil, err
		}
	}
	return bb, nil

}

// LoadBuilds loads a all builds for a given repository
func (r *SQLDB) LoadBuilds(org, name string) ([]*model.Build, error) {
	bb := make([]*model.Build, 0)

	rows, err := r.db.Query("SELECT org,name,number,comitters,status,commit,coverage,duration FROM builds WHERE ORG=$1 AND NAME=$2", org, name)
	if err != nil {
		return bb, err
	}
	defer rows.Close()

	for rows.Next() {
		b := &model.Build{}
		var c string
		err = rows.Scan(&b.Org, &b.Name, &b.Number, &c, &b.Status, &b.Commit, &b.Coverage, &b.Duration)
		b.Committers = strings.Split(c, ",")
		bb = append(bb, b)
		if err != nil {
			return nil, err
		}
	}
	return bb, nil
}

// LoadStep loads a given step in a repo based on the repo name, build id and step name
func (r *SQLDB) LoadStep(org, reponame string, build int, stepname string) (*model.Step, error) {
	result := &model.Step{}
	rows, err := r.db.Query("SELECT org, reponame, buildnumber,name,log,status,exitcode FROM steps WHERE ORG=$1 AND REPONAME=$2 AND buildnumber=$3 AND NAME=$4", org, reponame, build, stepname)
	if err != nil {
		return result, err
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&result.Org, &result.Reponame, &result.BuildNumber, &result.Name, &result.Log, &result.Status, &result.ExitCode)
		if err != nil {
			return nil, err
		}
	}
	return result, nil

}

// LoadStepInfos loads all step informations in a repo based on the repo name, build id and step name
func (r *SQLDB) LoadStepInfos(org string, name string, build int) ([]*model.StepInfo, error) {
	result := make([]*model.StepInfo, 0)
	rows, err := r.db.Query("SELECT org, reponame, buildnumber,name,status,exitcode FROM steps WHERE ORG=$1 AND REPONAME=$2 AND buildnumber=$3", org, name, build)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stepinfo model.StepInfo
		err = rows.Scan(&stepinfo.Org, &stepinfo.Reponame, &stepinfo.BuildNumber, &stepinfo.Name, &stepinfo.Status, &stepinfo.ExitCode)
		if err != nil {
			return nil, err
		}
		result = append(result, &stepinfo)
	}
	return result, nil

}

// LoadStepInfo loads a given step information in a repo based on the repo name, build id and step name
func (r *SQLDB) LoadStepInfo(org string, name string, stepname string, build int) (*model.StepInfo, error) {
	rows, err := r.db.Query("SELECT org, reponame, buildnumber,name,status,exitcode FROM steps WHERE ORG=$1 AND REPONAME=$2 AND NAME=$3 AND buildnumber=$4", org, name, stepname, build)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var stepinfo model.StepInfo
		err = rows.Scan(&stepinfo.Org, &stepinfo.Reponame, &stepinfo.BuildNumber, &stepinfo.Name, &stepinfo.Status, &stepinfo.ExitCode)
		if err != nil {
			return nil, err
		}
		return &stepinfo, nil
	}
	return nil, fmt.Errorf("couldn't find step %v in %v/%v with build number %v ", stepname, org, name, build)

}

// SaveRepo saves the repository in the database
func (r *SQLDB) SaveRepo(repo *model.Repo) error {
	if repo.Org == "" {
		return fmt.Errorf("org is required for a repository")
	}
	if repo.Name == "" {
		return fmt.Errorf("name is required for a repository")
	}

	stmt, err := r.db.Prepare("INSERT INTO repositories(org, name, url) VALUES($1, $2, $3)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(repo.Org, repo.Name, repo.URL)
	if err != nil {
		return err
	}
	return nil
}

// SaveBuild saves a build in the database with the repo id as the key
func (r *SQLDB) SaveBuild(build *model.Build) error {
	if build.Org == "" {
		return fmt.Errorf("org is required for the build struct")
	}
	if build.Name == "" {
		return fmt.Errorf("name is required for the build struct")
	}

	stmt, err := r.db.Prepare("INSERT INTO builds(org,name,number,comitters,status,success,commit,coverage,duration) " +
		"VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)" +
		"ON CONFLICT (org,name, number) DO UPDATE SET " +
		"comitters=$4, status=$5, success=$6, commit=$7, coverage=$8, duration=$9 WHERE builds.org=$1 AND builds.name=$2 AND builds.number=$3")
	if err != nil {
		log.Println(err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(build.Org, build.Name, build.Number, fmt.Sprintf("%v", build.Committers), build.Status, build.Success, build.Commit, build.Coverage, build.Duration)
	if err != nil {
		return err
	}
	return nil
}

// SaveStep saves a step in the database using the repo-buildnumber-stepname as key
func (r *SQLDB) SaveStep(step *model.Step) error {
	if step.Org == "" {
		return fmt.Errorf("org is required for the step struct")
	}
	if step.Reponame == "" {
		return fmt.Errorf("repository name is required for the step struct")
	}

	if step.BuildNumber <= 0 {
		return fmt.Errorf("a build number larger then 0 is required")
	}
	if step.BuildNumber <= 0 {
		return fmt.Errorf("the step must have a build number larger then 0")
	}

	stmt, err := r.db.Prepare("INSERT INTO steps(buildnumber, reponame, name, log, status, exitcode,org) " +
		"VALUES($1, $2, $3, $4, $5, $6,$7) ON CONFLICT (buildnumber, reponame, name,org) DO UPDATE SET " +
		"buildnumber=$1, reponame=$2,name=$3,log=$4,status=$5,exitcode=$6,org=$7 WHERE steps.buildnumber=$1 AND steps.reponame=$2 AND steps.name=$3 AND steps.org=$4")

	if err != nil {
		log.Println(err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(step.BuildNumber, step.Reponame, step.Name, step.Log, step.Status, step.ExitCode, step.Org)
	if err != nil {
		return err
	}
	return nil
}

// GetNextBuildNumber returns the next available build number, this is currently globally unique
func (r *SQLDB) GetNextBuildNumber() (int, error) {
	buildNumber := 1
	return buildNumber, nil

}

// Close the connection to the database
func (r SQLDB) Close() {
	r.db.Close()
}
