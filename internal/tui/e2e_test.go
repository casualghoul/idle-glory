package tui

// e2e_test.go — end-to-end "buy then quit saves to disk" integration test.
//
// This test uses the teatest running-program harness to exercise the real
// Update→Cmd→save.Save path: it starts the TUI program, sends a buy keypress
// (which triggers an immediate on-purchase save Cmd), then sends quit, waits
// for the program to exit, and loads the save file to assert the purchase
// persisted durably.
//
// This is the highest-value end-to-end path for data-loss prevention — it
// proves the on-purchase save Cmd writes to disk, not just that the state
// lives in memory.

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/andrewhorton/glory/internal/game"
	"github.com/andrewhorton/glory/internal/save"
)

// TestE2EBuyThenQuitSavesToDisk starts a running TUI program, buys an upgrade
// (which triggers an immediate save Cmd), sends quit, waits for exit, then
// asserts the save file on disk reflects the purchase.
func TestE2EBuyThenQuitSavesToDisk(t *testing.T) {
	dir := t.TempDir()
	clk := fakeClock{t: time.Unix(1000, 0)}

	// Start with enough munitions to afford supply_lines (cost=50).
	initialState := game.State{
		Munitions:     500.0,
		MunitionsRate: 2.0,
		ArmyPower:     10.0,
		EnemyPower:    10.0,
		LinePosition:  0.0,
		OwnedCounts:   map[string]int{},
	}

	m := New(initialState, dir, clk, AwaySummary{})

	tm := teatest.NewTestModel(
		t,
		m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Give the program a moment to start and process the initial WindowSizeMsg.
	time.Sleep(50 * time.Millisecond)

	// Buy upgrade [1] = supply_lines (cost 50, affordable from 500).
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})

	// Give the Cmd time to execute (the save Cmd runs asynchronously).
	// The buy triggers saveCmd immediately; we wait for it to complete.
	time.Sleep(100 * time.Millisecond)

	// Send quit.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	// Wait for the program to exit cleanly.
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	// Now load the save file and assert the purchase persisted.
	out, err := save.Load(dir, clk)
	if err != nil {
		t.Fatalf("save.Load after E2E test: %v", err)
	}
	if out.Result != save.LoadOK {
		t.Fatalf("expected LoadOK, got %v (msg=%q)", out.Result, out.Message)
	}

	// The purchase must have reduced munitions from 500 by the cost (50).
	if out.State.Munitions >= 500.0 {
		t.Fatalf("munitions should be <500 after purchase; got %v", out.State.Munitions)
	}
	// The owned count must reflect the purchase.
	if out.State.OwnedCounts["supply_lines"] < 1 {
		t.Fatalf("supply_lines owned count should be ≥1; got %v", out.State.OwnedCounts["supply_lines"])
	}
}
