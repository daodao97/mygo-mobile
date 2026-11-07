package backend

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNativeRequestLateReplyCannotFinishCanceledOrNewRequest(t *testing.T) {
	var r requestRegistry
	var calls atomic.Int32
	first := r.add(func(json.RawMessage, error) { calls.Add(1) })
	if fn := r.take(first); fn != nil {
		fn(nil, nil)
	}
	next := r.add(func(json.RawMessage, error) { calls.Add(1) })
	if fn := r.take(first); fn != nil {
		t.Fatal("late reply retained canceled callback")
	}
	if fn := r.take(next); fn != nil {
		fn(nil, nil)
	}
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
}
func TestNativeRequestConcurrentCancelAndReplyFinishOnce(t *testing.T) {
	var r requestRegistry
	var calls atomic.Int32
	id := r.add(func(json.RawMessage, error) { calls.Add(1) })
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			if fn := r.take(id); fn != nil {
				fn(nil, nil)
			}
		})
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("callback completed %d times", calls.Load())
	}
}
