package handler

import (
	"encoding/json"
	"net"

	"github.com/zYasser/tcp-p2p-chat-go.git/internal/errors"
	"github.com/zYasser/tcp-p2p-chat-go.git/internal/protocol"
)

type JoinEvent struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

func (m *Member) handleJoin(conn net.Conn, msg *protocol.Message) error {
	var eventBody JoinEvent
	err := json.Unmarshal(msg.Body, &eventBody)
	if err != nil {
		return errors.SerializationError

	}
	m.AddFriend(eventBody.ID, eventBody.Address, eventBody.Name, eventBody.Port)
	protocol.BuildMessage(protocol.ACK, nil, nil)
	return nil
}

