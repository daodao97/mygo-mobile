package backend

import (
	"sync"
	"testing"
)

func TestWorkQueueReentrantPostWithInputBurst(t *testing.T) {
	q := newWorkQueue()
	got := []int{}
	// Reproduce the full-queue shape: a frame posts while more than the
	// former 64-channel capacity of input is pending behind that frame.
	q.push(func() { q.push(func() { got = append(got, 256) }) })
	for i := 0; i < 256; i++ {
		i := i
		q.push(func() { got = append(got, i) })
	}
	for fn := q.pop(); fn != nil; fn = q.pop() {
		fn()
	}
	if len(got) != 257 {
		t.Fatalf("got %d tasks", len(got))
	}
	for i, v := range got {
		if v != i {
			t.Fatalf("task %d=%d", i, v)
		}
	}
}
func TestWorkQueueConcurrentProducerFIFO(t *testing.T) {
	q := newWorkQueue()
	var wg sync.WaitGroup
	got := make([][]int, 8)
	for p := range got {
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 128; n++ {
				n := n
				q.push(func() { got[p] = append(got[p], n) })
			}
		}()
	}
	wg.Wait()
	for fn := q.pop(); fn != nil; fn = q.pop() {
		fn()
	}
	for p, ns := range got {
		if len(ns) != 128 {
			t.Fatalf("producer %d count %d", p, len(ns))
		}
		for n, v := range ns {
			if v != n {
				t.Fatalf("producer %d order %v", p, ns)
			}
		}
	}
	if q.pop() != nil {
		t.Fatal("drained queue retained work")
	}
	q.push(func() {})
	select {
	case <-q.ready:
	default:
		t.Fatal("post did not wake pump")
	}
}

func TestWorkQueueSustainedTrafficReclaimsConsumedSlots(t *testing.T) {
	q := newWorkQueue()
	q.push(func() {})
	q.push(func() {})
	for range 10000 {
		q.pop()()
		q.push(func() {})
		if len(q.pending) > 66 {
			t.Fatalf("retained %d slots for two pending tasks", len(q.pending))
		}
	}
}
