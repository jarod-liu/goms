package main

import (
	"github.com/jarod/goms/rtmp"
	"log"
)

type Server struct {
}

func NewServer() *Server {
	return new(Server)
}

func (s *Server) ListenAndServe(addr string) {
	l, err := rtmp.Listen(addr)
	if err != nil {
		log.Fatalln(err)
	}
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println("accept", err)
			continue
		}
		go s.serveConn(c)
	}
}

func (s *Server) serveConn(c *rtmp.ServerConn) {
	err := c.Handshake()
	if err != nil {
		log.Println(err)
		return
	}

	for {
		m, err := c.Read()
		if err != nil {
			log.Println(err)
			break
		}
		log.Printf("id=%d,len=%d,read=%d,type=%d,ts=%d,fmt=%d",
			m.StreamId, m.Len, m.BytesRead, m.Type, m.Timestamp, m.HeaderFmt)
	}
}
