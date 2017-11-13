package webstream

import (
	"encoding/json"
	"fmt"

	"gitlab.com/sorenmat/seneferu/model"
	"gitlab.com/sorenmat/seneferu/storage"
)

var broker *Broker

type BucketWriter struct {
	Service storage.Service
	RepoID  string
	Build   *model.Build
	Step    string
}

type Payload struct {
	Step string
	Line string
}

func (b *BucketWriter) Write(p []byte) (n int, err error) {
	if broker == nil {
		broker = NewServer() // make this pretty
	}
	//for _, s := range sockets.GetSockets() {
	// Write
	payload := Payload{Line: string(p), Step: b.Step}
	d, err := json.Marshal(&payload)
	if err != nil {
		fmt.Println("JSON: ", err)
	}
	broker.Notifier <- d
	/*err = websocket.Message.Send(s, string(d))
	if err != nil {
		sockets.Remove <- s
		fmt.Println("WS: Removed socket because of error: ", err)
	}*/

	//}

	build := b.Build
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
	err = b.Service.SaveBuild(b.Build)
	if err != nil {
		return -1, err
	}
	return len(p), nil
}
