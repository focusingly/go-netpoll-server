package client

import (
	"context"
	"fmt"
	"log"
	"msg-protocol/codec"
	"msg-protocol/protocol"
	"os"
	"sync/atomic"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/cloudwego/netpoll"
)

type (
	Client struct {
		ctx        context.Context
		dialer     netpoll.Dialer
		conn       netpoll.Connection
		msgHandler *msgHandler
		// 一个带 1 缓存长度的通道, 用于表示一次 请求/响应 同步
		syncChan                chan struct{}
		onMessageReceiveHandler func(msg *protocol.ProtocolMessage)
		nextSeqID               uint64
		waitID                  uint64
	}

	msgHandler struct {
		codec  *codec.Codec
		client *Client
	}
)

// SendMessage 发送消息给服务端
//
// Parameters:
//   - ctx 包含一个超时设定的上下文包装, 用于处理等待上一条消息同步的超时处理, 在到达超时阈值后, 会放弃等待上一次发送消息的同步处理, 直接发送下一条消息
//   - msg 要发送的消息(无论是否手动设置了 RequestID, 都会由内部的 ID 生成器生成的标准化 ID 进行覆盖)
//
// Examples:
//   - timeoutCtx,cancel  := context.WithTimeout(context.Background(), time.Second*5)
//     defer cancel()
//     c.SendMessage(timeoutCtx, &protocol.ProtoMessage{ // ...})
func (c *Client) SendMessage(ctx context.Context, msg *protocol.ProtocolMessage) (err error) {
	writer := c.conn.Writer()

	// 创建下一条消息的 ID
	seqID := c.nextID()
	msg.RequestID = seqID
	bf, err := c.msgHandler.codec.Encode(msg)
	if err != nil {
		return err
	}

	select {
	// 等待上一已发送数据的回复消息同步
	case c.syncChan <- struct{}{}:
	// 允许的最大超时控制处理
	case <-ctx.Done():
		// 说明上一条发送出去的消息没有收到, 已经超时了
		// <-c.sendToken
		c.expireWaitingRequestID()
	}

	_, err = writer.WriteBinary(bf)
	if err != nil {
		return err
	}

	c.setMessageWaitID(seqID)
	if err = writer.Flush(); err != nil {
		c.expireWaitingRequestID()
	}

	return
}

func (m *msgHandler) handleRawRequestMessage(ctx context.Context, connection netpoll.Connection) error {
	reader := connection.Reader()
	defer reader.Release()
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("handle receive message error: ", err)
		}
	}()

	receivePackets, err := reader.ReadBinary(reader.Len())
	if err != nil {
		// cleanup
		m.codec.Reset()
		switch err {
		case netpoll.ErrConnClosed:
			m.client.conn.Close()
			os.Exit(-1)
		case netpoll.ErrEOF:
			fallthrough
		default:
			return err
		}
	}
	if err := m.codec.WriteBytes(receivePackets); err != nil {
		return err
	}

	msg, done, err := m.codec.Decode()

	// 出现了解码错误
	if err != nil {
		return err
	}

	// 数据还未完全接收完毕
	if !done {
		return nil
	}
	if m.client.waitID == msg.RequestID {
		// 消耗通道, 以允许下一条消息的发送
		<-m.client.syncChan
	} else {
		return fmt.Errorf("unknown receive message: %d; %v", msg.RequestID, msg)
	}
	if m.client.onMessageReceiveHandler != nil {
		m.client.onMessageReceiveHandler(msg)
	}

	return nil
}

func NewClient(ctx context.Context, addr string) (*Client, error) {
	dialer := netpoll.NewDialer()
	conn, retryConnErr := createConn(dialer, addr, 1000, time.Second*10)
	if retryConnErr != nil {
		log.Fatal(retryConnErr)
	}

	c := &Client{
		ctx:      ctx,
		dialer:   dialer,
		conn:     conn,
		syncChan: make(chan struct{}, 1),
		onMessageReceiveHandler: func(msg *protocol.ProtocolMessage) {
		},
		nextSeqID: 0,
	}
	c.msgHandler = &msgHandler{
		codec:  &codec.Codec{},
		client: c,
	}

	return c, nil
}

func createConn(dialer netpoll.Dialer, addr string, maxRetry int, delay time.Duration) (netpoll.Connection, error) {
	var conn netpoll.Connection
	var connErr error
	for {
		if conn, connErr = dialer.DialConnection("tcp", addr, time.Second*10); connErr != nil {
			maxRetry++
			time.Sleep(delay)
		} else {
			break
		}
	}

	return conn, connErr
}

func (c *Client) Spin(onEstablish ...func()) error {
	fnLen := len(onEstablish)
	switch fnLen {
	case 0:
	case 1:
	default:
		panic(fmt.Errorf("expect 0 or 1 established Listen func, but got: %d", fnLen))
	}
	err := c.conn.SetOnRequest(c.msgHandler.handleRawRequestMessage)
	if err != nil {
		return err
	}
	if fnLen == 1 {
		onEstablish[0]()
	}

	// 连接检查
	gopool.Go(func() {
		ticker := time.NewTicker(time.Second * 10)
		for {
			select {
			// 断连检测
			case <-ticker.C:
				if !c.conn.IsActive() {
					c.conn.Close()
					ticker.Stop()
					os.Exit(-1)
				}
			// 程序被用户主动退出
			case <-c.ctx.Done():
				ticker.Stop()
				return
			}
		}
	})

	// 阻塞等待外部的退出
	<-c.ctx.Done()
	return nil
}

// 利用原子操作获取下一组数据的编号信息
func (c *Client) nextID() uint64 {
	return atomic.AddUint64(&c.nextSeqID, 1)
}

// 获取当前的请求 ID 状态值
func (c *Client) CurrentSeqID() uint64 {
	return atomic.LoadUint64(&c.nextSeqID)
}

func (c *Client) expireWaitingRequestID() {
	atomic.StoreUint64(&c.waitID, 0)
}

func (c *Client) setMessageWaitID(seqID uint64) {
	atomic.StoreUint64(&c.waitID, seqID)
}

func (c *Client) SetOnMessageReceive(dispatcher func(msg *protocol.ProtocolMessage)) {
	c.onMessageReceiveHandler = dispatcher
}
