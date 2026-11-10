package mygo

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// MicrophoneSampleRate is the rate of the audio Microphone delivers: mono,
// signed 16-bit little-endian PCM.
const MicrophoneSampleRate = 16000

// MicrophoneChunk is a piece of captured audio. The last chunk has End set;
// Err tells why capture ended early, and is nil after Stop.
type MicrophoneChunk struct {
	PCM []byte
	End bool
	Err error
}

// MicrophoneModule streams raw audio from the device microphone, for
// speech providers and other code that processes audio itself. It is
// available on iOS and Android; other backends return ErrUnsupported.
type MicrophoneModule struct{}

var Microphone MicrophoneModule

// Start requests microphone permission and begins capture. Like
// Speech.Start it can wait for a permission prompt; cancel ctx to abandon
// the capture, which then ends with ctx's error.
//
// emit runs on the audio thread, one call at a time. It must not block or
// touch the user interface, and it must not call Stop or Cancel; hand the
// audio to another goroutine. Only one capture can run at a time.
//
// Deprecated: apps call microphone.Start (plugins/microphone), which calls this.
func (MicrophoneModule) Start(ctx context.Context, emit func(MicrophoneChunk)) (*MicrophoneCapture, error) {
	if ctx == nil || emit == nil {
		return nil, errors.New("mygo: microphone needs a context and a callback")
	}
	if !microphoneAvailable {
		return nil, ErrUnsupported
	}
	needsApp("Microphone.Start")
	status, err := Permissions.Request(PermissionMicrophone)
	if err != nil {
		return nil, err
	}
	if status != PermissionGranted {
		return nil, fmt.Errorf("mygo: microphone permission %s", status)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c := &MicrophoneCapture{emit: emit, done: make(chan struct{})}
	stop, cancel := startMicrophone(c.deliver)
	c.mu.Lock()
	c.stop, c.cancel = stop, cancel
	c.mu.Unlock()
	go func() {
		select {
		case <-ctx.Done():
			c.end(ctx.Err())
		case <-c.done:
		}
	}()
	return c, nil
}

// MicrophoneCapture is a running capture. Its methods are safe from any
// goroutine, idempotent, and do nothing after capture ended.
type MicrophoneCapture struct {
	mu                sync.Mutex
	emit              func(MicrophoneChunk)
	stop, cancel      func()
	done              chan struct{}
	finished, stopped bool
}

// Stop ends capture; the remaining audio arrives, then the End chunk.
func (c *MicrophoneCapture) Stop() {
	c.mu.Lock()
	if c.finished || c.stopped {
		c.mu.Unlock()
		return
	}
	c.stopped = true
	stop := c.stop
	c.mu.Unlock()
	if stop != nil {
		stop()
	}
}

// Cancel releases the microphone at once; emit is not called again.
func (c *MicrophoneCapture) Cancel() {
	c.mu.Lock()
	if c.finished {
		c.mu.Unlock()
		return
	}
	c.finished = true
	close(c.done)
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// end finishes capture with err, delivering it as the End chunk.
func (c *MicrophoneCapture) end(err error) {
	c.mu.Lock()
	if c.finished {
		c.mu.Unlock()
		return
	}
	c.finished = true
	close(c.done)
	cancel := c.cancel
	c.emit(MicrophoneChunk{End: true, Err: err})
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// deliver receives the backend's audio. The lock keeps chunks in order and
// keeps any from arriving after End or Cancel.
func (c *MicrophoneCapture) deliver(pcm []byte, err error, end bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.finished {
		return
	}
	end = end || err != nil
	if end {
		c.finished = true
		close(c.done)
	}
	c.emit(MicrophoneChunk{PCM: pcm, End: end, Err: err})
}
