//go:build android && cgo

package jni

/*
#include <jni.h>
#include <stdlib.h>
static void *mygo_audio_bytes(JNIEnv *e,jbyteArray a,int *n){
 *n=0;if(!a)return NULL;*n=(*e)->GetArrayLength(e,a);if(*n<1||*n>65536)return NULL;
 void*p=malloc(*n);if(p)(*e)->GetByteArrayRegion(e,a,0,*n,p);return p;
}
*/
import "C"
import (
	"encoding/json"
	"errors"
	"github.com/egoist/mygo/internal/android/backend"
	"sync"
	"sync/atomic"
)

var audioSerial atomic.Uint64
var audioCallbacks sync.Map

// BeginAudio starts 16 kHz mono PCM asynchronously. started runs on the Go
// pump; emit runs on the capture thread and must not block or access UI.
func BeginAudio(emit func([]byte, error, bool), started func(error)) (stop, cancel func()) {
	token := audioSerial.Add(1)
	audioCallbacks.Store(token, emit)
	b := backend.New()
	b.Post(func() {
		b.NativeRequest("audio-start", map[string]any{"stream": token}, func(_ json.RawMessage, err error) {
			if err != nil {
				audioCallbacks.Delete(token)
			}
			if started != nil {
				started(err)
			}
		})
	})
	stop = func() { b.Post(func() { _ = b.NativeFire("audio-stop", map[string]any{"stream": token}) }) }
	cancel = func() {
		audioCallbacks.Delete(token)
		b.Post(func() { _ = b.NativeFire("audio-cancel", map[string]any{"stream": token}) })
	}
	return
}

//export Java_org_mygo_Native_onAudio
func Java_org_mygo_Native_onAudio(e *C.JNIEnv, c C.jclass, token C.jlong, data C.jbyteArray, failure C.jbyteArray, end C.jboolean) {
	var value any
	var ok bool
	if end != 0 {
		value, ok = audioCallbacks.LoadAndDelete(uint64(token))
	} else {
		value, ok = audioCallbacks.Load(uint64(token))
	}
	if !ok {
		return
	}
	var n C.int
	p := C.mygo_audio_bytes(e, data, &n)
	var pcm []byte
	if p != nil {
		pcm = C.GoBytes(p, n)
		C.free(p)
	}
	p = C.mygo_audio_bytes(e, failure, &n)
	var err error
	if p != nil {
		err = errors.New(string(C.GoBytes(p, n)))
		C.free(p)
	}
	value.(func([]byte, error, bool))(pcm, err, end != 0)
}
