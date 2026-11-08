//go:build ios && cgo

package ios

/*
#include "native.h"
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"
)

// Callbacks arrive on AVAudioEngine's render thread as well as the main
// thread, so the table needs a lock.
var microphones = struct {
	sync.Mutex
	next      uint64
	callbacks map[uint64]func([]byte, error, bool)
}{callbacks: map[uint64]func([]byte, error, bool){}}

// StartMicrophone streams 16 kHz mono little-endian int16 PCM to emit. emit
// runs on the audio render thread and must not block; its last call has end
// set, with an error when capture failed. stop finishes capture and makes
// that last call; cancel releases the microphone without further calls.
func StartMicrophone(emit func(pcm []byte, err error, end bool)) (stop, cancel func()) {
	microphones.Lock()
	microphones.next++
	token := microphones.next
	microphones.callbacks[token] = emit
	microphones.Unlock()
	C.mygo_ios_microphone_start(C.uint64_t(token))
	stop = func() { C.mygo_ios_microphone_stop(C.uint64_t(token), 0) }
	cancel = func() {
		microphones.Lock()
		delete(microphones.callbacks, token)
		microphones.Unlock()
		C.mygo_ios_microphone_stop(C.uint64_t(token), 1)
	}
	return stop, cancel
}

//export goIOSMicrophoneAudio
func goIOSMicrophoneAudio(token C.uint64_t, data unsafe.Pointer, size C.int, failure *C.char, end C.int) {
	microphones.Lock()
	emit := microphones.callbacks[uint64(token)]
	if end != 0 {
		delete(microphones.callbacks, uint64(token))
	}
	microphones.Unlock()
	if emit == nil {
		return
	}
	var err error
	if failure != nil {
		err = errors.New(C.GoString(failure))
	}
	var pcm []byte
	if size > 0 {
		pcm = C.GoBytes(data, size)
	}
	emit(pcm, err, end != 0)
}
