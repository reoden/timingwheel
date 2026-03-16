package main

// TimerTaskEntry is a node in the bucket's doubly-linked list.
// It wraps a TimerTask and tracks which bucket it belongs to.
type TimerTaskEntry struct {
	expiration int64
	task       *TimerTask

	bucket *Bucket
	prev   *TimerTaskEntry
	next   *TimerTaskEntry
}

func newTimerTaskEntry(task *TimerTask, expiration int64) *TimerTaskEntry {
	entry := &TimerTaskEntry{
		expiration: expiration,
		task:       task,
	}
	task.setEntry(entry)
	return entry
}

// cancelled returns true if the entry is no longer associated with its task.
func (e *TimerTaskEntry) cancelled() bool {
	return e.task.getEntry() != e
}

// remove removes this entry from its bucket.
func (e *TimerTaskEntry) remove() {
	b := e.bucket
	for b != nil {
		b.Remove(e)
		b = e.bucket
	}
}
