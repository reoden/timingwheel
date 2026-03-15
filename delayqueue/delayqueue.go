package delayqueue

import (
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
	cond *sync.Cond
}

func New() *DelayQueue {

	pq := pq.New[Delayed](func(a, b Delayed) bool {
		return a.Expiration() < b.Expiration()
	})

	dq := &DelayQueue{
		pq: pq,
	}

	dq.cond = sync.NewCond(&dq.lock)

	return dq
}

func (dq *DelayQueue) Offer(d Delayed) {

	dq.lock.Lock()
	defer dq.lock.Unlock()

	dq.pq.Push(d)

	dq.cond.Signal()
}

func (dq *DelayQueue) Take() Delayed {

	dq.lock.Lock()
	defer dq.lock.Unlock()

	for {

		if dq.pq.Empty() {
			dq.cond.Wait()
			continue
		}

		item := dq.pq.Peek()

		now := time.Now().UnixMilli()

		exp := item.Value.Expiration()

		if exp <= now {
			return dq.pq.Pop().Value
		}

		wait := exp - now

		timer := time.NewTimer(time.Duration(wait) * time.Millisecond)

		dq.lock.Unlock()

		<-timer.C

		dq.lock.Lock()
	}
}
