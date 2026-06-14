// Package save implements durable JSON persistence for Glory game state.
// It provides atomic save/load operations, a Clock interface for testability,
// and away-progress computation (catching up idle earnings on resume).
package save

import "time"

// Clock abstracts wall-clock access so that save operations can be tested
// with a deterministic fake clock without patching global state.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production Clock implementation that delegates to time.Now.
type SystemClock struct{}

// Now returns the current wall-clock time.
func (SystemClock) Now() time.Time { return time.Now() }
