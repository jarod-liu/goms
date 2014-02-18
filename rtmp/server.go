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
		conn := NewConn(c)

		go conn.Handshake()
	}
}
