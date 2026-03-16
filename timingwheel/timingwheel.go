package timingwheel

import (
	"sync"

	"github.com/reoden/timewheel/delayqueue"
)

// TimingWheel is a hierarchical timing wheel.
// Each wheel has a fixed number of buckets, each spanning a tick duration.
// When a task's delay exceeds the wheel's range, it overflows to a higher-level wheel.
type TimingWheel struct {
	tick        int64 // duration of a single tick in milliseconds
	wheelSize   int64 // number of buckets in this wheel
	interval    int64 // tick * wheelSize — total time range of this wheel
	currentTime int64 // current time truncated to tick boundary

	buckets []*Bucket
	queue   *delayqueue.DelayQueue // shared across all levels

	overflowWheel *TimingWheel
	mu            sync.Mutex
}

func newTimingWheel(tick int64, wheelSize int64, startMs int64, queue *delayqueue.DelayQueue) *TimingWheel {
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
		queue:       queue,
	}
}

// add inserts a TimerTaskEntry into the appropriate bucket.
// Returns true if the entry was successfully added, false if it has already expired.
func (tw *TimingWheel) add(entry *TimerTaskEntry) bool {
	exp := entry.expiration

	if entry.cancelled() {
		// Task was cancelled, no need to add.
		return false
	} else if exp < tw.currentTime+tw.tick {
		// Already expired.
		return false
	} else if exp < tw.currentTime+tw.interval {
		// Fits in this wheel. Compute bucket index.
		virtualID := exp / tw.tick
		bucketIdx := virtualID % tw.wheelSize
		bucket := tw.buckets[bucketIdx]

		bucket.Add(entry)

		// Set the bucket's expiration to the beginning of the tick window.
		if bucket.SetExpiration(virtualID * tw.tick) {
			// Bucket expiration changed — (re-)insert it into the delay queue.
			tw.queue.Offer(bucket)
		}

		return true
	} else {
		// Overflow to higher-level wheel.
		tw.mu.Lock()
		if tw.overflowWheel == nil {
			tw.overflowWheel = newTimingWheel(tw.interval, tw.wheelSize, tw.currentTime, tw.queue)
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
		tw.mu.Lock()
		overflow := tw.overflowWheel
		tw.mu.Unlock()

		if overflow != nil {
			overflow.advanceClock(tw.currentTime)
		}
	}
}
