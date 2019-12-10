package gott

import (
	"bufio"
	"encoding/binary"
	"gott/bytes"
	"gott/utils"
	"log"
	"net"

	"github.com/google/uuid"
)

type Client struct {
	connection net.Conn
	connected  bool
	keepAlive  uint16
	broker     *Broker
	ClientId   string
}

func (c *Client) listen() {
	sockBuffer := bufio.NewReader(c.connection)

	for {
		fixedHeader := make([]byte, FIXED_HEADER_LEN)
		_, err := sockBuffer.Read(fixedHeader)
		if err != nil {
			log.Println("read error", err)
			break
		}
		log.Println("fixedHeader", fixedHeader)

		switch fixedHeader[0] {
		case TYPE_CONNECT_BYTE:
			log.Println("client trying to connect")
			if remLen, err := bytes.Decode(fixedHeader[1:]); err != nil {
				log.Println("malformed packet", err)
				break
			} else {
				payloadLen := remLen - CONNECT_VAR_HEADER_LEN
				if payloadLen == 0 {
					log.Println("connect error: payload len is zero")
					break
				}

				log.Println("rem len", remLen)
				varHeader := make([]byte, CONNECT_VAR_HEADER_LEN)
				if _, err = sockBuffer.Read(varHeader); err != nil {
					log.Println("error reading var header", err)
					break
				}
				log.Println("var header", varHeader)

				protocolNameLen := binary.BigEndian.Uint16(varHeader[0:2])
				if protocolNameLen != 4 {
					log.Println("malformed packet: protocol name length is incorrect", protocolNameLen)
					break
				}

				protocolName := string(varHeader[2:6])
				if protocolName != "MQTT" {
					log.Println("malformed packet: unknown protocol name. expected MQTT found", protocolName)
					break
				}

				if !utils.ByteInSlice(varHeader[6], SUPPORTED_PROTOCOL_VERSIONS) {
					log.Println("unsupported protocol", varHeader[6])
					// TODO: respond to the CONNECT Packet with a CONNACK return code 0x01 (unacceptable protocol level) and then disconnect the Client
					break
				}

				flagsBits := bytes.ByteToBinaryString(varHeader[7])
				connFlags := CheckConnectFlags(flagsBits)
				if connFlags.Reserved != "0" {
					log.Println("malformed packet: reserved flag in CONNECT packet is not zero", flagsBits[7:8])
					break
				}

				c.keepAlive = binary.BigEndian.Uint16(varHeader[8:])

				// payload parsing
				payload := make([]byte, payloadLen)
				if _, err = sockBuffer.Read(payload); err != nil {
					log.Println("error reading payload", err)
					break
				}
				log.Println("payload", payload)

				// client id parsing
				clientIdLen := int(binary.BigEndian.Uint16(payload[0:2]))
				if clientIdLen == 0 {
					if connFlags.CleanSession == "0" {
						// TODO: respond with a CONNACK return code 0x02 (Identifier rejected)
						log.Println("connect error: received zero byte client id with clean session flag set to 0")
						break
					}
					c.ClientId = uuid.New().String()
				} else {
					if len(payload) < 2+clientIdLen {
						log.Println("malformed packet: payload length is not valid")
						break
					}

					c.ClientId = string(payload[2 : 2+clientIdLen])
				}

				// TODO: verify payload with flags (Client Identifier, Will Topic, Will Message, User Name, Password)

				// TODO: implement Session
				// TODO: implement Will [3.1.3]

				// TODO: implement keep alive check and disconnect on timeout of (1.5 * keepalive) as per spec [3.1.2.10]
				// TODO: implement username and password auth [3.1.3]

				// connection succeeded
				log.Println("client connected successfully with id:", c.ClientId)
				c.broker.addClient(c.ClientId, c)
				c.emit(MakeConnAckPacket(0, CONNECT_ACCEPTED))
			}
		}
	}
	c.disconnect()
}

func (c *Client) disconnect() {
	c.connected = false
	c.connection.Close()
	c.broker.removeClient(c.ClientId)
}

func (c *Client) closeConnection() {
	c.connected = false
	c.connection.Close()
}

func (c *Client) emit(packet []byte) {
	if _, err := c.connection.Write(packet); err != nil {
		log.Println("error sending packet", err, packet)
	}
}
