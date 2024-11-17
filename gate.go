// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gate contains an alternative condition variable.
//
// A gate is a monitor (mutex + condition variable) with one bit of state.
//
// A gate exists in one of three states:
//   - locked
//   - unlocked and set
//   - unlocked and unset
//
// Lock operations may be unconditional, or wait for the condition to be set.
// Unlock operations record the new state of the condition.
//
// Gates have several advantages over sync.Cond:
//   - Wait operations can be easily bounded by a context.Context lifetime.
//   - A Wait operation only returns successfully when the gate condition is set.
//     For example, if a gate's condition is set when a queue is non-empty,
//     then a successful return from Wait guarantees that an item is in the queue.
//   - No need to call Signal/Broadcast to notify waiters of a change in the condition.
package gate

import "context"

// A gate is a monitor (mutex + condition variable) with one bit of state.
type Gate struct {
	// When unlocked, exactly one of set or unset contains a value.
	// When locked, neither chan contains a value.
	set   chan struct{}
	unset chan struct{}
}

// New returns a new, unlocked gate with the given condition state.
func New(set bool) Gate {
	g := Gate{
		set:   make(chan struct{}, 1),
		unset: make(chan struct{}, 1),
	}
	g.Unlock(set)
	return g
}

// Lock acquires the gate unconditionally.
// It reports whether the condition was set.
func (g *Gate) Lock() (set bool) {
	// This doesn't take a Context parameter because
	// we don't expect unconditional lock operations to be time-bounded.
	select {
	case <-g.set:
		return true
	case <-g.unset:
		return false
	}
}

// WaitAndLock waits until the condition is set before acquiring the gate.
// If the context expires, WaitAndLock returns an error and does not acquire the gate.
func (g *Gate) WaitAndLock(ctx context.Context) error {
	// If the gate is available and the context is expired,
	// prefer locking the gate.
	select {
	case <-g.set:
		return nil
	default:
	}
	select {
	case <-g.set:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// LockIfSet acquires the gate if and only if the condition is set.
func (g *Gate) LockIfSet() (acquired bool) {
	select {
	case <-g.set:
		return true
	default:
		return false
	}
}

// Unlock sets the condition and releases the gate.
func (g *Gate) Unlock(set bool) {
	if set {
		g.set <- struct{}{}
	} else {
		g.unset <- struct{}{}
	}
}
