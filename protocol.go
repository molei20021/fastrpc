package fastrpc

type Header [11]byte // 0: magicNumber | 1:(0): heartbeat | 1:(1-3): payloadType | 1:(4-6): compressType | 1:(7): status | 2-9: uuid | 10:(0-3): bodyLength

const (
	magicNumber = 0x1c
)

type Package struct {
	*Header
	RpcPath   string
	RpcMethod string
	Payload   []byte
	Data      []byte
}
