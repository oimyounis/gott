package gott

import (
	"fmt"
	"gott/utils"
	"sync/atomic"
	"time"
)

type brokerStats struct {
	ReceivedMessagesCount  int64     `json:"received_messages_count"`
	SentMessagesCount      int64     `json:"sent_messages_count"`
	SubscriptionCount      int64     `json:"subscription_count"`
	BytesInCount           int64     `json:"bytes_in"`
	BytesOutCount          int64     `json:"bytes_out_count"`
	ConnectedClientsCount  int64     `json:"connected_clients_count"`
	RetainedMessagesCount  int64     `json:"retained_messages_count"`
	AllTimeConnectionCount int64     `json:"all_time_connection_count"`
	SessionsOnDiskCount    int64     `json:"sessions_on_disk_count"`
	Started                time.Time `json:"time_started"`
	Uptime                 int64     `json:"uptime"`
}

func (s *brokerStats) received(delta int64) {
	atomic.AddInt64(&s.ReceivedMessagesCount, delta)
	if s.ReceivedMessagesCount < 0 {
		s.ReceivedMessagesCount = 0
	}
}

func (s *brokerStats) sent(delta int64) {
	atomic.AddInt64(&s.SentMessagesCount, delta)
	if s.SentMessagesCount < 0 {
		s.SentMessagesCount = 0
	}
}

func (s *brokerStats) subscription(delta int64) {
	atomic.AddInt64(&s.SubscriptionCount, delta)
	if s.SubscriptionCount < 0 {
		s.SubscriptionCount = 0
	}
}

func (s *brokerStats) bytesIn(delta int64) {
	atomic.AddInt64(&s.BytesInCount, delta)
	if s.BytesInCount < 0 {
		s.BytesInCount = 0
	}
}

func (s *brokerStats) bytesOut(delta int64) {
	atomic.AddInt64(&s.BytesOutCount, delta)
	if s.BytesOutCount < 0 {
		s.BytesOutCount = 0
	}
}

func (s *brokerStats) connectedClients(delta int64) {
	atomic.AddInt64(&s.ConnectedClientsCount, delta)
	if s.ConnectedClientsCount < 0 {
		s.ConnectedClientsCount = 0
	}
	s.maxConnectedClients(s.ConnectedClientsCount)
}

func (s *brokerStats) maxConnectedClients(count int64) {
	if atomic.LoadInt64(&s.AllTimeConnectionCount) < count {
		atomic.StoreInt64(&s.AllTimeConnectionCount, count)
	}
}

func (s *brokerStats) uptime() time.Duration {
	return time.Since(s.Started).Round(time.Second)
}

func (s *brokerStats) retained(delta int64) {
	atomic.AddInt64(&s.RetainedMessagesCount, delta)
	if s.RetainedMessagesCount < 0 {
		s.RetainedMessagesCount = 0
	}
}

func (s *brokerStats) sessionOnDisk(delta int64) {
	atomic.AddInt64(&s.SessionsOnDiskCount, delta)
	if s.SessionsOnDiskCount < 0 {
		s.SessionsOnDiskCount = 0
	}
}

func (s *brokerStats) Reset() {
	s.ReceivedMessagesCount = 0
	s.SentMessagesCount = 0
	s.BytesInCount = 0
	s.BytesOutCount = 0
}

func (s *brokerStats) JSON() []byte {
	s.Uptime = int64(s.uptime().Seconds())
	return utils.ToJSONBytes(s, "{}")
}

func (s *brokerStats) String() string {
	return fmt.Sprintf("\u001B[2J\u001B[0;0HBroker Stats:\nMax Connected Clients: %v\nConnected Clients: %v\nSessions On Disk: %v\nReceived Messages: %v\nSent Messages: %v\nSubscriptions: %v\nBytes In: %v\nBytes Out: %v\nUptime: %v\n", s.AllTimeConnectionCount, s.ConnectedClientsCount, s.SessionsOnDiskCount, s.ReceivedMessagesCount, s.SentMessagesCount, s.SubscriptionCount, s.BytesInCount, s.BytesOutCount, s.uptime())
}

func (s *brokerStats) StartMonitor() {
	go func() {
		for {
			fmt.Printf(s.String())
			time.Sleep(time.Second)
		}
	}()
}
