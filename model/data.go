package model

import (
	"time"
)

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
	BuildNumber int    `storm:"id"`
	Repo        string `storm:"index"`
	Name        string `json:"name"`
	Log         string `json:"build"`
	Status      string `json:"status"`
	ExitCode    int32  `json:"exitcode"`
}
type Service struct {
	Name string `json:"name"`
	Log  string `json:"build"`
}
type Repo struct {
	Id   string `json:"id" storm:"id"`
	Name string `json:"name"`
	Url  string `json:"url"`
}

// Deployment is a structure defining a Helm deployment
type Deployment struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Description string `json:"description"`
}
