package handler

import (
	"net"

	"github.com/zYasser/tcp-p2p-chat-go.git/internal/protocol"
)

func HandleMessage(conn net.Conn) []byte {
	msg, err := protocol.Decode(conn)
	if err != nil {
		msg = protocol.BuildErrorMessage(protocol.MESSAGE, err)

	}
	bytes, _ := protocol.Encode(*msg)

	return bytes

}
