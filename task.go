package main

import "sync"

// TimerTask represents a task to be executed after a delay.
type TimerTask struct {
	delay int64 // delay in milliseconds
	f     func()

	entry *TimerTaskEntry
	mu    sync.RWMutex
}

func newTimerTask(delay int64, f func()) *TimerTask {
	return &TimerTask{
		delay: delay,
		f:     f,
	}
}

// Cancel cancels the timer task so it will not be executed.
func (t *TimerTask) Cancel() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.entry != nil {
		t.entry.remove()
		t.entry = nil
	}
}

func (t *TimerTask) setEntry(entry *TimerTaskEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// If there is an existing entry, remove it from its bucket.
	if t.entry != nil && t.entry != entry {
		t.entry.remove()
	}
	t.entry = entry
}

func (t *TimerTask) getEntry() *TimerTaskEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.entry
}
