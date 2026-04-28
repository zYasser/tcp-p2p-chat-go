package client

import (
	"fmt"
	"log"
	"net"

	"github.com/zYasser/tcp-p2p-chat-go.git/internal/errors"
	"github.com/zYasser/tcp-p2p-chat-go.git/internal/protocol"
	"github.com/zYasser/tcp-p2p-chat-go.git/internal/transport"
)

type ClientP2p struct {
	conn net.Conn
}

func establishConnection(address string, port int) (net.Conn, error) {
	listener, err := net.Dial("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {

		return nil, err
	}
	return listener, nil
}

func CreateClient(address string, port int) (*ClientP2p, error) {
	l, err := establishConnection(address, port)
	if err != nil {
		log.Printf("Failed To connect to %s:%d error : %s", address, port, err.Error())
		return nil, errors.FailedToConnect
	}
	return &ClientP2p{
		conn: l,
	}, nil
}

func (c *ClientP2p) SendPing() {
	msg := protocol.BuildMessage(protocol.PING, nil, nil)
	bytes, _ := protocol.Encode(msg)
	transport.Write(c.conn, bytes)
	ack, _ := protocol.Decode(c.conn)
	fmt.Println(ack)
	c.conn.Close()
}
