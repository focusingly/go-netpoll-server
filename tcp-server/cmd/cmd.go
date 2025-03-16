package cmd

import (
	"context"
	"fmt"
	"log"
	"msg-protocol/codec"
	"sync"

	"github.com/cloudwego/netpoll"
)

var (
	mutex       sync.RWMutex
	codecBufMap = make(map[netpoll.Connection]*codec.Codec)
)

func handleConnection(ctx context.Context, conn netpoll.Connection) error {
	reader := conn.Reader()
	writer := conn.Writer()
	defer reader.Release()
	mutex.RLock()
	defer mutex.RUnlock()

	c, ok := codecBufMap[conn]
	if !ok {
		return fmt.Errorf("not codec setup")
	}

	bf, err := reader.ReadBinary(reader.Len())
	if err != nil {
		c.Reset()
		return err
	}

	if err := c.WriteBytes(bf); err != nil {
		return err
	}

	msg, done, err := c.Decode()
	if err != nil {
		return err
	}
	// 数据还未读取完成
	if !done {
		return nil
	}

	// 数据已经处理完成
	fmt.Println("server receive message", msg)
	out, _ := c.Encode(msg)
	if _, err := writer.WriteBinary(out); err != nil {
		return err
	}
	if writeErr := writer.Flush(); writeErr != nil {
		return writeErr
	}

	return nil
}

func startServer() {
	// 创建监听器
	listener, err := netpoll.CreateListener("tcp", ":9000")
	if err != nil {
		log.Fatal(err)
	}

	// 事件循环
	eventLoop, err := netpoll.NewEventLoop(
		// 当客户端建立连接成功
		handleConnection,
		// 当客户端建立连接
		netpoll.WithOnConnect(func(ctx context.Context, connection netpoll.Connection) context.Context {
			fmt.Println("client connect request: ", connection.RemoteAddr())
			mutex.Lock()
			defer mutex.Unlock()
			codecBufMap[connection] = &codec.Codec{}
			return ctx
		}),
		// 客户端连接断开
		netpoll.WithOnDisconnect(func(ctx context.Context, connection netpoll.Connection) {
			fmt.Printf("remote client close: %s\n", connection.RemoteAddr())
			mutex.Lock()
			defer mutex.Unlock()
			// 清理连接资源
			delete(codecBufMap, connection)
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 启动服务器
	if err := eventLoop.Serve(listener); err != nil {
		log.Fatal(err)
	}
}

func Spin() {
	startServer()
}
