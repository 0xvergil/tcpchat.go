package chat

import (
	"errors"
	"log"
	"net"
	"time"
)

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Printf("accept: %v", err)
			time.Sleep(50 * time.Millisecond)
			continue
		}
		conn.Close()
	}
}