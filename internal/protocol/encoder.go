package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
)

func Encode(m Message) ([]byte, error) {
	headers, err := json.Marshal(m.Headers)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal headers: %w", err)
	}

	totalBytes := 2 + 1 + 4 + len(headers) + len(m.Body)
	packet := make([]byte, 4+totalBytes)

	binary.BigEndian.PutUint32(packet[0:4], uint32(totalBytes))
	binary.BigEndian.PutUint16(packet[4:6], uint16(m.Type))
	packet[6] = byte(m.Status)
	binary.BigEndian.PutUint32(packet[7:11], uint32(len(headers)))
	copy(packet[11:], headers)
	copy(packet[11+len(headers):], m.Body)

	return packet, nil
}

func Decode(conn net.Conn) (*Message, error) {
	payload, err := readPacket(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read packet: %w", err)
	}

	mType, rest, err := extractType(payload)
	if err != nil {
		return nil, err
	}

	status, rest, err := extractStatus(rest)
	if err != nil {
		return nil, err
	}

	headers, body, err := extractHeaders(rest)
	if err != nil {
		return nil, err
	}

	return &Message{
		Response: Response{
			Headers: headers,
			Status:  status,
		},
		Type:     mType,
		Body:     body,
	}, nil
}

func extractType(b []byte) (MessageType, []byte, error) {
	if len(b) < 2 {
		return 0, nil, fmt.Errorf("payload too short for type field: got %d bytes", len(b))
	}
	return MessageType(binary.BigEndian.Uint16(b[:2])), b[2:], nil
}

func extractStatus(b []byte) (Status, []byte, error) {
	if len(b) < 1 {
		return 0, nil, fmt.Errorf("payload too short for status field: got %d bytes", len(b))
	}
	return Status(b[0]), b[1:], nil
}

func extractHeaders(b []byte) (map[string]string, []byte, error) {
	if len(b) < 4 {
		return nil, nil, fmt.Errorf("payload too short for header length field: got %d bytes", len(b))
	}
	headerLength := binary.BigEndian.Uint32(b[:4])
	if uint32(len(b)) < 4+headerLength {
		return nil, nil, fmt.Errorf("payload too short for headers: need %d bytes, got %d", 4+headerLength, len(b))
	}
	var headers map[string]string
	if err := json.Unmarshal(b[4:4+headerLength], &headers); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal headers (length=%d): %w", headerLength, err)
	}
	return headers, b[4+headerLength:], nil
}

func readPacket(conn net.Conn) ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, fmt.Errorf("failed to read packet length: %w", err)
	}
	totalLength := binary.BigEndian.Uint32(lenBuf)
	payload := make([]byte, totalLength)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, fmt.Errorf("failed to read packet body: %w", err)
	}
	return payload, nil
}
