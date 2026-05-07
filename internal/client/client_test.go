package client

import (
	"fmt"
	"os"
	"testing"

	"github.com/zYasser/tcp-p2p-chat-go.git/internal/protocol"
	"github.com/zYasser/tcp-p2p-chat-go.git/internal/transport"
)

var server *transport.Server

func initializeServer() {
    server = transport.InitiateServer()
    if err := server.Start(); err != nil {
        panic("failed to start server: " + err.Error())
    } 
}

func Test_Ping(t *testing.T) {
    client, err := CreateClient(server.Address, server.Port)
    if err != nil {
        t.Fatalf("failed to connect: %s", err)
    }

    result , err := client.SendPing();
    if  err != nil {
        t.Errorf("failed to send ping: %s", err)
    }
    fmt.Println(result.Type)
    if result.Type!=protocol.PONG{
        t.Errorf("Failed to send correct response %s", err)
    }
}

func Test_Sending_Wrong_Payload(t *testing.T) {
    client, err := CreateClient(server.Address, server.Port)
    if err != nil {
        t.Fatalf("failed to connect: %s", err)
    }

    result , err := client.SendPing();
    if  err != nil {
        t.Errorf("failed to send ping: %s", err)
    }
    fmt.Println(result.Type)
    if result.Type!=protocol.PONG{
        t.Errorf("Failed to send correct response %s", err)
    }
}


func TestMain(m *testing.M) {
    initializeServer()
    code := m.Run()
    server.Stop()
    os.Exit(code)
}