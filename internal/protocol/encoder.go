package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
)

// Encode serializes a Message into a length-prefixed binary packet suitable
// for sending over a network connection.
//
// Packet layout (big-endian):
//
//	[4 bytes] total payload length
//	[2 bytes] message type
//	[1 byte]  status code
//	[4 bytes] headers length (N)
//	[N bytes] headers (JSON-encoded map[string]string)
//	[4 bytes] error string length (E)
//	[E bytes] error string
//	[?bytes]  body (remainder)
func Encode(m Message) ([]byte, error) {
	// Marshal the headers map to JSON so it can be embedded in the binary packet.
	headers, err := json.Marshal(m.Headers)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal headers: %w", err)
	}

	errorBytes := []byte(m.Error)

	// Calculate the total payload size (everything after the 4-byte length prefix):
	//   2 (type) + 1 (status) + 4 (header len) + headers + 4 (error len) + error + body
	totalBytes := 2 + 1 + 4 + len(headers) + 4 + len(errorBytes) + len(m.Body)

	// Allocate the full packet: 4-byte length prefix + payload.
	packet := make([]byte, 4+totalBytes)

	// Write the 4-byte big-endian payload length at the start of the packet.
	binary.BigEndian.PutUint32(packet[0:4], uint32(totalBytes))

	// Write the 2-byte message type.
	binary.BigEndian.PutUint16(packet[4:6], uint16(m.Type))

	// Write the 1-byte status code.
	packet[6] = byte(m.Status)

	// Write the 4-byte header length, then copy the JSON-encoded headers.
	binary.BigEndian.PutUint32(packet[7:11], uint32(len(headers)))
	copy(packet[11:], headers)

	// Calculate where the error-length field starts (right after the headers).
	errorLenOffset := 11 + len(headers)

	// Write the 4-byte error string length, then copy the error bytes.
	binary.BigEndian.PutUint32(packet[errorLenOffset:errorLenOffset+4], uint32(len(errorBytes)))
	copy(packet[errorLenOffset+4:errorLenOffset+4+len(errorBytes)], errorBytes)

	// Copy the raw message body into the remaining space.
	copy(packet[errorLenOffset+4+len(errorBytes):], m.Body)

	return packet, nil
}

// Decode reads and deserializes a single Message from an open network connection.
// It first reads the length-prefixed packet via readPacket, then sequentially
// extracts each field from the payload.
func Decode(conn net.Conn) (*Message, error) {
	// Read the raw bytes of the next packet off the wire.
	payload, err := readPacket(conn)
	if err != nil {
		return nil, fmt.Errorf("decode: failed to read packet: %w", err)
	}

	// Extract the 2-byte message type; rest holds the remaining bytes.
	mType, rest, err := extractType(payload)
	if err != nil {
		return nil, err
	}

	// Extract the 1-byte status code.
	status, rest, err := extractStatus(rest)
	if err != nil {
		return nil, err
	}

	// Extract the length-prefixed, JSON-encoded headers map.
	headers, rest, err := extractHeaders(rest)
	if err != nil {
		return nil, err
	}

	// Extract the length-prefixed error string; body holds whatever is left.
	errorMessage, body, err := extractError(rest)
	if err != nil {
		return nil, err
	}

	return &Message{
		Response: Response{
			Headers: headers,
			Status:  status,
			Error:   errorMessage,
		},
		Type: mType,
		Body: body,
	}, nil
}

// extractType reads the first 2 bytes of b as a big-endian MessageType,
// and returns the type along with the remaining bytes.
func extractType(b []byte) (MessageType, []byte, error) {
	if len(b) < 2 {
		return 0, nil, fmt.Errorf("extract type: payload too short for type field: got %d bytes", len(b))
	}
	return MessageType(binary.BigEndian.Uint16(b[:2])), b[2:], nil
}

// extractStatus reads the first byte of b as a Status code,
// and returns the status along with the remaining bytes.
func extractStatus(b []byte) (Status, []byte, error) {
	if len(b) < 1 {
		return 0, nil, fmt.Errorf("extract status: payload too short for status field: got %d bytes", len(b))
	}
	return Status(b[0]), b[1:], nil
}

// extractHeaders reads a length-prefixed JSON blob from b, unmarshals it into
// a map[string]string, and returns the map along with the remaining bytes.
//
// Format: [4-byte big-endian length][N bytes of JSON]
func extractHeaders(b []byte) (map[string]string, []byte, error) {
	if len(b) < 4 {
		return nil, nil, fmt.Errorf("extract headers: payload too short for header length field: got %d bytes", len(b))
	}

	// Read how many bytes the JSON-encoded headers occupy.
	headerLength := binary.BigEndian.Uint32(b[:4])

	if uint32(len(b)) < 4+headerLength {
		return nil, nil, fmt.Errorf("extract headers: payload too short for headers: need %d bytes, got %d", 4+headerLength, len(b))
	}

	var headers map[string]string
	if err := json.Unmarshal(b[4:4+headerLength], &headers); err != nil {
		return nil, nil, fmt.Errorf("extract headers: failed to unmarshal headers (length=%d): %w", headerLength, err)
	}

	// Return the headers and advance past the length prefix + header bytes.
	return headers, b[4+headerLength:], nil
}

// extractError reads a length-prefixed string from b and returns it along
// with the remaining bytes (which become the message body).
//
// Format: [4-byte big-endian length][E bytes of UTF-8 text]
func extractError(b []byte) (string, []byte, error) {
	if len(b) < 4 {
		return "", nil, fmt.Errorf("extract error: payload too short for error length field: got %d bytes", len(b))
	}

	// Read how many bytes the error string occupies.
	errorLength := binary.BigEndian.Uint32(b[:4])

	if uint32(len(b)) < 4+errorLength {
		return "", nil, fmt.Errorf("extract error: payload too short for error body: need %d bytes, got %d", 4+errorLength, len(b))
	}

	// Return the error string and whatever bytes follow it (the body).
	return string(b[4 : 4+errorLength]), b[4+errorLength:], nil
}

// readPacket reads a single length-prefixed packet from conn.
// It first reads a 4-byte big-endian length, then reads exactly that many
// bytes as the packet payload. This ensures full packets are always consumed
// even if the underlying TCP stream delivers data in fragments.
func readPacket(conn net.Conn) ([]byte, error) {
	// Read the fixed-size 4-byte length header.
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, fmt.Errorf("failed to read packet length: %w", err)
	}

	totalLength := binary.BigEndian.Uint32(lenBuf)

	// Allocate a buffer exactly large enough for the declared payload and fill it.
	payload := make([]byte, totalLength)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, fmt.Errorf("failed to read packet body: %w", err)
	}

	return payload, nil
}
