// Package tui implements the live, real-time animated Bubble Tea interface for
// Glory. It wires the pure simulation (internal/game), persistence
// (internal/save), and animation primitives (internal/tui/anim) into an
// interactive game following the Elm architecture.
//
// Architecture:
//   - model.go  — the Model struct, construction, Init, and persistence helpers
//   - update.go — the Update reducer: tick clock, key handling, save cadence
//   - view.go   — pure rendering of the three stacked panes
//   - battle.go — the deterministic charge→fire→resolve FSM
//   - keys.go   — keybinding predicates
//
// Timing follows decision D2: a single dt-based clock on a fixed ~30fps tick.
// All accumulation flows through game.Tick(state, dt) so totals are independent
// of tick frequency. The springs are stepped once per tick at the same fps.
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/andrewhorton/glory/internal/game"
	"github.com/andrewhorton/glory/internal/save"
	"github.com/andrewhorton/glory/internal/tui/anim"
)

const (
	// fps is the spring/tick rate. The tick interval is 1000/fps ms. The spring
	// constructors MUST receive this same fps (decision D2 / anim contract).
	fps = 30
	// tickInterval is the wall-clock spacing between ticks (~33ms at 30fps).
	tickInterval = time.Second / fps
	// autosaveInterval is how often a periodic save Cmd is issued.
	autosaveInterval = 30 * time.Second
)

// tickMsg carries the wall-clock time of a tick. dt is derived as t-lastTick.
type tickMsg time.Time

// saveErrMsg reports the result of an asynchronous save Cmd. A nil Err means
// success; it is surfaced transiently so a failing disk does not pass silently.
type saveErrMsg struct{ Err error }

// AwaySummary is the optional "while you were away" banner data passed in by
// main from save.ApplyAwayProgress. A zero Duration means no banner.
type AwaySummary struct {
	Duration        time.Duration
	MunitionsGained float64
}

// Model is the root Bubble Tea model. It owns the game state and all mutable
// animation state; it is the single source of truth for the live game.
type Model struct {
	// Persistence (injected so tests can supply a fake clock / temp dir).
	saveDir string
	clk     save.Clock

	// Simulation state — the canonical game state main persists on quit.
	state game.State

	// Battle FSM (charge / fire / resolve).
	battle battle

	// Animation primitives, stepped once per tick at `fps`.
	munitionsSpring *anim.ValueSpring
	lineSpring      *anim.ValueSpring
	shake           *anim.ScreenShake

	// Terminal dimensions from the latest WindowSizeMsg.
	width  int
	height int

	// Clock bookkeeping for the dt-based tick (decision D2).
	lastTick   time.Time
	haveTick   bool
	sinceSave  time.Duration // accumulates toward autosaveInterval
	saveErr    error         // most recent save error (nil on success); banner shows while non-nil, cleared on next successful save
	awayBanner AwaySummary   // shown until the first key/fire
	showAway   bool

	quitting bool
}

// New constructs a Model from the resolved initial state, the save directory,
// a clock, and an optional away-progress summary for the first-render banner.
func New(state game.State, saveDir string, clk save.Clock, away AwaySummary) Model {
	if state.OwnedCounts == nil {
		state.OwnedCounts = make(map[string]int)
	}
	m := Model{
		saveDir:         saveDir,
		clk:             clk,
		state:           state,
		munitionsSpring: anim.NewValueSpring(fps),
		lineSpring:      anim.NewValueSpring(fps),
		shake:           anim.NewScreenShake(fps),
		awayBanner:      away,
		showAway:        away.Duration > 0,
		// Sensible defaults so View never renders into a zero-size box before
		// the first WindowSizeMsg arrives.
		width:  80,
		height: 24,
	}
	// Seed the springs at the current values so they don't animate from zero.
	m.munitionsSpring.SetValue(state.Munitions)
	m.lineSpring.SetValue(state.LinePosition)
	// Start the offensive charged so the first F feels immediately rewarding.
	m.battle.charge = chargeTime
	return m
}

// Init starts the tick loop.
func (m Model) Init() tea.Cmd {
	return tickCmd()
}

// FinalState returns the latest game state for main to persist after Run.
func (m Model) FinalState() game.State { return m.state }

// tickCmd schedules the next fixed-rate tick.
func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// saveCmd returns a Cmd that persists the given state and reports the result.
// state is captured by value so the save reflects the moment it was requested.
func (m Model) saveCmd(state game.State) tea.Cmd {
	dir, clk := m.saveDir, m.clk
	return func() tea.Msg {
		return saveErrMsg{Err: save.Save(dir, clk, state)}
	}
}
