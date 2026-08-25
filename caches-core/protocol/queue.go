package protocol

import "sync"

// ResultQueue is an unbounded FIFO of per-command result channels
// (E2: the fixed 5000-buffer channel deadlocks once a pipeline
// exceeds it, because the reader stops draining the socket while the
// writer waits on the queue).
type ResultQueue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	q      []chan *Result
	closed bool
}

// Result carries one command's reply.
type Result struct {
	V []byte
	E error
}

// NewResultQueue creates an empty queue.
func NewResultQueue() *ResultQueue {
	rq := &ResultQueue{}
	rq.cond = sync.NewCond(&rq.mu)
	return rq
}

// Push enqueues a result channel.
func (rq *ResultQueue) Push(c chan *Result) {
	rq.mu.Lock()
	rq.q = append(rq.q, c)
	rq.cond.Signal()
	rq.mu.Unlock()
}

// Pop blocks until a result channel is available, or returns ok=false
// after Close.
func (rq *ResultQueue) Pop() (chan *Result, bool) {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	for len(rq.q) == 0 && !rq.closed {
		rq.cond.Wait()
	}
	if len(rq.q) == 0 {
		return nil, false
	}
	c := rq.q[0]
	rq.q = rq.q[1:]
	return c, true
}

// Close wakes blocked Poppers; they drain the remaining queue before
// seeing closed.
func (rq *ResultQueue) Close() {
	rq.mu.Lock()
	rq.closed = true
	rq.cond.Broadcast()
	rq.mu.Unlock()
}
