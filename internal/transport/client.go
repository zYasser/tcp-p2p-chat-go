package transport

import (
	"fmt"
	"log"
	"net"
)

func establishConnection(address string, port int) (net.Conn, error) {
	listener, err := net.Dial("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return nil, err
	}
	return listener, nil
}

func send(con net.Conn, payload ...[]byte) {
	for _, byt := range payload {
		data := byt
		for len(data) > 0 {
			n, err := con.Write(data)
			if err != nil {
				log.Fatal("Failed To send data")
			}
			data = data[n:]
		}
	}
}

