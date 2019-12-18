package gott

import (
	gob "bytes"
	"encoding/binary"
	"log"
	"math"
	"net"
	"sync"
	"time"
)

var (
	// 4 is MQTT v3.1.1
	SUPPORTED_PROTOCOL_VERSIONS = []byte{4}
)

var GOTT *Broker

type Broker struct {
	address            string
	listener           net.Listener
	clients            map[string]*Client
	mutex              sync.RWMutex
	TopicFilterStorage *TopicStorage
	MessageStore       *MessageStore
}

func NewBroker(address string) *Broker {
	GOTT = &Broker{
		address:            address,
		listener:           nil,
		clients:            map[string]*Client{},
		TopicFilterStorage: &TopicStorage{},
		MessageStore:       NewMessageStore(),
	}
	return GOTT
}

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
	//return nil
}

func (b *Broker) addClient(client *Client) {
	b.mutex.RLock()
	if c, ok := b.clients[client.ClientId]; ok {
		// disconnect existing client
		log.Println("disconnecting existing client with id:", c.ClientId)
		c.closeConnection()
	}
	b.mutex.RUnlock()

	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.clients[client.ClientId] = client
}

func (b *Broker) removeClient(clientId string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	delete(b.clients, clientId)
}

func (b *Broker) handleConnection(conn net.Conn) {
	log.Printf("Accepted connection from %v", conn.RemoteAddr().String())
	//client := b.addClient(conn)
	c := &Client{
		connection:   conn,
		connected:    true,
		MessageStore: NewMessageStore(),
	}
	go c.listen()
}

func (b *Broker) Subscribe(client *Client, filter []byte, qos byte) {
	if !ValidFilter(filter) {
		return
	}

	segs := gob.Split(filter, TOPIC_DELIM)

	segsLen := len(segs)
	if segsLen == 0 {
		return
	}

	topLevel := segs[0]
	tl := b.TopicFilterStorage.Find(topLevel)
	if tl == nil {
		tl = &TopicLevel{bytes: topLevel}
		b.TopicFilterStorage.AddTopLevel(tl)
	}

	if segsLen == 1 {
		tl.CreateOrUpdateSubscription(client, qos)
		return
	}

	tl.ParseChildren(client, segs[1:], qos)
}

func (b *Broker) Unsubscribe(client *Client, filter []byte) {
	if !ValidFilter(filter) {
		return
	}

	segs := gob.Split(filter, TOPIC_DELIM)

	segsLen := len(segs)
	if segsLen == 0 {
		return
	}

	if tl := b.TopicFilterStorage.Find(segs[0]); tl != nil {
		if segsLen == 1 {
			tl.DeleteSubscription(client)
			return
		}

		tl.TraverseDelete(client, segs[1:])
	}
}

// TODO: re-implement as this traverses the whole topic tree and all subscriptions (need to optimize this)
func (b *Broker) UnsubscribeAll(client *Client) {
	for _, tl := range b.TopicFilterStorage.filters {
		tl.DeleteSubscription(client)
		tl.TraverseDeleteAll(client)
	}
}

func (b *Broker) Publish(topic, payload []byte, flags PublishFlags) {
	// NOTE: the server never upgrades QoS levels, downgrades only when necessary as in Min(pub.QoS, sub.QoS)
	if !ValidTopicName(topic) {
		return
	}

	matches := b.TopicFilterStorage.Match(topic)
	log.Println(string(topic), "matches", matches)

	for _, match := range matches {
		for _, sub := range match.Subscriptions {
			if sub.Client.connected {
				qos := byte(math.Min(float64(sub.QoS), float64(flags.QoS)))
				// dup is zero according to [MQTT-3.3.1.-1] and [MQTT-3.3.1-3]
				packet, packetId := MakePublishPacket(topic, payload, 0, qos, 0) // TODO: fix retainFlag
				if qos != 0 {
					msg := &ClientMessage{
						Topic:   topic,
						Payload: payload,
						QoS:     qos,
						Client:  sub.Client,
						Status:  STATUS_UNACKNOWLEDGED,
					}
					b.MessageStore.Store(packetId, msg)

					go Retry(packetId, msg)
				}
				sub.Client.emit(packet)
			}
		}
	}
}

func Retry(packetId uint16, msg *ClientMessage) {
	defer Recover()
	for {
		time.Sleep(time.Second * 5)
		if msg != nil && msg.Client != nil && msg.Client.connected {
			switch msg.Status {
			case STATUS_PUBACK_RECEIVED, STATUS_PUBCOMP_RECEIVED:
				return
			}
		} else {
			return
		}

		time.Sleep(time.Second * 15)

		if msg != nil && msg.Client != nil && msg.Client.connected {
			switch msg.Status {
			case STATUS_UNACKNOWLEDGED:
				msg.Client.emit(MakePublishPacketWithId(packetId, msg.Topic, msg.Payload, 1, msg.QoS, 0)) // TODO: fix retainFlag
			case STATUS_PUBREC_RECEIVED:
				packetIdBytes := make([]byte, 2)
				binary.BigEndian.PutUint16(packetIdBytes, packetId)
				msg.Client.emit(MakePubRelPacket(packetIdBytes))
			case STATUS_PUBREL_RECEIVED:
				packetIdBytes := make([]byte, 2)
				binary.BigEndian.PutUint16(packetIdBytes, packetId)
				msg.Client.emit(MakePubCompPacket(packetIdBytes))
			case STATUS_PUBACK_RECEIVED, STATUS_PUBCOMP_RECEIVED:
				return
			}
		} else {
			return
		}
	}
}
