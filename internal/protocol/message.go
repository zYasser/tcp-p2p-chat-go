package protocol

type MessageType int

const (
	PING MessageType = iota
	SYNC
	KILL
	MESSAGE
)

type Message struct {
	Type    MessageType
	Headers map[string]string
	Body    []byte
}
