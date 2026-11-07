package ui

import (
	"github.com/egoist/mygo/internal/platform"
	"unicode/utf8"
)

// Native handles use UTF-16. Update selection without a replacement edit, so
// moving a handle neither deletes text nor creates an undo entry.
func (w *widgetTextInput) SetSelection(selection TextInputSelection) {
	w.prepare()
	if w.state() == nil || w.ed.password {
		return
	}
	ed := w.ed
	r := platform.NormalizeTextRange(selection.Range)
	if ed.compose != "" {
		mark := ed.buf.utf16At(ed.caret)
		if r.Start >= mark && r.End <= mark+countUnits(ed.compose) {
			offset := func(i int) int { return countUnits(ed.compose[:UTF16ByteOffset(ed.compose, i-mark)]) }
			ed.composeSelected = TextInputRange{Start: offset(r.Start), End: offset(r.End)}
			if selection.Reversed {
				ed.composeSelected.Start, ed.composeSelected.End = ed.composeSelected.End, ed.composeSelected.Start
			}
			caret := selection.Caret() - mark
			ed.composeCaret = utf8.RuneCountInString(ed.compose[:UTF16ByteOffset(ed.compose, caret)])
			w.publish()
			return
		}
	}
	a, z := w.virtualRune(r.Start), w.virtualRune(r.End)
	ed.commitCompose()
	ed.editorSelection = editorSelection{anchor: a, caret: z}
	if selection.Reversed {
		ed.anchor, ed.caret = z, a
	}
	ed.coalesce = false
	ed.hasDesired = false
	if ed.area != nil {
		ed.area.reveal = true
	}
	w.publish()
}
