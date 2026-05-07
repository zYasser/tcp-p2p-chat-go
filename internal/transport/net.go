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
	Port     int
	Address  string
	Ready    chan struct{}
}

func InitiateServer() *Server {
	address := flag.String("address", "localhost", "server address")
	port := flag.Int("port", 0, "server port")
	flag.Parse()

	return &Server{
		Address: *address,
		Port:    *port,
		Ready:   make(chan struct{}),
	}
}

func InitiateServerWithArgs(address string, port int) *Server {
	return &Server{
		Address: address,
		Port:    port,
		Ready:   make(chan struct{}),
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Address, s.Port))
	if err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}
	s.listener = listener

	if s.Port == 0 {
		s.Port = listener.Addr().(*net.TCPAddr).Port
	}

	log.Printf("Server initialized at %s:%d", s.Address, s.Port)
	close(s.Ready)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("accept failed: %v", err)
				return
			}
			go handleConnection(conn)
		}
	}()

	return nil
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
