package cmd

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/cloudwego/netpoll"
)

// 解析 TLV 数据
func parseTLV(reader netpoll.Reader) (byte, []byte, error) {
	// 读取长度字段(前两个字节表示数据的长度)
	lengthBuf, err := reader.Peek(2)
	if err != nil {
		return 0, nil, err
	}
	// 表示数据长度
	length := binary.BigEndian.Uint16(lengthBuf)
	// 读取完整数据
	data, err := reader.ReadBinary(int(length))
	if err != nil {
		return 0, nil, err
	}

	// 解析 Type 和 Payload
	msgType := data[0]
	payload := data[1:]

	return msgType, payload, nil
}

func handleConnection(ctx context.Context, conn netpoll.Connection) error {
	reader := conn.Reader()
	defer conn.Close()
	for {
		msgType, payload, err := parseTLV(reader)
		if err != nil {
			log.Println("Read error:", err)
			return err
		}
		fmt.Printf("Received: Type=%d, Payload=%s\n", msgType, string(payload))
		// 回写响应
		conn.Write([]byte("ACK"))
	}
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
			return ctx
		}),
		// 客户端连接断开
		netpoll.WithOnDisconnect(func(ctx context.Context, connection netpoll.Connection) {
			fmt.Printf("remote client close: %s\n", connection.RemoteAddr())
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
