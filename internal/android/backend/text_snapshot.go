package backend

import (
	"github.com/egoist/mygo/internal/platform"
	"unicode/utf8"
)

// Read-only snapshots use rune offsets; InputConnection uses UTF-16. Geometry
// and selection edits always return to the same Go layout that paints the text.
type readOnlyTextClient struct {
	state       platform.TextInputState
	geometry    func(platform.TextGeometryQuery) platform.TextGeometry
	selectRange func(int, int)
}

func (c *readOnlyTextClient) runeOffset(i int) int {
	return utf8.RuneCountInString(c.state.Text[:platform.UTF16ByteOffset(c.state.Text, i)])
}
func (c *readOnlyTextClient) nativeOffset(i int) int {
	i = max(0, i)
	n := 0
	for b := range c.state.Text {
		if n == i {
			return platform.UTF16Len(c.state.Text[:b])
		}
		n++
	}
	return platform.UTF16Len(c.state.Text)
}
func (c *readOnlyTextClient) TextForRange(r platform.TextRange) (string, platform.TextRange) {
	r = platform.NormalizeTextRange(r)
	a, z := platform.UTF16ByteOffset(c.state.Text, r.Start), platform.UTF16ByteOffset(c.state.Text, r.End)
	return c.state.Text[a:z], platform.TextRange{Start: platform.UTF16Len(c.state.Text[:a]), End: platform.UTF16Len(c.state.Text[:z])}
}
func (c *readOnlyTextClient) Selection() platform.TextSelection {
	return platform.TextSelection{Range: platform.TextRange{Start: c.nativeOffset(c.state.Start), End: c.nativeOffset(c.state.End)}}
}
func (c *readOnlyTextClient) SetSelection(s platform.TextSelection) {
	a, z := c.runeOffset(s.Range.Start), c.runeOffset(s.Range.End)
	c.selectRange(a, z)
	// A second drag can arrive before the next frame publishes a snapshot.
	c.state.Start, c.state.End = a, z
}
func (*readOnlyTextClient) MarkedRange() (platform.TextRange, bool) {
	return platform.TextRange{}, false
}
func (*readOnlyTextClient) ReplaceText(*platform.TextRange, string)                       {}
func (*readOnlyTextClient) SetMarkedText(*platform.TextRange, string, platform.TextRange) {}
func (*readOnlyTextClient) UnmarkText()                                                   {}
func (c *readOnlyTextClient) query(kind string, r platform.TextRange, x, y float64) platform.TextGeometry {
	return c.geometry(platform.TextGeometryQuery{ID: c.state.ID, Kind: kind, Start: c.runeOffset(r.Start), End: c.runeOffset(r.End), X: x, Y: y})
}
func (c *readOnlyTextClient) BoundsForRange(r platform.TextRange) (platform.RectF, platform.TextRange, bool) {
	g := c.query("caret", r, 0, 0)
	return g.Caret, r, g.Valid
}
func (c *readOnlyTextClient) SelectionRectsForRange(r platform.TextRange) []platform.TextSelectionRect {
	return c.query("selection", r, 0, 0).Rects
}
func (c *readOnlyTextClient) IndexForPoint(x, y float64) (int, bool) {
	g := c.query("hit", platform.TextRange{}, x, y)
	return c.nativeOffset(g.Index), g.Valid
}
