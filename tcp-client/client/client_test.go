package client_test

import (
	"context"
	"fmt"
	"log"
	"msg-protocol/protocol"
	"sync"
	"sync/atomic"
	"tcp-client/client"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestClientMessageSendAndReceive(t *testing.T) {
	var wg sync.WaitGroup

	cl, err := client.NewClient(context.TODO(), "localhost:9000")
	if err != nil {
		log.Fatal(err)
	}

	var count uint64
	cl.SetOnMessageReceive(func(msg *protocol.ProtocolMessage) {
		wg.Done()
		// s, _ := json.MarshalIndent(msg, "", "  ")
		atomic.AddUint64(&count, 1)
		// fmt.Printf("client receive server message: %s\n\n", s)
	})

	wg.Add(1)
	// 异步启动
	go func() {
		if err := cl.Spin(func() {
			wg.Done()
		}); err != nil {
			log.Fatal(err)
		}
	}()
	wg.Wait()

	wc := 0
	start := time.Now()

	for wc < 1_0000 {
		wg.Add(1)
		err := cl.SendMessage(context.TODO(), &protocol.ProtocolMessage{
			Version:     1,
			Flags:       1,
			MessageType: 1,
			BodyPayload: ([]byte)(uuid.NewString()),
		})
		if err != nil {
			log.Fatalf("could not send message %v", err)
		}
		wc++
	}
	wg.Wait()
	cost := time.Since(start).Milliseconds()
	fmt.Printf("client incr ID is %d, count: %d, cost time: %d ms\n", cl.CurrentSeqID(), atomic.LoadUint64(&count), cost)
	assert.Equal(t, cl.CurrentSeqID(), atomic.LoadUint64(&count))
}
