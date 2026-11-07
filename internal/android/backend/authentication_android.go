//go:build android && cgo

package backend

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/egoist/mygo/internal/platform"
)

func (b *Backend) Authenticator() platform.Authenticator { return androidAuthenticator{b} }

type androidAuthenticator struct{ b *Backend }

func (a androidAuthenticator) Query(done func(platform.BiometricStatus, error)) {
	a.b.NativeRequest("auth-query", nil, func(v json.RawMessage, err error) {
		var status platform.BiometricStatus
		if err == nil {
			err = json.Unmarshal(v, &status)
		}
		done(status, err)
	})
}
func (a androidAuthenticator) Authenticate(o platform.AuthenticationOptions, done func(error)) func() {
	return a.b.NativeRequest("authenticate", o, func(_ json.RawMessage, err error) {
		var native *NativeError
		if errors.As(err, &native) {
			err = &platform.AuthenticationError{Code: platform.AuthenticationErrorCode(native.Code), Message: native.Message}
		}
		if errors.Is(err, context.Canceled) {
			err = &platform.AuthenticationError{Code: platform.AuthenticationCanceled, Message: "Authentication canceled"}
		}
		done(err)
	})
}
