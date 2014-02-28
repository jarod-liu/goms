package rtmp

import (
	"log"
	"net"
)

type Server struct {
}

func NewServer() *Server {
	return new(Server)
}

func (s *Server) ListenAndServe(addr string) {
	a, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Fatalln(err)
	}
	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		log.Fatalln(err)
	}
	for {
		c, err := l.AcceptTCP()
		if err != nil {
			log.Println("accept", err)
			continue
		}
		go s.serveConn(c)
	}
}

func (s *Server) serveConn(conn *net.TCPConn) {
	c := NewConn(conn)
	err := c.Handshake()
	if err != nil {
		log.Println(err)
		return
	}

	for {
		c.readChunk()
		log.Printf("id=%d,len=%d,type=%d,ts=%d,tsd=%d,ts_ext=%v",
			c.messageStreamId, c.messageLength, c.messageType, c.messageTimestamp, c.messageTimestampDelta, c.extendedTimestamp)
	}
}
