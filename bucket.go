package timerWheel

import (
	"sync"
	"sync/atomic"
)

// Bucket is a doubly-linked list of TimerTaskEntry nodes.
// It represents one slot in the timing wheel and implements the Delayed interface
// so it can be inserted into the delay queue.
type Bucket struct {
	expiration atomic.Int64
	mu         sync.Mutex
	root       TimerTaskEntry // sentinel node
}

func newBucket() *Bucket {
	b := &Bucket{}
	b.expiration.Store(-1)
	b.root.prev = &b.root
	b.root.next = &b.root
	return b
}

// Expiration implements the delayqueue.Delayed interface.
func (b *Bucket) Expiration() int64 {
	return b.expiration.Load()
}

// SetExpiration sets the bucket's expiration time.
// Returns true if the expiration was changed (bucket needs to be re-inserted into the delay queue).
func (b *Bucket) SetExpiration(exp int64) bool {
	return b.expiration.Swap(exp) != exp
}

// Add appends a timer task entry to this bucket.
func (b *Bucket) Add(entry *TimerTaskEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if entry.bucket != nil && entry.bucket != b {
		entry.remove()
	}

	entry.bucket = b
	// Insert before the root sentinel (i.e., at the tail).
	tail := b.root.prev
	entry.prev = tail
	entry.next = &b.root
	tail.next = entry
	b.root.prev = entry
}

// Remove removes a timer task entry from this bucket.
func (b *Bucket) Remove(entry *TimerTaskEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if entry.bucket == b {
		entry.prev.next = entry.next
		entry.next.prev = entry.prev
		entry.bucket = nil
		entry.prev = nil
		entry.next = nil
	}
}

// Flush removes all entries from this bucket and calls the reinsert function for each.
// This is used when advancing the timing wheel — entries are moved to a lower-level wheel or executed.
func (b *Bucket) Flush(reinsert func(*TimerTaskEntry)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry := b.root.next
	for entry != &b.root {
		next := entry.next

		// Unlink from this bucket.
		entry.prev = nil
		entry.next = nil
		entry.bucket = nil

		reinsert(entry)

		entry = next
	}

	// Reset sentinel.
	b.root.prev = &b.root
	b.root.next = &b.root
	b.expiration.Store(-1)
}
