package gott

import (
	"bufio"
	"log"
	"net"
)

type Client struct {
	connection net.Conn
	connected  bool
}

func (c *Client) listen() {
	sockBuffer := bufio.NewReader(c.connection)

	for {
		fixedHeader := make([]byte, 2)
		_, err := sockBuffer.Read(fixedHeader)
		if err != nil {
			log.Println("read error", err)
			break
		}

		switch fixedHeader[0] {
		case TYPE_CONNECT_BYTE:
			log.Println("client trying to connect")
			remLen := fixedHeader[1]
			//varHeader :=
		}
	}
	c.disconnect()
}

func (c *Client) disconnect() {
	c.connection.Close()
	c.connected = false
}
