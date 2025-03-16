package main

import (
	"encoding/binary"
	"msg-protocol/protocol"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec/frame"
)

func main() {
	bootstrap := netty.NewBootstrap(netty.WithChildInitializer(func(c netty.Channel) {
		c.
			Pipeline().
			AddLast(
				frame.LengthFieldCodec(
					binary.BigEndian,
					1024*1024,
					protocol.FixedHeaderByteLen,
					20,
					4,
					0,
				),
				frame.PacketCodec(12),
			)
	}))

	bootstrap.Listen(":5531").Sync()
}
