package game

import (
	"fmt"
	"math"
)

// FormatNum formats n as a human-readable string:
//
//   - |n| < 1 000              → up to two decimal places (e.g. "1.23", "999")
//   - 1 000 ≤ |n| < 1 000 000 → suffix K  (e.g. "1.50K")
//   - 1 000 000 ≤ |n| < 1e9   → suffix M  (e.g. "1.50M")
//   - 1e9 ≤ |n| < 1e12        → suffix B  (e.g. "1.50B")
//   - |n| ≥ 1e12               → scientific notation (e.g. "1.23e+12")
//
// Plain float64 is used throughout; no type aliases.
func FormatNum(n float64) string {
	// Use the rounded absolute value for threshold comparisons so the suffix
	// matches what fmt.Sprintf will actually print (e.g. 999.995 rounds to
	// 1000.00, so it must be formatted with the K suffix).
	abs := math.Abs(n)
	absRounded := math.Round(abs*100) / 100

	var s string
	switch {
	case absRounded < 1_000:
		// Use fixed-point with up to 2 decimal places; strip trailing zeros.
		s = fmt.Sprintf("%.2f", n)
		// Trim trailing zeros and unnecessary decimal point for integers.
		trimmed := s
		for len(trimmed) > 0 && trimmed[len(trimmed)-1] == '0' {
			trimmed = trimmed[:len(trimmed)-1]
		}
		if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '.' {
			trimmed = trimmed[:len(trimmed)-1]
		}
		s = trimmed
	case absRounded < 1_000_000:
		s = fmt.Sprintf("%.2fK", n/1_000)
	case absRounded < 1_000_000_000:
		s = fmt.Sprintf("%.2fM", n/1_000_000)
	case absRounded < 1e12:
		s = fmt.Sprintf("%.2fB", n/1_000_000_000)
	default:
		s = fmt.Sprintf("%.2e", n)
	}
	return s
}
