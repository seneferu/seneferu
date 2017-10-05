package main

import (
	"github.com/asdine/storm"
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
}

type Step struct {
	Name   string `json:"name"`
	Log    string `json:"build"`
	Status string `json:"status"`
}
type Service struct {
	Name string `json:"name"`
	Log  string `json:"build"`
}
type Repo struct {
	db    *storm.DB
	Id    string   `json:"id" storm:"id"`
	Name  string   `json:"name"`
	Url   string   `json:"url"`
	Build []*Build `json:"builds"`
}
