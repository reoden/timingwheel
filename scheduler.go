package main

import (
	"context"
	"time"

	"github.com/reoden/timerwheel/delayqueue"
)

// scheduler abstracts the driving strategy for the timing wheel.
type scheduler interface {
	// notify is called when a bucket's expiration changes.
	notify(b *Bucket)
	// run is the main polling loop, blocking until ctx is cancelled.
	run(ctx context.Context, tw *TimingWheel, addEntry func(*TimerTaskEntry))
}

// delayQueueScheduler drives the timing wheel using a DelayQueue (min-heap).
// It only wakes when a bucket expires, making it efficient when idle.
type delayQueueScheduler struct {
	queue *delayqueue.DelayQueue
}

func newDelayQueueScheduler() *delayQueueScheduler {
	return &delayQueueScheduler{
		queue: delayqueue.New(),
	}
}

func (s *delayQueueScheduler) notify(b *Bucket) {
	s.queue.Offer(b)
}

func (s *delayQueueScheduler) run(ctx context.Context, tw *TimingWheel, addEntry func(*TimerTaskEntry)) {
	for {
		d, ok := s.queue.Poll(ctx)
		if !ok {
			return
		}

		bucket := d.(*Bucket)
		tw.advanceClock(bucket.Expiration())
		bucket.Flush(addEntry)
	}
}

// tickerScheduler drives the timing wheel using a fixed-interval ticker.
// Simpler and O(1) for task insertion, but wakes every tick even when idle.
type tickerScheduler struct {
	tick  time.Duration
	clock Clock
}

func newTickerScheduler(tick time.Duration, clock Clock) *tickerScheduler {
	return &tickerScheduler{tick: tick, clock: clock}
}

func (s *tickerScheduler) notify(b *Bucket) {
	// No-op: the ticker loop discovers expired buckets by scanning.
}

func (s *tickerScheduler) run(ctx context.Context, tw *TimingWheel, addEntry func(*TimerTaskEntry)) {
	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := s.clock()
			tw.advanceAndFlush(now, addEntry)
		case <-ctx.Done():
			return
		}
	}
}
