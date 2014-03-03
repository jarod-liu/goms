package rtmp

const (
	HANKSHAKE_MESSAGE_LEN = 1536

	EXTENDED_TIMESTAMP = 0xFFFFFF

	DEFAULT_CHUNK_SIZE = 128
)

type Chunk struct {
	HeaderSize int
	Header     []byte
	Data       []byte
}

type Message struct {
	Cs *ChunkStream

	HeaderFmt      uint8
	Type           uint8
	IsAbsTimestamp bool
	Timestamp      uint32
	StreamId       uint32

	Len       uint32
	BytesRead uint32
	Body      []byte
}

type ChunkStream struct {
	Id        uint32
	Prev      *Message
	Timestamp uint32
}

func NewMessage() *Message {
	return &Message{
		IsAbsTimestamp: true,
	}
}

func (m *Message) Ready() bool {
	return m.Len == m.BytesRead
}

func NewChunkStream(id uint32) *ChunkStream {
	cs := &ChunkStream{Id: id}
	return cs
}

func decodeUint24(b []byte) uint32 {
	v := uint32(b[0]) << 16
	v += uint32(b[1]) << 8
	v += uint32(b[2])
	return v
}
