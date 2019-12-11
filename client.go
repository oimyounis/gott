package gott

import (
	"bufio"
	"log"
	"net"
)

type Client struct {
	connection    net.Conn
	connected     bool
	keepAliveSecs int
	ClientId      string
}

func (c *Client) listen() {
	if GOTT == nil {
		return
	}

	sockBuffer := bufio.NewReader(c.connection)

loop:
	for {
		if !c.connected {
			break
		}

		fixedHeader := make([]byte, 2)
		_, err := sockBuffer.Read(fixedHeader)
		if err != nil {
			log.Println("read error", err)
			break
		}
		log.Println("fixedHeader", fixedHeader)

		lastByte := fixedHeader[1]
		remLenEncoded := []byte{lastByte}

		for lastByte >= 128 {
			lastByte, err = sockBuffer.ReadByte()
			if err != nil {
				log.Println("read error", err)
				break loop
			}
			remLenEncoded = append(remLenEncoded, lastByte)
		}

		packetType := 0

		//switch fixedHeader[0] {
		//case TYPE_CONNECT_BYTE:
		//	log.Println("client trying to connect")
		//	if remLen, err := bytes.Decode(fixedHeader[1:]); err != nil {
		//		log.Println("malformed packet", err)
		//		break loop
		//	} else {
		//		payloadLen := remLen - CONNECT_VAR_HEADER_LEN
		//		if payloadLen == 0 {
		//			log.Println("connect error: payload len is zero")
		//			break loop
		//		}
		//
		//		log.Println("rem len", remLen)
		//		varHeader := make([]byte, CONNECT_VAR_HEADER_LEN)
		//		if _, err = io.ReadFull(sockBuffer, varHeader); err != nil {
		//			log.Println("error reading var header", err)
		//			break loop
		//		}
		//		log.Println("var header", varHeader)
		//
		//		protocolNameLen := binary.BigEndian.Uint16(varHeader[0:2])
		//		if protocolNameLen != 4 {
		//			log.Println("malformed packet: protocol name length is incorrect", protocolNameLen)
		//			break loop
		//		}
		//
		//		protocolName := string(varHeader[2:6])
		//		if protocolName != "MQTT" {
		//			log.Println("malformed packet: unknown protocol name. expected MQTT found", protocolName)
		//			break loop
		//		}
		//
		//		if !utils.ByteInSlice(varHeader[6], SUPPORTED_PROTOCOL_VERSIONS) {
		//			log.Println("unsupported protocol", varHeader[6])
		//			c.emit(MakeConnAckPacket(0, CONNECT_UNACCEPTABLE_PROTO))
		//			break loop
		//		}
		//
		//		flagsBits := bytes.ByteToBinaryString(varHeader[7])
		//		connFlags := ExtractConnectFlags(flagsBits)
		//		if connFlags.Reserved != "0" {
		//			log.Println("malformed packet: reserved flag in CONNECT packet header is not zero", flagsBits[7:8])
		//			break loop
		//		}
		//
		//		c.keepAliveSecs = int(binary.BigEndian.Uint16(varHeader[8:]))
		//
		//		// payload parsing
		//		payload := make([]byte, payloadLen)
		//		if _, err = io.ReadFull(sockBuffer, payload); err != nil {
		//			log.Println("error reading payload", err)
		//			break loop
		//		}
		//		log.Println("payload", payload)
		//
		//		// client id parsing
		//		clientIdLen := int(binary.BigEndian.Uint16(payload[0:2])) // maximum is 65535 (16-bit)
		//		if clientIdLen == 0 {
		//			if connFlags.CleanSession == "0" {
		//				log.Println("connect error: received zero byte client id with clean session flag set to 0")
		//				c.emit(MakeConnAckPacket(0, CONNECT_ID_REJECTED))
		//				break loop
		//			}
		//			c.ClientId = uuid.New().String()
		//		} else {
		//			if len(payload) < 2+clientIdLen {
		//				log.Println("malformed packet: payload length is not valid")
		//				break loop
		//			}
		//
		//			c.ClientId = string(payload[2 : 2+clientIdLen])
		//		}
		//
		//		// TODO: verify payload with flags (Client Identifier, Will Topic, Will Message, User Name, Password)
		//
		//		// TODO: implement Session
		//		// TODO: implement Will [3.1.3]
		//
		//		// TODO: implement keep alive check and disconnect on timeout of (1.5 * keepalive) as per spec [3.1.2.10]
		//		// TODO: implement username and password auth [3.1.3]
		//
		//		// connection succeeded
		//		log.Println("client connected successfully with id:", c.ClientId)
		//		GOTT.addClient(c)
		//		c.emit(MakeConnAckPacket(0, CONNECT_ACCEPTED))
		//
		//		go func(c *Client) {
		//			time.Sleep(time.Second * 2)
		//			//c.publish([]byte(""), []byte(""), 0, 0, 0)
		//		}(c)
		//	}
		//}
	}
	c.disconnect()
}

func (c *Client) disconnect() {
	if GOTT == nil {
		return
	}

	c.closeConnection()
	GOTT.removeClient(c.ClientId)
}

func (c *Client) closeConnection() {
	c.connected = false
	_ = c.connection.Close()
}

func (c *Client) publish(topicName, payload []byte, dup, qos, retain byte) {
	if qos == 0 {
		dup = 0
	}
	MakePublishPacket(topicName, payload, dup, qos, retain)
}

func (c *Client) emit(packet []byte) {
	if _, err := c.connection.Write(packet); err != nil {
		log.Println("error sending packet", err, packet)
	}
}
