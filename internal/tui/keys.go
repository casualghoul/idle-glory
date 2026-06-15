package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/casualghoul/idle-glory/internal/game"
)

// upgradeKeyFor maps a single-digit key ("1".."9") to its zero-based index in
// game.Upgrades. The index is bounds-checked against the live Upgrades table so
// adding or removing upgrades never silently drops or activates a key binding.
func upgradeKeyFor(s string) (idx int, ok bool) {
	if len(s) != 1 || s[0] < '1' || s[0] > '9' {
		return 0, false
	}
	idx = int(s[0] - '1') // '1' → 0, '2' → 1, …, '9' → 8
	if idx >= len(game.Upgrades) {
		return 0, false
	}
	return idx, true
}

// isFireKey reports whether the key press should fire an offensive.
func isFireKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "f", "F", " ", "spacebar":
		return true
	}
	return false
}

// isQuitKey reports whether the key press should quit the program.
func isQuitKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "q", "Q", "ctrl+c", "esc":
		return true
	}
	return false
}
