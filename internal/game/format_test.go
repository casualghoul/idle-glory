package game_test

import (
	"testing"

	"github.com/andrewhorton/glory/internal/game"
)

func TestFormatNum(t *testing.T) {
	cases := []struct {
		n        float64
		contains string // substring the result must contain
		desc     string
	}{
		{0, "0", "zero"},
		{1, "1", "one"},
		{999, "999", "just below 1K"},
		{1000, "K", "exactly 1K"},
		{1500, "K", "1.5K"},
		{999999, "K", "just below 1M — still K range"},
		{1_000_000, "M", "exactly 1M"},
		{1_500_000, "M", "1.5M"},
		{1_000_000_000, "B", "exactly 1B"},
		{1.5e9, "B", "1.5B"},
		{1e12, "e", "1 trillion -> scientific"},
		{1.23456, "1.23", "small decimal"},
		{-5, "-", "negative small"},
		{-1500, "K", "negative K"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := game.FormatNum(tc.n)
			found := false
			for i := 0; i+len(tc.contains) <= len(got); i++ {
				if got[i:i+len(tc.contains)] == tc.contains {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("FormatNum(%v) = %q, expected it to contain %q", tc.n, got, tc.contains)
			}
		})
	}
}

func TestFormatNum_Boundaries(t *testing.T) {
	// 999 should NOT contain K
	got := game.FormatNum(999)
	for i := 0; i+1 <= len(got); i++ {
		if got[i] == 'K' {
			t.Errorf("FormatNum(999) = %q, should not contain K", got)
		}
	}

	// 1000 should contain K
	got = game.FormatNum(1000)
	found := false
	for _, c := range got {
		if c == 'K' {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("FormatNum(1000) = %q, expected K", got)
	}
}
