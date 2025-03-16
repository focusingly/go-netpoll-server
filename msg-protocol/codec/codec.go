package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"msg-protocol/protocol"
)

type (
	Codec struct {
		buffer bytes.Buffer
	}
)

func (c *Codec) WriteBytes(buf []byte) error {
	if _, err := c.buffer.Write(buf); err != nil {
		c.buffer.Reset()
		return err
	}

	return nil
}

func (c *Codec) Encode(msg *protocol.ProtocolMessage) (encodes []byte, err error) {
	if msg == nil {
		return nil, fmt.Errorf("msg should not be empty")
	}
	var bf bytes.Buffer
	// 写入头部
	if _, err = bf.Write(binary.BigEndian.AppendUint32([]byte{}, protocol.MessageMagic)); err != nil {
		return
	}
	// 写入空白填充
	if err = bf.WriteByte(msg.OffsetPadding); err != nil {
		return
	}
	// 写入版本号
	if err = bf.WriteByte(msg.Version); err != nil {
		return
	}
	// 写入标志位
	if err = bf.WriteByte(msg.Flags); err != nil {
		return
	}
	// 写入消息类型
	if err = bf.WriteByte(msg.MessageType); err != nil {
		return
	}
	// 写入消息 ID
	if _, err = bf.Write(binary.BigEndian.AppendUint64([]byte{}, msg.RequestID)); err != nil {
		return
	}
	// 写入校验和
	if _, err = bf.Write(binary.BigEndian.AppendUint32([]byte{}, crc32.ChecksumIEEE(msg.BodyPayload))); err != nil {
		return
	}
	// 写入正文长度
	if _, err = bf.Write(binary.BigEndian.AppendUint32([]byte{}, uint32(len(msg.BodyPayload)))); err != nil {
		return
	}
	// 写入附带的正文数据
	if _, err = bf.Write(msg.BodyPayload); err != nil {
		return
	}

	return bf.Bytes(), nil
}

func (c *Codec) Decode() (msg *protocol.ProtocolMessage, processDone bool, err error) {
	// 小于最少要求的字节数
	if c.buffer.Len() < protocol.FixedHeaderByteLen {
		return
	}

	// 读取固定的头部消息
	fixedHeaders := make([]byte, protocol.FixedHeaderByteLen)
	c.buffer.Read(fixedHeaders)

	// 判断消息头部是否匹配
	if binary.BigEndian.Uint32(fixedHeaders[:protocol.MagicHeaderBytesLen]) != protocol.MessageMagic {
		err = fmt.Errorf("expected magic header values: %v, but got: %v", protocol.MessageMagic, fixedHeaders[:4])
		c.Reset()
		return
	}

	// 读取变长部分
	bodyLen := binary.BigEndian.Uint32(fixedHeaders[protocol.MagicHeaderBytesLen+16 : protocol.MagicHeaderBytesLen+20])

	// 无负载消息, 直接返回即可
	if bodyLen == 0 {
		msg = &protocol.ProtocolMessage{
			MagicNumbers:  protocol.MessageMagic,
			OffsetPadding: fixedHeaders[protocol.MagicHeaderBytesLen],
			Version:       fixedHeaders[protocol.MagicHeaderBytesLen+1],
			Flags:         fixedHeaders[protocol.MagicHeaderBytesLen+2],
			MessageType:   fixedHeaders[protocol.MagicHeaderBytesLen+3],
			RequestID:     binary.BigEndian.Uint64(fixedHeaders[protocol.MagicHeaderBytesLen+4 : protocol.MagicHeaderBytesLen+12]),
			BodyPayload:   []byte{},
		}
		processDone = true
		return
	}

	// 存在负载消息但容量不足的情况将数据放回给缓冲空间头部
	if bodyLen > uint32(c.buffer.Len()) {
		c.buffer.Grow(protocol.FixedHeaderByteLen)
		tmpBuf := c.buffer.Bytes()
		c.buffer.Reset()
		// 放回数据, 供下一次请求使用
		if _, err = c.buffer.Write(fixedHeaders); err != nil {
			c.buffer.Reset()
			return
		}
		if _, err = c.buffer.Write(tmpBuf); err != nil {
			c.buffer.Reset()
			return
		}
	}

	payload := make([]byte, bodyLen)
	if _, err = c.buffer.Read(payload); err != nil {
		c.buffer.Reset()
		return
	}

	// 负载数据已经完整
	// 判断校验和
	respCrc32Val := binary.BigEndian.Uint32(fixedHeaders[protocol.MagicHeaderBytesLen+12 : protocol.MagicHeaderBytesLen+16])
	calcCrc32Val := crc32.ChecksumIEEE(payload)

	if respCrc32Val != calcCrc32Val {
		err = fmt.Errorf("expect crc32 value is: %d, but got: %d", respCrc32Val, calcCrc32Val)
		return
	}

	msg = &protocol.ProtocolMessage{
		MagicNumbers:  protocol.MessageMagic,
		OffsetPadding: fixedHeaders[protocol.MagicHeaderBytesLen],
		Version:       fixedHeaders[protocol.MagicHeaderBytesLen+1],
		Flags:         fixedHeaders[protocol.MagicHeaderBytesLen+2],
		MessageType:   fixedHeaders[protocol.MagicHeaderBytesLen+3],
		RequestID:     binary.BigEndian.Uint64(fixedHeaders[protocol.MagicHeaderBytesLen+4 : protocol.MagicHeaderBytesLen+12]),
		BodyPayload:   payload,
		CRC32Checksum: calcCrc32Val,
		BodyLength:    bodyLen,
	}
	processDone = true

	return
}

func (c *Codec) Reset() {
	c.buffer.Reset()
}
