# timewheel

![License](https://img.shields.io/github/license/reoden/timewheel)
![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.23-blue)
[![Test](https://github.com/reoden/timewheel/actions/workflows/test.yml/badge.svg)](https://github.com/reoden/timewheel/actions)

A high-performance hierarchical timing wheel implementation in Go, based on the algorithm described in [Hashed and Hierarchical Timing Wheels](http://www.cs.columbia.edu/~aho/CS6991/04-28-2014.pdf).

## Overview

A timing wheel is a data structure for efficiently scheduling timer events. Instead of using a min-heap with O(log n) insertion and removal, a timing wheel achieves O(1) amortized time complexity by organizing timers into a circular buffer of "buckets". Each bucket represents a time slot, and timers are placed in the bucket corresponding to their expiration time.

This implementation uses **hierarchical timing wheels** to support arbitrary timeout durations:
- Level 0: precision wheel (e.g., 1ms tick, 20 buckets → 20ms range)
- Level 1: overflow wheel (20ms × 20 = 400ms range)
- Level N: continues cascading upward

When a timer's delay exceeds the current wheel's range, it overflows to the next level.

## Features

- **O(1) timer insertion** — Efficient scheduling for high-throughput scenarios
- **Two scheduling modes**:
  - DelayQueue mode (default) — Uses a min-heap, only wakes when needed, ideal for sparse timers
  - Ticker mode — Fixed-interval polling, better for dense workloads
- **Task cancellation** — Support for cancelling scheduled tasks
- **Custom time source** — Use a custom clock (useful for game servers with manipulated time)
- **Thread-safe** — Safe for concurrent use

## Installation

```bash
go get github.com/reoden/timewheel
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/reoden/timewheel"
)

func main() {
    // Create a timer with 1ms tick and 20 buckets per wheel
    timer := timingwheel.NewTimer(time.Millisecond, 20)
    timer.Start()
    defer timer.Stop()
    
    // Schedule a task to run after 50ms
    task := timer.AfterFunc(50*time.Millisecond, func() {
        fmt.Println("Task executed!")
    })
    
    // Optionally cancel the task
    // task.Cancel()
    
    // Keep main alive
    time.Sleep(100 * time.Millisecond)
}
```

## Configuration Options

### Ticker Mode

Use `WithTickerMode()` to switch from the default DelayQueue scheduler to a ticker-based scheduler:

```go
timer := timingwheel.NewTimer(time.Millisecond, 20, timingwheel.WithTickerMode())
timer.Start()
```

**When to use**:
- Ticker mode: moderate tick durations (≥10ms) or task-dense workloads
- DelayQueue mode (default): sparse timers, power-sensitive applications

### Custom Clock

Use `WithClock()` to provide a custom time source:

```go
gameTime := int64(0)

// Fast-forward game time for testing
timer := timingwheel.NewTimer(time.Millisecond, 20, timingwheel.WithClock(func() int64 {
    return gameTime
}))
timer.Start()

// Advance time manually
gameTime += 5000 // 5 seconds
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    TimingWheel (Level 0)                  │
│  tick=1ms, wheelSize=20, interval=20ms                  │
├─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┬─────┤
│  0  │  1  │  2  │ ... │ 17  │ 18  │ 19  │     │     │
│[Bkt]│[Bkt]│[Bkt]│     │[Bkt]│[Bkt]│[Bkt]│     │     │
└─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┴─────┘
       │
       │ overflow (delay > 20ms)
       ▼
┌─────────────────────────────────────────────────────────────┐
│                 TimingWheel (Level 1)                     │
│  tick=20ms, wheelSize=20, interval=400ms               │
└─────────────────────────────────────────────────────────────┘
```

### Key Components

| Component | Description |
|-----------|-------------|
| `TimingWheel` | Hierarchical wheel managing multiple levels of buckets |
| `Bucket` | Doubly-linked list storing tasks expiring in the same time slot |
| `TimerTaskEntry` | Node in bucket, wraps a timer task |
| `TimerTask` | User's callable task with cancel support |
| `scheduler` | Drives the wheel (DelayQueue or Ticker) |

### Workflow

1. **Schedule**: `AfterFunc(delay, fn)` → creates entry → calculates bucket index → inserts into appropriate bucket
2. **Poll**: Scheduler wakes on next expiration → advances wheel clock → flushes expired bucket
3. **Execute**: Expired entries is reinserted or executed immediately if already past time

## API Reference

### `NewTimer(tick time.Duration, wheelSize int64, opts ...Option) *Timer`

Creates a new Timer with specified tick duration and wheel size.

- `tick`: Base time resolution (e.g., `time.Millisecond`)
- `wheelSize`: Number of buckets per wheel (e.g., 20)
- `opts`: Optional configuration (see below)

### `Timer.Start()`

Starts the timer goroutine. Must be called before scheduling tasks.

### `Timer.Stop()`

Stops the timer and waits for the goroutine to exit.

### `Timer.AfterFunc(d time.Duration, f func()) *TimerTask`

Schedules a function to be called after the specified delay. Returns a `TimerTask` that can be used to cancel.

### `TimerTask.Cancel()`

Cancels the scheduled task.

### Options

- `WithTickerMode()` — Use ticker-based scheduler
- `WithClock(clock Clock)` — Provide custom time source

## Performance

The hierarchical timing wheel is ideal for:
- High-volume timer management (e.g., connection timeouts, rate limiting)
- Scenarios requiring low overhead per timer
- Systems where most timers are cancelled before expiration

For most use cases, `tick=1ms` and `wheelSize=20` provide a good balance of precision and memory usage.

## License

MIT License - see [LICENSE](LICENSE) for details.