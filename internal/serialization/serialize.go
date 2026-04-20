package serialization

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"strings"

	"github.com/zYasser/tcp-p2p-chat-go.git/pkg/errors"
)

func Serialize(conn net.Conn) (*map[string]interface{}, error) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Read error: %v", err)
		return nil, errors.SerializationError
	}
	line = strings.ReplaceAll(line, "\r\n", "")

	var msg map[string]interface{}
	err = json.Unmarshal([]byte(strings.TrimSpace(line)), &msg)
	if err != nil {
		log.Printf("JSON parse error: %v", err)
		return nil, errors.SerializationError
	}
	return &msg, nil

}
