package gott

import (
	"sort"
	"sync"
)

type clientMessage struct {
	client         *Client
	Topic, Payload []byte
	QoS, Retain    byte
	Status         byte
}

func (cm *clientMessage) Client() *Client {
	return cm.client
}

type messageStore struct {
	Messages map[uint16]*clientMessage
	mutex    sync.RWMutex
}

func newMessageStore() *messageStore {
	return &messageStore{Messages: map[uint16]*clientMessage{}}
}

func (ms *messageStore) delete(packetID uint16) {
	ms.mutex.Lock()
	delete(ms.Messages, packetID)
	ms.mutex.Unlock()
}

func (ms *messageStore) store(packetID uint16, msg *clientMessage) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	ms.Messages[packetID] = msg
}

func (ms *messageStore) acknowledge(packetID uint16, status byte, delete bool) {
	if cm := ms.get(packetID); cm != nil {
		cm.Status = status

		if delete {
			ms.delete(packetID)
		}
	}
}

func (ms *messageStore) get(packetID uint16) *clientMessage {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	if m, ok := ms.Messages[packetID]; ok {
		return m
	}
	return nil
}

// Range iterates over the underlying message store.
func (ms *messageStore) Range(iterator func(packetID uint16, cm *clientMessage) bool) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	for packetID, cm := range ms.Messages {
		if next := iterator(packetID, cm); !next {
			return
		}
	}
}

// RangeSorted iterates over the underlying message store after sorting by key.
func (ms *messageStore) RangeSorted(iterator func(packetID uint16, cm *clientMessage) bool) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	keys := make([]uint16, 0)
	for k := range ms.Messages {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	for _, packetID := range keys {
		if next := iterator(packetID, ms.Messages[packetID]); !next {
			return
		}
	}
}
