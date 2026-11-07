package backend

import "sync"

// workQueue is the Android pump's FIFO. Posting must never wait for the
// pump: Java callbacks must return promptly, and the pump posts its own
// frames and dispatches. A bounded channel deadlocks when input fills it
// while the pump is itself posting the next frame.
// ready is only a coalesced wakeup; pending owns the actual work.
type workQueue struct {
	mu      sync.Mutex
	pending []func()
	head    int
	ready   chan struct{}
}

func newWorkQueue() *workQueue { return &workQueue{ready: make(chan struct{}, 1)} }
func (q *workQueue) push(fn func()) {
	if fn == nil {
		return
	}
	q.mu.Lock()
	// Reuse consumed storage while traffic keeps the queue nonempty.
	if q.head >= 64 && q.head >= len(q.pending)/2 {
		remaining := copy(q.pending, q.pending[q.head:])
		clear(q.pending[remaining:])
		q.pending = q.pending[:remaining]
		q.head = 0
	}
	q.pending = append(q.pending, fn)
	q.mu.Unlock()
	select {
	case q.ready <- struct{}{}:
	default:
	}
}
func (q *workQueue) pop() func() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.head == len(q.pending) {
		q.pending = nil
		q.head = 0
		return nil
	}
	fn := q.pending[q.head]
	q.pending[q.head] = nil
	q.head++
	if q.head == len(q.pending) {
		q.pending = nil
		q.head = 0
	}
	return fn
}
