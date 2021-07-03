package gott

import (
	"bytes"

	"go.uber.org/zap"
)

type session struct {
	Client        *Client       `json:"-"`
	Clean         bool          `json:"clean"`
	ID            string        `json:"client_id"`
	MessageStore  *messageStore `json:"message_store"`
	Subscriptions []*filter     `json:"subscriptions"`
}

func newSession(client *Client, cleanFlag bool) *session {
	return &session{
		Client:       client,
		Clean:        cleanFlag,
		MessageStore: newMessageStore(),
		ID:           client.ClientID,
	}
}

func (s *session) load() error {
	//start := time.Now()
	err := GOTT.SessionStore.get(s.ID, s)
	//end := time.Since(start)
	//LogBench("session load took:", end)
	return err
}

func (s *session) put() error {
	//start := time.Now()
	err := GOTT.SessionStore.set(s.ID, s)
	//end := time.Since(start)
	//LogBench("session put took:", end)
	return err
}

func (s *session) storeMessage(packetID uint16, msg *clientMessage) {
	s.MessageStore.store(packetID, msg)
	if !s.Clean {
		_ = s.put()
	}
}

func (s *session) acknowledge(packetID uint16, status int32, delete bool) {
	s.MessageStore.acknowledge(packetID, status, delete)
	if !s.Clean {
		_ = s.put()
	}
}

func (s *session) addSubscription(topicFilter filter) bool {
	if !s.Clean {
		for _, sub := range s.Subscriptions {
			if bytes.Equal(sub.Filter, topicFilter.Filter) {
				sub.QoS = topicFilter.QoS
				return true
			}
		}
		s.Subscriptions = append(s.Subscriptions, &topicFilter)
		return true
	}
	return false
}

func (s *session) replay() {
	if s.Clean {
		return
	}

	if s.Client != nil {
		s.MessageStore.RangeSorted(func(packetId uint16, cm *clientMessage) bool {
			GOTT.PublishToClient(s.Client, packetId, cm)
			return true
		})

		for _, sub := range s.Subscriptions {
			if !GOTT.invokeOnBeforeSubscribe(s.Client.ClientID, s.Client.Username, sub.Filter, sub.QoS) {
				continue
			}

			if GOTT.Subscribe(s.Client, sub.Filter, sub.QoS) {
				GOTT.invokeOnSubscribe(s.Client.ClientID, s.Client.Username, sub.Filter, sub.QoS)
				GOTT.Logger.Info("subscribe on session load", zap.String("clientID", s.Client.ClientID), zap.ByteString("filter", sub.Filter), zap.Int("qos", int(sub.QoS)))
			}
		}
	}
}
