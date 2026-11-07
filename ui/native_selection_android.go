//go:build android

package ui

// Read-only snapshots must not switch indexed editable clients to the snapshot
// IME path: those clients own their text and composition.
func (rt *engine) nativeSelectionAvailable(readOnly bool) bool {
	if readOnly {
		if h, ok := rt.host.(interface{ nativeReadOnlyTextSelection() bool }); ok {
			return h.nativeReadOnlyTextSelection()
		}
	}
	h, ok := rt.host.(interface{ nativeTextSelection() bool })
	return ok && h.nativeTextSelection()
}
func (h *windowHost) nativeReadOnlyTextSelection() bool {
	s, ok := h.conn.Surface.(interface{ NativeReadOnlyTextSelection() bool })
	return ok && s.NativeReadOnlyTextSelection()
}

// The adapter's legacy InputSelect fallback is not universally handled by
// custom editors. Android may draw draggable handles only for a real setter.
func (a *textInputAdapter) SupportsNativeTextSelection() bool {
	c, _ := a.client()
	_, ok := c.(TextInputSelectionClient)
	return ok
}
