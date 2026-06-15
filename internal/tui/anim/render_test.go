package anim_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/andrewhorton/glory/internal/tui/anim"
)

// --- Bar tests ---

func TestBar_FullFill(t *testing.T) {
	b := anim.Bar(100, 100, 8)
	// All filled
	if strings.Contains(b, "░") {
		t.Errorf("Bar(100,100,8) contains empty rune: %q", b)
	}
	if utf8.RuneCountInString(b) != 8 {
		t.Errorf("Bar(100,100,8) width = %d, want 8", utf8.RuneCountInString(b))
	}
}

func TestBar_ZeroFill(t *testing.T) {
	b := anim.Bar(0, 100, 8)
	if strings.Contains(b, "█") {
		t.Errorf("Bar(0,100,8) contains filled rune: %q", b)
	}
	if utf8.RuneCountInString(b) != 8 {
		t.Errorf("Bar(0,100,8) width = %d, want 8", utf8.RuneCountInString(b))
	}
}

func TestBar_HalfFill(t *testing.T) {
	b := anim.Bar(50, 100, 8)
	// 4 filled, 4 empty
	filledCount := strings.Count(b, "█")
	emptyCount := strings.Count(b, "░")
	if filledCount != 4 {
		t.Errorf("Bar(50,100,8) filled count = %d, want 4; bar=%q", filledCount, b)
	}
	if emptyCount != 4 {
		t.Errorf("Bar(50,100,8) empty count = %d, want 4; bar=%q", emptyCount, b)
	}
}

func TestBar_ClampsBelowZero(t *testing.T) {
	b := anim.Bar(-10, 100, 8)
	// same as 0
	b0 := anim.Bar(0, 100, 8)
	if b != b0 {
		t.Errorf("Bar(-10,...) = %q, want same as Bar(0,...) = %q", b, b0)
	}
}

func TestBar_ClampsAboveMax(t *testing.T) {
	b := anim.Bar(200, 100, 8)
	bMax := anim.Bar(100, 100, 8)
	if b != bMax {
		t.Errorf("Bar(200,100,...) = %q, want same as Bar(100,...) = %q", b, bMax)
	}
}

func TestBar_WidthOne(t *testing.T) {
	b := anim.Bar(100, 100, 1)
	if utf8.RuneCountInString(b) != 1 {
		t.Errorf("Bar width=1: got len %d", utf8.RuneCountInString(b))
	}
}

func TestBar_WidthZero(t *testing.T) {
	b := anim.Bar(50, 100, 0)
	if b != "" {
		t.Errorf("Bar width=0: got %q, want empty", b)
	}
}

func TestBar_ExactWidth(t *testing.T) {
	for _, w := range []int{1, 4, 10, 20} {
		b := anim.Bar(50, 100, w)
		got := utf8.RuneCountInString(b)
		if got != w {
			t.Errorf("Bar width=%d: got %d runes in %q", w, got, b)
		}
	}
}

// --- PlotShell (rune grid) tests ---

func TestPlotShell_LandsInCorrectCell(t *testing.T) {
	// 10 cols x 5 rows grid
	rows, cols := 5, 10
	grid := anim.NewRuneGrid(rows, cols, ' ')

	anim.PlotShell(grid, 3, 2, '●')

	if grid[2][3] != '●' {
		t.Errorf("PlotShell: glyph at (col=3,row=2) = %c, want ●", grid[2][3])
	}
}

func TestPlotShell_NoPanicOutOfBounds(t *testing.T) {
	grid := anim.NewRuneGrid(5, 10, ' ')

	// Out-of-bounds calls — must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PlotShell panicked on out-of-bounds: %v", r)
		}
	}()
	anim.PlotShell(grid, -1, -1, '●')
	anim.PlotShell(grid, 100, 100, '●')
	anim.PlotShell(grid, 9, 4, '●') // last valid cell
}

func TestPlotShell_ClampsToGridEdge(t *testing.T) {
	// x=15 clamped to col 9 (last col in 10-wide grid)
	grid := anim.NewRuneGrid(5, 10, ' ')
	anim.PlotShell(grid, 15, 2, '▶')

	if grid[2][9] != '▶' {
		t.Errorf("PlotShell: out-of-bounds x should clamp to last col; grid[2][9]=%c", grid[2][9])
	}
}

func TestPlotShell_OnlyModifiesOneCell(t *testing.T) {
	grid := anim.NewRuneGrid(5, 10, ' ')
	anim.PlotShell(grid, 3, 2, '●')

	for r := 0; r < 5; r++ {
		for c := 0; c < 10; c++ {
			if r == 2 && c == 3 {
				if grid[r][c] != '●' {
					t.Errorf("expected ● at [%d][%d], got %c", r, c, grid[r][c])
				}
			} else {
				if grid[r][c] != ' ' {
					t.Errorf("expected space at [%d][%d], got %c", r, c, grid[r][c])
				}
			}
		}
	}
}

func TestNewRuneGrid_InitializesCorrectly(t *testing.T) {
	grid := anim.NewRuneGrid(3, 4, '.')
	if len(grid) != 3 {
		t.Errorf("rows = %d, want 3", len(grid))
	}
	for r := 0; r < 3; r++ {
		if len(grid[r]) != 4 {
			t.Errorf("row %d len = %d, want 4", r, len(grid[r]))
		}
		for c := 0; c < 4; c++ {
			if grid[r][c] != '.' {
				t.Errorf("grid[%d][%d] = %c, want .", r, c, grid[r][c])
			}
		}
	}
}

func TestPlotShell_ZeroColumnGrid_NoPanic(t *testing.T) {
	// A grid with rows but zero columns must not panic (covers the guard
	// in PlotShell that returns early when cols == 0).
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PlotShell panicked on zero-column grid: %v", r)
		}
	}()
	// Manually construct a jagged grid: one row with zero columns.
	grid := anim.RuneGrid{[]rune{}}
	anim.PlotShell(grid, 0, 0, '●')
	anim.PlotShell(grid, 5, 0, '●')
}

func TestBar_NegativeMax(t *testing.T) {
	// Negative max must produce an all-empty bar of the given width.
	b := anim.Bar(50, -1, 8)
	if strings.Contains(b, "█") {
		t.Errorf("Bar(50,-1,8) contains filled rune: %q", b)
	}
	if utf8.RuneCountInString(b) != 8 {
		t.Errorf("Bar(50,-1,8) width = %d, want 8", utf8.RuneCountInString(b))
	}

	// Zero max: also all-empty.
	b0 := anim.Bar(50, 0, 8)
	if strings.Contains(b0, "█") {
		t.Errorf("Bar(50,0,8) contains filled rune: %q", b0)
	}
	if utf8.RuneCountInString(b0) != 8 {
		t.Errorf("Bar(50,0,8) width = %d, want 8", utf8.RuneCountInString(b0))
	}
}
