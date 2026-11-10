// Package microphone streams raw audio from the device's microphone, for
// speech providers and other code that processes audio itself: mono,
// signed 16-bit little-endian PCM at SampleRate. It is available on iOS and
// Android; elsewhere Start reports mygo.ErrUnsupported.
//
// It is the fork's (#940 has no counterpart), in the shape of #940's
// plugins (FORK.md, "向 #940 收敛"); until the fork takes #940's backends
// it runs on mygo.Microphone.
package microphone

import (
	"context"

	"github.com/egoist/mygo"
)

// SampleRate is the rate of the audio Start delivers.
const SampleRate = mygo.MicrophoneSampleRate

// Chunk is a piece of captured audio. The last chunk has End set; Err tells
// why capture ended early, and is nil after Stop.
type Chunk = mygo.MicrophoneChunk

// Capture is a running capture. Its methods are safe from any goroutine,
// idempotent, and do nothing after capture ended: Stop delivers the rest of
// the audio, then the End chunk; Cancel releases the microphone at once.
type Capture = mygo.MicrophoneCapture

// Start asks for the microphone's permission and begins capture. Cancel ctx
// to abandon it, which then ends with ctx's error.
//
// emit runs on the audio thread, one call at a time. It must not block or
// touch the user interface, and must not call Stop or Cancel. Only one
// capture runs at a time.
func Start(ctx context.Context, emit func(Chunk)) (*Capture, error) {
	return mygo.Microphone.Start(ctx, emit)
}
