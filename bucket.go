package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
)

type BucketWriter struct {
	repo  *Repo
	build *Build
	Step  string
}
type Payload struct {
	Step string
	Line string
}

func (b *BucketWriter) Write(p []byte) (n int, err error) {

	for _, s := range sockets {
		// Write
		payload := Payload{Line: string(p), Step: b.Step}
		d, err := json.Marshal(&payload)
		if err != nil {
			fmt.Println("JSON: ", err)
		}
		err = websocket.Message.Send(s, string(d))
		if err != nil {
			fmt.Println("WS: ", err)
		}

	}

	build := b.build
	for _, s := range build.Steps {
		if s.Name == b.Step {
			s.Log = s.Log + string(p)
		}
	}
	for _, s := range build.Services {
		if s.Name == b.Step {
			s.Log = s.Log + string(p)
		}
	}

	err = b.repo.Save(b.build)
	if err != nil {
		return -1, err
	}
	return len(p), nil
}
