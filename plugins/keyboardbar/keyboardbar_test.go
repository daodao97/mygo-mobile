package keyboardbar

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestLatchAndBar(t *testing.T) {
	var latch ModifierLatch
	modifier := func(id string) ui.Modifiers { return map[string]ui.Modifiers{"ctrl": ui.Ctrl, "shift": ui.Shift}[id] }
	defs := []Action{{ID: "ctrl", Label: "Ctrl", LongPressID: "lock:ctrl"}, {ID: "more", Label: "More", Items: []Action{{ID: "shift", Label: "Shift"}, {ID: "tab", Label: "Tab"}}}}
	var typed []string
	expanded := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Label("Terminal").Size(300, 100).AutoFocus().TextCaret(ui.Rect{X: 4, Y: 4, W: 1, H: 20}).HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind == ui.InputText {
				typed = append(typed, ev.Text)
			}
			return true
		}).InputModifiers(latch.Active(), latch.Consume)
		Bar(c, latch.Decorate(defs, modifier), &expanded, func(id string) {
			switch {
			case id == "lock:ctrl":
				latch.Lock(ui.Ctrl)
			case modifier(id) != 0:
				latch.Tap(modifier(id))
			default:
				typed = append(typed, id)
			}
		})
	}, 400, 300)
	tt.Click("Ctrl")
	if latch.Active() != ui.Ctrl || latch.Locked() != 0 || !tt.Focused("Terminal") {
		t.Fatal("tapping Ctrl did not arm it with the input focused")
	}
	if defs[0].Selected {
		t.Fatal("decorating changed the definitions")
	}
	latch.Consume()
	if latch.Active() != 0 {
		t.Fatal("a one-shot modifier outlived its key")
	}
	latch.Lock(ui.Ctrl)
	latch.Consume()
	if latch.Active() != ui.Ctrl || latch.Locked() != ui.Ctrl {
		t.Fatal("a locked modifier was released by a key")
	}
	tt.Frame()
	if a := latch.Decorate(defs, modifier)[0]; !a.Selected || !a.Locked {
		t.Fatal("decoration lost the lock")
	}
	tt.Click("Ctrl")
	if latch.Active() != 0 || latch.Locked() != 0 {
		t.Fatal("tapping a locked modifier did not release it")
	}
	if tt.HasText("Tab") {
		t.Fatal("a closed panel showed its items")
	}
	tt.Click("More")
	tt.Click("Tab")
	tt.Click("Shift")
	if !expanded || latch.Active() != ui.Shift || len(typed) == 0 || typed[len(typed)-1] != "tab" {
		t.Fatalf("panel actions: expanded %v latch %v typed %v", expanded, latch.Active(), typed)
	}
	if !tt.Focused("Terminal") {
		t.Fatal("the panel took the input's focus")
	}
	if !latch.Clear() || latch.Clear() {
		t.Fatal("Clear did not report its change once")
	}
}
