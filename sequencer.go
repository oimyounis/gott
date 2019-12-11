package gott

import "sync"

type Sequencer struct {
	current        int64
	mutex          sync.Mutex
	UpperBoundBits uint
	Start          int64
}

func (seq *Sequencer) Next() int64 {
	seq.mutex.Lock()
	defer seq.mutex.Unlock()
	if seq.current == 1<<seq.UpperBoundBits-1 || seq.current == 0 {
		seq.current = seq.Start
	}

	current := seq.current
	seq.current++
	return current
}

func (seq *Sequencer) Val() int64 {
	if seq.current == 0 {
		return seq.current
	}
	return seq.current - 1
}
