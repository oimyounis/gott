package gott

import (
	gob "bytes"
	"encoding/binary"
	"log"
	"math"
	"net"
	"sort"
	"sync"
	"time"
)

var (
	// 4 is MQTT v3.1.1
	supportedProtocolVersions = []byte{4}
)

// GOTT is a singleton used to save memory instead of keeping a reference inside each Client
var GOTT *Broker

// Broker is the main broker struct. Should not be used directly. Use the global GOTT var instead.
type Broker struct {
	address            string
	listener           net.Listener
	clients            map[string]*Client
	mutex              sync.RWMutex
	config             Config
	TopicFilterStorage *topicStorage
	MessageStore       *messageStore
	SessionStore       *sessionStore
}

// NewBroker initializes a new object of type Broker. You can either use the returned pointer or the global GOTT var.
// It takes an address to bind the connection to.
// Returns a pointer of type Broker which is also assigned to the global GOTT var and an error.
// It creates/opens an on-disk session store.
func NewBroker(address string) (*Broker, error) {
	GOTT = &Broker{
		address:            address,
		listener:           nil,
		clients:            map[string]*Client{},
		config:             NewConfig(),
		TopicFilterStorage: &topicStorage{},
		MessageStore:       newMessageStore(),
	}

	ss, err := loadSessionStore()
	if err != nil {
		return nil, err
	}
	GOTT.SessionStore = ss

	GOTT.bootstrapPlugins()

	return GOTT, nil
}

// Listen starts the broker and listens for new connections on the "address" provided earlier.
func (b *Broker) Listen() error {
	l, err := net.Listen("tcp", b.address)
	if err != nil {
		return err
	}
	defer l.Close()

	b.listener = l
	log.Println("Broker listening on " + b.listener.Addr().String())

	for {
		conn, err := b.listener.Accept()
		if err != nil {
			log.Printf("Couldn't accept connection: %v\n", err)
		} else {
			go b.handleConnection(conn)
		}
	}
}

func (b *Broker) addClient(client *Client) {
	b.mutex.RLock()
	if c, ok := b.clients[client.ClientID]; ok {
		// disconnect existing client
		log.Println("disconnecting existing client with id:", c.ClientID)
		c.closeConnection()
	}
	b.mutex.RUnlock()

	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.clients[client.ClientID] = client
}

func (b *Broker) removeClient(clientID string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	delete(b.clients, clientID)
}

func (b *Broker) handleConnection(conn net.Conn) {
	log.Printf("Accepted connection from %v", conn.RemoteAddr().String())

	c := &Client{
		connection: conn,
		connected:  true,
	}
	go c.listen()
}

// Subscribe receives a client, a filter and qos level to create or update a subscription.
func (b *Broker) Subscribe(client *Client, filter []byte, qos byte) {
	if !validFilter(filter) {
		return
	}

	segs := gob.Split(filter, TopicDelim)

	segsLen := len(segs)
	if segsLen == 0 {
		return
	}

	topLevel := segs[0]
	tl := b.TopicFilterStorage.find(topLevel)
	if tl == nil {
		tl = &topicLevel{Bytes: topLevel}
		b.TopicFilterStorage.addTopLevel(tl)
	}

	if segsLen == 1 {
		tl.createOrUpdateSubscription(client, qos)
	} else {
		tl.parseChildren(client, segs[1:], qos)
	}

	if topicNames := b.TopicFilterStorage.reverseMatch(filter); topicNames != nil {
		sort.SliceStable(topicNames, func(i, j int) bool { // spec REQUIRES topics to be "Ordered" by default
			return topicNames[i].RetainedMessage.Timestamp.Before(topicNames[j].RetainedMessage.Timestamp)
		})

		for _, topic := range topicNames {
			b.PublishRetained(topic.RetainedMessage, &subscription{
				Session: client.Session,
				QoS:     qos,
			})
		}
	}
}

// Unsubscribe receives a client and a filter to remove a subscription.
func (b *Broker) Unsubscribe(client *Client, filter []byte) {
	if !validFilter(filter) {
		return
	}

	segs := gob.Split(filter, TopicDelim)

	segsLen := len(segs)
	if segsLen == 0 {
		return
	}

	if tl := b.TopicFilterStorage.find(segs[0]); tl != nil {
		if segsLen == 1 {
			tl.DeleteSubscription(client, true)
			return
		}

		tl.traverseDelete(client, segs[1:])
	}
}

// UnsubscribeAll is used to remove all subscriptions of a client.
// Currently used when the Client disconnects.
func (b *Broker) UnsubscribeAll(client *Client) {
	for _, tl := range b.TopicFilterStorage.Filters {
		tl.DeleteSubscription(client, false)
		tl.traverseDeleteAll(client)
	}
}

// Retain stores a msg in a specific topic as a retained message.
func (b *Broker) Retain(msg *message, topic []byte) {
	if msg != nil && !validTopicName(msg.Topic) {
		return
	}

	segs := gob.Split(topic, TopicDelim)

	segsLen := len(segs)
	if segsLen == 0 {
		return
	}

	topLevel := segs[0]
	tl := b.TopicFilterStorage.find(topLevel)
	if tl == nil {
		tl = &topicLevel{Bytes: topLevel}
		b.TopicFilterStorage.addTopLevel(tl)
	}

	if segsLen == 1 && !gob.Equal(tl.Bytes, TopicSingleLevelWildcard) {
		tl.retain(msg)
		return
	}

	tl.parseChildrenRetain(msg, segs[1:])
}

// Publish sends out a payload to all clients with subscriptions on a provided topic given the passed publish flags.
func (b *Broker) Publish(topic, payload []byte, flags publishFlags) {
	// NOTE: the server never upgrades QoS levels, downgrades only when necessary as in Min(pub.QoS, sub.QoS)
	if !validTopicName(topic) {
		return
	}

	matches := b.TopicFilterStorage.match(topic)
	log.Println(string(topic), "matches", matches)

	if flags.Retain {
		if len(payload) != 0 {
			b.Retain(&message{
				Topic:     topic,
				Payload:   payload,
				QoS:       flags.QoS,
				Timestamp: time.Now(),
			}, topic)
		} else {
			b.Retain(nil, topic)
		}
	}

	for _, match := range matches {
		for _, sub := range match.Subscriptions {
			qos := byte(math.Min(float64(sub.QoS), float64(flags.QoS)))
			// dup is zero according to [MQTT-3.3.1.-1] and [MQTT-3.3.1-3]
			packet, packetID := makePublishPacket(topic, payload, 0, qos, 0)

			var msg *clientMessage

			if qos != 0 {
				msg = &clientMessage{
					Topic:   topic,
					Payload: payload,
					QoS:     qos,
					Retain:  0,
					client:  sub.Session.client,
					Status:  StatusUnacknowledged,
				}
			}

			if sub.Session.client != nil && sub.Session.client.connected {
				sub.Session.client.emit(packet)
				if qos != 0 {
					b.MessageStore.store(packetID, msg)
					go Retry(packetID, msg)
				}
			} else if !sub.Session.clean {
				if qos != 0 {
					sub.Session.storeMessage(packetID, msg)
				}
			}
		}
	}
}

// PublishRetained is used to publish retained messages to a subscribing client.
func (b *Broker) PublishRetained(msg *message, sub *subscription) {
	if msg == nil || sub == nil {
		return
	}

	if sub.Session.client != nil && sub.Session.client.connected {
		qosOut := byte(math.Min(float64(sub.QoS), float64(msg.QoS)))
		packet, packetID := makePublishPacket(msg.Topic, msg.Payload, 0, qosOut, 1)
		if qosOut != 0 {
			msg := &clientMessage{
				Topic:   msg.Topic,
				Payload: msg.Payload,
				QoS:     qosOut,
				Retain:  1,
				client:  sub.Session.client,
				Status:  StatusUnacknowledged,
			}
			b.MessageStore.store(packetID, msg)

			go Retry(packetID, msg)
		}
		sub.Session.client.emit(packet)
	}
}

// PublishToClient is used to publish a message with a provided packetId to a specific client.
func (b *Broker) PublishToClient(client *Client, packetID uint16, cm *clientMessage) {
	if client.connected {
		packet := makePublishPacketWithID(packetID, cm.Topic, cm.Payload, 0, cm.QoS, 0)
		if cm.QoS != 0 {
			cm.client = client
			cm.Retain = 0
			b.MessageStore.store(packetID, cm)

			if cm.Status == StatusUnacknowledged {
				client.emit(packet)
			}

			go Retry(packetID, cm)
		}
	}
}

// Retry will check for a msg's status and resend it if it hasn't been acknowledged within 20 seconds.
func Retry(packetID uint16, msg *clientMessage) {
	defer Recover(nil)
	for {
		time.Sleep(time.Second * 5)
		if msg != nil && msg.client != nil && msg.client.connected {
			switch msg.Status {
			case StatusPubackReceived, StatusPubcompReceived:
				return
			}
		} else {
			return
		}

		time.Sleep(time.Second * 15)

		if msg.client.connected {
			switch msg.Status {
			case StatusUnacknowledged:
				msg.client.emit(makePublishPacketWithID(packetID, msg.Topic, msg.Payload, 1, msg.QoS, msg.Retain))
			case StatusPubrecReceived:
				packetIDBytes := make([]byte, 2)
				binary.BigEndian.PutUint16(packetIDBytes, packetID)
				msg.client.emit(makePubRelPacket(packetIDBytes))
			case StatusPubrelReceived:
				packetIDBytes := make([]byte, 2)
				binary.BigEndian.PutUint16(packetIDBytes, packetID)
				msg.client.emit(makePubCompPacket(packetIDBytes))
			case StatusPubackReceived, StatusPubcompReceived:
				return
			}
		} else {
			return
		}
	}
}
