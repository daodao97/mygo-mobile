package backend

import (
	"context"
	"encoding/json"
	"github.com/egoist/mygo/internal/platform"
	"sync"
)

// NativeServiceBridge only enqueues Java UI work. It must never wait for a
// Java UI result: that result is delivered later on the Go pump.
type NativeServiceBridge interface {
	Request([]byte) error
	Cancel(uint64)
}

type NativeError struct{ Code, Message string }

func (e *NativeError) Error() string { return "mygo: Android " + e.Code + ": " + e.Message }

var nativeBridge struct {
	sync.RWMutex
	bridge NativeServiceBridge
}

func SetNativeServiceBridge(b NativeServiceBridge) {
	nativeBridge.Lock()
	nativeBridge.bridge = b
	nativeBridge.Unlock()
}
func nativeServiceBridge() NativeServiceBridge {
	nativeBridge.RLock()
	defer nativeBridge.RUnlock()
	return nativeBridge.bridge
}

type nativeCallback func(json.RawMessage, error)
type requestRegistry struct {
	sync.Mutex
	next    uint64
	pending map[uint64]nativeCallback
}

func (r *requestRegistry) add(fn nativeCallback) uint64 {
	r.Lock()
	defer r.Unlock()
	r.next++
	if r.pending == nil {
		r.pending = make(map[uint64]nativeCallback)
	}
	r.pending[r.next] = fn
	return r.next
}
func (r *requestRegistry) take(id uint64) nativeCallback {
	r.Lock()
	defer r.Unlock()
	fn := r.pending[id]
	delete(r.pending, id)
	return fn
}

var nativeRequests requestRegistry

func nativeFailure(code, message string) error {
	if code == "" {
		return nil
	}
	if code == "notifications-denied" {
		return platform.ErrNotificationsDenied
	}
	if code == "unsupported" {
		return platform.ErrUnsupported
	}
	if code == "canceled" {
		return context.Canceled
	}
	return &NativeError{Code: code, Message: message}
}
