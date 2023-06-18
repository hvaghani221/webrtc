package main

import (
	"time"

	"github.com/hvaghani221/webrtc/firebase"
)

func main() {
	client, err := firebase.Init()
	if err != nil {
		panic(err)
	}
	time.Sleep(10 * time.Second)
	defer client.Close()
}
