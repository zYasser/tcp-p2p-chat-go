package protocol

type MessageType int

const (
	PING MessageType = iota
	SYNC
	KILL
	MESSAGE
)

type Status byte

const (
	StatusOK Status = iota
	StatusError
)

type Response struct {
	Headers map[string]string
	Status  Status
}

type Message struct {
	Response
	Type MessageType
	Body []byte
}
