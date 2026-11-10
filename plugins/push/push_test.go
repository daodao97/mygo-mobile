package push

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/internal/pushhub"
)

func TestDataOf(t *testing.T) {
	if string(dataOf(pushhub.Event{})) != "{}" {
		t.Fatal("no data is not an empty object")
	}
	if got := string(dataOf(pushhub.Event{Data: map[string]string{"thread": "4"}})); got != `{"thread":"4"}` {
		t.Fatalf("data %s", got)
	}
}

func TestHooksAreSet(t *testing.T) {
	if pushhub.Received == nil || pushhub.Registered == nil {
		t.Fatal("the plugin did not set pushhub's hooks")
	}
}

func TestReceived(t *testing.T) {
	defer SetPresentation(nil)
	pushhub.Received(pushhub.Event{ID: "cold", Remote: true, Clicked: true, Data: map[string]string{"n": "1"}})
	var opens, msgs []string
	offOpen := OnOpened(func(d []byte) { opens = append(opens, string(d)) })
	defer offOpen()
	offMsg := OnMessage(func(d []byte) { msgs = append(msgs, string(d)) })
	defer offMsg()
	SetPresentation(func(d []byte) bool { return string(d) != `{"thread":"4"}` })
	if pushhub.Received(pushhub.Event{Remote: true, Data: map[string]string{"thread": "4"}}) {
		t.Error("a push SetPresentation hid was shown")
	}
	if !pushhub.Received(pushhub.Event{Remote: true}) {
		t.Error("a push SetPresentation showed was hidden")
	}
	if pushhub.Received(pushhub.Event{Data: map[string]string{"thread": "4"}}) {
		t.Error("the app's own notification was not asked about")
	}
	if !pushhub.Received(pushhub.Event{Clicked: true, Data: map[string]string{"thread": "4"}}) {
		t.Error("a tap was hidden")
	}
	pushhub.Received(pushhub.Event{Remote: true, Clicked: true})
	if len(opens) != 2 || opens[0] != `{"n":"1"}` || opens[1] != "{}" {
		t.Fatalf("opened %q", opens)
	}
	if len(msgs) != 2 {
		t.Fatalf("messages %q", msgs)
	}
	offOpen()
	var again []string
	off := OnOpened(func(d []byte) { again = append(again, string(d)) })
	defer off()
	if len(again) != 0 {
		t.Fatal("a tap was replayed twice")
	}
}

func TestRegistered(t *testing.T) {
	var got []string
	off := OnToken(func(s string) { got = append(got, s) })
	defer off()
	pushhub.Registered("abc", nil)
	pushhub.Registered("", errors.New("denied"))
	off()
	pushhub.Registered("def", nil)
	if len(got) != 1 || got[0] != "abc" {
		t.Fatalf("tokens %q", got)
	}
}
