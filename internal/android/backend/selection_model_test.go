package backend

import (
	"github.com/egoist/mygo/internal/platform"
	"testing"
)

type clippedSelectionClient struct {
	recordingClient
	visible bool
	rect    platform.RectF
}

func (c *clippedSelectionClient) SetSelection(s platform.TextSelection) { c.sel = s }
func (c *clippedSelectionClient) BoundsForRange(r platform.TextRange) (platform.RectF, platform.TextRange, bool) {
	return c.rect, r, c.visible
}
func TestSelectionMenuSurvivesOffscreenEndpoints(t *testing.T) {
	c := &clippedSelectionClient{recordingClient: recordingClient{text: "selected", sel: platform.TextSelection{Range: platform.TextRange{Start: 0, End: 8}}}}
	state := platform.TextInputState{Active: true, ID: 42, Bounds: platform.RectF{X: 10, Y: 20, W: 100, H: 30}}
	v := selectionPresentation(state, c)
	if !v.Active || v.StartVisible || v.EndVisible || len(v.Commands) == 0 {
		t.Fatal(v)
	}
	c.visible = true
	c.rect = platform.RectF{X: 15, Y: 25, W: 1, H: 20}
	v = selectionPresentation(state, c)
	if !v.StartVisible || !v.EndVisible {
		t.Fatal(v)
	}
	// Still inside the full window, but outside the text's scroll viewport.
	c.rect.Y = 80
	v = selectionPresentation(state, c)
	if !v.Active || v.StartVisible || v.EndVisible {
		t.Fatal(v)
	}
	state.Password = true
	v = selectionPresentation(state, c)
	if v.Active || len(v.Commands) != 0 {
		t.Fatal(v)
	}
}
