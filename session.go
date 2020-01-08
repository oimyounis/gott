package gott

type session struct {
	client       *Client
	clean        bool
	ID           string
	MessageStore *messageStore
}

func newSession(client *Client, cleanFlag bool) *session {
	return &session{
		client:       client,
		clean:        cleanFlag,
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
	if !s.clean {
		_ = s.put()
	}
}

func (s *session) acknowledge(packetID uint16, status byte, delete bool) {
	s.MessageStore.acknowledge(packetID, status, delete)
	if !s.clean {
		_ = s.put()
	}
}

func (s *session) replay() {
	if s.clean {
		return
	}

	if s.client != nil {
		s.MessageStore.RangeSorted(func(packetId uint16, cm *clientMessage) bool {
			GOTT.PublishToClient(s.client, packetId, cm)
			return true
		})
	}
}
