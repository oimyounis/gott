package gott

import "sync"

type PacketStore struct {
	packets map[uint16]*PacketStatus
	mutex   sync.RWMutex
}

func NewPacketStore() *PacketStore {
	return &PacketStore{packets: map[uint16]*PacketStatus{}}
}

func (ps *PacketStore) delete(packetId uint16) {
	ps.mutex.Lock()
	delete(ps.packets, packetId)
	ps.mutex.Unlock()
}

func (ps *PacketStore) Store(packetId uint16, msg *PacketStatus) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.packets[packetId] = msg
}

func (ps *PacketStore) Acknowledge(packetId uint16) {
	ps.delete(packetId)
}

func (ps *PacketStore) Get(packetId uint16) *PacketStatus {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	if p, ok := ps.packets[packetId]; ok {
		return p
	} else {
		return nil
	}
}
