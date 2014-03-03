package rtmp

import (
	"encoding/binary"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
)

func Dial(addr string) (c *ClientConn, err error) {
	a, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}
	tl, err := net.DialTCP("tcp", nil, a)
	if err != nil {
		return
	}
	c = newClientConn(tl)
	return
}

type ClientConn struct {
	conn
}

func newClientConn(tcpConn *net.TCPConn) *ClientConn {
	return &ClientConn{conn{
		c:            tcpConn,
		chunkStreams: make(map[uint32]*ChunkStream),
		inChunkSize:  DEFAULT_CHUNK_SIZE,
		outChunkSize: DEFAULT_CHUNK_SIZE}}
}

// server side handshake
func (c *ClientConn) Handshake() (err error) {
	b := make([]byte, HANKSHAKE_MESSAGE_LEN)
	b[0] = 0x03
	_, err = c.c.Write(b[:1])
	if err != nil {
		return
	}
	log.Println("C0 sent")

	c.ts = time.Now()
	binary.BigEndian.PutUint32(b, 0)
	// a little random to the data
	binary.BigEndian.PutUint32(b[8+rand.Intn(HANKSHAKE_MESSAGE_LEN-8):], rand.Uint32())
	_, err = c.c.Write(b)
	if err != nil {
		return
	}
	log.Println("C1 sent")

	_, err = c.c.Read(b[:1])
	if err != nil {
		return
	}
	log.Printf("Get S0=%d\n", b[0])

	_, err = io.ReadFull(c.c, b)
	if err != nil {
		return
	}
	st1 := binary.BigEndian.Uint32(b)
	log.Printf("Get S1 time=%d\n", st1)

	binary.BigEndian.PutUint32(b[4:], c.Timestamp())
	// a little random to the data
	binary.BigEndian.PutUint32(b[8+rand.Intn(HANKSHAKE_MESSAGE_LEN-8):], rand.Uint32())
	_, err = c.c.Write(b)
	if err != nil {
		return
	}
	log.Println("C2 sent")

	_, err = io.ReadFull(c.c, b)
	if err != nil {
		return
	}
	log.Println("Get S2. handshake completed.")
	return
}
