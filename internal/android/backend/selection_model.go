package backend

import "github.com/egoist/mygo/internal/platform"

type nativeTextSelection struct {
	ID                       int64
	Active, ReadOnly         bool
	Start, End, Bounds       platform.RectF
	StartVisible, EndVisible bool
	Rects                    []platform.TextSelectionRect
	Commands                 []int
}

// A range stays copyable even when its endpoints have scrolled out of view.
// Handle visibility belongs to the text viewport, not the whole SurfaceView.
func selectionPresentation(t platform.TextInputState, c platform.TextInputClient) nativeTextSelection {
	v := nativeTextSelection{ID: int64(t.ID), ReadOnly: t.ReadOnly, Bounds: t.Bounds}
	if !t.Active || t.Password || c == nil {
		return v
	}
	r := platform.NormalizeTextRange(c.Selection().Range)
	if r.Start == r.End {
		return v
	}
	v.Active = t.Bounds.W > 0 && t.Bounds.H > 0
	a, _, aok := c.BoundsForRange(platform.TextRange{Start: r.Start, End: r.Start})
	z, _, zok := c.BoundsForRange(platform.TextRange{Start: r.End, End: r.End})
	v.Start, v.End = a, z
	visible := func(r platform.RectF) bool {
		b := t.Bounds
		return r.H > 0 && r.Y >= b.Y-1 && r.Y+r.H <= b.Y+b.H+1 && r.X >= b.X-1 && r.X <= b.X+b.W+1
	}
	_, settable := nativeSelectionSetter(c)
	v.StartVisible, v.EndVisible = settable && aok && visible(a), settable && zok && visible(z)
	if g, ok := c.(platform.TextInputSelectionGeometryClient); ok {
		v.Rects = g.SelectionRectsForRange(r)
	}
	v.Commands = []int{1, 4}
	if !t.ReadOnly {
		v.Commands = []int{1, 2, 3, 4, 5, 6}
	}
	return v
}
