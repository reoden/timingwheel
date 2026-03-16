package delayqueue

import (
	"context"
	"sync"
	"time"

	"github.com/reoden/timewheel/pq"
)

type Delayed interface {
	Expiration() int64
}

type DelayQueue struct {
	pq   *pq.PriorityQueue[Delayed]
	lock sync.Mutex
	wake chan struct{} // buffered channel of size 1 for wakeup signaling
}

func New() *DelayQueue {
	pq := pq.New[Delayed](func(a, b Delayed) bool {
		return a.Expiration() < b.Expiration()
	})

	return &DelayQueue{
		pq:   pq,
		wake: make(chan struct{}, 1),
	}
}

// Offer adds a delayed item to the queue.
func (dq *DelayQueue) Offer(d Delayed) {
	dq.lock.Lock()
	item := dq.pq.Push(d)
	index := item.Index()
	dq.lock.Unlock()

	// If the new item became the head of the queue, wake up Poll.
	if index == 0 {
		select {
		case dq.wake <- struct{}{}:
		default:
		}
	}
}

// Poll waits for an expired item or until the context is done.
// Returns the expired item and true, or nil and false if context is cancelled.
func (dq *DelayQueue) Poll(ctx context.Context) (Delayed, bool) {
	for {
		now := time.Now().UnixMilli()

		dq.lock.Lock()

		if !dq.pq.Empty() {
			item := dq.pq.Peek()
			exp := item.Value.Expiration()

			if exp <= now {
				val := dq.pq.Pop().Value
				dq.lock.Unlock()
				return val, true
			}

			wait := time.Duration(exp-now) * time.Millisecond
			dq.lock.Unlock()

			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, false
			case <-timer.C:
				// Time elapsed, re-check.
			case <-dq.wake:
				timer.Stop()
				// New item may have arrived with earlier expiration, re-check.
			}
		} else {
			dq.lock.Unlock()

			// Queue is empty — wait for a signal or context cancellation.
			select {
			case <-ctx.Done():
				return nil, false
			case <-dq.wake:
				// Item was offered, re-check.
			}
		}
	}
}

// Size returns the number of items in the queue.
func (dq *DelayQueue) Size() int {
	dq.lock.Lock()
	defer dq.lock.Unlock()
	return dq.pq.Len()
}
