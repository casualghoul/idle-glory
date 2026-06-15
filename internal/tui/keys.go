package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/andrewhorton/glory/internal/game"
)

// upgradeKeys maps a 1-based digit key to its index in game.Upgrades.
// "1" buys the first upgrade, "2" the second, and so on. Generated lazily so
// the binding always tracks the (read-only) Upgrades table length.
func upgradeKeyFor(s string) (idx int, ok bool) {
	switch s {
	case "1":
		idx = 0
	case "2":
		idx = 1
	case "3":
		idx = 2
	case "4":
		idx = 3
	case "5":
		idx = 4
	case "6":
		idx = 5
	case "7":
		idx = 6
	case "8":
		idx = 7
	case "9":
		idx = 8
	default:
		return 0, false
	}
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
