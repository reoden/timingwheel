package pq

import (
	"math/rand"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
)

func equal(t *testing.T, act, exp interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		t.Logf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n",
			filepath.Base(file), line, exp, act)
		t.FailNow()
	}
}

func TestMinPriorityQueue(t *testing.T) {
	c := 100
	pq := NewPriorityQueue[int](func(a, b int) bool {
		return a < b
	})

	for i := 0; i < c+1; i++ {
		pq.Push(i)
	}
	equal(t, pq.Len(), c+1)

	for i := 0; i < c+1; i++ {
		item := pq.Pop()
		equal(t, item.Value, i)
	}
}

func TestMaxPriorityQueue(t *testing.T) {
	c := 100
	pq := NewPriorityQueue[int](func(a, b int) bool {
		return a > b
	})

	for i := 0; i < c+1; i++ {
		pq.Push(i)
	}
	equal(t, pq.Len(), c+1)

	for i := c; i >= 0; i-- {
		item := pq.Pop()
		equal(t, item.Value, i)
	}
}

func TestUnsortedInsert(t *testing.T) {
	c := 100
	pq := NewPriorityQueue[int](func(a, b int) bool {
		return a < b
	})
	ints := make([]int, 0, c)

	for i := 0; i < c; i++ {
		v := rand.Int()
		ints = append(ints, v)
		pq.Push(v)
	}
	equal(t, pq.Len(), c)

	sort.Ints(ints)

	for i := 0; i < c; i++ {
		item := pq.Pop()
		equal(t, item.Value, ints[i])
	}
}

func TestRemove(t *testing.T) {
	c := 100
	pq := NewPriorityQueue[int](func(a, b int) bool {
		return a < b
	})

	items := make([]*Item[int], 0, c)
	for i := 0; i < c; i++ {
		v := rand.Int()
		item := pq.Push(v)
		items = append(items, item)
	}

	// Remove 10 random items.
	for i := 0; i < 10; i++ {
		idx := rand.Intn(len(items))
		pq.Remove(items[idx])
		items = append(items[:idx], items[idx+1:]...)
	}

	equal(t, pq.Len(), c-10)

	// Verify remaining items come out in sorted order.
	prev := pq.Pop()
	for i := 0; i < c-10-1; i++ {
		item := pq.Pop()
		equal(t, prev.Value <= item.Value, true)
		prev = item
	}
}
