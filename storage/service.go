package storage

import (
	"time"

	"gitlab.com/sorenmat/seneferu/model"
)

type Service interface {
	All() ([]*model.Repo, error)
	LoadByOrgAndName(string, string) (*model.Repo, error)
	LoadBuilds(string, string) ([]*model.Build, error)
	LoadBuild(string, string, int) (*model.Build, error)
	LoadStep(string, string, int, string) (*model.Step, error)
	LoadStepInfos(org string, name string, build int) ([]*model.StepInfo, error)
	LoadStepInfo(org string, name string, stepname string, build int) (*model.StepInfo, error)
	SaveRepo(*model.Repo) error
	SaveBuild(*model.Build) error
	SaveStep(*model.Step) error
	GetNextBuildNumber(string, string) (int, error)
	Close()
}

type Build struct {
	Number     int        `json:"number" storm:"id"`
	Committers []string   `json:"committers"`
	Timestamp  time.Time  `json:"timestamp"`
	Success    bool       `json:"success"`
	Status     string     `json:"status"`
	Steps      []*Step    `json:"steps"`
	Services   []*Service `json:"services"`
	Repo       string     `json:"repository"`
	Owner      string     `json:"owner"`
	Commit     string     `json:"commit"`
	Coverage   string     `json:"coverage"`
	Took       string     `json:"took"`
}

type Step struct {
	Name     string `json:"name"`
	Log      string `json:"build"`
	Status   string `json:"status"`
	ExitCode int32  `json:"exitcode"`
}

type ServiceLog struct {
	Name string `json:"name"`
	Log  string `json:"build"`
}
type Repo struct {
	Id    string   `json:"id" storm:"id"`
	Name  string   `json:"name"`
	Url   string   `json:"url"`
	Build []*Build `json:"builds"`
}

// Deployment is a structure defining a Helm deployment
type Deployment struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Description string `json:"description"`
}
