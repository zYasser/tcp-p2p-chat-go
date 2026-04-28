package protocol

type MessageType int

const (
	PING MessageType = iota
	SYNC
	KILL
	MESSAGE
	PONG
)

type Status byte

const (
	StatusOK Status = iota
	StatusError
)

type Response struct {
	Headers map[string]string
	Status  Status
	Error   string
}

type Message struct {
	Response
	Type MessageType
	Body []byte
}
