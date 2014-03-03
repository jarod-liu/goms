package rtmp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

type conn struct {
	c  *net.TCPConn
	ts time.Time

	inChunkSize, outChunkSize uint32

	chunkStreams map[uint32]*ChunkStream
}

func (c *conn) Timestamp() uint32 {
	return uint32((time.Now().UnixNano() - c.ts.UnixNano()) / 1000000)
}

func (c *conn) Handshake() (err error) {
	err = errors.New("not implemented in Conn")
	return
}

func (c *conn) chunkStream(id uint32) *ChunkStream {
	cs, ok := c.chunkStreams[id]
	if !ok {
		cs = NewChunkStream(id)
		c.chunkStreams[id] = cs
	}
	return cs
}

const maxChunkBasicHeaderSize = 11

var chunkHeaderSize = []int{12, 8, 4, 1}

func (c *conn) Read() (m *Message, err error) {
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
		csid = uint32(binary.LittleEndian.Uint16(hb[1:])) + 64
	}
	cs := c.chunkStream(csid)

	if hfmt == 0 {
		m = NewMessage()
		m.Cs = cs
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

		m.Timestamp = decodeUint24(hb)

		if m.HeaderFmt < 2 {
			m.Len = decodeUint24(hb[3:])
			m.BytesRead = 0
			m.Type = uint8(hb[6])

			if m.HeaderFmt == 0 {
				m.StreamId = binary.LittleEndian.Uint32(hb[7:])
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

	if m.Body == nil && m.Len > 0 {
		m.Body = make([]byte, m.Len)
	}
	bytesRemain := m.Len - m.BytesRead
	chunkSize := c.inChunkSize
	if bytesRemain < chunkSize {
		chunkSize = bytesRemain
	}
	_, err = io.ReadFull(c.c, m.Body[m.BytesRead:m.BytesRead+chunkSize])
	if err != nil {
		err = errors.New(err.Error() + ": Failed to read body")
		return
	}
	m.BytesRead += chunkSize
	m.Cs.Prev = m

	if m.Ready() {
		if !m.IsAbsTimestamp {
			m.Timestamp += m.Cs.Timestamp
		}
		m.Cs.Timestamp = m.Timestamp
	}
	return
}
