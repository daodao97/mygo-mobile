package ui

import "testing"

func TestWidgetNativeSelectionMovesWithoutEditing(t *testing.T) {
	for _, area := range []bool{false, true} {
		value := "A👩中文 B"
		tt := NewTester(func(c *Context) {
			if area {
				TextArea(c, &value).Label("input")
			} else {
				TextInput(c, &value).Label("input")
			}
		}, 320, 180)
		tt.Click("input")
		client := tt.h.ime.Client
		setter, ok := client.(interface{ SetSelection(TextInputSelection) })
		if !ok {
			t.Fatal("selection setter absent")
		}
		wanted := TextInputSelection{Range: TextInputRange{Start: 1, End: 5}, Reversed: true}
		setter.SetSelection(wanted)
		if client.Selection() != wanted || value != "A👩中文 B" {
			t.Fatal(client.Selection(), value)
		}
		client.ReplaceText(nil, "X")
		tt.Frame()
		if value != "AX B" {
			t.Fatal(value)
		}
		tt.rt.close()
	}
}

func TestWidgetNativeComposingSelectionKeepsDirection(t *testing.T) {
	value := ""
	tt := inputTester(&value, false)
	defer tt.rt.close()
	c := tt.h.ime.Client
	c.SetMarkedText(nil, "nihao", TextInputRange{Start: 5, End: 5})
	wanted := TextInputSelection{Range: TextInputRange{Start: 1, End: 3}, Reversed: true}
	c.(interface{ SetSelection(TextInputSelection) }).SetSelection(wanted)
	if c.Selection() != wanted || c.Selection().Caret() != 1 {
		t.Fatal(c.Selection())
	}
	if mark, ok := c.MarkedRange(); !ok || mark != (TextInputRange{Start: 0, End: 5}) {
		t.Fatal(mark, ok)
	}
	if value != "" {
		t.Fatal("preedit was committed", value)
	}
}
