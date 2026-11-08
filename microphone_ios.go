//go:build ios && cgo

package mygo

import "github.com/egoist/mygo/internal/ios"

const microphoneAvailable = true

func startMicrophone(emit func([]byte, error, bool)) (stop, cancel func()) {
	return ios.StartMicrophone(emit)
}
