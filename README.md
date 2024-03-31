# Go-cond
[![Go Reference](https://pkg.go.dev/badge/github.com/nursik/go-cond.svg)](https://pkg.go.dev/github.com/nursik/go-cond)

Safer `sync.Cond` alternative

## Overview
This library provides `Cond` and `RWCond` with safer and enhanced [API](https://pkg.go.dev/github.com/nursik/go-cond). Standard `sync.Cond` is prone to hanging goroutines and this library addresses these issues. Built on top of [wake library](https://pkg.go.dev/github.com/nursik/wake), it provides the same API layout. 

Slower than `sync.Cond` ~3x (used sync/cond_test.go, which benchmarks only broadcast).

## Features

| Operation                          |    sync.Cond    |                 go-cond                 | Notes                                                                                                                                                               |
| ---------------------------------- | :-------------: | :-------------------------------------: | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Wake a goroutine (if any)          |  `c.Signal()`   |           `m := c.Signal(1)`            | Unlike standard sync.Cond, `Signal` reports, how many goroutines were awoken by this call                                                                           |
| Wake all goroutines                | `c.Broadcast()` |             `c.Broadcast()`             |                                                                                                                                                                     |
| Wait for signal                    |   `c.Wait()`    |            `ok := c.Wait()`             | `Wait` reports, if it was unblocked due receiving signal/broadcast or `Cond` was closed                                                                             |
| Wake "n" goroutines (if any)       |                 |           `m := c.Signal(n)`            | You can wake N goroutines                                                                                                                                           |
| Wake exactly "n" goroutines        |                 | `m, err := c.SignalWithContext(ctx, n)` | You can wake exactly N goroutines. Blocks until wakes all N, context is cancelled or cond is closed                                                                 |
| Wait with context for signal       |                 |   `ok, err := c.WaitWithContext(ctx)`   | Wait with context. Same as `Wait` + unblocks in case of context cancellation                                                                                        |
| Get a number of waiting goroutines |                 |          `n := c.WaitCount()`           |                                                                                                                                                                     |
| Close Cond                         |                 |          `first := c.Close()`           | Close cond. All waiting goroutines will be awoken. Reported value indicates, if it is the first `Close` call. All methods are safe to use even after cond is closed |
| Use RWMutex + RLock/RUnlock        |                 |               `NewRW(&l)`               | You can create `RWCond`, which uses `RLock` and `RUnlock` in `Wait*` methods.                                                                                       |

## Example
```bash
go get github.com/nursik/go-cond
```

```go
import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nursik/go-cond"
)

func Wait(c *cond.Cond, mp map[int]int, key int) {
	c.L.Lock()
	for {
		value, ok := mp[key]
		if ok {
			fmt.Printf("Found %v at key %v\n", key, value)
			break
		}
		// Cond.Wait returns false only if Cond was closed.
		if !c.Wait() {
			fmt.Printf("Not found key %v\n", key)
			break
		}
	}
	c.L.Unlock()

	fmt.Printf("Key %v checker exit\n", key)
}

func main() {
	var locker sync.Mutex
	mp := make(map[int]int)

	c := cond.New(&locker)
	defer c.Close()

	go Wait(c, mp, 1)
	go Wait(c, mp, 2)
	go Wait(c, mp, 3)

	locker.Lock()
	mp[1] = 2
	locker.Unlock()

	c.Broadcast()

	locker.Lock()
	mp[2] = 4
	locker.Unlock()

	c.Broadcast()

	locker.Lock()
	mp[3] = 6
	locker.Unlock()

	c.Signal(1)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer cancel()

	locker.Lock()
	// Nobody can wake. Unblocks with (true, context.DeadlineExceeded).
	notClosed, err := c.WaitWithContext(ctx)
	fmt.Println(notClosed, err)
	locker.Unlock()
}
```