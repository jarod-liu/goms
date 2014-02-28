package rtmp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
)

type Conn struct {
	c  *net.TCPConn
	ts time.Time

	inChunkSize, outChunkSize uint32

	chunkStreams map[uint32]*ChunkStream
}

func NewConn(conn *net.TCPConn) *Conn {
	return &Conn{
		c:            conn,
		chunkStream:  make(map[uint32]*ChunkStream),
		inChunkSize:  DEFAULT_CHUNK_SIZE,
		outChunkSize: DEFAULT_CHUNK_SIZE}
}

func (c *Conn) Timestamp() uint32 {
	return uint32((time.Now().UnixNano() - c.ts.UnixNano()) / 1000000)
}

// server side handshake
func (c *Conn) Handshake() (err error) {
	b := make([]byte, handshakeLen)
	_, err = c.c.Read(b[:1])
	if err != nil {
		return
	}

	log.Printf("Get C0=%v, sending S0...\n", b[:1])
	c.c.Write(b[:1])
	c.ts = time.Now()
	binary.BigEndian.PutUint32(b, 0)
	// a little random to the data
	binary.BigEndian.PutUint32(b[8+rand.Intn(handshakeLen-8):], rand.Uint32())
	_, err = c.c.Write(b[:handshakeLen])
	if err != nil {
		return
	}
	log.Println("S1 sent")

	_, err = io.ReadFull(c.c, b[:handshakeLen])
	if err != nil {
		return
	}
	ct1 := binary.BigEndian.Uint32(b)
	log.Printf("Get C1 time=%d\n", ct1)
	binary.BigEndian.PutUint32(b[4:], c.Timestamp())
	// a little random to the data
	binary.BigEndian.PutUint32(b[8+rand.Intn(handshakeLen-8):], rand.Uint32())
	_, err = c.c.Write(b[:handshakeLen])
	if err != nil {
		return
	}
	log.Println("S2 sent")

	_, err = io.ReadFull(c.c, b[:handshakeLen])
	if err != nil {
		return
	}
	log.Println("Get C2. handshake completed.")
	return
}

func (c *Conn) chunkStream(id uint32) *ChunkStream {
	cs, ok := c.chunkStreams[id]
	if !ok {
		cs = NewChunkStream(id)
		c.chunkStreams[id] = cs
	}
	return cs
}

const maxChunkBasicHeaderSize = 11

var chunkHeaderSize = []int{12, 8, 4, 1}

func (c *Conn) ReadMessage() (m *Message, err error) {
	hb := make([]byte, maxChunkBasicHeaderSize)
	_, err = io.ReadFull(c.c, hb[:1])
	if err != nil {
		err = errors.New(err.Error() + ": Failed to read Basic Header 1st byte")
		return
	}

	hfmt := uint8((hb[0] & 0xC0) >> 6)
	csid := uint32(hb[0] & 0x3F)

	switch csid {
	case 0:
		_, err = io.ReadFull(c.c, hb[1:2])
		if err != nil {
			err = errors.New(err.Error() + ": Failed to read Basic Header 2nd byte(cs id 0)")
			return
		}
		csid = (uint32(hb[1]) + 64)

	case 1:
		_, err = io.ReadFull(c.c, hb[1:3])
		if err != nil {
			err = errors.New(err.Error() + ": Failed to read Basic Header 3rd byte(cs id 1)")
			return
		}
		csid = binary.LittleEndian.Uint16(hb[1:]) + 64
	}
	cs := c.chunkStream(csid)

	if hfmt == 0 {
		m = NewMessage()
	} else {
		m = cs.Prev
		m.HeaderFmt = hfmt
	}
	hsize := chunkHeaderSize[m.HeaderFmt] - 1
	if m.HeaderFmt < 3 {
		_, err = io.ReadFull(c.c, hb[:hsize])
		if err != nil {
			err = errors.New(fmt.Sprintf("%v: Failed to read Message Basic Header fmt=%d", err, m.HeaderFmt))
			return
		}

		m.Timestamp = readUint24(hb)

		if m.HeaderFmt < 2 {
			m.Len = readUint24(hb[3:])
			m.Type = uint8(hb[6])

			if m.HeaderFmt == 0 {
				m.StreamId = binary.LittleEndian.Uint32(b[7:])
			}
		}
	}

	if m.Timestamp == EXTENDED_TIMESTAMP {
		_, err = io.ReadFull(c.c, hb[:4])
		if err != nil {
			err = errors.New(err.Error() + ": Failed to read Extended Timestamp")
			return
		}
		m.Timestamp = binary.BigEndian.Uint32(hb)
	}
	if !m.IsAbsTimestamp {
		m.Timestamp += m.Cs.Timestamp
	}
	m.Cs.Timestamp = m.Timestamp

	io.ReadFull(c.c, b[:c.messageLength])
	return
}

func (c *Conn) readChunk() (err error) {
	b := make([]byte, handshakeLen)
	c.c.Read(b[:1])
	fmt := uint32(b[0]) & 0xC0
	csid := uint32(b[0]) & 0x3F

	log.Printf("csid=%v,fmt=%v", csid, fmt)
	switch fmt {
	case 0:
		_, err = io.ReadFull(c.c, b[:11])
		if err != nil {
			return
		}
		c.messageTimestamp = readUint24(b)
		if c.messageTimestamp == 0xFFFFFF {
			c.extendedTimestamp = true
		}
		c.messageLength = readUint24(b[3:])
		c.messageType = uint8(b[6])
		c.messageStreamId = binary.LittleEndian.Uint32(b[7:])

	case 1:
		_, err = io.ReadFull(c.c, b[:7])
		if err != nil {
			return
		}
		c.messageTimestampDelta = readUint24(b)
		if c.messageTimestampDelta != 0xFFFFFF {
			c.messageTimestamp += c.messageTimestampDelta
		} else {
			c.extendedTimestamp = true
		}
		c.messageLength = readUint24(b[3:])
		c.messageType = uint8(b[6])

	case 2:
		_, err = io.ReadFull(c.c, b[:3])
		if err != nil {
			return
		}
		c.messageTimestampDelta = readUint24(b)
		if c.messageTimestampDelta != 0xFFFFFF {
			c.messageTimestamp += c.messageTimestampDelta
		} else {
			c.extendedTimestamp = true
		}
	case 3:
		if !c.extendedTimestamp {
			c.messageTimestamp += c.messageTimestampDelta
		}
	}
	if c.extendedTimestamp {
		_, err = io.ReadFull(c.c, b[:4])
		if err != nil {
			return
		}
		if fmt > 0 {
			c.messageTimestampDelta = binary.BigEndian.Uint32(b)
			c.messageTimestamp += c.messageTimestampDelta
		} else {
			c.messageTimestamp = binary.BigEndian.Uint32(b)
		}
	}
	io.ReadFull(c.c, b[:c.messageLength])
	return
}

func readUint24(b []byte) uint32 {
	v := uint32(b[0]) << 16
	v += uint32(b[1]) << 8
	v += uint32(b[2])
	return v
}
