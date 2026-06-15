package save

import (
	"time"

	"github.com/casualghoul/idle-glory/internal/game"
)

// MaxAwayDuration is the upper bound on catch-up time applied when computing
// away-progress. Saves older than this are treated as exactly this duration.
// 24 hours: generous enough to reward overnight play but prevents an instant
// win after weeks of inactivity on a laptop that was sleeping the whole time.
const MaxAwayDuration = 24 * time.Hour

// AwayResult holds the result of ApplyAwayProgress.
type AwayResult struct {
	// State is the new game state after applying idle earnings.
	State game.State
	// Duration is the clamped time interval that was applied (0 … MaxAwayDuration).
	Duration time.Duration
	// MunitionsGained is the munitions earned during the away period.
	MunitionsGained float64
}

// ApplyAwayProgress computes idle earnings between savedAt and now, clamps the
// interval to [0, MaxAwayDuration], ticks the state forward by that amount, and
// returns an AwayResult so the caller can show a "while you were away…" message.
//
// One accumulation path: this calls game.Tick with the clamped dt, exactly as
// the live tick loop does. Battles are NOT resolved here (v1 battles are manual).
//
// Negative dt (clock skew, time-zone switch, NTP correction) is clamped to 0 so
// the player never loses progress.
func ApplyAwayProgress(state game.State, savedAt time.Time, clk Clock) AwayResult {
	now := clk.Now()
	raw := now.Sub(savedAt)

	// Clamp to [0, MaxAwayDuration].
	dt := raw
	if dt < 0 {
		dt = 0
	}
	if dt > MaxAwayDuration {
		dt = MaxAwayDuration
	}

	before := state.Munitions
	newState := game.Tick(state, dt)

	return AwayResult{
		State:           newState,
		Duration:        dt,
		MunitionsGained: newState.Munitions - before,
	}
}
