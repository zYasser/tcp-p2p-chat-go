package transport

import "net"

func Write(conn net.Conn, packet []byte) error {
	return writeAll(conn, packet)
}

func writeAll(conn net.Conn, data []byte) error {
	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}
