package tui

import (
	"time"

	"github.com/casualghoul/idle-glory/internal/game"
	"github.com/casualghoul/idle-glory/internal/tui/anim"
)

// phase enumerates the presentation states of the artillery offensive.
//
// The battle OUTCOME is computed exactly once (game.ResolveBattle) at the
// instant the shell lands; everything else here is timing and juice, which the
// design assigns to the TUI layer (decision D6).
type phase int

const (
	// phaseIdle: charge meter is filling. Pressing F once charged fires.
	phaseIdle phase = iota
	// phaseFiring: a shell is in flight across no-man's-land.
	phaseFiring
	// phaseResolving: the shell has landed; impact flash, shake, and number
	// pops are settling before returning to Idle.
	phaseResolving
)

func (p phase) String() string {
	switch p {
	case phaseIdle:
		return "Idle"
	case phaseFiring:
		return "Firing"
	case phaseResolving:
		return "Resolving"
	default:
		return "Unknown"
	}
}

// Charge / timing constants. These are presentation-only tunables.
const (
	// chargeTime is how long the meter takes to fill from empty to ready.
	chargeTime = 3 * time.Second
	// flightTime is the shell's time aloft. Within the design's 0.6–1.2s band.
	flightTime = 900 * time.Millisecond
	// resolveTime is the settle window after impact before returning to Idle.
	resolveTime = 1200 * time.Millisecond
	// shakeMagnitude is the initial screen-shake displacement on impact.
	shakeMagnitude = 6.0
	// arcHeight is the peak height of the shell's arc, in grid rows.
	arcHeight = 4.0
)

// battle is the explicit, deterministic FSM driving the offensive. It is
// advanced purely by accumulated dt fed from tickMsgs, so it is unit-testable
// by feeding synthetic ticks — no wall clock, no goroutines.
type battle struct {
	phase phase

	// charge accumulates toward chargeTime while Idle. Ready when full.
	charge time.Duration
	// elapsed is time spent in the current Firing or Resolving phase.
	elapsed time.Duration

	// outcome is the result of the most recent resolved battle, retained for
	// the duration of the Resolving phase so the view can show the pops.
	outcome   game.Outcome
	resolved  bool // true once ResolveBattle has been applied this cycle
	flight    anim.ShellFlight
	shellX    float64
	shellY    float64
	shellLive bool // true while a shell glyph should be drawn
}

// ready reports whether the charge meter is full and F will fire.
func (b *battle) ready() bool {
	return b.phase == phaseIdle && b.charge >= chargeTime
}

// chargeFraction returns the charge meter fill in [0,1].
func (b *battle) chargeFraction() float64 {
	if b.charge >= chargeTime {
		return 1
	}
	return float64(b.charge) / float64(chargeTime)
}

// configureFlight sets the shell trajectory for the no-man's-land grid of the
// given size. Called when firing so the arc spans the current viewport.
func (b *battle) configureFlight(cols, rows int) {
	startX := 1.0
	endX := float64(cols - 2)
	if endX < startX {
		endX = startX
	}
	baseline := float64(rows - 1)
	if baseline < 0 {
		baseline = 0
	}
	b.flight = anim.ShellFlight{
		StartX:    startX,
		StartY:    baseline,
		EndX:      endX,
		EndY:      baseline,
		ArcHeight: arcHeight,
		Duration:  flightTime,
	}
}

// fire transitions Idle→Firing if charged. Returns true if it fired.
// cols/rows describe the no-man's-land grid so the flight can be scaled.
func (b *battle) fire(cols, rows int) bool {
	if !b.ready() {
		return false
	}
	b.phase = phaseFiring
	b.elapsed = 0
	b.charge = 0
	b.resolved = false
	b.shellLive = true
	b.configureFlight(cols, rows)
	return true
}

// advance steps the FSM by dt. When the shell lands it calls resolve exactly
// once, mutating the provided state pointer and triggering the juice springs.
// state, shake, and the value springs are owned by the model and passed in so
// the FSM stays free of Bubble Tea types.
func (b *battle) advance(dt time.Duration, state *game.State, shake *anim.ScreenShake) {
	switch b.phase {
	case phaseIdle:
		if b.charge < chargeTime {
			b.charge += dt
			if b.charge > chargeTime {
				b.charge = chargeTime
			}
		}

	case phaseFiring:
		b.elapsed += dt
		x, y, done := b.flight.At(b.elapsed)
		b.shellX, b.shellY = x, y
		if done {
			// Impact: resolve exactly once.
			b.resolve(state, shake)
			b.phase = phaseResolving
			b.elapsed = 0
			b.shellLive = false
		}

	case phaseResolving:
		b.elapsed += dt
		if b.elapsed >= resolveTime {
			b.phase = phaseIdle
			b.elapsed = 0
		}
	}
}

// resolve computes the battle outcome once and applies it to state. It is the
// single call site of game.ResolveBattle in the whole TUI (decision D6).
func (b *battle) resolve(state *game.State, shake *anim.ScreenShake) {
	if b.resolved {
		return
	}
	b.outcome = game.ResolveBattle(*state)
	state.LinePosition += b.outcome.GroundGained
	state.ArmyPower -= b.outcome.PlayerAttrition
	if state.ArmyPower < 0 {
		state.ArmyPower = 0
	}
	state.EnemyPower -= b.outcome.EnemyAttrition
	if state.EnemyPower < 0 {
		state.EnemyPower = 0
	}
	b.resolved = true
	if shake != nil {
		shake.Trigger(shakeMagnitude)
	}
}

// resolveFraction returns how far through the Resolving settle window we are,
// in [0,1]. Used for the impact flash fade.
func (b *battle) resolveFraction() float64 {
	if b.phase != phaseResolving {
		return 0
	}
	f := float64(b.elapsed) / float64(resolveTime)
	if f > 1 {
		f = 1
	}
	return f
}
