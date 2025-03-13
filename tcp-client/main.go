package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

func sendTLV(conn net.Conn, msgType byte, payload string) error {
	data := []byte(payload)
	length := uint16(len(data) + 1)

	// 组装数据包
	packet := make([]byte, 2+1+len(data))
	binary.BigEndian.PutUint16(packet[:2], length)
	packet[2] = msgType
	copy(packet[3:], data)

	// 发送数据
	_, err := conn.Write(packet)
	return err
}

func main() {
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		fmt.Println("Connection error:", err)
		return
	}
	defer conn.Close()

	// 发送消息
	err = sendTLV(conn, 1, "Hello, Server!")
	if err != nil {
		fmt.Println("Send error:", err)
		return
	}

	// 接收响应
	buf := make([]byte, 1024)
	n, _ := conn.Read(buf)
	fmt.Println("Response:", string(buf[:n]))
}
