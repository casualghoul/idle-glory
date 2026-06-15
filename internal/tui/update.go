package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/andrewhorton/glory/internal/game"
)

// Update is the Elm reducer. It is deterministic given its messages, which is
// what makes the FSM unit-testable: feed synthetic tickMsgs and tea.KeyMsgs.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		return m.handleTick(time.Time(msg))

	case saveErrMsg:
		m.saveErr = msg.Err // nil on success, clears any prior error banner
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleTick advances the single dt-based clock (decision D2): it derives
// dt = t - lastTick (guarding the first tick and non-positive dt), accumulates
// the economy via game.Tick, advances the battle FSM, steps every spring once,
// and folds in the autosave cadence. It always re-arms the tick.
func (m Model) handleTick(t time.Time) (tea.Model, tea.Cmd) {
	if !m.haveTick {
		// First tick establishes the baseline; no dt elapsed yet.
		m.haveTick = true
		m.lastTick = t
		return m, tickCmd()
	}

	dt := t.Sub(m.lastTick)
	m.lastTick = t
	if dt <= 0 {
		// Clock skew or duplicate tick: advance nothing but keep ticking.
		return m, tickCmd()
	}

	// Economy accumulation — the single accumulation path (rate*dt).
	m.state = game.Tick(m.state, dt)

	// Battle FSM advances on the same dt; it may resolve and mutate state.
	m.battle.advance(dt, &m.state, m.shake)

	// Step the animation springs once per tick at the constructor fps.
	m.munitionsSpring.Step(m.state.Munitions)
	m.lineSpring.Step(m.state.LinePosition)
	m.shake.Step()

	// Autosave cadence.
	var cmds []tea.Cmd
	cmds = append(cmds, tickCmd())
	m.sinceSave += dt
	if m.sinceSave >= autosaveInterval {
		m.sinceSave -= autosaveInterval
		cmds = append(cmds, m.saveCmd(m.state))
	}

	return m, tea.Batch(cmds...)
}

// handleKey routes a key press: quit (save then quit), fire, or buy an upgrade.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key dismisses the away banner.
	m.showAway = false

	switch {
	case isQuitKey(msg):
		m.quitting = true
		// Quit immediately. main performs the authoritative final save after
		// prog.Run() returns (covering quit, ctrl-c, and SIGTERM uniformly), so
		// we do not race a save Cmd against tea.Quit here. m.FinalState() carries
		// the latest state to main.
		return m, tea.Quit

	case isFireKey(msg):
		cols, rows := m.noMansLandSize()
		m.battle.fire(cols, rows)
		return m, nil
	}

	// Upgrade purchase by digit key.
	if idx, ok := upgradeKeyFor(msg.String()); ok {
		return m.buyUpgrade(idx)
	}

	return m, nil
}

// buyUpgrade attempts to purchase the upgrade at the given index. On success it
// updates state, re-seeds the relevant springs' targets implicitly (they track
// state on the next tick), and issues a save Cmd (purchase persistence cadence).
//
// Factored out of handleKey so tests can drive it directly and assert both the
// state change and that a save was requested (the returned Cmd is non-nil).
func (m Model) buyUpgrade(idx int) (tea.Model, tea.Cmd) {
	if idx < 0 || idx >= len(game.Upgrades) {
		return m, nil
	}
	id := game.Upgrades[idx].ID
	next, ok := game.Buy(m.state, id)
	if !ok {
		return m, nil // unaffordable / unknown: no-op, no save
	}
	m.state = next
	return m, m.saveCmd(m.state)
}

// noMansLandSize returns the grid dimensions for the middle pane, used to scale
// the shell flight. It mirrors the sizing logic in view.go.
func (m Model) noMansLandSize() (cols, rows int) {
	return noMansLandDims(m.width, m.height)
}
