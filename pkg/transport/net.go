package transport

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/zYasser/tcp-p2p-chat-go.git/pkg/serialization"
)

type Server struct {
	lp      *net.Listener
	port    int
	address string
}

func InitiateServer() *Server {

	address := flag.String("address", "localhost", "server address")
	port := flag.Int("port", 0, "server port")
	flag.Parse()

	return &Server{
		address: *address,
		port:    *port,
	}
}

func InitiateServerWithArgs(address string, port int) *Server {

	return &Server{
		address: address,
		port:    port,
	}
}

func (s *Server) Start() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.address, s.port))
	if err != nil {
		log.Fatal("Error listening:", err)
	}
	if s.port == 0 {
		s.port = listener.Addr().(*net.TCPAddr).Port
	}
	defer listener.Close()

	log.Printf("Server Initialize it at %s:%d", s.address, s.port)

	s.lp = &listener
	for {

		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting conn:", err)
			continue
		}

		go handleConnection(conn)
	}

}
func handleConnection(conn net.Conn) {

	defer conn.Close()
	start := time.Now()
	defer func() {
		log.Printf("This connection took %v ms", time.Since(start).Milliseconds())
	}()
	msg, err := serialization.Serialize(conn)
	if err != nil {
		fmt.Printf("Failed To process the task %v", err)
		return
	}

	response, _ := json.Marshal(msg)
	write(conn, response)

}

func write(conn net.Conn, parts ...[]byte) error {
	total := 0
	for _, p := range parts {
		total += len(p)
	}

	packet := make([]byte, 4+total)
	binary.BigEndian.PutUint32(packet[:4], uint32(total))

	offset := 4
	for _, p := range parts {
		copy(packet[offset:], p)
		offset += len(p)
	}

	return writeAll(conn, packet)
}

func writeAll(conn net.Conn, data []byte) error {
	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}
