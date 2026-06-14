// Package game implements the pure simulation engine for Glory, a WW1-themed
// idle game. All functions are pure (same inputs → same outputs); they never
// call time.Now(), perform I/O, or reference global mutable state.
package game

import "time"

// State holds the complete, JSON-serialisable game state.
// All fields are exported so that internal/save can marshal/unmarshal them
// without importing this package transitively.
type State struct {
	// Economy
	Munitions     float64 // current stockpile of munitions
	MunitionsRate float64 // passive munitions gained per second

	// Army
	ArmyPower float64 // player's total combat power

	// Enemy
	EnemyPower   float64 // enemy's current combat power
	LinePosition float64 // front-line position: positive = player advance, negative = enemy advance

	// Upgrade ownership – keyed by Upgrade.ID.
	OwnedCounts map[string]int
}

// Tick advances the simulation by dt. It is a pure function: it does not call
// time.Now() or read any global state. Passing dt=0 returns an identical
// state. Large dt values produce correct catch-up accumulation because
// accumulation = rate * dt (linear, independent of tick frequency).
func Tick(s State, dt time.Duration) State {
	if dt == 0 {
		return s
	}
	seconds := dt.Seconds()
	s.Munitions += s.MunitionsRate * seconds
	return s
}
