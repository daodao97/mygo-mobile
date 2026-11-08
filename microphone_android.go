//go:build android && cgo

package mygo

import "github.com/egoist/mygo/android/jni"

const microphoneAvailable = true

func startMicrophone(emit func([]byte, error, bool)) (stop, cancel func()) {
	return jni.BeginAudio(emit, func(err error) {
		if err != nil {
			emit(nil, err, true)
		}
	})
}
