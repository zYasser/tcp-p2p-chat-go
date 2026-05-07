package client

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/zYasser/tcp-p2p-chat-go.git/internal/errors"
	"github.com/zYasser/tcp-p2p-chat-go.git/internal/protocol"
	"github.com/zYasser/tcp-p2p-chat-go.git/internal/transport"
)

type ClientP2p struct {
	conn    net.Conn
	context context.Context
	cancel  context.CancelFunc
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	client := &ClientP2p{
		conn:    l,
		context: ctx,
		cancel:  cancel,
	}

	go func() {
		<-ctx.Done()
		client.conn.Close()
	}()
	return client, nil
}

func (c *ClientP2p) SendPing() (*protocol.Message, error) {
	msg := protocol.BuildMessage(protocol.PING, nil, nil)

	bytes, err := protocol.Encode(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode ping message: %w", err)
	}

	if err := transport.Write(c.conn, bytes); err != nil {
		return nil, fmt.Errorf("failed to write ping message: %w", err)
	}

	ack, err := protocol.Decode(c.conn)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ack: %w", err)
	}

	fmt.Println(ack)

	if err := c.conn.Close(); err != nil {
		return nil, fmt.Errorf("failed to close connection: %w", err)
	}

	return ack, nil
}
