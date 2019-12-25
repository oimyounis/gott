package gott

import (
	"log"
	"time"
)

type Session struct {
	client       *Client
	clean        bool
	Id           string
	MessageStore *MessageStore
}

func NewSession(client *Client, cleanFlag string) *Session {
	return &Session{
		client:       client,
		clean:        cleanFlag == "1",
		MessageStore: NewMessageStore(),
		Id:           client.ClientId,
	}
}

func (s *Session) Load() error {
	start := time.Now()
	err := GOTT.SessionStore.Get(s.Id, s)
	end := time.Since(start)
	LogBench("session load took:", end)
	return err
}

func (s *Session) Update(value map[string]interface{}) error {
	return GOTT.SessionStore.Set(s.Id, value)
}

func (s *Session) Put() error {
	start := time.Now()
	err := GOTT.SessionStore.Set(s.Id, s)
	end := time.Since(start)
	LogBench("session put took:", end)
	return err
}

func (s *Session) Client() *Client {
	return s.client
}

func (s *Session) Clean() bool {
	return s.clean
}

func (s *Session) StoreMessage(packetId uint16, msg *ClientMessage) {
	s.MessageStore.Store(packetId, msg)
	if !s.clean {
		_ = s.Put()
	}
}

func (s *Session) Acknowledge(packetId uint16, status byte, delete bool) {
	s.MessageStore.Acknowledge(packetId, status, delete)
	if !s.clean {
		_ = s.Put()
	}
}

func (s *Session) Replay() {
	s.MessageStore.RangeSorted(func(packetId uint16, cm *ClientMessage) bool {
		log.Println(packetId, string(cm.Topic), string(cm.Payload))
		return true
	})
}
