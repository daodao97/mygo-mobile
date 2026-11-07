//go:build android && cgo

package backend

import (
	"bytes"
	"encoding/json"
	"github.com/egoist/mygo/internal/platform"
)

// NativeSelectionUpdate only posts to Java's UI queue. The UI never waits for
// these asynchronous actions, so InputConnection cannot deadlock the pump.
var NativeSelectionUpdate func([]byte)
var selectionSnapshot readOnlyTextClient
var selectionLast []byte

func (*Surface) NativeReadOnlyTextSelection() bool { return true }
func (b *Backend) setSelectionInput(t platform.TextInputState) {
	if t.Active && t.ReadOnly && t.Client == nil {
		selectionSnapshot.state = t
		selectionSnapshot.geometry = func(q platform.TextGeometryQuery) platform.TextGeometry {
			if w := b.win; w != nil {
				if h, ok := w.h.(platform.TextGeometryHandler); ok {
					return h.TextGeometry(q)
				}
			}
			return platform.TextGeometry{}
		}
		selectionSnapshot.selectRange = func(a, z int) { b.deliver(platform.SurfaceEvent{Kind: platform.TextSelectionChanged, From: a, To: z}) }
		t.Client = &selectionSnapshot
	} else {
		selectionSnapshot.state = platform.TextInputState{}
	}
	theIme.SetTextInput(t)
	b.publishSelection()
}
func (b *Backend) publishSelection() {
	v := selectionPresentation(theIme.state, theIme.Client())
	data, _ := json.Marshal(v)
	if !bytes.Equal(data, selectionLast) && NativeSelectionUpdate != nil {
		selectionLast = append(selectionLast[:0], data...)
		NativeSelectionUpdate(data)
	}
}

// Action IDs deliberately match Android's menu mapping, without passing text
// or a client pointer through JNI. Reject late callbacks after focus changes.
func (b *Backend) SelectionAction(id uint64, action int) {
	t, c := theIme.state, theIme.Client()
	if !t.Active || t.ID != id || c == nil || t.Password {
		return
	}
	commands := map[int]string{1: "copy", 2: "cut", 3: "paste", 4: "selectAll", 5: "undo", 6: "redo"}
	if action == 0 {
		if t.ReadOnly {
			r := c.Selection().Range
			theIme.SetSelection(r.End, r.End)
			b.deliver(platform.SurfaceEvent{Kind: platform.SurfaceKeyboardDismiss})
		}
	} else if command := commands[action]; command != "" && (!t.ReadOnly || action == 1 || action == 4) {
		b.deliver(platform.SurfaceEvent{Kind: platform.SurfaceCommand, Text: command})
		if t.ReadOnly && action == 1 {
			r := c.Selection().Range
			theIme.SetSelection(r.End, r.End)
			b.deliver(platform.SurfaceEvent{Kind: platform.SurfaceKeyboardDismiss})
		}
	}
	b.postFrame()
}
func (b *Backend) SelectionDrag(id uint64, edge int, x, y float64) {
	if t := theIme.state; !t.Active || t.ID != id || t.Password {
		return
	}
	c := theIme.Client()
	if c == nil {
		return
	}
	at, ok := c.IndexForPoint(x, y)
	if !ok {
		return
	}
	r := platform.NormalizeTextRange(c.Selection().Range)
	if edge == 0 {
		r.Start = min(at, r.End)
	} else {
		r.End = max(at, r.Start)
	}
	theIme.SetSelection(r.Start, r.End)
	b.postFrame()
}

// Window focus can return after an earlier UI update was discarded. Force a
// fresh snapshot, rather than replaying a queued snapshot of the old field.
func (b *Backend) RefreshSelection() { selectionLast = nil; b.publishSelection() }
