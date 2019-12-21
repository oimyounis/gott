package gott

import (
	"sync"
)

type ClientMessage struct {
	client         *Client
	Topic, Payload []byte
	QoS, Retain    byte
	Status         byte
}

func (cm *ClientMessage) Client() *Client {
	return cm.client
}

type MessageStore struct {
	Messages map[uint16]*ClientMessage
	mutex    sync.RWMutex
}

func NewMessageStore() *MessageStore {
	return &MessageStore{Messages: map[uint16]*ClientMessage{}}
}

func (ms *MessageStore) delete(packetId uint16) {
	ms.mutex.Lock()
	delete(ms.Messages, packetId)
	ms.mutex.Unlock()
}

func (ms *MessageStore) Store(packetId uint16, msg *ClientMessage) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	ms.Messages[packetId] = msg
}

func (ms *MessageStore) Acknowledge(packetId uint16, status byte, delete bool) {
	if cm := ms.Get(packetId); cm != nil {
		cm.Status = status

		if delete {
			ms.delete(packetId)
		}
	}
}

func (ms *MessageStore) Get(packetId uint16) *ClientMessage {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	if m, ok := ms.Messages[packetId]; ok {
		return m
	} else {
		return nil
	}
}
