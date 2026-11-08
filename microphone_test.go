package mygo

import (
	"context"
	"errors"
	"testing"
)

func testCapture(t *testing.T) (*MicrophoneCapture, *[]MicrophoneChunk, *int, *int) {
	t.Helper()
	var got []MicrophoneChunk
	var stops, cancels int
	c := &MicrophoneCapture{emit: func(ch MicrophoneChunk) { got = append(got, ch) }, done: make(chan struct{})}
	c.stop = func() { stops++ }
	c.cancel = func() { cancels++ }
	return c, &got, &stops, &cancels
}

func TestMicrophoneUnsupportedOnDesktop(t *testing.T) {
	if microphoneAvailable {
		t.Skip("backend has a microphone")
	}
	if _, err := Microphone.Start(context.Background(), func(MicrophoneChunk) {}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Start = %v, want ErrUnsupported", err)
	}
}

func TestMicrophoneStopDeliversEndOnce(t *testing.T) {
	c, got, stops, _ := testCapture(t)
	c.deliver([]byte{1, 2}, nil, false)
	c.Stop()
	c.Stop()
	c.deliver([]byte{3, 4}, nil, false)
	c.deliver(nil, nil, true)
	c.deliver([]byte{5}, nil, false)
	c.Stop()
	if *stops != 1 {
		t.Fatalf("stops = %d, want 1", *stops)
	}
	if len(*got) != 3 || !(*got)[2].End || (*got)[2].Err != nil {
		t.Fatalf("chunks = %+v", *got)
	}
	select {
	case <-c.done:
	default:
		t.Fatal("done stays open after End")
	}
}

func TestMicrophoneErrorEndsCapture(t *testing.T) {
	c, got, _, _ := testCapture(t)
	failure := errors.New("interrupted")
	c.deliver(nil, failure, false)
	c.deliver([]byte{1}, nil, false)
	if len(*got) != 1 || !(*got)[0].End || !errors.Is((*got)[0].Err, failure) {
		t.Fatalf("chunks = %+v", *got)
	}
}

func TestMicrophoneCancelIsSilent(t *testing.T) {
	c, got, _, cancels := testCapture(t)
	c.Cancel()
	c.Cancel()
	c.deliver([]byte{1}, nil, true)
	if len(*got) != 0 || *cancels != 1 {
		t.Fatalf("chunks = %+v, cancels = %d", *got, *cancels)
	}
}

func TestMicrophoneContextEndReportsError(t *testing.T) {
	c, got, _, cancels := testCapture(t)
	c.end(context.Canceled)
	c.deliver([]byte{1}, nil, false)
	if len(*got) != 1 || !errors.Is((*got)[0].Err, context.Canceled) || *cancels != 1 {
		t.Fatalf("chunks = %+v, cancels = %d", *got, *cancels)
	}
}
