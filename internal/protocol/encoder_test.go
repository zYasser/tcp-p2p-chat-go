package protocol

import (
	"bytes"
	"encoding/binary"
	"net"
	"reflect"
	"testing"
	"time"
)

func encodeOrFail(t *testing.T, m Message) []byte {
	t.Helper()
	b, err := Encode(m)
	if err != nil {
		t.Fatalf("Encode() unexpected error: %v", err)
	}
	return b
}

func pipeConn() (net.Conn, net.Conn) {
	return net.Pipe()
}

func writeAndDecode(t *testing.T, raw []byte) (*Message, error) {
	t.Helper()
	server, client := pipeConn()
	defer server.Close()
	defer client.Close()

	go func() {
		server.Write(raw)
		server.Close()
	}()

	client.SetDeadline(time.Now().Add(time.Second))
	return Decode(client)
}

func TestEncode_FrameLength(t *testing.T) {
	m := Message{Response: Response{Headers: map[string]string{"k": "v"}, Status: StatusOK}, Type: 1, Body: []byte("hello")}
	packet := encodeOrFail(t, m)

	declared := binary.BigEndian.Uint32(packet[:4])
	if int(declared)+4 != len(packet) {
		t.Errorf("frame length field=%d, actual payload=%d", declared, len(packet)-4)
	}
}

func TestEncode_TypeField(t *testing.T) {
	m := Message{Response: Response{Headers: map[string]string{}}, Type: 42, Body: nil}
	packet := encodeOrFail(t, m)

	gotType := binary.BigEndian.Uint16(packet[4:6])
	if gotType != 42 {
		t.Errorf("type field: got %d, want 42", gotType)
	}
}

func TestEncode_StatusField(t *testing.T) {
	m := Message{Response: Response{Headers: map[string]string{}, Status: StatusError}, Type: 1, Body: nil}
	packet := encodeOrFail(t, m)

	if gotStatus := Status(packet[6]); gotStatus != StatusError {
		t.Errorf("status field: got %d, want %d", gotStatus, StatusError)
	}
}

func TestEncode_ErrorField(t *testing.T) {
	m := Message{Response: Response{Headers: map[string]string{}, Status: StatusError, Error: "bad request"}, Type: 1, Body: nil}
	packet := encodeOrFail(t, m)

	headersLen := binary.BigEndian.Uint32(packet[7:11])
	errorLenOffset := 11 + headersLen
	gotErrorLen := binary.BigEndian.Uint32(packet[errorLenOffset : errorLenOffset+4])
	if gotErrorLen != uint32(len("bad request")) {
		t.Errorf("error field length: got %d, want %d", gotErrorLen, len("bad request"))
	}
}

func TestEncode_NilHeaders(t *testing.T) {
	m := Message{Response: Response{Headers: nil}, Type: 1, Body: []byte("body")}
	_, err := Encode(m)
	if err != nil {
		t.Errorf("Encode() with nil headers should not error, got: %v", err)
	}
}

func TestEncode_NilBody(t *testing.T) {
	m := Message{Response: Response{Headers: map[string]string{"a": "b"}}, Type: 1, Body: nil}
	_, err := Encode(m)
	if err != nil {
		t.Errorf("Encode() with nil body should not error, got: %v", err)
	}
}

func TestEncode_EmptyMessage(t *testing.T) {
	m := Message{Response: Response{Headers: map[string]string{}}, Type: 0, Body: []byte{}}
	_, err := Encode(m)
	if err != nil {
		t.Errorf("Encode() with empty message should not error, got: %v", err)
	}
}

func TestEncode_LargeBody(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 1<<16)
	m := Message{Response: Response{Headers: map[string]string{}}, Type: 7, Body: body}
	packet := encodeOrFail(t, m)

	declared := int(binary.BigEndian.Uint32(packet[:4]))
	if declared+4 != len(packet) {
		t.Errorf("frame length mismatch for large body")
	}
}

func TestEncode_MultipleHeaders(t *testing.T) {
	m := Message{
		Response: Response{Headers: map[string]string{"content-type": "application/json", "x-request-id": "abc123", "auth": "bearer xyz"}, Status: StatusOK},
		Type:     3,
		Body:     []byte(`{"key":"value"}`),
	}
	_, err := Encode(m)
	if err != nil {
		t.Errorf("Encode() unexpected error: %v", err)
	}
}

func TestExtractType_Valid(t *testing.T) {
	b := []byte{0x00, 0x05, 0xFF, 0xFF}
	mType, rest, err := extractType(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mType != 5 {
		t.Errorf("got type %d, want 5", mType)
	}
	if !bytes.Equal(rest, []byte{0xFF, 0xFF}) {
		t.Errorf("rest not advanced correctly: %v", rest)
	}
}

func TestExtractType_TooShort(t *testing.T) {
	_, _, err := extractType([]byte{0x01})
	if err == nil {
		t.Error("expected error for 1-byte input, got nil")
	}
}

func TestExtractType_Empty(t *testing.T) {
	_, _, err := extractType([]byte{})
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestExtractType_MaxValue(t *testing.T) {
	b := []byte{0xFF, 0xFF}
	mType, _, err := extractType(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mType != 0xFFFF {
		t.Errorf("got %d, want 65535", mType)
	}
}

func TestExtractStatus_Valid(t *testing.T) {
	status, rest, err := extractStatus([]byte{byte(StatusError), 0xFF})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != StatusError {
		t.Errorf("got status %d, want %d", status, StatusError)
	}
	if !bytes.Equal(rest, []byte{0xFF}) {
		t.Errorf("rest not advanced correctly: %v", rest)
	}
}

func TestExtractStatus_Empty(t *testing.T) {
	_, _, err := extractStatus([]byte{})
	if err == nil {
		t.Error("expected error for empty status field")
	}
}

func TestExtractHeaders_Valid(t *testing.T) {
	jsonBytes := []byte(`{"foo":"bar"}`)
	buf := make([]byte, 4+len(jsonBytes))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(jsonBytes)))
	copy(buf[4:], jsonBytes)
	buf = append(buf, []byte("body")...)

	headers, body, err := extractHeaders(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if headers["foo"] != "bar" {
		t.Errorf("got header foo=%q, want bar", headers["foo"])
	}
	if !bytes.Equal(body, []byte("body")) {
		t.Errorf("body not extracted correctly: %q", body)
	}
}

func TestExtractHeaders_TooShortForLengthField(t *testing.T) {
	_, _, err := extractHeaders([]byte{0x00, 0x01})
	if err == nil {
		t.Error("expected error for input shorter than 4 bytes")
	}
}

func TestExtractHeaders_LengthFieldExceedsBuffer(t *testing.T) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, 999)
	_, _, err := extractHeaders(buf)
	if err == nil {
		t.Error("expected error when header length exceeds buffer size")
	}
}

func TestExtractHeaders_InvalidJSON(t *testing.T) {
	invalid := []byte(`not-json`)
	buf := make([]byte, 4+len(invalid))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(invalid)))
	copy(buf[4:], invalid)

	_, _, err := extractHeaders(buf)
	if err == nil {
		t.Error("expected error for invalid JSON headers")
	}
}

func TestExtractHeaders_EmptyHeaders(t *testing.T) {
	jsonBytes := []byte(`{}`)
	buf := make([]byte, 4+len(jsonBytes))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(jsonBytes)))
	copy(buf[4:], jsonBytes)

	headers, body, err := extractHeaders(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 0 {
		t.Errorf("expected empty headers, got %v", headers)
	}
	if len(body) != 0 {
		t.Errorf("expected empty body, got %v", body)
	}
}

func TestExtractError_Valid(t *testing.T) {
	errorBytes := []byte("bad request")
	buf := make([]byte, 4+len(errorBytes))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(errorBytes)))
	copy(buf[4:], errorBytes)
	buf = append(buf, []byte("body")...)

	gotError, body, err := extractError(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotError != "bad request" {
		t.Errorf("got error %q, want %q", gotError, "bad request")
	}
	if !bytes.Equal(body, []byte("body")) {
		t.Errorf("body not extracted correctly: %q", body)
	}
}

func TestExtractError_TooShortForLengthField(t *testing.T) {
	_, _, err := extractError([]byte{0x00, 0x01})
	if err == nil {
		t.Error("expected error for input shorter than 4 bytes")
	}
}

func roundTrip(t *testing.T, m Message) *Message {
	t.Helper()
	packet := encodeOrFail(t, m)

	server, client := pipeConn()
	defer server.Close()
	defer client.Close()

	go func() {
		server.Write(packet)
		server.Close()
	}()

	client.SetDeadline(time.Now().Add(time.Second))
	got, err := Decode(client)
	if err != nil {
		t.Fatalf("Decode() unexpected error: %v", err)
	}
	return got
}

func TestRoundTrip_Basic(t *testing.T) {
	m := Message{
		Response: Response{Headers: map[string]string{"key": "value"}, Status: StatusOK},
		Type:     1,
		Body:     []byte("hello world"),
	}
	got := roundTrip(t, m)

	if got.Type != m.Type {
		t.Errorf("Type: got %d, want %d", got.Type, m.Type)
	}
	if !reflect.DeepEqual(got.Headers, m.Headers) {
		t.Errorf("Headers: got %v, want %v", got.Headers, m.Headers)
	}
	if got.Status != m.Status {
		t.Errorf("Status: got %d, want %d", got.Status, m.Status)
	}
	if got.Error != m.Error {
		t.Errorf("Error: got %q, want %q", got.Error, m.Error)
	}
	if !bytes.Equal(got.Body, m.Body) {
		t.Errorf("Body: got %q, want %q", got.Body, m.Body)
	}
}

func TestRoundTrip_EmptyBody(t *testing.T) {
	m := Message{Response: Response{Headers: map[string]string{"x": "y"}}, Type: 2, Body: []byte{}}
	got := roundTrip(t, m)
	if !bytes.Equal(got.Body, []byte{}) && got.Body != nil {
		t.Errorf("Body: got %v, want empty", got.Body)
	}
}

func TestRoundTrip_EmptyHeaders(t *testing.T) {
	m := Message{Response: Response{Headers: map[string]string{}, Status: StatusError, Error: "bad request"}, Type: 3, Body: []byte("data")}
	got := roundTrip(t, m)
	if got.Status != m.Status {
		t.Errorf("Status: got %d, want %d", got.Status, m.Status)
	}
	if got.Error != m.Error {
		t.Errorf("Error: got %q, want %q", got.Error, m.Error)
	}
	if !bytes.Equal(got.Body, m.Body) {
		t.Errorf("Body: got %q, want %q", got.Body, m.Body)
	}
}

func TestRoundTrip_BinaryBody(t *testing.T) {
	body := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	m := Message{Response: Response{Headers: map[string]string{}}, Type: 9, Body: body}
	got := roundTrip(t, m)
	if !bytes.Equal(got.Body, body) {
		t.Errorf("Body: got %v, want %v", got.Body, body)
	}
}

func TestRoundTrip_MultipleHeaders(t *testing.T) {
	m := Message{
		Response: Response{Headers: map[string]string{
			"content-type":  "application/json",
			"authorization": "Bearer token123",
			"x-request-id":  "req-abc",
		}, Status: StatusOK},
		Type: 4,
		Body: []byte(`{"status":"ok"}`),
	}
	got := roundTrip(t, m)
	if !reflect.DeepEqual(got.Headers, m.Headers) {
		t.Errorf("Headers: got %v, want %v", got.Headers, m.Headers)
	}
	if got.Status != m.Status {
		t.Errorf("Status: got %d, want %d", got.Status, m.Status)
	}
}

func TestRoundTrip_ZeroType(t *testing.T) {
	m := Message{Response: Response{Headers: map[string]string{}}, Type: 0, Body: []byte("zero")}
	got := roundTrip(t, m)
	if got.Type != 0 {
		t.Errorf("Type: got %d, want 0", got.Type)
	}
}

func TestRoundTrip_MaxType(t *testing.T) {
	m := Message{Response: Response{Headers: map[string]string{}}, Type: 0xFFFF, Body: []byte("max")}
	got := roundTrip(t, m)
	if got.Type != 0xFFFF {
		t.Errorf("Type: got %d, want 65535", got.Type)
	}
}

func TestRoundTrip_LargeBody(t *testing.T) {
	body := bytes.Repeat([]byte("ab"), 1<<15)
	m := Message{Response: Response{Headers: map[string]string{"size": "large"}}, Type: 5, Body: body}
	got := roundTrip(t, m)
	if !bytes.Equal(got.Body, body) {
		t.Errorf("large body mismatch (lengths: got %d, want %d)", len(got.Body), len(body))
	}
}

func TestDecode_EmptyStream(t *testing.T) {
	server, client := pipeConn()
	server.Close()
	defer client.Close()

	client.SetDeadline(time.Now().Add(time.Second))
	_, err := Decode(client)
	if err == nil {
		t.Error("expected error on empty stream, got nil")
	}
}

func TestDecode_TruncatedAfterLengthField(t *testing.T) {
	raw := make([]byte, 4)
	binary.BigEndian.PutUint32(raw, 100)
	_, err := writeAndDecode(t, raw)
	if err == nil {
		t.Error("expected error on truncated payload, got nil")
	}
}

func TestDecode_PacketTooShortForTypeField(t *testing.T) {
	inner := []byte{0x01}
	raw := make([]byte, 4+len(inner))
	binary.BigEndian.PutUint32(raw[:4], uint32(len(inner)))
	copy(raw[4:], inner)

	_, err := writeAndDecode(t, raw)
	if err == nil {
		t.Error("expected error for payload too short for type field")
	}
}

func TestDecode_PacketTooShortForStatusField(t *testing.T) {
	inner := []byte{0x00, 0x01}
	raw := make([]byte, 4+len(inner))
	binary.BigEndian.PutUint32(raw[:4], uint32(len(inner)))
	copy(raw[4:], inner)

	_, err := writeAndDecode(t, raw)
	if err == nil {
		t.Error("expected error for payload too short for status field")
	}
}

func TestDecode_PacketTooShortForHeaderLengthField(t *testing.T) {
	inner := []byte{0x00, 0x01, byte(StatusOK)}
	raw := make([]byte, 4+len(inner))
	binary.BigEndian.PutUint32(raw[:4], uint32(len(inner)))
	copy(raw[4:], inner)

	_, err := writeAndDecode(t, raw)
	if err == nil {
		t.Error("expected error for payload too short for header length field")
	}
}

func TestDecode_PacketTooShortForErrorLengthField(t *testing.T) {
	inner := []byte{0x00, 0x01, byte(StatusOK), 0x00, 0x00, 0x00, 0x02, '{', '}'}
	raw := make([]byte, 4+len(inner))
	binary.BigEndian.PutUint32(raw[:4], uint32(len(inner)))
	copy(raw[4:], inner)

	_, err := writeAndDecode(t, raw)
	if err == nil {
		t.Error("expected error for payload too short for error length field")
	}
}

func TestDecode_InvalidHeaderJSON(t *testing.T) {
	badJSON := []byte(`{invalid}`)
	inner := make([]byte, 2+1+4+len(badJSON))
	binary.BigEndian.PutUint16(inner[0:2], 1)
	inner[2] = byte(StatusOK)
	binary.BigEndian.PutUint32(inner[3:7], uint32(len(badJSON)))
	copy(inner[7:], badJSON)

	raw := make([]byte, 4+len(inner))
	binary.BigEndian.PutUint32(raw[:4], uint32(len(inner)))
	copy(raw[4:], inner)

	_, err := writeAndDecode(t, raw)
	if err == nil {
		t.Error("expected error for invalid header JSON")
	}
}
