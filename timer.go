package timerWheel

import (
	"context"
	"sync"
	"time"
)

// Clock is a function that returns the current time in milliseconds.
// Use WithClock to provide a custom clock (e.g., game server time).
// Defaults to time.Now().UnixMilli().
type Clock func() int64

// Option configures a Timer.
type Option func(*Timer)

// WithTickerMode configures the Timer to use a ticker-based scheduler
// instead of the default delay queue scheduler.
// Better suited for moderate tick durations (>=10ms) or task-dense workloads.
func WithTickerMode() Option {
	return func(t *Timer) {
		t.useTicker = true
	}
}

// WithClock sets a custom clock source for the Timer.
// The clock function should return the current time in milliseconds.
// This is useful for game servers that manipulate time independently.
func WithClock(clock Clock) Option {
	return func(t *Timer) {
		t.clock = clock
	}
}

// Timer is the public API for the hierarchical timing wheel.
// It manages task scheduling, the polling goroutine, and shutdown.
type Timer struct {
	tw     *TimingWheel
	sched  scheduler
	tick   time.Duration
	clock  Clock
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	useTicker bool // set by options before Start
}

// NewTimer creates a new Timer with the given tick duration and wheel size.
// tick is the base time resolution, wheelSize is the number of buckets per wheel.
// Use WithTickerMode() to switch from the default delay queue scheduler to a ticker scheduler.
// Use WithClock() to provide a custom time source.
func NewTimer(tick time.Duration, wheelSize int64, opts ...Option) *Timer {
	tickMs := int64(tick / time.Millisecond)
	if tickMs <= 0 {
		tickMs = 1
	}

	ctx, cancel := context.WithCancel(context.Background())

	t := &Timer{
		tick:   tick,
		clock:  func() int64 { return time.Now().UnixMilli() },
		ctx:    ctx,
		cancel: cancel,
	}

	for _, opt := range opts {
		opt(t)
	}

	// Create the scheduler based on the selected mode.
	if t.useTicker {
		t.sched = newTickerScheduler(tick, t.clock)
	} else {
		t.sched = newDelayQueueScheduler()
	}

	startMs := t.clock()
	t.tw = newTimingWheel(tickMs, wheelSize, startMs, t.sched)

	return t
}

// Start starts the timer's polling goroutine.
func (t *Timer) Start() {
	t.wg.Go(func() {
		t.sched.run(t.ctx, t.tw, t.addEntry)
	})
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
	exp := t.clock() + delayMs

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
