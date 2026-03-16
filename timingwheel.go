package timerWheel

import (
	"sync"
)

// TimingWheel is a hierarchical timing wheel.
// Each wheel has a fixed number of buckets, each spanning a tick duration.
// When a task's delay exceeds the wheel's range, it overflows to a higher-level wheel.
type TimingWheel struct {
	tick        int64 // duration of a single tick in milliseconds
	wheelSize   int64 // number of buckets in this wheel
	interval    int64 // tick * wheelSize — total time range of this wheel
	currentTime int64 // current time truncated to tick boundary

	buckets   []*Bucket
	scheduler scheduler // shared across all levels

	overflowWheel *TimingWheel
	mu            sync.RWMutex
}

func newTimingWheel(tick int64, wheelSize int64, startMs int64, sched scheduler) *TimingWheel {
	buckets := make([]*Bucket, wheelSize)
	for i := range buckets {
		buckets[i] = newBucket()
	}

	return &TimingWheel{
		tick:        tick,
		wheelSize:   wheelSize,
		interval:    tick * wheelSize,
		currentTime: startMs - (startMs % tick), // truncate to tick boundary
		buckets:     buckets,
		scheduler:   sched,
	}
}

// add inserts a TimerTaskEntry into the appropriate bucket.
// Returns true if the entry was successfully added, false if it has already expired.
func (tw *TimingWheel) add(entry *TimerTaskEntry) bool {
	exp := entry.expiration

	if entry.cancelled() {
		// Task was cancelled, no need to add.
		return false
	}

	tw.mu.RLock()
	currentTime := tw.currentTime
	tw.mu.RUnlock()

	if exp < currentTime+tw.tick {
		// Already expired.
		return false
	} else if exp < currentTime+tw.interval {
		// Fits in this wheel. Compute bucket index.
		virtualID := exp / tw.tick
		bucketIdx := virtualID % tw.wheelSize
		bucket := tw.buckets[bucketIdx]

		bucket.Add(entry)

		// Set the bucket's expiration to the beginning of the tick window.
		if bucket.SetExpiration(virtualID * tw.tick) {
			// Bucket expiration changed — notify the scheduler.
			tw.scheduler.notify(bucket)
		}

		return true
	} else {
		// Overflow to higher-level wheel.
		tw.mu.Lock()
		if tw.overflowWheel == nil {
			tw.overflowWheel = newTimingWheel(tw.interval, tw.wheelSize, tw.currentTime, tw.scheduler)
		}
		tw.mu.Unlock()

		return tw.overflowWheel.add(entry)
	}
}

// advanceClock advances the timing wheel's current time.
// This is called when a bucket expires in the delay queue.
func (tw *TimingWheel) advanceClock(expiration int64) {
	if expiration >= tw.currentTime+tw.tick {
		tw.currentTime = expiration - (expiration % tw.tick)

		// Propagate to overflow wheel.
		tw.mu.RLock()
		overflow := tw.overflowWheel
		tw.mu.RUnlock()

		if overflow != nil {
			overflow.advanceClock(tw.currentTime)
		}
	}
}

// advanceAndFlush advances the clock and flushes all expired buckets at every level.
// Used by the ticker scheduler.
func (tw *TimingWheel) advanceAndFlush(now int64, reinsert func(*TimerTaskEntry)) {
	tw.mu.Lock()
	currentTime := now - (now % tw.tick)
	if currentTime <= tw.currentTime {
		tw.mu.Unlock()
		return
	}
	tw.currentTime = currentTime
	tw.mu.Unlock()

	for _, bucket := range tw.buckets {
		if exp := bucket.Expiration(); exp != -1 && exp <= currentTime {
			bucket.Flush(reinsert)
		}
	}

	// Propagate to overflow wheel.
	tw.mu.RLock()
	overflow := tw.overflowWheel
	tw.mu.RUnlock()

	if overflow != nil {
		overflow.advanceAndFlush(now, reinsert)
	}
}
