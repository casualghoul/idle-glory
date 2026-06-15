package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/casualghoul/idle-glory/internal/game"
	"github.com/casualghoul/idle-glory/internal/save"
)

// fakeClock is a deterministic save.Clock for tests.
type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

// testState returns a state where the player out-powers the enemy so a battle
// resolves to a clear WinnerPlayer with non-zero ground + attrition.
func testState() game.State {
	return game.State{
		Munitions:     1000,
		MunitionsRate: 2,
		ArmyPower:     50,
		EnemyPower:    10,
		LinePosition:  0,
		OwnedCounts:   map[string]int{},
	}
}

// newTestModel builds a Model with a fake clock and a temp save dir, sized for
// a normal terminal.
func newTestModel(t *testing.T) Model {
	t.Helper()
	m := New(testState(), t.TempDir(), fakeClock{t: time.Unix(0, 0)}, AwaySummary{})
	m.width, m.height = 80, 30
	return m
}

// tick advances the model by feeding a tickMsg dt after the last tick time.
// It returns the updated model. The first call establishes the baseline.
func (m Model) tickBy(dt time.Duration) Model {
	var base time.Time
	if m.haveTick {
		base = m.lastTick
	} else {
		// First feed a zero-baseline tick, then the real dt.
		base = time.Unix(100, 0)
		next, _ := m.Update(tickMsg(base))
		m = next.(Model)
	}
	next, _ := m.Update(tickMsg(base.Add(dt)))
	return next.(Model)
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// --- Charge / Fire ---

func TestFireWhenChargedTransitionsToFiring(t *testing.T) {
	m := newTestModel(t)
	if m.battle.phase != phaseIdle {
		t.Fatalf("want initial phase Idle, got %v", m.battle.phase)
	}
	if !m.battle.ready() {
		t.Fatalf("model should start charged & ready")
	}

	next, _ := m.Update(key("f"))
	m = next.(Model)
	if m.battle.phase != phaseFiring {
		t.Fatalf("after F want Firing, got %v", m.battle.phase)
	}
	if !m.battle.shellLive {
		t.Fatalf("shell should be live after firing")
	}
}

func TestFireWhenNotChargedIsNoOp(t *testing.T) {
	m := newTestModel(t)
	m.battle.charge = 0 // drain charge
	next, _ := m.Update(key("f"))
	m = next.(Model)
	if m.battle.phase != phaseIdle {
		t.Fatalf("uncharged F should stay Idle, got %v", m.battle.phase)
	}
}

// --- Firing → Resolving applies ResolveBattle exactly once ---

func TestFiringResolvesExactlyOnce(t *testing.T) {
	m := newTestModel(t)
	before := m.state

	next, _ := m.Update(key("f"))
	m = next.(Model)

	// Feed enough ticks to exceed flightTime. Use small dt steps to simulate
	// real frames; the shell should advance then land.
	landed := false
	for i := 0; i < 200 && !landed; i++ {
		m = m.tickBy(30 * time.Millisecond)
		if m.battle.phase == phaseResolving {
			landed = true
		}
	}
	if !landed {
		t.Fatalf("flight never resolved within budget")
	}
	if !m.battle.resolved {
		t.Fatalf("battle should be marked resolved")
	}

	// Outcome should be a player victory given testState; assert it was applied.
	if m.battle.outcome.Winner != game.WinnerPlayer {
		t.Fatalf("want WinnerPlayer, got %v", m.battle.outcome.Winner)
	}
	if m.state.LinePosition <= before.LinePosition {
		t.Fatalf("line should have advanced: before=%v after=%v",
			before.LinePosition, m.state.LinePosition)
	}
	if m.state.EnemyPower >= before.EnemyPower {
		t.Fatalf("enemy should have taken attrition: before=%v after=%v",
			before.EnemyPower, m.state.EnemyPower)
	}

	// Now keep feeding ticks during the remainder of Resolving and assert the
	// outcome is NOT applied again (line should only move by GroundGained once).
	lineAtResolve := m.state.LinePosition
	enemyAtResolve := m.state.EnemyPower
	for i := 0; i < 5; i++ {
		m = m.tickBy(30 * time.Millisecond)
		if m.battle.phase != phaseResolving {
			break
		}
	}
	// While still resolving, the per-tick attrition must not re-apply.
	if m.battle.phase == phaseResolving {
		if m.state.LinePosition != lineAtResolve {
			t.Fatalf("line changed during resolve (double-apply): %v -> %v",
				lineAtResolve, m.state.LinePosition)
		}
		if m.state.EnemyPower != enemyAtResolve {
			t.Fatalf("enemy power changed during resolve (double-apply): %v -> %v",
				enemyAtResolve, m.state.EnemyPower)
		}
	}
}

// --- Resolving settles back to Idle with charge reset ---

func TestResolveSettlesToIdleAndResetsCharge(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(key("f"))
	m = next.(Model)

	// Run through flight + full resolve window.
	for i := 0; i < 400; i++ {
		m = m.tickBy(30 * time.Millisecond)
		if m.battle.phase == phaseIdle {
			break
		}
	}
	if m.battle.phase != phaseIdle {
		t.Fatalf("should return to Idle after resolve, got %v", m.battle.phase)
	}
	// Charge resets to empty after firing (was zeroed on fire, refills in Idle).
	if m.battle.ready() {
		// It may have partially recharged during the resolve window's idle ticks,
		// but should not be fully ready immediately after such a short time.
		t.Fatalf("charge should not be full right after returning to Idle")
	}
}

// --- Frequency independence: route through game.Tick ---

func TestMunitionsAccumulationFrequencyIndependent(t *testing.T) {
	d := 500 * time.Millisecond

	// One tick of 2d.
	m1 := newTestModel(t)
	start := m1.state.Munitions
	m1 = m1.tickBy(2 * d)
	gainedOnce := m1.state.Munitions - start

	// Two ticks of d.
	m2 := newTestModel(t)
	m2 = m2.tickBy(d)
	m2 = m2.tickBy(d)
	gainedTwice := m2.state.Munitions - start

	if diff := gainedOnce - gainedTwice; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("frequency dependence: one 2d tick gained %v, two d ticks gained %v",
			gainedOnce, gainedTwice)
	}
	// Sanity: rate*dt = 2 * 1.0s = 2.0.
	if gainedOnce < 1.99 || gainedOnce > 2.01 {
		t.Fatalf("expected ~2.0 munitions over 1s at rate 2, got %v", gainedOnce)
	}
}

// --- Window resize ---

func TestWindowSizeMsgUpdatesDimsAndViewSurvives(t *testing.T) {
	m := newTestModel(t)
	for _, sz := range []struct{ w, h int }{{80, 24}, {120, 40}, {20, 8}, {1, 1}, {0, 0}} {
		next, _ := m.Update(tea.WindowSizeMsg{Width: sz.w, Height: sz.h})
		m = next.(Model)
		if m.width != sz.w || m.height != sz.h {
			t.Fatalf("dims not stored: want %dx%d got %dx%d", sz.w, sz.h, m.width, m.height)
		}
		// View must not panic at any size.
		_ = m.View()
	}
}

// --- Buy upgrade ---

func TestBuyUpgradeReducesMunitionsIncrementsOwnedAndSaves(t *testing.T) {
	m := newTestModel(t)
	beforeMun := m.state.Munitions
	id := game.Upgrades[0].ID

	next, cmd := m.Update(key("1"))
	m = next.(Model)

	if m.state.Munitions >= beforeMun {
		t.Fatalf("munitions should drop after purchase: %v -> %v", beforeMun, m.state.Munitions)
	}
	if m.state.OwnedCounts[id] != 1 {
		t.Fatalf("owned count should be 1, got %d", m.state.OwnedCounts[id])
	}
	if cmd == nil {
		t.Fatalf("buy should return a save Cmd")
	}
	// Execute the Cmd and assert it is a save (returns saveErrMsg with no error).
	msg := cmd()
	se, ok := msg.(saveErrMsg)
	if !ok {
		t.Fatalf("buy Cmd should produce saveErrMsg, got %T", msg)
	}
	if se.Err != nil {
		t.Fatalf("save Cmd reported error: %v", se.Err)
	}
}

func TestBuyUnaffordableIsNoOp(t *testing.T) {
	m := newTestModel(t)
	m.state.Munitions = 0
	next, cmd := m.Update(key("1"))
	m = next.(Model)
	if m.state.OwnedCounts[game.Upgrades[0].ID] != 0 {
		t.Fatalf("unaffordable buy should not increment owned")
	}
	if cmd != nil {
		t.Fatalf("unaffordable buy should not return a save Cmd")
	}
}

// --- Quit ---

func TestQuitReturnsQuitCmdAndHoldsState(t *testing.T) {
	m := newTestModel(t)
	// Advance some so state is non-trivial.
	m = m.tickBy(time.Second)
	want := m.state

	next, cmd := m.Update(key("q"))
	m = next.(Model)

	if cmd == nil {
		t.Fatalf("quit should return a command")
	}
	// The quit handler returns tea.Quit directly; main performs the final save after Run() returns.
	if !cmdYieldsQuit(cmd) {
		t.Fatalf("quit command should ultimately yield tea.Quit")
	}
	// Model retains the latest state for main to persist.
	if m.FinalState().Munitions != want.Munitions {
		t.Fatalf("FinalState lost munitions: want %v got %v", want.Munitions, m.FinalState().Munitions)
	}
}

func TestCtrlCAlsoQuits(t *testing.T) {
	m := newTestModel(t)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = next.(Model)
	if cmd == nil || !cmdYieldsQuit(cmd) {
		t.Fatalf("ctrl+c should yield tea.Quit")
	}
}

// cmdYieldsQuit executes a command and reports whether it produces tea.QuitMsg.
// The quit handler returns tea.Quit directly, so a single invocation suffices.
func cmdYieldsQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// --- View never panics across phases ---

func TestViewNeverPanicsAcrossPhases(t *testing.T) {
	m := newTestModel(t)

	// Idle
	_ = m.View()

	// Firing
	next, _ := m.Update(key("f"))
	m = next.(Model)
	_ = m.View()

	// Drive into Resolving.
	for i := 0; i < 200; i++ {
		m = m.tickBy(30 * time.Millisecond)
		if m.battle.phase == phaseResolving {
			break
		}
	}
	if m.battle.phase != phaseResolving {
		t.Fatalf("expected to reach Resolving phase")
	}
	_ = m.View()
}

func TestAwayBannerShownThenDismissed(t *testing.T) {
	m := New(testState(), t.TempDir(), fakeClock{}, AwaySummary{Duration: 2 * time.Hour, MunitionsGained: 500})
	m.width, m.height = 80, 30
	if !m.showAway {
		t.Fatalf("away banner should be shown initially")
	}
	out := m.View()
	if out == "" {
		t.Fatalf("view empty")
	}
	// Any key dismisses it.
	next, _ := m.Update(key("1"))
	m = next.(Model)
	if m.showAway {
		t.Fatalf("away banner should be dismissed after a key press")
	}
}

// Ensure FinalState compiles against save.Clock usage (interface sanity).
var _ save.Clock = fakeClock{}
