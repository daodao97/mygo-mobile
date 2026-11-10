// Package push receives push notifications on phones: APNs on iOS. It tells
// the token that the app's server sends to. iOS needs the Push Notifications
// capability in the app's provisioning profile. Elsewhere its functions
// report mygo.ErrUnsupported.
//
// Its API is upstream #940's plugins/push (FORK.md, "向 #940 收敛"), on the
// fork's backends until it takes #940's. Unlike #940's, Register does not
// ask the user to allow notifications (plugins/permissionflow does), and
// Android has no Firebase: the fork's Android host tries the push channels
// the app registers itself (mygo.App.OnPushSubscription). SetPresentation
// is the fork's.
package push

import (
	"context"
	"encoding/json"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/pushhub"
)

// Firebase is what Android's messaging needs of the app's Firebase project.
type Firebase struct {
	ProjectID string `json:"projectId"`
	AppID     string `json:"appId"`
	APIKey    string `json:"apiKey"`
	SenderID  string `json:"senderId"`
}

// Configure sets up Firebase, on Android. Call it before Register. iOS ignores
// it. The fork's Android has no Firebase: it reports mygo.ErrUnsupported.
func Configure(ctx context.Context, f Firebase) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if runtime.GOOS == "ios" {
		return nil
	}
	return mygo.ErrUnsupported
}

// Register returns the token that the app's server sends pushes to, which
// may change: see OnToken. Call it off the main goroutine, as it waits for
// the system.
func Register(ctx context.Context) (string, error) {
	tokenCh, errCh := make(chan string, 1), make(chan error, 1)
	offToken := tokens.add(func(t string) {
		select {
		case tokenCh <- t:
		default:
		}
	})
	defer offToken()
	offErr := errs.add(func(err error) {
		select {
		case errCh <- err:
		default:
		}
	})
	defer offErr()
	if err := mygo.App.RegisterPushNotifications(); err != nil {
		return "", err
	}
	select {
	case t := <-tokenCh:
		return t, nil
	case err := <-errCh:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// OnToken calls fn on the main thread as the token changes.
func OnToken(fn func(token string)) (off func()) { return tokens.add(fn) }

// OnMessage calls fn on the main thread as a push arrives while the app runs,
// with what it carries, as JSON.
func OnMessage(fn func(data []byte)) (off func()) {
	return messages.add(func(e pushhub.Event) { fn(dataOf(e)) })
}

// OnOpened calls fn on the main thread as the user taps a push's
// notification, with what it carries, as JSON: also the tap that launched
// the app, once this listens.
func OnOpened(fn func(data []byte)) (off func()) {
	return onOpened(func(e pushhub.Event) { fn(dataOf(e)) })
}

// SetPresentation decides, on the main thread, whether a notification that
// arrives while the app is in front shows (iOS): show is given what it
// carries, as JSON, the app's own notifications' data included. It is the
// fork's; nil shows them all.
func SetPresentation(show func(data []byte) bool) {
	var fn func(pushhub.Event) bool
	if show != nil {
		fn = func(e pushhub.Event) bool { return show(dataOf(e)) }
	}
	mu.Lock()
	present = fn
	mu.Unlock()
}

// dataOf is what e carries, as a JSON object.
func dataOf(e pushhub.Event) []byte {
	if len(e.Data) == 0 {
		return []byte("{}")
	}
	b, err := json.Marshal(e.Data)
	if err != nil {
		return []byte("{}")
	}
	return b
}
