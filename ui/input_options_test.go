package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestInputOptionsFollowFieldAndReset(t *testing.T) {
	value, keyboard, ret := "", KeyboardEmail, ReturnNext
	enabled := true
	tt := NewTester(func(c *Context) {
		e := TextInput(c, &value).Label("Email").Width(300).AutoFocus()
		if enabled {
			e.Keyboard(keyboard).ReturnKey(ret).Autofill(AutofillEmail).Autocorrect(false).Capitalize(CapitalizeNone)
		}
	}, 320, 200)
	if got := tt.h.ime.Options; got.Keyboard != "email" || got.Return != "next" || got.Content != "emailAddress" || got.Correction != "off" || got.Capitalization != "none" {
		t.Fatalf("keyboard options not forwarded: %+v", got)
	}
	tt.Type("中文👋@example.com")
	if value != "中文👋@example.com" {
		t.Fatalf("keyboard hint filtered text: %q", value)
	}
	keyboard, ret = KeyboardNumber, ReturnDone
	tt.Frame()
	if tt.h.ime.Options.Keyboard != "number" || tt.h.ime.Options.Return != "done" {
		t.Fatal("changing traits was not propagated")
	}
	enabled = false
	tt.Frame()
	if tt.h.ime.Options != (platform.TextInputOptions{}) {
		t.Fatal("omitted traits leaked from a previous frame")
	}
}

func TestInputReturnPreservesSubmitAndMultiline(t *testing.T) {
	for _, multiline := range []bool{false, true} {
		value, submitted := "", false
		tt := NewTester(func(c *Context) {
			var e Element
			if multiline {
				e = TextArea(c, &value)
			} else {
				e = TextInput(c, &value)
			}
			e.ReturnKey(ReturnSend).Width(300).AutoFocus()
			if e.Submitted() {
				submitted = true
			}
		}, 320, 200)
		tt.Type("hello")
		tt.Key(0, KeyEnter)
		if multiline && (value != "hello\n" || submitted) {
			t.Fatalf("text area submitted or lost newline: %q %v", value, submitted)
		}
		if !multiline && (value != "hello" || !submitted) {
			t.Fatalf("single line did not submit: %q %v", value, submitted)
		}
	}
}

func TestCustomTextCaretKeyboardOptions(t *testing.T) {
	enabled := true
	var received string
	tt := NewTester(func(c *Context) {
		e := Box(c).Size(300, 100).AutoFocus().TextCaret(Rect{X: 8, Y: 8, W: 1, H: 18}).HandleInput(func(ev InputEvent) bool {
			if ev.Kind == InputText {
				received += ev.Text
			}
			return true
		})
		if enabled {
			e.Keyboard(KeyboardASCII).Autocorrect(false).Capitalize(CapitalizeNone)
		}
	}, 320, 200)
	if got := tt.h.ime.Options; !tt.h.ime.Active || got.Keyboard != "ascii" || got.Correction != "off" || got.Capitalization != "none" {
		t.Fatalf("custom input did not receive keyboard options: %+v", tt.h.ime)
	}
	tt.Type("echo 中文")
	if received != "echo 中文" {
		t.Fatalf("custom input changed committed text: %q", received)
	}
	enabled = false
	tt.Frame()
	if tt.h.ime.Options != (platform.TextInputOptions{}) {
		t.Fatal("custom input retained omitted keyboard options")
	}
}

func TestTextInputClientKeyboardOptions(t *testing.T) {
	client := &primitiveClient{}
	step := 0
	tt := NewTester(func(c *Context) {
		e := Box(c).Fill().HandleTextInput(client).AutoFocus()
		switch step {
		case 0:
			e.Keyboard(KeyboardASCII).ReturnKey(ReturnSend).Autocorrect(false).Capitalize(CapitalizeNone).KeyboardDismiss(KeyboardDismissOnDrag)
		case 1:
			e.Keyboard(KeyboardEmail).ReturnKey(ReturnDone)
		}
	}, 320, 200)
	defer tt.rt.close()
	want := platform.TextInputOptions{Keyboard: "ascii", Return: "send", Correction: "off", Capitalization: "none", Dismiss: "on-drag"}
	if got := tt.h.ime; got.Client == nil || !got.Active || got.Options != want {
		t.Fatalf("client keyboard options not forwarded: %+v", got)
	}
	tt.Type("中文👋")
	if client.text != "中文👋" {
		t.Fatalf("keyboard hint changed committed text: %q", client.text)
	}
	step = 1
	tt.Frame()
	if want := (platform.TextInputOptions{Keyboard: "email", Return: "done"}); tt.h.ime.Options != want {
		t.Fatal("changed client keyboard options were not propagated")
	}
	step = 2
	tt.Frame()
	if tt.h.ime.Options != (platform.TextInputOptions{}) {
		t.Fatal("client retained omitted keyboard options")
	}
}
