package backend

import (
	"github.com/egoist/mygo/internal/platform"
	"testing"
)

func TestReadOnlySnapshotUnicodeSelectionAndGeometry(t *testing.T) {
	var selected platform.TextRange
	var query platform.TextGeometryQuery
	c := &readOnlyTextClient{
		state:       platform.TextInputState{Active: true, ReadOnly: true, ID: 7, Text: "A👩中文\nB", Start: 1, End: 4},
		selectRange: func(a, z int) { selected = platform.TextRange{Start: a, End: z} },
		geometry: func(q platform.TextGeometryQuery) platform.TextGeometry {
			query = q
			return platform.TextGeometry{Valid: true, Index: 3, Caret: platform.RectF{X: 12, Y: 34, W: 1, H: 20}}
		},
	}
	if got := c.Selection().Range; got != (platform.TextRange{Start: 1, End: 5}) {
		t.Fatal(got)
	}
	text, r := c.TextForRange(platform.TextRange{Start: 2, End: 5})
	if text != "👩中文" || r != (platform.TextRange{Start: 1, End: 5}) {
		t.Fatal(text, r)
	}
	c.SetSelection(platform.TextSelection{Range: platform.TextRange{Start: 1, End: 6}})
	if c.Selection().Range != (platform.TextRange{Start: 1, End: 6}) {
		t.Fatal("stale range before frame", c.Selection())
	}
	if selected != (platform.TextRange{Start: 1, End: 5}) {
		t.Fatal(selected)
	}
	_, _, ok := c.BoundsForRange(platform.TextRange{Start: 4, End: 4})
	if !ok || query.ID != 7 || query.Start != 3 || query.Kind != "caret" {
		t.Fatal(query)
	}
	at, ok := c.IndexForPoint(20, 30)
	if !ok || at != 4 || query.Kind != "hit" || query.X != 20 || query.Y != 30 {
		t.Fatal(at, query)
	}
	c.SelectionRectsForRange(platform.TextRange{Start: 1, End: 6})
	if query.Start != 1 || query.End != 5 || query.Kind != "selection" {
		t.Fatal(query)
	}
	old := c.state
	c.ReplaceText(nil, "changed")
	c.SetMarkedText(nil, "changed", platform.TextRange{})
	c.UnmarkText()
	if c.state != old {
		t.Fatal("read-only client changed text")
	}
}

type selectionRecordingClient struct{ recordingClient }

func (c *selectionRecordingClient) SetSelection(s platform.TextSelection) { c.sel = s }
func TestIMESelectionUsesOptionalSetterWithoutDeletingText(t *testing.T) {
	c := &selectionRecordingClient{recordingClient: recordingClient{text: "A👩中文", sel: platform.TextSelection{Range: platform.TextRange{Start: 5, End: 5}}}}
	m := &Ime{}
	m.SetTextInput(platform.TextInputState{Active: true, Client: c})
	if !m.SetSelection(1, 5) || c.sel.Range != (platform.TextRange{Start: 1, End: 5}) || c.text != "A👩中文" || len(c.ops) != 0 {
		t.Fatal(c)
	}
	if !m.CommitText("OK", 1) || c.text != "AOK" {
		t.Fatal(c.text)
	}
}
func TestIMEReadOnlyDoesNotOpenKeyboardOrAcceptEdits(t *testing.T) {
	c := &recordingClient{text: "unchanged"}
	keyboard := true
	m := &Ime{OnActive: func(on bool) { keyboard = on }}
	m.SetTextInput(platform.TextInputState{Active: true, ReadOnly: true, Client: c})
	if keyboard || m.CommitText("x", 1) || m.SetComposingText("x", 1) || m.DeleteSurroundingText(1, 1) || m.FinishComposingText() {
		t.Fatal("read-only editor accepted keyboard edits")
	}
	if c.text != "unchanged" || len(c.ops) != 0 {
		t.Fatal(c)
	}
}

func TestIMEUnchangedSelectionRetainsPendingComposition(t *testing.T) {
	c := &selectionRecordingClient{}
	m := &Ime{}
	m.SetTextInput(platform.TextInputState{Active: true, Client: c})
	if !m.SetComposingText("ni", 1) || !m.FinishComposingText() {
		t.Fatal("composition failed")
	}
	caret := c.Selection().Caret()
	if !m.SetSelection(caret, caret) || !m.CommitText("你", 1) || c.text != "你" {
		t.Fatal(c.text, c.ops)
	}
}
func TestIMESelectionRetainsActiveEndpoint(t *testing.T) {
	c := &selectionRecordingClient{recordingClient: recordingClient{text: "A👩中文"}}
	m := &Ime{}
	m.SetTextInput(platform.TextInputState{Active: true, Client: c})
	if !m.SetSelection(5, 1) || !c.sel.Reversed || c.sel.Caret() != 1 || c.text != "A👩中文" {
		t.Fatal(c)
	}
}

type legacySelectionAdapter struct{ recordingClient }

func (*legacySelectionAdapter) SetSelection(platform.TextSelection) {}
func (*legacySelectionAdapter) SupportsNativeTextSelection() bool   { return false }
func TestIMELegacyAdapterMovesCaretWithoutClaimingRange(t *testing.T) {
	c := &legacySelectionAdapter{recordingClient: recordingClient{text: "A👩中文", sel: platform.TextSelection{Range: platform.TextRange{Start: 5, End: 5}}}}
	m := &Ime{}
	m.SetTextInput(platform.TextInputState{Active: true, Client: c})
	if !m.SetSelection(1, 1) || c.Selection().Caret() != 1 || c.text != "A👩中文" {
		t.Fatal(c)
	}
	if m.SetSelection(1, 5) || c.Selection().Caret() != 1 || c.text != "A👩中文" {
		t.Fatal(c)
	}
}
func TestIMEChangedRangeCommitsPreeditBeforeMovingItsBase(t *testing.T) {
	c := &selectionRecordingClient{recordingClient: recordingClient{text: "abcdef", sel: platform.TextSelection{Range: platform.TextRange{Start: 1, End: 1}}}}
	m := &Ime{}
	m.SetTextInput(platform.TextInputState{Active: true, Client: c})
	m.SetComposingText("ni", 1)
	if !m.SetSelection(2, 4) || c.text != "anibcdef" || c.mark != nil || c.Selection().Range != (platform.TextRange{Start: 2, End: 4}) {
		t.Fatal(c)
	}
}
