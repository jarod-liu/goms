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
	Len            uint32
	Type           uint8
	IsAbsTimestamp bool
	Timestamp      uint32
	StreamId       uint32
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

func (m *Message) UsePrevValues() {
	p := m.Cs.Prev
	if p == nil {
		return
	}
	m.Timestamp = p.Timestamp
}

func NewChunkStream(id uint32) *ChunkStream {
	cs := &ChunkStream{Id: id}
	return cs
}
