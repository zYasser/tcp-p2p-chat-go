package main

import "github.com/zYasser/tcp-p2p-chat-go.git/internal/transport"

func main() {
	socket := transport.InitiateServerWithArgs("localhost", 8080)
	socket.Start()

}
