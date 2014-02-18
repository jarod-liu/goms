package rtmp

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"
)

const (
	handshakeLen = 1536
)

type Conn struct {
	c    *net.TCPConn
	hbuf []byte
	ts   time.Time

	messageStreamId       uint32
	messageLength         uint32
	extendedTimestamp     bool
	messageTimestamp      uint32
	messageTimestampDelta uint32
	messageType           uint8
}

func NewConn(conn *net.TCPConn) *Conn {
	return &Conn{c: conn, hbuf: make([]byte, 2048)}
}

func (c *Conn) Timestamp() uint32 {
	return uint32((time.Now().UnixNano() - c.ts.UnixNano()) / 1000000)
}

func (c *Conn) Handshake() {
	b := c.hbuf
	_, err := c.c.Read(b[:1])
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Get C0=%v, sending S0...\n", b[:1])
	c.c.Write(b[:1])
	c.ts = time.Now()
	binary.BigEndian.PutUint32(b, 0)
	_, err = c.c.Write(b[:handshakeLen])
	if err != nil {
		log.Println(err)
		return
	}
	_, err = io.ReadFull(c.c, b[:handshakeLen])
	if err != nil {
		log.Println(err)
		return
	}
	ct1 := binary.BigEndian.Uint32(b)
	log.Printf("Get C1 time=%d\n", ct1)
	binary.BigEndian.PutUint32(b[4:], c.Timestamp())
	_, err = c.c.Write(b[:handshakeLen])
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("S2 sent")
	_, err = io.ReadFull(c.c, b[:handshakeLen])
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Get C2. handshake completed.")

	c.readChunk()
	log.Printf("id=%d,len=%d,type=%d,ts=%d,tsd=%d,ts_ext=%v",
		c.messageStreamId, c.messageLength, c.messageType, c.messageTimestamp, c.messageTimestampDelta, c.extendedTimestamp)
	c.readChunk()
	log.Printf("id=%d,len=%d,type=%d,ts=%d,tsd=%d,ts_ext=%v",
		c.messageStreamId, c.messageLength, c.messageType, c.messageTimestamp, c.messageTimestampDelta, c.extendedTimestamp)
	for {
		n, err := c.c.Read(b)
		if err != nil {
			log.Println(err)
			break
		}
		log.Printf("%v\n", b[:n])
	}
}

func (c *Conn) readChunk() (err error) {
	b := c.hbuf
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
