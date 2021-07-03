package gott

import "sync"

type atomicBool struct {
	val   bool
	mutex sync.Mutex
}

func (ab *atomicBool) Load() bool {
	ab.mutex.Lock()
	defer ab.mutex.Unlock()
	return ab.val
}

func (ab *atomicBool) Store(val bool) {
	ab.mutex.Lock()
	defer ab.mutex.Unlock()
	ab.val = val
}

type subscriptionList struct {
	Subs  []*subscription `json:"subs"`
	mutex sync.Mutex
}

func (s *subscriptionList) delete(index int) {
	var newSubs []*subscription
	newSubs = append(newSubs, s.Subs[:index]...)
	if index != len(s.Subs)-1 {
		newSubs = append(newSubs, s.Subs[index+1:]...)
	}
	s.Subs = newSubs
	GOTT.Stats.subscription(-1)
}

func (s *subscriptionList) Len() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.Subs)
}

func (s *subscriptionList) Add(sub *subscription) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Subs = append(s.Subs, sub)
	GOTT.Stats.subscription(1)
}

func (s *subscriptionList) Delete(index int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.delete(index)
}

func (s *subscriptionList) Range(iterator func(i int, sub *subscription) bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for i, sub := range s.Subs {
		if next := iterator(i, sub); !next {
			break
		}
	}
}

func (s *subscriptionList) RangeDelete(iterator func(i int, sub *subscription, delete func(index int)) bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for i, sub := range s.Subs {
		if next := iterator(i, sub, s.delete); !next {
			break
		}
	}
}
