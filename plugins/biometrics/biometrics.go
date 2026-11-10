// Package biometrics confirms it is the device's owner with what the device
// has: Face ID or Touch ID on iOS (LocalAuthentication), and the system's
// prompt on Android, falling back to the device's passcode. Elsewhere its
// functions report mygo.ErrUnsupported.
//
// Its API is upstream #940's plugins/biometrics (FORK.md, "向 #940 收敛");
// until the fork takes #940's backends it runs on mygo.Biometrics. The
// errors after ErrFailed are the fork's: #940 tells only a cancel and a
// failure apart.
package biometrics

import (
	"context"
	"errors"
	"fmt"

	"github.com/egoist/mygo"
)

// Kind is what the device confirms its owner with.
type Kind string

const (
	None       Kind = ""
	FaceID     Kind = "face"
	TouchID    Kind = "fingerprint"
	Iris       Kind = "iris"
	DeviceCode Kind = "passcode"
	Biometric  Kind = "biometric"
)

// ErrCanceled is what Authenticate fails with as the user goes back.
var ErrCanceled = errors.New("biometrics: canceled")

// ErrFailed is what Authenticate fails with as the device didn't recognize
// the user.
var ErrFailed = errors.New("biometrics: not recognized")

// The fork's: why Authenticate could not ask at all. The error Authenticate
// returns wraps one of them and the system's own.
var (
	// ErrUnavailable: the device has nothing to confirm its owner with.
	ErrUnavailable = errors.New("biometrics: unavailable")
	// ErrNotEnrolled: biometrics are not set up.
	ErrNotEnrolled = errors.New("biometrics: not enrolled")
	// ErrLockedOut: too many failures; the system locked biometrics.
	ErrLockedOut = errors.New("biometrics: locked out")
	// ErrPasscodeNotSet: the device has no passcode to fall back to.
	ErrPasscodeNotSet = errors.New("biometrics: no device passcode")
	// ErrBusy: another prompt of the system is showing.
	ErrBusy = errors.New("biometrics: busy")
	// ErrInactive: the app is not in front, where the prompt shows.
	ErrInactive = errors.New("biometrics: app inactive")
	// ErrMissingPurpose: the app's bundle lacks the usage description the
	// system asks for (NSFaceIDUsageDescription).
	ErrMissingPurpose = errors.New("biometrics: missing usage description")
)

// Available tells what the device can confirm its owner with now, None when
// nothing, as without a passcode set.
func Available(ctx context.Context) (Kind, error) {
	if err := ctx.Err(); err != nil {
		return None, err
	}
	s, err := mygo.Biometrics.Query()
	if err != nil {
		return None, err
	}
	return kindOf(s), nil
}

func kindOf(s mygo.BiometricStatus) Kind {
	switch {
	case s.Available && s.Kind == mygo.BiometricFaceID:
		return FaceID
	case s.Available && s.Kind == mygo.BiometricTouchID:
		return TouchID
	case s.Available:
		return Biometric
	case s.DeviceAuthenticationAvailable:
		return DeviceCode
	}
	return None
}

// Authenticate asks the user to confirm it's them, saying reason. It returns
// nil once they did. Call it off the main goroutine, as it waits for the user.
func Authenticate(ctx context.Context, reason string) error {
	return classify(mygo.Biometrics.Authenticate(ctx, mygo.AuthenticationOptions{Reason: reason, AllowDevicePasscode: true}))
}

// classify wraps the system's error in the one of this package it is.
func classify(err error) error {
	var a *mygo.AuthenticationError
	if !errors.As(err, &a) {
		return err
	}
	var kind error
	switch a.Code {
	case mygo.AuthenticationCanceled, mygo.AuthenticationFallback:
		kind = ErrCanceled
	case mygo.AuthenticationFailed:
		kind = ErrFailed
	case mygo.AuthenticationUnavailable:
		kind = ErrUnavailable
	case mygo.AuthenticationNotEnrolled:
		kind = ErrNotEnrolled
	case mygo.AuthenticationLockedOut:
		kind = ErrLockedOut
	case mygo.AuthenticationPasscodeNotSet:
		kind = ErrPasscodeNotSet
	case mygo.AuthenticationBusy:
		kind = ErrBusy
	case mygo.AuthenticationInactive:
		kind = ErrInactive
	case mygo.AuthenticationMissingPurpose:
		kind = ErrMissingPurpose
	default:
		return err
	}
	return fmt.Errorf("%w: %w", kind, err)
}
