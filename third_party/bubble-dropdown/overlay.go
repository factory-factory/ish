package dropdownv2

// Local patch for ish: these functions replace the dropdown's use of
// github.com/madicen/bubble-overlay. That package's root imports Bubble Tea
// v1, whose package init queries the terminal's background color before the
// program starts, stalling start-up and swallowing keys in terminals that
// don't answer. The geometry functions are ported from bubble-overlay
// (MIT License, Copyright (c) 2026 Michael Madicen); the panel is composited
// with Lip Gloss v2's layers instead of bubble-overlay's v1-based renderer.

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// overlayView draws modalView over mainView with its top-left corner at
// (left, top), in cells from the top-left of mainView. viewWidth and
// viewHeight are accepted to match bubble-overlay's OverlayView; the
// position is clamped to them by clampedOrigin before this is called.
func overlayView(mainView, modalView string, viewWidth, viewHeight, top, left int) string {
	_, _ = viewWidth, viewHeight

	return lipgloss.NewCompositor(
		lipgloss.NewLayer(mainView),
		lipgloss.NewLayer(modalView).X(left).Y(top).Z(1),
	).Render()
}

// modalCellSize returns the width of the widest line of modal, in cells, and
// its number of lines.
func modalCellSize(modal string) (w, h int) {
	lines := strings.Split(modal, "\n")
	h = len(lines)

	for _, line := range lines {
		w = max(w, lipgloss.Width(line))
	}

	return w, h
}

// cellInModal reports whether the cell (x, y) is inside the modal rectangle
// [left, left+mw) × [top, top+mh). A modal with no size contains every cell.
func cellInModal(x, y, top, left, mw, mh int) bool {
	if mw <= 0 || mh <= 0 {
		return true
	}

	return x >= left && x < left+mw && y >= top && y < top+mh
}

// clampedOrigin returns the position for a modal of size modalW × modalH
// placed at (left, top), moved so it fits within a view of size
// viewW × viewH where possible.
func clampedOrigin(top, left, modalW, modalH, viewW, viewH int) (int, int) {
	top, left = max(top, 0), max(left, 0)

	if left+modalW > viewW {
		left = max(0, viewW-modalW)
	}

	if top+modalH > viewH {
		top = max(0, viewH-modalH)
	}

	return top, left
}
