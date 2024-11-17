// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gate_test

import (
	"context"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/neild/gate"
)

// A Queue is an unbounded queue of some item.
type Queue[T any] struct {
	gate gate.Gate // set if queue is non-empty or closed
	err  error
	q    []T
}

// NewQueue returns a new queue.
func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{
		gate: gate.New(false),
	}
}

// Close closes the queue, causing pending and future pop operations
// to return immediately with err.
func (q *Queue[T]) Close(err error) {
	q.gate.Lock()
	defer q.unlock()
	if q.err == nil {
		q.err = err
	}
}

// Put appends an item to the queue.
// It returns true if the item was added, false if the queue is closed.
func (q *Queue[T]) Put(v T) bool {
	q.gate.Lock()
	defer q.unlock()
	if q.err != nil {
		return false
	}
	q.q = append(q.q, v)
	return true
}

// Get removes the first item from the queue, blocking until ctx is done, an item is available,
// or the queue is closed.
func (q *Queue[T]) Get(ctx context.Context) (T, error) {
	var zero T
	if err := q.gate.WaitAndLock(ctx); err != nil {
		return zero, err
	}
	defer q.unlock()

	// WaitAndLock blocks until the gate condition is set,
	// so either the queue is closed (q.err != nil) or
	// there is at least one item in the queue.
	if q.err != nil {
		return zero, q.err
	}
	v := q.q[0]
	q.q = slices.Delete(q.q, 0, 1)
	return v, nil
}

// unlock unlocks the queue's gate,
// setting the condition to true if the queue is non-empty or closed.
func (q *Queue[T]) unlock() {
	q.gate.Unlock(q.err != nil || len(q.q) > 0)
}

func Example_queue() {
	q := NewQueue[int]()

	go func() {
		time.Sleep(1 * time.Millisecond)
		q.Put(1)
		time.Sleep(1 * time.Millisecond)
		q.Put(2)
		q.Close(io.EOF)
	}()

	fmt.Println(q.Get(context.Background()))
	fmt.Println(q.Get(context.Background()))
	fmt.Println(q.Get(context.Background()))
	// Output:
	// 1 <nil>
	// 2 <nil>
	// 0 EOF
}
