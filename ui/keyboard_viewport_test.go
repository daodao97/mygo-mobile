package ui

import "testing"

func TestShrinkingViewportRevealsFocusedTextInput(t *testing.T) {
	for _, password := range []bool{false, true} {
		t.Run(map[bool]string{false: "text", true: "password"}[password], func(t *testing.T) {
			value := "test code"
			var scroll ScrollState
			tt := coreNewTester(func(c *context) {
				coreScroll(c).Grow(1).MinHeight(0).TrackScroll(&scroll).Children(func() {
					coreBox(c).Height(500).Shrink(0)
					e := coreTextInput(c, &value).Label("Code").Height(44).Shrink(0)
					if password {
						e.Password()
					}
					coreBox(c).Height(300).Shrink(0)
				})
				coreBox(c).Height(60).Shrink(0)
			}, 390, 844)
			if err := tt.Click("Code"); err != nil {
				t.Fatal(err)
			}
			for _, height := range []int{700, 600, 510, 440} {
				tt.SetSize(390, height)
				r, ok := tt.Find("Code")
				caret, active := tt.TextCaret()
				if !ok || r.H < 44 || r.Y < 0 || r.Y+r.H > float32(height-60) || !active || caret.Y < r.Y || caret.Y+caret.H > r.Y+r.H+1 {
					t.Fatalf("height=%d input=%v caret=%v active=%v", height, r, caret, active)
				}
			}
			// The user can scroll away without the focus dragging the view back.
			scroll.Y = 0
			tt.Frame()
			if scroll.Y != 0 {
				t.Fatalf("ordinary scrolling was overridden: %v", scroll.Y)
			}
			// Dismissing the keyboard also keeps the user's place.
			tt.SetSize(390, 844)
			if scroll.Y != 0 {
				t.Fatalf("growing viewport moved to focus: %v", scroll.Y)
			}
		})
	}
}

func TestShrinkingViewportDoesNotRevealNonTextFocus(t *testing.T) {
	var scroll ScrollState
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().TrackScroll(&scroll).Children(func() {
			coreBox(c).Height(500).Shrink(0)
			coreButton(c, "Continue").Height(44).Shrink(0)
			coreBox(c).Height(300).Shrink(0)
		})
	}, 390, 844)
	if err := tt.Click("Continue"); err != nil {
		t.Fatal(err)
	}
	tt.SetSize(390, 440)
	if scroll.Y != 0 {
		t.Fatalf("non-text focus moved the viewport: %v", scroll.Y)
	}
}
