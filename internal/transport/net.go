package transport

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/zYasser/tcp-p2p-chat-go.git/internal/handler"
)

type Server struct {
	listener net.Listener
	port     int
	address  string
	Ready    chan struct{}
}

func InitiateServer() *Server {
	address := flag.String("address", "localhost", "server address")
	port := flag.Int("port", 0, "server port")
	flag.Parse()

	return &Server{
		address: *address,
		port:    *port,
		Ready:   make(chan struct{}),
	}
}

func InitiateServerWithArgs(address string, port int) *Server {
	return &Server{
		address: address,
		port:    port,
		Ready:   make(chan struct{}),
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.address, s.port))
	if err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}

	s.listener = listener

	if s.port == 0 {
		s.port = listener.Addr().(*net.TCPAddr).Port
	}

	log.Printf("Server initialized at %s:%d", s.address, s.port)
	close(s.Ready)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept failed: %w", err)
		}

		go handleConnection(conn)
	}
}

func (s *Server) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	result := handler.HandleMessage(conn)
	if err := writeAll(conn, result); err != nil {
		log.Println("write failed:", err)
	}
}
