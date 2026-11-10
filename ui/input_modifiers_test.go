package ui

import "testing"

func TestSoftwareInputModifiersRespectCompositionAndSingleKeyConsumption(t *testing.T) {
	mods, consumed := Ctrl, 0
	var events []InputEvent
	tt := NewTester(func(c *Context) {
		Box(c).Fill().AutoFocus().TextCaret(Rect{W: 1, H: 20}).InputModifiers(mods, func() { consumed++; mods = 0 }).HandleInput(func(ev InputEvent) bool { events = append(events, ev); return true })
	}, 320, 180)
	tt.Type("c")
	if events[len(events)-1].Mods != Ctrl || consumed != 1 {
		t.Fatal("software Ctrl did not reach next letter", events, consumed)
	}
	tt.Type("d")
	if events[len(events)-1].Mods != 0 {
		t.Fatal("one-shot Ctrl leaked")
	}
	mods = Alt
	tt.Frame()
	tt.Compose("zhong", 5)
	tt.Type("中")
	if events[len(events)-1].Mods != 0 || consumed != 1 || mods != Alt {
		t.Fatal("candidate commit consumed the modifier")
	}
	tt.Type("b")
	if events[len(events)-1].Mods != Alt || consumed != 2 {
		t.Fatal("modifier not available after candidate commit")
	}
	mods = Super
	tt.Frame()
	tt.Type("pasted text")
	if events[len(events)-1].Mods != 0 || consumed != 2 {
		t.Fatal("multi-character text turned into shortcuts")
	}
}


func TestSoftwareControlKeyReleaseKeepsConsumedModifiers(t *testing.T) {
	mods := Alt
	var events []InputEvent
	tt := NewTester(func(c *Context) {
		Box(c).Fill().AutoFocus().TextCaret(Rect{W: 1, H: 20}).InputModifiers(mods, func() { mods = 0 }).HandleInput(func(ev InputEvent) bool { events = append(events, ev); return true })
	}, 320, 180)
	tt.Key(0, KeyBackspace)
	n := len(events)
	if n < 2 || events[n-2].Mods != Alt || events[n-1].Mods != Alt || !events[n-1].Software || mods != 0 {
		t.Fatal("release lost one-shot modifiers", events)
	}
	tt.Key(0, KeyBackspace)
	if events[len(events)-1].Mods != 0 {
		t.Fatal("modifier leaked into following key")
	}
}
