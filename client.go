package gott

import (
	"bufio"
	"encoding/binary"
	"gott/bytes"
	"gott/utils"
	"io"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
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
		log.Println("read", fixedHeader)

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

		packetType, flagsBits := ParseFixedHeaderFirstByte(fixedHeader[0])
		remLen, err := bytes.Decode(remLenEncoded)
		if err != nil {
			log.Println("malformed packet", err)
			break loop
		}

		//log.Println("packetType:", packetType, "remLen:", remLen)

		switch packetType {
		case TYPE_CONNECT:
			//log.Println("client trying to connect")

			payloadLen := remLen - CONNECT_VAR_HEADER_LEN
			if payloadLen == 0 {
				log.Println("connect error: payload len is zero")
				break loop
			}

			//log.Println("rem len", remLen)
			varHeader := make([]byte, CONNECT_VAR_HEADER_LEN)
			if _, err = io.ReadFull(sockBuffer, varHeader); err != nil {
				log.Println("error reading var header", err)
				break loop
			}
			//log.Println("var header", varHeader)

			protocolNameLen := binary.BigEndian.Uint16(varHeader[0:2])
			if protocolNameLen != 4 {
				log.Println("malformed packet: protocol name length is incorrect", protocolNameLen)
				break loop
			}

			protocolName := string(varHeader[2:6])
			if protocolName != "MQTT" {
				log.Println("malformed packet: unknown protocol name. expected MQTT found", protocolName)
				break loop
			}

			if !utils.ByteInSlice(varHeader[6], SUPPORTED_PROTOCOL_VERSIONS) {
				log.Println("unsupported protocol", varHeader[6])
				c.emit(MakeConnAckPacket(0, CONNECT_UNACCEPTABLE_PROTO))
				break loop
			}

			flagsBits := bytes.ByteToBinaryString(varHeader[7])
			connFlags := ExtractConnectFlags(flagsBits)
			if connFlags.Reserved != "0" {
				log.Println("malformed packet: reserved flag in CONNECT packet header is not zero", flagsBits[7:8])
				break loop
			}

			c.keepAliveSecs = int(binary.BigEndian.Uint16(varHeader[8:]))

			// payload parsing
			payload := make([]byte, payloadLen)
			if _, err = io.ReadFull(sockBuffer, payload); err != nil {
				log.Println("error reading payload", err)
				break loop
			}
			//log.Println("payload", payload)

			// client id parsing
			clientIdLen := int(binary.BigEndian.Uint16(payload[0:2])) // maximum is 65535 bytes
			if clientIdLen == 0 {
				if connFlags.CleanSession == "0" {
					log.Println("connect error: received zero byte client id with clean session flag set to 0")
					c.emit(MakeConnAckPacket(0, CONNECT_ID_REJECTED))
					break loop
				}
				c.ClientId = uuid.New().String()
			} else {
				if len(payload) < 2+clientIdLen {
					log.Println("malformed packet: payload length is not valid")
					break loop
				}
				c.ClientId = string(payload[2 : 2+clientIdLen])
			}

			// TODO: verify payload with flags (Client Identifier, Will Topic, Will Message, User Name, Password)

			// TODO: implement Session
			// TODO: implement Will [3.1.3]

			// TODO: implement keep alive check and disconnect on timeout of (1.5 * keepalive) as per spec [3.1.2.10]
			// TODO: implement username and password auth [3.1.3]

			// connection succeeded
			log.Println("client connected with id:", c.ClientId)
			GOTT.addClient(c)
			c.emit(MakeConnAckPacket(0, CONNECT_ACCEPTED)) // TODO: fix sessionPresent after implementing sessions

			go func(c *Client) {
				time.Sleep(time.Second * 2)
				//c.publish([]byte(""), []byte(""), 0, 0, 0)
			}(c)
		case TYPE_PUBLISH:
			publishFlags := ExtractPublishFlags(flagsBits)

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading publish packet", err)
				break loop
			}

			topicLen := int(binary.BigEndian.Uint16(remBytes[:2]))
			if topicLen == 0 {
				log.Println("received empty topic. disconnecting client.")
				break loop
			}

			if len(remBytes) < 2+topicLen {
				log.Println("malformed packet: topic length is not valid")
				break loop
			}

			topicEnd := 2 + topicLen
			topic := string(remBytes[2:topicEnd])
			var packetId uint16
			var packetIdBytes []byte

			varHeaderEnd := topicEnd

			if publishFlags.QoS != "00" { // QoS = 1 or 2
				packetIdBytes = remBytes[topicEnd : 2+topicEnd]
				packetId = binary.BigEndian.Uint16(packetIdBytes)
				varHeaderEnd += 2
			}

			payload := string(remBytes[varHeaderEnd:])

			log.Println("pub in ::", "dup:", publishFlags.DUP, "qos:", publishFlags.QoS, "retain:", publishFlags.Retain, "packetid:", packetId, "topic:", topic, "payload:", payload)

			if publishFlags.QoS == "01" { // QoS level 1
				// return a PUBACK
				c.emit(MakePubAckPacket(packetIdBytes))
			} else if publishFlags.QoS == "10" { // QoS level 2
				// return a PUBREC
				c.emit(MakePubRecPacket(packetIdBytes))
			}
			// TODO: implement topic filter
			// TODO: implement re-publishing to other clients
			// TODO: implement retention policy
		case TYPE_PUBACK:
			break
		case TYPE_PUBREC:
			break
		case TYPE_PUBREL:
			if flagsBits != "0010" { // as per [MQTT-3.6.1-1]
				log.Println("malformed PUBREL packet: flags bits != 0010")
				break loop
			}

			packetIdBytes := make([]byte, PUBREL_REM_LEN)
			if _, err = io.ReadFull(sockBuffer, packetIdBytes); err != nil {
				log.Println("error reading var header", err)
				break loop
			}

			//packetId := binary.BigEndian.Uint16(packetIdBytes)
			//log.Println("PUBREL in", packetId)
			c.emit(MakePubCompPacket(packetIdBytes))
		case TYPE_PUBCOMP:
			break
		case TYPE_SUBSCRIBE:
			if flagsBits != "0010" { // as per [MQTT-3.8.1-1]
				log.Println("malformed SUBSCRIBE packet: flags bits != 0010")
				break loop
			}

			if remLen < 3 {
				// TODO: handle protocol violation (See section 4.8 for information about handling errors)
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading SUBSCRIBE packet", err)
				break loop
			}

			packetIdBytes := remBytes[0:2]
			packetId := binary.BigEndian.Uint16(packetIdBytes)
			payload := remBytes[2:]

			if len(payload) < 3 { // 3 is used to make sure there are at least 2 bytes for topic length and 1 byte for topic name of at least 1 character (eg. 00 01 97)
				// TODO: handle protocol violation (See section 4.8 for information about handling errors)
			}

			filterList, err := ExtractTopicFilters(payload)
			if err != nil {
				log.Println("malformed SUBSCRIBE packet:", err)
				break loop
			}

			log.Printf("SUBSCRIBE in > id: %v - list: %#v", packetId, filterList)

			// TODO: implement SUB list in broker and process SUBs
			// NOTE: see [MQTT-3.8.4-3] when implementing the SUB list, retention policy and server > clients publish

			// NOTE: If a Server receives a SUBSCRIBE packet that contains multiple Topic Filters it MUST handle that packet as if it had received a sequence of multiple SUBSCRIBE packets, except that it combines their responses into a single SUBACK response [MQTT-3.8.4-4].
			c.emit(MakeSubAckPacket(packetIdBytes, filterList))
		}
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
