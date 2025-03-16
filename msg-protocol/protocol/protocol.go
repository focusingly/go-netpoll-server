package protocol

const (
	MagicHeaderBytesLen        = 4 // 头部魔术字节占用长度
	MessageMagic        uint32 = (0x32 << 24) | (0xff << 16) | (0x64 << 8) | 0x32
	FixedHeaderByteLen         = 24 // 固定的头部占用长度
)

type (
	ProtocolMessage struct {
		// (自定义头部)魔术标识 4 字节
		MagicNumbers uint32
		// 字节填充占位(默认为 0)
		OffsetPadding byte
		// 协议版本号
		Version byte
		// 标识位(是否压缩, 加密等)
		Flags byte
		// 消息协议类型(表示请求, 响应, 心跳)
		MessageType byte
		// 消息 ID, 用于追踪
		RequestID uint64
		// 校验和(正文数据的校验和), offset: 20
		CRC32Checksum uint32
		// 正文长度
		BodyLength uint32
		// 可选的正文数据
		BodyPayload []byte
	}
)
