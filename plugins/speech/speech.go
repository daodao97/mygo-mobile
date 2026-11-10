// Package speech turns speech into text as the user speaks: with iOS's
// Speech framework (the "apple" provider), or with a provider the app
// registers, such as a cloud service in Go capturing with
// plugins/microphone. Elsewhere the "apple" provider reports
// mygo.ErrUnsupported.
//
// It is the fork's (#940 has no counterpart), in the shape of #940's
// plugins (FORK.md, "向 #940 收敛"); until the fork takes #940's backends
// it runs on mygo.Speech.
package speech

import (
	"context"

	"github.com/egoist/mygo"
)

// Options configure a session: the Provider ("apple" when empty), its
// Language (a BCP-47 tag, the system's when empty), OnDevice, which forbids
// Apple's network recognition, ContextualStrings, hints for names, and
// MaxDuration, 60 seconds when zero.
type Options = mygo.SpeechOptions

// Result replaces the whole transcript. Final ends the session; an error
// ends it too, and may keep a provisional transcript in Text.
type Result = mygo.SpeechResult

// Recording controls a provider's capture. Its methods are safe from any
// goroutine, idempotent, and safe after it ended.
type Recording = mygo.SpeechRecording

// Provider is a speech-to-text service. Start honors ctx while preparing
// capture; emit may run on any goroutine, also during Start.
type Provider = mygo.SpeechProvider

// Session is one capture: Results streams the newest transcript, and never
// loses the final one.
type Session = mygo.SpeechSession

// RegisterProvider adds a provider under name, which Options.Provider
// names. "apple" is built in.
func RegisterProvider(name string, p Provider) error { return mygo.Speech.RegisterProvider(name, p) }

// Start asks for the permissions the provider needs and begins capture.
// It can wait for a prompt: call it off the main goroutine, and cancel ctx
// as the user lets go, navigates away or the app goes to the background.
func Start(ctx context.Context, o Options) (*Session, error) { return mygo.Speech.Start(ctx, o) }

// Authorize asks for Apple's microphone and speech permissions without
// capturing, as a voice panel opens, so that the prompts do not interrupt
// a press-and-hold. Providers of the app's own ask for theirs.
func Authorize(ctx context.Context) error { return mygo.Speech.Authorize(ctx) }
