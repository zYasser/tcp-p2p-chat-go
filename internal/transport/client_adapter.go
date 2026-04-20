package transport

import (
	"context"
)

type ClientAdp struct {
	receiveChan chan []byte
	context     context.Context
}

func sendPing() {
	
}
