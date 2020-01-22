package gott

import (
	"bufio"
	"encoding/binary"
	"gott/bytes"
	"gott/utils"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"
)

// Client is the main struct for every client that connects to GOTT.
// Holds all the info needed to process its messages and maintain state.
type Client struct {
	connection           net.Conn
	connected            bool
	keepAliveSecs        int
	lastPacketReceivedOn time.Time
	gracefulDisconnect   bool
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
			//log.Println("fixedHeader read error", err)
			break
		}

		lastByte := fixedHeader[1]
		remLenEncoded := []byte{lastByte}

		for lastByte >= 128 {
			lastByte, err = sockBuffer.ReadByte()
			if err != nil {
				//log.Println("read error", err)
				GOTT.Logger.Error("malformed", zap.String("reason", "rem len parsing"),
					zap.Error(err))
				break loop
			}
			remLenEncoded = append(remLenEncoded, lastByte)
		}

		packetType, flagsBits := parseFixedHeaderFirstByte(fixedHeader[0])
		remLen, err := bytes.Decode(remLenEncoded)

		GOTT.Stats.bytesIn(1 + int64(len(remLenEncoded)) + int64(remLen))

		if err != nil {
			log.Println("malformed packet", err)
			GOTT.Logger.Error("malformed", zap.String("reason", "rem len decoding"))
			break loop
		}

		c.lastPacketReceivedOn = time.Now()

		switch packetType {
		case TypeConnect:
			payloadLen := remLen - ConnectVarHeaderLen
			if payloadLen == 0 {
				log.Println("connect error: payload len is zero")
				GOTT.Logger.Error("malformed", zap.String("reason", "connect error: payload len is zero"))
				break loop
			}

			varHeader := make([]byte, ConnectVarHeaderLen)
			if _, err = io.ReadFull(sockBuffer, varHeader); err != nil {
				log.Println("error reading var header", err)
				GOTT.Logger.Error("malformed", zap.String("reason", "connect error: reading var header"),
					zap.Error(err))
				break loop
			}

			protocolNameLen := binary.BigEndian.Uint16(varHeader[0:2])
			if protocolNameLen != 4 {
				log.Println("malformed packet: protocol name length is incorrect", protocolNameLen)
				GOTT.Logger.Error("malformed", zap.String("reason", "connect error: protocol name length is incorrect"))
				break loop
			}

			protocolName := string(varHeader[2:6])
			if protocolName != "MQTT" {
				log.Println("malformed packet: unknown protocol name. expected MQTT found", protocolName)
				GOTT.Logger.Error("malformed", zap.String("reason", "unknown protocol name. expected MQTT found "+protocolName))
				break loop
			}

			if !utils.ByteInSlice(varHeader[6], supportedProtocolVersions) {
				log.Println("unsupported protocol", varHeader[6])
				GOTT.Logger.Error("malformed", zap.String("reason", "unsupported protocol "+strconv.Itoa(int(varHeader[6]))))
				c.emit(makeConnAckPacket(0, ConnectUnacceptableProto))
				break loop
			}

			connFlags, err := extractConnectFlags(varHeader[7])
			if err != nil {
				log.Println("malformed packet: ", err)
				GOTT.Logger.Error("malformed", zap.String("reason", "connect flags extraction"),
					zap.Error(err))
				break loop
			}

			c.keepAliveSecs = int(binary.BigEndian.Uint16(varHeader[8:]))

			// payload parsing
			payload := make([]byte, payloadLen)
			if _, err = io.ReadFull(sockBuffer, payload); err != nil {
				log.Println("error reading payload", err)
				GOTT.Logger.Error("malformed", zap.String("reason", "reading payload"), zap.Error(err))
				break loop
			}

			head := 0

			// connect flags parsing
			clientIDLen := int(binary.BigEndian.Uint16(payload[head:2])) // maximum client ID length is 65535 bytes
			head += 2
			if clientIDLen == 0 {
				if !connFlags.CleanSession {
					log.Println("connect error: received zero byte client id with clean session flag set to 0")
					GOTT.Logger.Error("malformed", zap.String("reason", "connect error: received zero byte client id with clean session flag set to 0"))
					c.emit(makeConnAckPacket(0, ConnectIDRejected))
					break loop
				}
				c.ClientID = uuid.New().String()
			} else {
				if payloadLen < 2+clientIDLen {
					log.Println("malformed packet: payload length is not valid")
					GOTT.Logger.Error("malformed", zap.String("reason", "payload length is not valid (clientID)"))
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
					GOTT.Logger.Error("malformed", zap.String("reason", "payload length is not valid (UserNameFlag)"))
					break loop
				}
				usernameLen := int(binary.BigEndian.Uint16(payload[head : head+2]))
				head += 2
				if usernameLen == 0 {
					break loop
				}
				if payloadLen < head+usernameLen {
					log.Println("malformed packet: payload length is not valid")
					GOTT.Logger.Error("malformed", zap.String("reason", "payload length is not valid (usernameLen)"))
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
					GOTT.Logger.Error("malformed", zap.String("reason", "payload length is not valid (PasswordFlag)"))
					break loop
				}
				passwordLen := int(binary.BigEndian.Uint16(payload[head : head+2]))
				head += 2
				if passwordLen == 0 {
					break loop
				}
				if payloadLen < head+passwordLen {
					log.Println("malformed packet: payload length is not valid")
					GOTT.Logger.Error("malformed", zap.String("reason", "payload length is not valid (passwordLen)"))
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

				//log.Printf("session for id: %s, session: %#v", c.ClientID, c.Session)
			} else {
				if err := c.Session.put(); err != nil {
					log.Println("error putting session to store:", err)
					GOTT.Logger.Error("malformed", zap.String("reason", "error putting session to store"),
						zap.Error(err))
					break loop
				}
			}

			// TODO: implement keep alive check and disconnect on timeout of (1.5 * keepalive) as per spec [3.1.2.10]

			// connection succeeded
			//log.Println("client connected with id:", c.ClientID)
			GOTT.addClient(c)
			c.emit(makeConnAckPacket(sessionPresent, ConnectAccepted))

			c.Session.replay()

			if !GOTT.invokeOnConnect(c.ClientID, c.Username, c.Password) {
				break loop
			}

			GOTT.Logger.Info("device connected", zap.String("id", c.ClientID), zap.String("username", c.Username), zap.Bool("cleanSession", connFlags.CleanSession))
		case TypePublish:
			publishFlags, err := extractPublishFlags(flagsBits)
			if err != nil {
				log.Println("error reading publish packet", err)
				GOTT.Logger.Error("malformed", zap.String("reason", "reading publish packet"))
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading publish packet", err)
				GOTT.Logger.Error("malformed", zap.String("reason", "reading publish packet"),
					zap.Error(err))
				break loop
			}

			topicLen := int(binary.BigEndian.Uint16(remBytes[:2]))
			if topicLen == 0 {
				log.Println("received empty topic. disconnecting client.")
				GOTT.Logger.Error("malformed", zap.String("reason", "received empty topic"))
				break loop
			}

			if len(remBytes) < 2+topicLen {
				log.Println("malformed packet: topic length is not valid")
				GOTT.Logger.Error("malformed", zap.String("reason", "topic length is not valid"))
				break loop
			}

			topicEnd := 2 + topicLen
			topic := remBytes[2:topicEnd]

			if !validTopicName(topic) {
				log.Println("malformed packet: invalid topic name")
				GOTT.Logger.Error("malformed", zap.String("reason", "invalid topic name"))
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

			GOTT.Stats.received(1)

			GOTT.invokeOnMessage(c.ClientID, c.Username, topic, payload, publishFlags.DUP, publishFlags.QoS, publishFlags.Retain)

			if !GOTT.invokeOnBeforePublish(c.ClientID, c.Username, topic, payload, publishFlags.DUP, publishFlags.QoS, publishFlags.Retain) {
				break
			}

			if GOTT.Publish(topic, payload, publishFlags) {
				GOTT.invokeOnPublish(c.ClientID, c.Username, topic, payload, publishFlags.DUP, publishFlags.QoS, false)

				GOTT.Logger.Info("publish", zap.ByteString("topic", topic), zap.ByteString("payload", payload), zap.Int("qos", int(publishFlags.QoS)))
			}
		case TypePubAck:
			if remLen != 2 {
				log.Println("malformed PUBACK packet: invalid remaining length")
				GOTT.Logger.Error("malformed", zap.String("reason", "PUBACK packet: invalid remaining length"))
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading PUBACK packet", err)
				GOTT.Logger.Error("malformed", zap.String("reason", "reading PUBACK packet"),
					zap.Error(err))
				break loop
			}

			var packetID uint16
			var packetIDBytes []byte

			packetIDBytes = remBytes
			packetID = binary.BigEndian.Uint16(packetIDBytes)

			GOTT.MessageStore.acknowledge(packetID, StatusPubackReceived, true)
			c.Session.acknowledge(packetID, StatusPubackReceived, true)

			GOTT.Logger.Debug("PUBACK", zap.Uint16("packetID", packetID))
		case TypePubRec:
			if remLen != 2 {
				log.Println("malformed PUBREC packet: invalid remaining length")
				GOTT.Logger.Error("malformed", zap.String("reason", "PUBREC packet: invalid remaining length"))
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading PUBREC packet", err)
				GOTT.Logger.Error("malformed", zap.String("reason", "reading PUBREC packet"),
					zap.Error(err))
				break loop
			}

			var packetID uint16
			var packetIDBytes []byte

			packetIDBytes = remBytes
			packetID = binary.BigEndian.Uint16(packetIDBytes)

			GOTT.MessageStore.acknowledge(packetID, StatusPubrecReceived, false)
			c.Session.acknowledge(packetID, StatusPubrecReceived, false)
			c.emit(makePubRelPacket(packetIDBytes))

			GOTT.Logger.Debug("PUBREC", zap.Uint16("packetID", packetID))
		case TypePubRel:
			if flagsBits != "0010" { // as per [MQTT-3.6.1-1]
				log.Println("malformed PUBREL packet: flags bits != 0010")
				GOTT.Logger.Error("malformed", zap.String("reason", "PUBREL packet: flags bits != 0010"))
				break loop
			}

			packetIDBytes := make([]byte, PubrelRemLen)
			if _, err = io.ReadFull(sockBuffer, packetIDBytes); err != nil {
				log.Println("error reading var header", err)
				GOTT.Logger.Error("malformed", zap.String("reason", "PUBREL packet: reading var header"), zap.Error(err))
				break loop
			}

			packetID := binary.BigEndian.Uint16(packetIDBytes)

			c.Session.MessageStore.acknowledge(packetID, StatusPubrelReceived, true)
			c.emit(makePubCompPacket(packetIDBytes))

			GOTT.Logger.Debug("PUBREL", zap.Uint16("packetID", packetID))
		case TypePubComp:
			if remLen != 2 {
				log.Println("malformed PUBCOMP packet: invalid remaining length")
				GOTT.Logger.Error("malformed", zap.String("reason", "PUBCOMP packet: invalid remaining length"))
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading PUBCOMP packet", err)
				GOTT.Logger.Error("malformed", zap.String("reason", "reading PUBCOMP packet"), zap.Error(err))
				break loop
			}

			var packetID uint16
			var packetIDBytes []byte

			packetIDBytes = remBytes
			packetID = binary.BigEndian.Uint16(packetIDBytes)

			GOTT.MessageStore.acknowledge(packetID, StatusPubcompReceived, true)
			c.Session.acknowledge(packetID, StatusPubcompReceived, true)

			GOTT.Logger.Debug("PUBCOMP", zap.Uint16("packetID", packetID))
		case TypeSubscribe:
			if flagsBits != "0010" { // as per [MQTT-3.8.1-1]
				log.Println("malformed SUBSCRIBE packet: flags bits != 0010")
				GOTT.Logger.Error("malformed", zap.String("reason", "SUBSCRIBE packet: flags bits != 0010"))
				break loop
			}

			if remLen < 3 {
				log.Println("malformed SUBSCRIBE packet: remLen < 3")
				GOTT.Logger.Error("malformed", zap.String("reason", "SUBSCRIBE packet: remLen < 3"))
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading SUBSCRIBE packet", err)
				GOTT.Logger.Error("malformed", zap.String("reason", "reading SUBSCRIBE packet"), zap.Error(err))
				break loop
			}

			packetIDBytes := remBytes[0:2]
			//packetID := binary.BigEndian.Uint16(packetIDBytes)
			payload := remBytes[2:]

			if len(payload) < 3 { // 3 is used to make sure there are at least 2 bytes for topic length and 1 byte for topic name of at least 1 character (eg. 00 01 97)
				log.Println("malformed SUBSCRIBE packet: remLen < 3")
				GOTT.Logger.Error("malformed", zap.String("reason", "SUBSCRIBE packet: remLen < 3"))
				break loop
			}

			filterList, err := extractSubTopicFilters(payload)
			if err != nil {
				log.Println("malformed SUBSCRIBE packet:", err)
				GOTT.Logger.Error("malformed", zap.String("reason", "SUBSCRIBE packet"), zap.Error(err))
				break loop
			}

			// NOTE: If a Server receives a SUBSCRIBE packet that contains multiple Topic Filters it MUST handle that packet as if it had received a sequence of multiple SUBSCRIBE packets, except that it combines their responses into a single SUBACK response [MQTT-3.8.4-4].

			for _, filter := range filterList {
				if !GOTT.invokeOnBeforeSubscribe(c.ClientID, c.Username, filter.Filter, filter.QoS) {
					continue
				}

				if GOTT.Subscribe(c, filter.Filter, filter.QoS) {
					GOTT.Stats.subscription(1)
					GOTT.invokeOnSubscribe(c.ClientID, c.Username, filter.Filter, filter.QoS)
					GOTT.Logger.Info("subscribe", zap.String("clientID", c.ClientID), zap.ByteString("filter", filter.Filter), zap.Int("qos", int(filter.QoS)))
				}
			}

			c.emit(makeSubAckPacket(packetIDBytes, filterList))
		case TypeUnsubscribe:
			if flagsBits != "0010" { // as per [MQTT-3.10.1-1]
				log.Println("malformed UNSUBSCRIBE packet: flags bits != 0010")
				GOTT.Logger.Error("malformed", zap.String("reason", "UNSUBSCRIBE packet: flags bits != 0010"))
				break loop
			}

			if remLen < 3 {
				break loop
			}

			remBytes := make([]byte, remLen)
			if _, err := io.ReadFull(sockBuffer, remBytes); err != nil {
				log.Println("error reading UNSUBSCRIBE packet", err)
				GOTT.Logger.Error("malformed", zap.String("reason", "reading UNSUBSCRIBE packet"), zap.Error(err))
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
				GOTT.Logger.Error("malformed", zap.String("reason", "UNSUBSCRIBE packet"), zap.Error(err))
				break loop
			}

			for _, filter := range filterList {
				if !GOTT.invokeOnBeforeUnsubscribe(c.ClientID, c.Username, filter) {
					continue
				}

				if GOTT.Unsubscribe(c, filter) {
					GOTT.invokeOnUnsubscribe(c.ClientID, c.Username, filter)
					GOTT.Logger.Info("unsubscribe", zap.String("clientID", c.ClientID), zap.ByteString("filter", filter))
				}
			}

			c.emit(makeUnSubAckPacket(packetIDBytes))
		case TypePingReq:
			c.emit(makePingRespPacket())
		case TypeDisconnect:
			c.WillMessage = nil // as per [MQTT-3.1.2-10]
			c.gracefulDisconnect = true
			break loop
		default:
			log.Println("UNKNOWN PACKET TYPE")
			GOTT.Logger.Error("malformed", zap.String("reason", "UNKNOWN PACKET TYPE"))
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

	//log.Printf("client id %s was disconnected", c.ClientID)

	GOTT.UnsubscribeAll(c)

	if c.WillMessage != nil {
		if GOTT.invokeOnBeforePublish(c.ClientID, c.Username, c.WillMessage.Topic, c.WillMessage.Payload, 0, c.WillMessage.QoS, c.WillMessage.Retain) {
			if GOTT.Publish(c.WillMessage.Topic, c.WillMessage.Payload, publishFlags{
				Retain: c.WillMessage.Retain,
				QoS:    c.WillMessage.QoS,
			}) {
				GOTT.invokeOnPublish(c.ClientID, c.Username, c.WillMessage.Topic, c.WillMessage.Payload, 0, c.WillMessage.QoS, false)
			}
		}

	}

	if connected {
		GOTT.invokeOnDisconnect(c.ClientID, c.Username, c.gracefulDisconnect)

		GOTT.Logger.Info("client disconnected", zap.String("id", c.ClientID), zap.Bool("graceful", c.gracefulDisconnect))
	}
}

func (c *Client) closeConnection() {
	c.connected = false
	_ = c.connection.Close()
}

func (c *Client) emit(packet []byte) {
	if _, err := c.connection.Write(packet); err != nil {
		GOTT.Logger.Error("error sending packet", zap.Error(err))
		return
	}
	if packet[0]>>4 == TypePublish {
		GOTT.Stats.sent(1)
	}
	GOTT.Stats.bytesOut(int64(len(packet)))
}
