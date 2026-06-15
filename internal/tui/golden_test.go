package tui

// golden_test.go — static-frame golden tests for the TUI (decision D11, task T6).
//
// Three goldens are captured at fixed 80×24 terminal size with deterministic model
// state and settled springs (SetValue teleports the spring to its target so no
// in-progress animation is visible). The "while you were away" banner is OFF for
// all three frames.
//
// Rules for determinism:
//   - Fixed terminal size 80×24.
//   - No away banner (AwaySummary{}).
//   - Springs seeded via SetValue (already done by New for munitions + line).
//   - Phase is static (Idle) — no in-flight shell, no shake.
//   - Clock fixed to Unix epoch 0; no wall-clock text is rendered by View().
//
// To regenerate golden files:
//
//	go test ./internal/tui/ -run Golden -update
//
// To verify determinism run three times (CI does this):
//
//	go test ./internal/tui/ -run Golden -count=3

import (
	"testing"
	"time"

	"github.com/charmbracelet/x/exp/teatest"

	"github.com/andrewhorton/glory/internal/game"
	"github.com/andrewhorton/glory/internal/save"
)

// goldenModel builds a model in a known static state at 80×24, springs settled,
// no away banner. The returned model is ready to call View() on deterministically.
func goldenModel(state game.State, clk save.Clock) Model {
	m := New(state, "", clk, AwaySummary{})
	m.width = 80
	m.height = 24
	// Seed springs exactly to current values so View shows settled numbers,
	// not mid-animation values.
	m.munitionsSpring.SetValue(state.Munitions)
	m.lineSpring.SetValue(state.LinePosition)
	// Charge meter: seed to full so it shows "READY" in the bottom pane for
	// idle/post-buy (which gives a deterministic label). For the post-resolve
	// golden the charge is empty (just fired), also deterministic.
	m.battle.charge = chargeTime
	return m
}

// TestGoldenIdle: fresh state, phase Idle, charge full, springs settled.
func TestGoldenIdle(t *testing.T) {
	state := game.State{
		Munitions:     500.0,
		MunitionsRate: 2.0,
		ArmyPower:     10.0,
		EnemyPower:    10.0,
		LinePosition:  0.0,
		OwnedCounts:   map[string]int{},
	}
	m := goldenModel(state, fakeClock{t: time.Unix(0, 0)})
	view := m.View()
	teatest.RequireEqualOutput(t, []byte(view))
}

// TestGoldenPostBuy: after buying one Supply Lines upgrade (munitions reduced,
// rate increased, owned=1), springs settled to new values, phase Idle.
func TestGoldenPostBuy(t *testing.T) {
	// Start state then apply a buy.
	initial := game.State{
		Munitions:     500.0,
		MunitionsRate: 2.0,
		ArmyPower:     10.0,
		EnemyPower:    10.0,
		LinePosition:  0.0,
		OwnedCounts:   map[string]int{},
	}
	// Buy supply_lines (cost 50, effect +1 rate).
	postBuy, ok := game.Buy(initial, "supply_lines")
	if !ok {
		t.Fatal("buy should succeed with 500 munitions vs cost 50")
	}
	m := goldenModel(postBuy, fakeClock{t: time.Unix(0, 0)})
	view := m.View()
	teatest.RequireEqualOutput(t, []byte(view))
}

// TestGoldenPostResolve: game.State after a player-victory has been applied
// (line advanced, enemy attrition taken), phase Idle, springs settled.
func TestGoldenPostResolve(t *testing.T) {
	// Simulate the state you'd see after ResolveBattle has been applied.
	// Use a known outcome: player (50) >> enemy (10) → clear victory.
	baseState := game.State{
		Munitions:     1000.0,
		MunitionsRate: 2.0,
		ArmyPower:     50.0,
		EnemyPower:    10.0,
		LinePosition:  0.0,
		OwnedCounts:   map[string]int{},
	}
	outcome := game.ResolveBattle(baseState)
	postResolve := baseState
	postResolve.LinePosition += outcome.GroundGained
	postResolve.ArmyPower -= outcome.PlayerAttrition
	if postResolve.ArmyPower < 0 {
		postResolve.ArmyPower = 0
	}
	postResolve.EnemyPower -= outcome.EnemyAttrition
	if postResolve.EnemyPower < 0 {
		postResolve.EnemyPower = 0
	}

	m := goldenModel(postResolve, fakeClock{t: time.Unix(0, 0)})
	// Post-resolve: charge is empty (was zeroed on fire).
	m.battle.charge = 0
	view := m.View()
	teatest.RequireEqualOutput(t, []byte(view))
}
