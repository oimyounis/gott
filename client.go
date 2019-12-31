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

// Client is the main struct for every client that connects to GOTT.
// Holds all the info needed to process its messages and maintain state.
type Client struct {
	connection           net.Conn
	connected            bool
	keepAliveSecs        int
	lastPacketReceivedOn time.Time
	ClientID             string
	WillMessage          *message
	Username, Password   string
	Session              *session
}

func (c *Client) listen() {
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

		packetType, flagsBits := parseFixedHeaderFirstByte(fixedHeader[0])
		remLen, err := bytes.Decode(remLenEncoded)
		if err != nil {
			log.Println("malformed packet", err)
			break loop
		}

		c.lastPacketReceivedOn = time.Now()

		switch packetType {
		case TypeConnect:
			payloadLen := remLen - ConnectVarHeaderLen
			if payloadLen == 0 {
				log.Println("connect error: payload len is zero")
				break loop
			}

			varHeader := make([]byte, ConnectVarHeaderLen)
			if _, err = io.ReadFull(sockBuffer, varHeader); err != nil {
				log.Println("error reading var header", err)
				break loop
			}

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

			if !utils.ByteInSlice(varHeader[6], supportedProtocolVersions) {
				log.Println("unsupported protocol", varHeader[6])
				c.emit(makeConnAckPacket(0, ConnectUnacceptableProto))
				break loop
			}

			connFlags, err := extractConnectFlags(varHeader[7])
			if err != nil {
				log.Println("malformed packet: ", err)
				break loop
			}

			c.keepAliveSecs = int(binary.BigEndian.Uint16(varHeader[8:]))

			// payload parsing
			payload := make([]byte, payloadLen)
			if _, err = io.ReadFull(sockBuffer, payload); err != nil {
				log.Println("error reading payload", err)
				break loop
			}

			head := 0

			// connect flags parsing
			clientIDLen := int(binary.BigEndian.Uint16(payload[head:2])) // maximum client ID length is 65535 bytes
			head += 2
			if clientIDLen == 0 {
				if !connFlags.CleanSession {
					log.Println("connect error: received zero byte client id with clean session flag set to 0")
					c.emit(makeConnAckPacket(0, ConnectIDRejected))
					break loop
				}
				c.ClientID = uuid.New().String()
			} else {
				if payloadLen < 2+clientIDLen {
					log.Println("malformed packet: payload length is not valid")
					break loop
				}
				c.ClientID = string(payload[head : head+clientIDLen])
				head += clientIDLen
			}

			if connFlags.WillFlag {
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

				c.WillMessage = &message{
					Topic:   willTopic,
					Payload: willPayload,
					QoS:     connFlags.WillQoS,
					Retain:  connFlags.WillRetain,
				}
			}

			if connFlags.UserNameFlag {
				if payloadLen < head+2 {
					log.Println("malformed packet: payload length is not valid")
					break loop
				}
				usernameLen := int(binary.BigEndian.Uint16(payload[head : head+2]))
				head += 2
				if usernameLen == 0 {
					break loop
				}
				if payloadLen < head+usernameLen {
					log.Println("malformed packet: payload length is not valid")
					break loop
				}
				username := payload[head : head+usernameLen]
				head += usernameLen
				if len(username) == 0 {
					break loop
				}
				c.Username = string(username)
			}

			if connFlags.PasswordFlag {
				if payloadLen < head+2 {
					log.Println("malformed packet: payload length is not valid")
					break loop
				}
				passwordLen := int(binary.BigEndian.Uint16(payload[head : head+2]))
				head += 2
				if passwordLen == 0 {
					break loop
				}
				if payloadLen < head+passwordLen {
					log.Println("malformed packet: payload length is not valid")
					break loop
				}
				password := payload[head : head+passwordLen]
				head += passwordLen
				if len(password) == 0 {
					break loop
				}
				c.Password = string(password)
			}

			// Invoke OnBeforeConnect handlers of all plugins before initializing sessions
			if !GOTT.invokeOnBeforeConnect(c.ClientID, c.Username, c.Password) {
				break loop
			}

			var sessionPresent byte

			c.Session = newSession(c, connFlags.CleanSession)

			if connFlags.CleanSession {
				_ = GOTT.SessionStore.delete(c.ClientID) // as per [MQTT-3.1.2-6]
			} else if GOTT.SessionStore.exists(c.ClientID) {
				sessionPresent = 1
				if err := c.Session.load(); err != nil {
					// try to delete stored session in case it was malformed
					_ = GOTT.SessionStore.delete(c.ClientID)
				}

				log.Printf("session for id: %s, session: %#v", c.ClientID, c.Session)
			} else {
				if err := c.Session.put(); err != nil {
					log.Println("error putting session to store:", err)
					break loop
				}
			}

			// TODO: implement keep alive check and disconnect on timeout of (1.5 * keepalive) as per spec [3.1.2.10]

			// connection succeeded
			log.Println("client connected with id:", c.ClientID)
			GOTT.addClient(c)
			c.emit(makeConnAckPacket(sessionPresent, ConnectAccepted))

			c.Session.replay()

			if !GOTT.invokeOnConnect(c.ClientID, c.Username, c.Password) {
				break loop
			}
		case TypePublish:
			publishFlags, err := extractPublishFlags(flagsBits)
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

			if !validTopicName(topic) {
				log.Println("malformed packet: invalid topic name")
				break loop
			}

			var packetID uint16
			var packetIDBytes []byte

			varHeaderEnd := topicEnd

			if publishFlags.QoS != 0 {
				packetIDBytes = remBytes[topicEnd : 2+topicEnd]
				packetID = binary.BigEndian.Uint16(packetIDBytes)
				varHeaderEnd += 2
				publishFlags.PacketID = packetID
			}

			payload := remBytes[varHeaderEnd:]

			if publishFlags.QoS == 1 {
				// return a PUBACK
				c.emit(makePubAckPacket(packetIDBytes))
			} else if publishFlags.QoS == 2 {
				// return a PUBREC
				if publishFlags.DUP == 1 {
					if msg := c.Session.MessageStore.get(packetID); msg != nil {
						c.emit(makePubRecPacket(packetIDBytes))
						break // skip resending message
					}
				}
				c.Session.MessageStore.store(packetID, &clientMessage{
					Topic:   topic,
					Payload: payload,
					QoS:     publishFlags.QoS,
					Status:  StatusPubrecReceived,
				})
				c.emit(makePubRecPacket(packetIDBytes))
			}

			if !GOTT.invokeOnBeforePublish(c.ClientID, c.Username, topic, payload, publishFlags.DUP, publishFlags.QoS, false) {
				break
			}

			if ok := GOTT.Publish(topic, payload, publishFlags); ok {
				GOTT.invokeOnPublish(c.ClientID, c.Username, topic, payload, publishFlags.DUP, publishFlags.QoS, false)
			}
		case TypePubAck:
			if remLen != 2 {
				log.Println("malformed PUBACK packet: invalid remaining length")
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading PUBACK packet", err)
				break loop
			}

			var packetID uint16
			var packetIDBytes []byte

			packetIDBytes = remBytes
			packetID = binary.BigEndian.Uint16(packetIDBytes)

			GOTT.MessageStore.acknowledge(packetID, StatusPubackReceived, true)
			c.Session.acknowledge(packetID, StatusPubackReceived, true)
		case TypePubRec:
			if remLen != 2 {
				log.Println("malformed PUBREC packet: invalid remaining length")
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading PUBREC packet", err)
				break loop
			}

			var packetID uint16
			var packetIDBytes []byte

			packetIDBytes = remBytes
			packetID = binary.BigEndian.Uint16(packetIDBytes)

			GOTT.MessageStore.acknowledge(packetID, StatusPubrecReceived, false)
			c.Session.acknowledge(packetID, StatusPubrecReceived, false)
			c.emit(makePubRelPacket(packetIDBytes))
		case TypePubRel:
			if flagsBits != "0010" { // as per [MQTT-3.6.1-1]
				log.Println("malformed PUBREL packet: flags bits != 0010")
				break loop
			}

			packetIDBytes := make([]byte, PubrelRemLen)
			if _, err = io.ReadFull(sockBuffer, packetIDBytes); err != nil {
				log.Println("error reading var header", err)
				break loop
			}

			packetID := binary.BigEndian.Uint16(packetIDBytes)

			c.Session.MessageStore.acknowledge(packetID, StatusPubrelReceived, true)
			c.emit(makePubCompPacket(packetIDBytes))
		case TypePubComp:
			if remLen != 2 {
				log.Println("malformed PUBCOMP packet: invalid remaining length")
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading PUBCOMP packet", err)
				break loop
			}

			var packetID uint16
			var packetIDBytes []byte

			packetIDBytes = remBytes
			packetID = binary.BigEndian.Uint16(packetIDBytes)

			GOTT.MessageStore.acknowledge(packetID, StatusPubcompReceived, true)
			c.Session.acknowledge(packetID, StatusPubcompReceived, true)
		case TypeSubscribe:
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

			packetIDBytes := remBytes[0:2]
			//packetID := binary.BigEndian.Uint16(packetIDBytes)
			payload := remBytes[2:]

			if len(payload) < 3 { // 3 is used to make sure there are at least 2 bytes for topic length and 1 byte for topic name of at least 1 character (eg. 00 01 97)
				log.Println("malformed SUBSCRIBE packet: remLen < 3")
				break loop
			}

			filterList, err := extractSubTopicFilters(payload)
			if err != nil {
				log.Println("malformed SUBSCRIBE packet:", err)
				break loop
			}

			// NOTE: If a Server receives a SUBSCRIBE packet that contains multiple Topic Filters it MUST handle that packet as if it had received a sequence of multiple SUBSCRIBE packets, except that it combines their responses into a single SUBACK response [MQTT-3.8.4-4].

			for _, filter := range filterList {
				if !GOTT.invokeOnBeforeSubscribe(c.ClientID, c.Username, filter.Filter, filter.QoS) {
					continue
				}

				if ok := GOTT.Subscribe(c, filter.Filter, filter.QoS); ok {
					GOTT.invokeOnSubscribe(c.ClientID, c.Username, filter.Filter, filter.QoS)
				}
			}

			c.emit(makeSubAckPacket(packetIDBytes, filterList))
		case TypeUnsubscribe:
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

			packetIDBytes := remBytes[0:2]
			//packetID := binary.BigEndian.Uint16(packetIDBytes)
			payload := remBytes[2:]

			if len(payload) < 3 { // 3 is used to make sure there are at least 2 bytes for topic length and 1 byte for topic name of at least 1 character (eg. 00 01 97)
				break loop
			}

			filterList, err := extractUnSubTopicFilters(payload)
			if err != nil {
				log.Println("malformed UNSUBSCRIBE packet:", err)
				break loop
			}

			for _, filter := range filterList {
				if !GOTT.invokeOnBeforeUnsubscribe(c.ClientID, c.Username, filter) {
					continue
				}

				GOTT.Unsubscribe(c, filter)

				GOTT.invokeOnUnsubscribe(c.ClientID, c.Username, filter)
			}

			c.emit(makeUnSubAckPacket(packetIDBytes))
		case TypePingReq:
			c.emit(makePingRespPacket())
		case TypeDisconnect:
			c.WillMessage = nil // as per [MQTT-3.1.2-10]
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

	connected := c.connected

	c.closeConnection()
	GOTT.removeClient(c.ClientID)

	log.Printf("client id %s was disconnected", c.ClientID)

	GOTT.UnsubscribeAll(c)

	if c.WillMessage != nil {
		// TODO: see [MQTT-3.1.2-10] after implementing sessions
		GOTT.Publish(c.WillMessage.Topic, c.WillMessage.Payload, publishFlags{
			Retain: c.WillMessage.Retain,
			QoS:    c.WillMessage.QoS,
		})
	}

	if connected {
		GOTT.invokeOnDisconnect(c.ClientID, c.Username)
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
}
