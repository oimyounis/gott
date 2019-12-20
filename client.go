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
	connection           net.Conn
	connected            bool
	keepAliveSecs        int
	lastPacketReceivedOn time.Time
	ClientId             string
	MessageStore         *MessageStore
	WillMessage          *Message
}

func (c *Client) listen() {
	if GOTT == nil {
		return
	}

	defer Recover(func(c *Client) func(string, string) {
		return func(err, stack string) {
			c.disconnect()
		}
	}(c))

	sockBuffer := bufio.NewReader(c.connection)

loop:
	for {
		if !c.connected {
			break
		}

		fixedHeader := make([]byte, 2)

		_, err := sockBuffer.Read(fixedHeader)
		if err != nil {
			log.Println("fixedHeader read error", err)
			break
		}
		//log.Println("read", fixedHeader)

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

		c.lastPacketReceivedOn = time.Now()

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

			head := 0

			// client id parsing
			clientIdLen := int(binary.BigEndian.Uint16(payload[head:2])) // maximum client ID length is 65535 bytes
			head += 2
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
				c.ClientId = string(payload[head : head+clientIdLen])
				head += clientIdLen
			}

			if connFlags.WillFlag == "1" {
				willTopicLen := int(binary.BigEndian.Uint16(payload[head : head+2]))
				head += 2
				if willTopicLen == 0 {
					break loop
				}
				willTopic := payload[head : head+willTopicLen]
				head += willTopicLen
				if len(willTopic) == 0 {
					break loop
				}

				willPayloadLen := int(binary.BigEndian.Uint16(payload[head : head+2]))
				head += 2
				if willPayloadLen == 0 {
					break loop
				}
				willPayload := payload[head : head+willPayloadLen]
				head += willPayloadLen
				if len(willPayload) == 0 {
					break loop
				}

				c.WillMessage = &Message{
					Topic:   willTopic,
					Payload: willPayload,
				}

				log.Println("will:", string(c.WillMessage.Topic), "-", string(c.WillMessage.Payload))
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
		case TYPE_PUBLISH:
			publishFlags, err := ExtractPublishFlags(flagsBits)
			if err != nil {
				log.Println("error reading publish packet", err)
				break loop
			}

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
			topic := remBytes[2:topicEnd]

			if !ValidTopicName(topic) {
				log.Println("malformed packet: invalid topic name")
				break loop
			}

			var packetId uint16
			var packetIdBytes []byte

			varHeaderEnd := topicEnd

			if publishFlags.QoS != 0 {
				packetIdBytes = remBytes[topicEnd : 2+topicEnd]
				packetId = binary.BigEndian.Uint16(packetIdBytes)
				varHeaderEnd += 2
				publishFlags.PacketId = packetId
			}

			payload := remBytes[varHeaderEnd:]

			//log.Println("pub in ::", "dup:", publishFlags.DUP, "qos:", publishFlags.QoS, "retain:", publishFlags.Retain, "packetid:", packetId, "topic:", topic, "payload:", payload)

			if publishFlags.QoS == 1 {
				// return a PUBACK
				c.emit(MakePubAckPacket(packetIdBytes))
			} else if publishFlags.QoS == 2 {
				// return a PUBREC
				if publishFlags.DUP == 1 {
					if msg := c.MessageStore.Get(packetId); msg != nil {
						c.emit(MakePubRecPacket(packetIdBytes))
						break // skip resending message
					}
				}
				c.MessageStore.Store(packetId, &ClientMessage{
					Topic:   topic,
					Payload: payload,
					QoS:     publishFlags.QoS,
					Status:  STATUS_PUBREC_RECEIVED,
				})
				c.emit(MakePubRecPacket(packetIdBytes))
			}

			GOTT.Publish(topic, payload, publishFlags)

			GOTT.TopicFilterStorage.Print()
		case TYPE_PUBACK:
			if remLen != 2 {
				log.Println("malformed PUBACK packet: invalid remaining length")
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading PUBACK packet", err)
				break loop
			}

			var packetId uint16
			var packetIdBytes []byte

			packetIdBytes = remBytes
			packetId = binary.BigEndian.Uint16(packetIdBytes)

			GOTT.MessageStore.Acknowledge(packetId, STATUS_PUBACK_RECEIVED, true)
		case TYPE_PUBREC:
			if remLen != 2 {
				log.Println("malformed PUBREC packet: invalid remaining length")
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading PUBREC packet", err)
				break loop
			}

			var packetId uint16
			var packetIdBytes []byte

			packetIdBytes = remBytes
			packetId = binary.BigEndian.Uint16(packetIdBytes)

			GOTT.MessageStore.Acknowledge(packetId, STATUS_PUBREC_RECEIVED, false)
			c.emit(MakePubRelPacket(packetIdBytes))
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

			packetId := binary.BigEndian.Uint16(packetIdBytes)

			c.MessageStore.Acknowledge(packetId, STATUS_PUBREL_RECEIVED, true)
			c.emit(MakePubCompPacket(packetIdBytes))
		case TYPE_PUBCOMP:
			if remLen != 2 {
				log.Println("malformed PUBCOMP packet: invalid remaining length")
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading PUBCOMP packet", err)
				break loop
			}

			var packetId uint16
			var packetIdBytes []byte

			packetIdBytes = remBytes
			packetId = binary.BigEndian.Uint16(packetIdBytes)

			GOTT.MessageStore.Acknowledge(packetId, STATUS_PUBCOMP_RECEIVED, true)
		case TYPE_SUBSCRIBE:
			if flagsBits != "0010" { // as per [MQTT-3.8.1-1]
				log.Println("malformed SUBSCRIBE packet: flags bits != 0010")
				break loop
			}

			if remLen < 3 {
				log.Println("malformed SUBSCRIBE packet: remLen < 3")
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading SUBSCRIBE packet", err)
				break loop
			}

			packetIdBytes := remBytes[0:2]
			//packetId := binary.BigEndian.Uint16(packetIdBytes)
			payload := remBytes[2:]

			if len(payload) < 3 { // 3 is used to make sure there are at least 2 bytes for topic length and 1 byte for topic name of at least 1 character (eg. 00 01 97)
				log.Println("malformed SUBSCRIBE packet: remLen < 3")
				break loop
			}

			filterList, err := ExtractSubTopicFilters(payload)
			if err != nil {
				log.Println("malformed SUBSCRIBE packet:", err)
				break loop
			}

			// NOTE: see [MQTT-3.8.4-3] when implementing the SUB list, retention policy and server > clients publish

			// NOTE: If a Server receives a SUBSCRIBE packet that contains multiple Topic Filters it MUST handle that packet as if it had received a sequence of multiple SUBSCRIBE packets, except that it combines their responses into a single SUBACK response [MQTT-3.8.4-4].
			c.emit(MakeSubAckPacket(packetIdBytes, filterList))

			for _, filter := range filterList {
				GOTT.Subscribe(c, filter.Filter, filter.QoS)
			}
		case TYPE_UNSUBSCRIBE:
			if flagsBits != "0010" { // as per [MQTT-3.10.1-1]
				log.Println("malformed UNSUBSCRIBE packet: flags bits != 0010")
				break loop
			}

			if remLen < 3 {
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading UNSUBSCRIBE packet", err)
				break loop
			}

			packetIdBytes := remBytes[0:2]
			//packetId := binary.BigEndian.Uint16(packetIdBytes)
			payload := remBytes[2:]

			if len(payload) < 3 { // 3 is used to make sure there are at least 2 bytes for topic length and 1 byte for topic name of at least 1 character (eg. 00 01 97)
				break loop
			}

			filterList, err := ExtractUnSubTopicFilters(payload)
			if err != nil {
				log.Println("malformed UNSUBSCRIBE packet:", err)
				break loop
			}

			for _, filter := range filterList {
				GOTT.Unsubscribe(c, filter)
			}

			c.emit(MakeUnSubAckPacket(packetIdBytes))
		case TYPE_PINGREQ:
			log.Printf("PINGREQ in > %v", fixedHeader)
			c.emit(MakePingRespPacket())
		case TYPE_DISCONNECT:
			c.WillMessage = nil
			break loop
		default:
			log.Println("UNKNOWN PACKET TYPE")
			break loop
		}

		//log.Printf("last packet on %v", c.lastPacketReceivedOn)
	}
	c.disconnect()
}

func (c *Client) disconnect() {
	if GOTT == nil {
		return
	}

	c.closeConnection()
	GOTT.removeClient(c.ClientId)

	log.Printf("client id %s was disconnected", c.ClientId)

	GOTT.UnsubscribeAll(c)

	if c.WillMessage != nil {
		GOTT.Publish(c.WillMessage.Topic, c.WillMessage.Payload, PublishFlags{})
	}
}

func (c *Client) closeConnection() {
	c.connected = false
	_ = c.connection.Close()
}

func (c *Client) emit(packet []byte) {
	if _, err := c.connection.Write(packet); err != nil {
		log.Println("error sending packet", err, packet)
	}
	//log.Println("emit out <", c.ClientId, packet)
}
