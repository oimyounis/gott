package gott

import (
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	json "github.com/json-iterator/go"
)

type brokerStats struct {
	receivedCount, sentCount, subscriptionCount, bytesInCount, bytesOutCount, connectedClientsCount int64
	started                                                                                         time.Time
}

func (s *brokerStats) received(delta int64) {
	atomic.AddInt64(&s.receivedCount, delta)
	if s.receivedCount < 0 {
		s.receivedCount = 0
	}
}

func (s *brokerStats) sent(delta int64) {
	atomic.AddInt64(&s.sentCount, delta)
	if s.sentCount < 0 {
		s.sentCount = 0
	}
}

func (s *brokerStats) subscription(delta int64) {
	atomic.AddInt64(&s.subscriptionCount, delta)
	if s.subscriptionCount < 0 {
		s.subscriptionCount = 0
	}
}

func (s *brokerStats) bytesIn(delta int64) {
	atomic.AddInt64(&s.bytesInCount, delta)
	if s.bytesInCount < 0 {
		s.bytesInCount = 0
	}
}

func (s *brokerStats) bytesOut(delta int64) {
	atomic.AddInt64(&s.bytesOutCount, delta)
	if s.bytesOutCount < 0 {
		s.bytesOutCount = 0
	}
}

func (s *brokerStats) connectedClients(delta int64) {
	atomic.AddInt64(&s.connectedClientsCount, delta)
	if s.connectedClientsCount < 0 {
		s.connectedClientsCount = 0
	}
}

func (s *brokerStats) uptime() time.Duration {
	return time.Since(s.started).Round(time.Second)
}

func (s *brokerStats) Reset() {
	s.receivedCount = 0
	s.sentCount = 0
	s.bytesInCount = 0
	s.bytesOutCount = 0
}

func (s *brokerStats) String() string {
	return fmt.Sprintf(`Broker Stats:
  Received Messages: %v
  Sent Messages: %v
  Subscriptions: %v
  Bytes In: %v
  Bytes Out: %v
  Connected Clients: %v
  Uptime: %v`, atomic.LoadInt64(&s.receivedCount), atomic.LoadInt64(&s.sentCount), atomic.LoadInt64(&s.subscriptionCount), atomic.LoadInt64(&s.bytesInCount), atomic.LoadInt64(&s.bytesOutCount), atomic.LoadInt64(&s.connectedClientsCount), s.uptime())
}

func (s *brokerStats) Json() []byte {
	stats := map[string]int64{
		"received":      atomic.LoadInt64(&s.receivedCount),
		"sent":          atomic.LoadInt64(&s.sentCount),
		"subscriptions": atomic.LoadInt64(&s.subscriptionCount),
		"bytesIn":       atomic.LoadInt64(&s.bytesInCount),
		"bytesOut":      atomic.LoadInt64(&s.bytesOutCount),
		"clients":       atomic.LoadInt64(&s.connectedClientsCount),
		"uptime":        int64(s.uptime().Seconds()),
	}
	b, err := json.Marshal(stats)
	if err != nil {
		GOTT.Logger.Debug("broker stats JSON marshalling failed", zap.Error(err))
		return []byte{}
	}
	return b
}

func (s *brokerStats) StartMonitor() {
	go func() {
		for {
			fmt.Print("\033[2J\033[0;0H", s.String())
			time.Sleep(time.Second)
		}
	}()
}
