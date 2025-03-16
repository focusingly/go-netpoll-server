package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"msg-protocol/protocol"
	"sync"
	"tcp-client/client"
	"time"

	"github.com/google/uuid"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(1)
	c, err := client.NewClient(context.TODO(), "localhost:9000")
	if err != nil {
		log.Fatal(err)
	}

	c.SetOnMessageReceive(func(msg *protocol.ProtocolMessage) {
		s, _ := json.MarshalIndent(msg, "", "  ")
		fmt.Printf("client receive server message: %s\n\n", s)
	})

	go func() {
		if err := c.Serve(func() {
			wg.Done()
		}); err != nil {
			log.Fatal(err)
		}
	}()
	wg.Wait()

	for {
		c.SendMessage(context.TODO(), &protocol.ProtocolMessage{
			Version:     1,
			Flags:       1,
			MessageType: 1,
			BodyPayload: ([]byte)(uuid.NewString()),
		})
		time.Sleep(time.Second * 1)
	}
}
