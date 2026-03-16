package timingwheel

import (
	"context"
	"sync"
	"time"

	"github.com/reoden/timewheel/delayqueue"
)

// Timer is the public API for the hierarchical timing wheel.
// It manages task scheduling, the polling goroutine, and shutdown.
type Timer struct {
	tw     *TimingWheel
	queue  *delayqueue.DelayQueue
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewTimer creates a new Timer with the given tick duration and wheel size.
// tick is the base time resolution, wheelSize is the number of buckets per wheel.
func NewTimer(tick time.Duration, wheelSize int64) *Timer {
	queue := delayqueue.New()
	tickMs := int64(tick / time.Millisecond)
	if tickMs <= 0 {
		tickMs = 1
	}
	startMs := time.Now().UnixMilli()
	tw := newTimingWheel(tickMs, wheelSize, startMs, queue)

	ctx, cancel := context.WithCancel(context.Background())

	return &Timer{
		tw:     tw,
		queue:  queue,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the timer's polling goroutine.
// The goroutine waits on the delay queue for expired buckets,
// advances the timing wheel clock, and flushes the entries.
func (t *Timer) Start() {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.poll()
	}()
}

// Stop stops the timer and waits for the polling goroutine to exit.
func (t *Timer) Stop() {
	t.cancel()
	t.wg.Wait()
}

// AfterFunc schedules f to be called after delay d.
// Returns a TimerTask that can be used to cancel the scheduled call.
func (t *Timer) AfterFunc(d time.Duration, f func()) *TimerTask {
	delayMs := int64(d / time.Millisecond)
	exp := time.Now().UnixMilli() + delayMs

	task := newTimerTask(delayMs, f)
	entry := newTimerTaskEntry(task, exp)

	t.addEntry(entry)

	return task
}

func (t *Timer) addEntry(entry *TimerTaskEntry) {
	if !t.tw.add(entry) {
		// Entry already expired or cancelled.
		if !entry.cancelled() {
			// Execute the task asynchronously.
			go entry.task.f()
		}
	}
}

func (t *Timer) poll() {
	for {
		d, ok := t.queue.Poll(t.ctx)
		if !ok {
			// Context was cancelled; shut down.
			return
		}

		bucket := d.(*Bucket)
		t.tw.advanceClock(bucket.Expiration())
		// Flush the bucket: each entry is either re-inserted into a lower-level
		// wheel or executed if it has expired.
		bucket.Flush(t.addEntry)
	}
}
