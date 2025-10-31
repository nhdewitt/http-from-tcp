package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"

	"github.com/nhdewitt/http-from-tcp/internal/request"
	"github.com/nhdewitt/http-from-tcp/internal/response"
)

type Server struct {
	listener    net.Listener
	isListening atomic.Bool
	handler     func(w *response.Writer, req *request.Request)
}

func Serve(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	s := &Server{
		listener: listener,
		handler:  handler,
	}
	s.isListening.Store(true)
	go s.listen()

	return s, nil
}

func (s *Server) Close() error {
	if !s.isListening.CompareAndSwap(true, false) {
		return nil
	}

	if s.listener != nil {
		return s.listener.Close()
	}

	return nil
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if !s.isListening.Load() {
				return
			}
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	req, err := request.RequestFromReader(conn)
	if err != nil {
		return
	}

	resp := response.NewWriter(conn)
	s.handler(resp, req)
}
