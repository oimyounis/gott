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
	SessionStore       *SessionStore
}

func NewBroker(address string) (*Broker, error) {
	GOTT = &Broker{
		address:            address,
		listener:           nil,
		clients:            map[string]*Client{},
		TopicFilterStorage: &TopicStorage{},
		MessageStore:       NewMessageStore(),
	}

	ss, err := LoadSessionStore()
	if err != nil {
		return nil, err
	}
	GOTT.SessionStore = ss

	return GOTT, nil
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
		connection: conn,
		connected:  true,
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
		tl = &TopicLevel{Bytes: topLevel}
		b.TopicFilterStorage.AddTopLevel(tl)
	}

	if segsLen == 1 {
		tl.CreateOrUpdateSubscription(client, qos)
		return
	}

	tl.ParseChildren(client, segs[1:], qos)

	if topicNames := b.TopicFilterStorage.ReverseMatch(filter); topicNames != nil {
		sort.SliceStable(topicNames, func(i, j int) bool { // spec REQUIRES topics to be "Ordered" by default
			return topicNames[i].RetainedMessage.Timestamp.Before(topicNames[j].RetainedMessage.Timestamp)
		})

		for _, topic := range topicNames {
			b.PublishRetained(topic.RetainedMessage, &Subscription{
				Session: client.Session,
				QoS:     qos,
			})
		}
	}
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
			tl.DeleteSubscription(client, true)
			return
		}

		tl.TraverseDelete(client, segs[1:])
	}
}

// TODO: re-implement as this traverses the whole topic tree and all subscriptions (need to optimize this)
func (b *Broker) UnsubscribeAll(client *Client) {
	for _, tl := range b.TopicFilterStorage.Filters {
		tl.DeleteSubscription(client, false)
		tl.TraverseDeleteAll(client)
	}
}

func (b *Broker) Retain(msg *Message, topic []byte) {
	if msg != nil && !ValidTopicName(msg.Topic) {
		return
	}

	segs := gob.Split(topic, TOPIC_DELIM)

	segsLen := len(segs)
	if segsLen == 0 {
		return
	}

	topLevel := segs[0]
	tl := b.TopicFilterStorage.Find(topLevel)
	if tl == nil {
		tl = &TopicLevel{Bytes: topLevel}
		b.TopicFilterStorage.AddTopLevel(tl)
	}

	if segsLen == 1 && !gob.Equal(tl.Bytes, TOPIC_SINGLE_LEVEL_WILDCARD) {
		tl.Retain(msg)
		return
	}

	tl.ParseChildrenRetain(msg, segs[1:])
}

func (b *Broker) Publish(topic, payload []byte, flags PublishFlags) {
	// NOTE: the server never upgrades QoS levels, downgrades only when necessary as in Min(pub.QoS, sub.QoS)
	if !ValidTopicName(topic) {
		return
	}

	matches := b.TopicFilterStorage.Match(topic)
	log.Println(string(topic), "matches", matches)

	//if len(matches) == 0 || !FilterInSlice(topic, matches) {
	if flags.Retain == 1 {
		if len(payload) != 0 {
			b.Retain(&Message{
				Topic:     topic,
				Payload:   payload,
				QoS:       flags.QoS,
				Timestamp: time.Now(),
			}, topic)
		} else {
			b.Retain(nil, topic)
		}
	}
	//}

	for _, match := range matches {
		//if flags.Retain == 1 {
		//	if len(payload) != 0 {
		//		match.Retain(&Message{
		//			Topic:   topic,
		//			Payload: payload,
		//			QoS:     flags.QoS,
		//		})
		//	} else {
		//		match.Retain(nil)
		//	}
		//}
		for _, sub := range match.Subscriptions {
			qos := byte(math.Min(float64(sub.QoS), float64(flags.QoS)))
			// dup is zero according to [MQTT-3.3.1.-1] and [MQTT-3.3.1-3]
			packet, packetId := MakePublishPacket(topic, payload, 0, qos, 0)

			var msg *ClientMessage

			if qos != 0 {
				msg = &ClientMessage{
					Topic:   topic,
					Payload: payload,
					QoS:     qos,
					Retain:  0,
					client:  sub.Session.client,
					Status:  STATUS_UNACKNOWLEDGED,
				}
			}

			if sub.Session.client != nil && sub.Session.client.connected {
				sub.Session.client.emit(packet)
				if qos != 0 {
					b.MessageStore.Store(packetId, msg)
					go Retry(packetId, msg)
				}
			} else if !sub.Session.clean {
				if qos != 0 {
					sub.Session.StoreMessage(packetId, msg)
				}
			}
		}
	}
}

func (b *Broker) PublishToClient(c *Client, topic, payload []byte, flags PublishFlags) {

}

func (b *Broker) PublishRetained(msg *Message, sub *Subscription) {
	if msg == nil || sub == nil {
		return
	}

	//log.Println("sending retained msg", string(msg.Topic), string(msg.Payload))

	if sub.Session.client != nil && sub.Session.client.connected {
		qosOut := byte(math.Min(float64(sub.QoS), float64(msg.QoS)))
		packet, packetId := MakePublishPacket(msg.Topic, msg.Payload, 0, qosOut, 1)
		if qosOut != 0 {
			msg := &ClientMessage{
				Topic:   msg.Topic,
				Payload: msg.Payload,
				QoS:     qosOut,
				Retain:  1,
				client:  sub.Session.client,
				Status:  STATUS_UNACKNOWLEDGED,
			}
			b.MessageStore.Store(packetId, msg)

			go Retry(packetId, msg)
		}
		sub.Session.client.emit(packet)
	}
}

func Retry(packetId uint16, msg *ClientMessage) {
	defer Recover(nil)
	for {
		time.Sleep(time.Second * 5)
		if msg != nil && msg.client != nil && msg.client.connected {
			switch msg.Status {
			case STATUS_PUBACK_RECEIVED, STATUS_PUBCOMP_RECEIVED:
				return
			}
		} else {
			return
		}

		time.Sleep(time.Second * 15)

		if msg != nil && msg.client != nil && msg.client.connected {
			switch msg.Status {
			case STATUS_UNACKNOWLEDGED:
				msg.client.emit(MakePublishPacketWithId(packetId, msg.Topic, msg.Payload, 1, msg.QoS, msg.Retain))
			case STATUS_PUBREC_RECEIVED:
				packetIdBytes := make([]byte, 2)
				binary.BigEndian.PutUint16(packetIdBytes, packetId)
				msg.client.emit(MakePubRelPacket(packetIdBytes))
			case STATUS_PUBREL_RECEIVED:
				packetIdBytes := make([]byte, 2)
				binary.BigEndian.PutUint16(packetIdBytes, packetId)
				msg.client.emit(MakePubCompPacket(packetIdBytes))
			case STATUS_PUBACK_RECEIVED, STATUS_PUBCOMP_RECEIVED:
				return
			}
		} else {
			return
		}
	}
}
