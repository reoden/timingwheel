package timingwheel

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAfterFunc_SingleTask(t *testing.T) {
	timer := NewTimer(time.Millisecond, 20)
	timer.Start()
	defer timer.Stop()

	done := make(chan struct{})
	timer.AfterFunc(50*time.Millisecond, func() {
		close(done)
	})

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("task did not fire within timeout")
	}
}

func TestAfterFunc_Ordering(t *testing.T) {
	timer := NewTimer(time.Millisecond, 20)
	timer.Start()
	defer timer.Stop()

	var mu sync.Mutex
	var order []int

	var wg sync.WaitGroup
	wg.Add(3)

	timer.AfterFunc(150*time.Millisecond, func() {
		mu.Lock()
		order = append(order, 3)
		mu.Unlock()
		wg.Done()
	})
	timer.AfterFunc(50*time.Millisecond, func() {
		mu.Lock()
		order = append(order, 1)
		mu.Unlock()
		wg.Done()
	})
	timer.AfterFunc(100*time.Millisecond, func() {
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()
		wg.Done()
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("tasks did not complete within timeout")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(order) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(order))
	}

	for i, v := range order {
		if v != i+1 {
			t.Errorf("expected order[%d]=%d, got %d", i, i+1, v)
		}
	}
}

func TestAfterFunc_Cancel(t *testing.T) {
	timer := NewTimer(time.Millisecond, 20)
	timer.Start()
	defer timer.Stop()

	var executed atomic.Bool
	task := timer.AfterFunc(100*time.Millisecond, func() {
		executed.Store(true)
	})

	task.Cancel()

	time.Sleep(200 * time.Millisecond)

	if executed.Load() {
		t.Fatal("cancelled task should not have been executed")
	}
}

func TestAfterFunc_OverflowWheel(t *testing.T) {
	// tick=1ms, wheelSize=20 → level 0 covers 20ms.
	// A 500ms delay will overflow to higher-level wheels.
	timer := NewTimer(time.Millisecond, 20)
	timer.Start()
	defer timer.Stop()

	done := make(chan struct{})
	start := time.Now()

	timer.AfterFunc(500*time.Millisecond, func() {
		close(done)
	})

	select {
	case <-done:
		elapsed := time.Since(start)
		if elapsed < 450*time.Millisecond {
			t.Fatalf("task fired too early: %v", elapsed)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("task did not fire within timeout")
	}
}

func TestAfterFunc_ManyTasks(t *testing.T) {
	timer := NewTimer(time.Millisecond, 20)
	timer.Start()
	defer timer.Stop()

	const n = 100
	var count atomic.Int64
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		delay := time.Duration(10+i*2) * time.Millisecond
		timer.AfterFunc(delay, func() {
			count.Add(1)
			wg.Done()
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if count.Load() != n {
			t.Fatalf("expected %d tasks, got %d", n, count.Load())
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("only %d/%d tasks completed within timeout", count.Load(), n)
	}
}

func TestAfterFunc_ImmediateExpiration(t *testing.T) {
	timer := NewTimer(time.Millisecond, 20)
	timer.Start()
	defer timer.Stop()

	done := make(chan struct{})
	timer.AfterFunc(0, func() {
		close(done)
	})

	select {
	case <-done:
		// success — immediate tasks should execute promptly
	case <-time.After(2 * time.Second):
		t.Fatal("immediate task did not fire within timeout")
	}
}
