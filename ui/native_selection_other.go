//go:build !android

package ui

func (rt *engine) nativeSelectionAvailable(readOnly bool) bool {
	h, ok := rt.host.(interface{ nativeTextSelection() bool })
	return ok && h.nativeTextSelection()
}
