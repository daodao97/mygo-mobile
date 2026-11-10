package biometrics

import (
	"errors"
	"testing"

	"github.com/egoist/mygo"
)

func TestClassify(t *testing.T) {
	for code, want := range map[mygo.AuthenticationErrorCode]error{
		mygo.AuthenticationCanceled:       ErrCanceled,
		mygo.AuthenticationFallback:       ErrCanceled,
		mygo.AuthenticationFailed:         ErrFailed,
		mygo.AuthenticationUnavailable:    ErrUnavailable,
		mygo.AuthenticationNotEnrolled:    ErrNotEnrolled,
		mygo.AuthenticationLockedOut:      ErrLockedOut,
		mygo.AuthenticationPasscodeNotSet: ErrPasscodeNotSet,
		mygo.AuthenticationBusy:           ErrBusy,
		mygo.AuthenticationInactive:       ErrInactive,
		mygo.AuthenticationMissingPurpose: ErrMissingPurpose,
	} {
		system := &mygo.AuthenticationError{Code: code}
		got := classify(system)
		var a *mygo.AuthenticationError
		if !errors.Is(got, want) || !errors.As(got, &a) || a != system {
			t.Errorf("%s: %v does not wrap %v and the system's error", code, got, want)
		}
	}
	other := errors.New("other")
	if classify(other) != other || classify(nil) != nil {
		t.Error("errors of other kinds changed")
	}
	unknown := &mygo.AuthenticationError{Code: mygo.AuthenticationUnknown}
	if classify(unknown) != error(unknown) {
		t.Error("an unknown code was classified")
	}
}

func TestKindOf(t *testing.T) {
	for _, c := range []struct {
		s    mygo.BiometricStatus
		want Kind
	}{
		{mygo.BiometricStatus{Kind: mygo.BiometricFaceID, Available: true}, FaceID},
		{mygo.BiometricStatus{Kind: mygo.BiometricTouchID, Available: true}, TouchID},
		{mygo.BiometricStatus{Kind: mygo.BiometricFaceID, DeviceAuthenticationAvailable: true}, DeviceCode},
		{mygo.BiometricStatus{Kind: mygo.BiometricNone}, None},
	} {
		if got := kindOf(c.s); got != c.want {
			t.Errorf("%+v: %q, want %q", c.s, got, c.want)
		}
	}
}
