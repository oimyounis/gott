package gott

import (
	"time"
)

type Session struct {
	client       *Client
	MessageStore *MessageStore
	clean        bool
}

func NewSession(client *Client, cleanFlag string) *Session {
	return &Session{
		client:       client,
		MessageStore: NewMessageStore(),
		clean:        cleanFlag == "1",
	}
}

func (s *Session) Load() error {
	start := time.Now()
	err := GOTT.SessionStore.Get(s.client.ClientId, s)
	Log("[BENCHMARK]", "session load took:", time.Since(start))
	return err
}

func (s *Session) Update(value map[string]interface{}) error {
	return GOTT.SessionStore.Set(s.client.ClientId, value)
}

func (s *Session) Put() error {
	start := time.Now()
	err := GOTT.SessionStore.Set(s.client.ClientId, s)
	Log("[BENCHMARK]", "session put took:", time.Since(start))
	return err
}

func (s *Session) Client() *Client {
	return s.client
}

func (s *Session) Clean() bool {
	return s.clean
}
