package handler

import (
	"fmt"
	"net"

	"github.com/zYasser/tcp-p2p-chat-go.git/internal/protocol"
)

func HandleMessage(conn net.Conn) []byte {
	msg, err := protocol.Decode(conn)
	if err != nil {
		msg = protocol.BuildErrorMessage(protocol.ERROR, err)

	}
	var result []byte
	switch msg.Type {
	case protocol.PING:
		result, err = handlePing(conn, msg)
	}
	if err != nil {
		msg = protocol.BuildErrorMessage(protocol.ERROR, err)

	}
	return result

}

func handlePing(conn net.Conn, msg *protocol.Message) ([]byte, error) {
	result := protocol.BuildMessage(protocol.PONG, nil, nil)
	bytes, err := protocol.Encode(result)
	if err != nil {
		err := fmt.Errorf("%w failed to encode this message %v", result)
		return nil, err
	}
	return bytes, nil
}
