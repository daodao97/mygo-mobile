//go:build android && cgo

package backend

import (
	"context"
	"encoding/json"
)

// NativeRequest and its cancel function run on the pump. A callback is removed
// before delivery, so cancellation and a late Java reply cannot finish twice.
func (b *Backend) NativeRequest(kind string, value any, done func(json.RawMessage, error)) func() {
	bridge := nativeServiceBridge()
	if bridge == nil {
		done(nil, ErrPlatformBridge)
		return func() {}
	}
	id := nativeRequests.add(done)
	data, err := json.Marshal(struct {
		ID    uint64 `json:"id"`
		Kind  string `json:"kind"`
		Value any    `json:"value"`
	}{id, kind, value})
	if err == nil {
		err = bridge.Request(data)
	}
	if err != nil {
		if fn := nativeRequests.take(id); fn != nil {
			fn(nil, err)
		}
	}
	return func() {
		if fn := nativeRequests.take(id); fn != nil {
			bridge.Cancel(id)
			fn(nil, context.Canceled)
		}
	}
}
func (b *Backend) NativeResult(id uint64, value json.RawMessage, code, message string) {
	b.Post(func() {
		if fn := nativeRequests.take(id); fn != nil {
			fn(value, nativeFailure(code, message))
		}
	})
}

// NativeFire queues a fire-and-forget system effect. Delivery failures before
// queuing are returned; device feedback may still be suppressed by the OS.
func (b *Backend) NativeFire(kind string, value any) error {
	bridge := nativeServiceBridge()
	if bridge == nil {
		return ErrPlatformBridge
	}
	data, err := json.Marshal(struct {
		ID    uint64 `json:"id"`
		Kind  string `json:"kind"`
		Value any    `json:"value"`
	}{0, kind, value})
	if err != nil {
		return err
	}
	return bridge.Request(data)
}
