package cond

import (
	"context"
	"sync"

	"github.com/nursik/wake"
)

type commonCond struct {
	s *wake.Signaller
	r *wake.Receiver
}

// Signal wakes n goroutines (if there are any) and reports how many goroutines were awoken.
// If n <= 0 it wakes all goroutines and returns 0 (same as [commonCond.Broadcast]).
func (c *commonCond) Signal(n int) int {
	if n <= 0 {
		c.s.Broadcast()
		return 0
	}

	var x int
	// we need to notify at least one receiver if we know that at least one is waiting.
	// we are doing it in for loop, because unlike golang's sync.Cond we may start waiting after sending Signal.
	// golang's sync.Cond Wait() appends to notification_list before unlocking.
	for c.s.WaitCount() > 0 {
		x = c.s.Signal(n)
		n = n - x
		if x > 0 {
			break
		}
	}
	// don't accidentally broadcast
	if n == 0 {
		return x
	}
	return x + c.s.Signal(n)
}

// SignalWithContext wakes n goroutines and reports how many goroutines were awoken and ctx.Err() if context was cancelled.
// It is a blocking operation and will be finished when all n goroutines are awoken, context is cancelled or Cond/RWCond was closed.
// If n <= 0, it wakes all goroutines (same as [commonCond.Broadcast]) regardless of context cancellation.
func (c *commonCond) SignalWithContext(ctx context.Context, n int) (int, error) {
	return c.s.SignalWithContext(ctx, n)
}

// Broadcast wakes up all goroutines.
func (c *commonCond) Broadcast() {
	c.s.Broadcast()
}

// Close closes Cond/RWCond and wakes all waiting goroutines.
// The first Close() returns true and subsequent calls always return false.
func (c *commonCond) Close() bool {
	return c.s.Close()
}

// IsClosed reports if Cond/RWCond is closed.
func (c *commonCond) IsClosed() bool {
	return c.s.IsClosed()
}

// WaitCount returns current number of goroutines waiting for signal.
func (c *commonCond) WaitCount() int {
	return c.s.WaitCount()
}

type Cond struct {
	L sync.Locker
	commonCond
}

// Wait Unlocks locker, blocks until awaken (returns true) or Cond was closed (returns false), and at the end Locks locker again.
func (c *Cond) Wait() bool {
	return wake.UnsafeWait(c.r, c.L)
}

// WaitWithContext Unlocks locker, blocks until awaken, context was cancelled or Cond was closed, and at the end Locks locker again.
// Returns true and nil, if awaken by signal/broadcast.
// Returns false and nil, if Cond was closed.
// Returns false and ctx.Err(), if context was cancelled.
func (c *Cond) WaitWithContext(ctx context.Context) (bool, error) {
	return wake.UnsafeWaitContext(c.r, c.L, ctx)
}

// New returns Cond with associated locker. Same as sync.Cond in terms of usage, but has more functionality.
// Only Wait and WaitWithContext methods use associated locker and other methods do not use locker. Using closed Cond is safe.
// Slower than sync.Cond by ~3 times (sync.Cond's tests which only benchmarks broadcast).
func New(l sync.Locker) *Cond {
	s, r := wake.New()
	return &Cond{
		L: l,
		commonCond: commonCond{
			s: s,
			r: r,
		},
	}
}

type RWCond struct {
	L   *sync.RWMutex
	rwl rlocker
	commonCond
}

// Wait RUnlocks locker, blocks until awaken (returns true) or RWCond was closed (returns false), and at the end RLocks locker again.
func (c *RWCond) Wait() bool {
	return wake.UnsafeWait(c.r, c.rwl)
}

// WaitWithContext RUnlocks locker, blocks until awaken, context was cancelled or RWCond was closed, and at the end RLocks locker again.
// Returns true and nil, if awaken by signal/broadcast.
// Returns false and nil, if RWCond was closed.
// Returns false and ctx.Err(), if context was cancelled.
func (c *RWCond) WaitWithContext(ctx context.Context) (bool, error) {
	return wake.UnsafeWaitContext(c.r, c.rwl, ctx)
}

type rlocker struct {
	mtx *sync.RWMutex
}

func (l rlocker) Lock() {
	l.mtx.RLock()
}

func (l rlocker) Unlock() {
	l.mtx.RUnlock()
}

// NewRW returns RWCond with associated sync.RWMutex. Uses RUnlock and RLock for Wait and WaitWithContext methods. Other methods do not use associated sync.RWMutex.
func NewRW(l *sync.RWMutex) *RWCond {
	s, r := wake.New()
	return &RWCond{
		L:   l,
		rwl: rlocker{mtx: l},
		commonCond: commonCond{
			s: s,
			r: r,
		},
	}
}
