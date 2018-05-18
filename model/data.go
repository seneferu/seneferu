package model

import (
	"time"
)

// Build is the structure describing a build in Seneferu
type Build struct {
	Org  string `json:"organisation"`
	Name string `json:"name"`

	Number     int        `json:"number"`
	Committers []string   `json:"committers"`
	Timestamp  time.Time  `json:"timestamp"`
	Success    bool       `json:"success"`
	Status     string     `json:"status"`
	Steps      []*Step    `json:"steps"`
	Services   []*Service `json:"services"`
	Commit     string     `json:"commit"`
	Coverage   string     `json:"coverage"`
	Duration   string     `json:"duration"`

	Ref       string
	TreesURL  string
	StatusURL string
}

// StepInfo contains information about each build step
type StepInfo struct {
	BuildNumber int    `json:"buildnumber"`
	Org         string `json:"org"`
	Reponame    string `json:"reponame"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	ExitCode    int32  `json:"exitcode"`
}

// Step contains step information and the log entry
type Step struct {
	StepInfo
	Log string `json:"log"`
}

// Service is used to describe a service which is a thing builds can depend on
// typically this could be postgres database or something like that
type Service struct {
	Name string `json:"name"`
	Log  string `json:"log"`
}

// Repo basic information about a repository
type Repo struct {
	Org  string `json:"org"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Deployment is a structure defining a Helm deployment
type Deployment struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Description string `json:"description"`
}
