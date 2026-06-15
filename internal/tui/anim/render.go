package anim

import "strings"

// RuneGrid is a 2-D slice of runes indexed as grid[row][col].
// It represents the raw cell content of a terminal region before styling.
type RuneGrid [][]rune

// NewRuneGrid allocates a RuneGrid of the given dimensions, filled with fill.
func NewRuneGrid(rows, cols int, fill rune) RuneGrid {
	g := make(RuneGrid, rows)
	for r := range g {
		g[r] = make([]rune, cols)
		for c := range g[r] {
			g[r][c] = fill
		}
	}
	return g
}

// PlotShell writes glyph into grid at the cell (col, row), clamped to the
// grid's bounds. The function never panics, even for negative or out-of-range
// coordinates.
//
// Parameters use (col, row) order matching (x, y) from ShellFlight.At.
func PlotShell(g RuneGrid, col, row int, glyph rune) {
	rows := len(g)
	if rows == 0 {
		return
	}
	cols := len(g[0])
	if cols == 0 {
		return
	}

	// Clamp both axes.
	if col < 0 {
		col = 0
	} else if col >= cols {
		col = cols - 1
	}
	if row < 0 {
		row = 0
	} else if row >= rows {
		row = rows - 1
	}

	g[row][col] = glyph
}

// Bar renders a horizontal ASCII progress bar of the given width.
// value is clamped to [0, max]. The bar uses '█' for filled cells and
// '░' for empty cells, returning a plain string with no ANSI styling.
//
// Example: Bar(50, 100, 8) → "████░░░░"
func Bar(value, max float64, width int) string {
	if width <= 0 {
		return ""
	}
	if value < 0 {
		value = 0
	}
	if value > max {
		value = max
	}

	filled := 0
	if max > 0 {
		filled = int(value / max * float64(width))
	}
	// Guard against rounding pushing filled beyond width.
	if filled > width {
		filled = width
	}

	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
