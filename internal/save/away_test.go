package save

import (
	"testing"
	"time"

	"github.com/andrewhorton/glory/internal/game"
)

// baseState returns a predictable State for away-progress tests.
func baseState() game.State {
	return game.State{
		Munitions:     50.0,
		MunitionsRate: 3.0, // +3 munitions/sec
		OwnedCounts:   map[string]int{},
	}
}

// ---- Negative dt (clock skew) --------------------------------------------

func TestAwayProgress_NegativeDt_Clamped(t *testing.T) {
	state := baseState()
	savedAt := epoch
	// now is BEFORE savedAt → negative raw dt.
	clk := newClock(epoch.Add(-1 * time.Hour))

	res := ApplyAwayProgress(state, savedAt, clk)
	if res.Duration != 0 {
		t.Errorf("expected Duration=0 for negative dt, got %v", res.Duration)
	}
	if res.MunitionsGained != 0 {
		t.Errorf("expected no munitions gained, got %v", res.MunitionsGained)
	}
	if res.State.Munitions != state.Munitions {
		t.Errorf("State.Munitions changed on zero tick: got %v, want %v", res.State.Munitions, state.Munitions)
	}
}

// ---- Huge dt (beyond cap) ------------------------------------------------

func TestAwayProgress_HugeDt_ClampedToCap(t *testing.T) {
	state := baseState()
	savedAt := epoch
	// now is 3 weeks after savedAt — well beyond MaxAwayDuration.
	clk := newClock(epoch.Add(3 * 7 * 24 * time.Hour))

	res := ApplyAwayProgress(state, savedAt, clk)

	if res.Duration != MaxAwayDuration {
		t.Errorf("expected Duration=MaxAwayDuration (%v), got %v", MaxAwayDuration, res.Duration)
	}

	// Earnings must equal exactly game.Tick(state, MaxAwayDuration).
	expected := game.Tick(state, MaxAwayDuration)
	if res.State.Munitions != expected.Munitions {
		t.Errorf("Munitions: got %v, want %v", res.State.Munitions, expected.Munitions)
	}
	expectedGain := expected.Munitions - state.Munitions
	if res.MunitionsGained != expectedGain {
		t.Errorf("MunitionsGained: got %v, want %v", res.MunitionsGained, expectedGain)
	}
}

// ---- Normal dt -----------------------------------------------------------

func TestAwayProgress_NormalDt(t *testing.T) {
	state := baseState()
	savedAt := epoch
	dt := 2 * time.Hour
	clk := newClock(epoch.Add(dt))

	res := ApplyAwayProgress(state, savedAt, clk)

	if res.Duration != dt {
		t.Errorf("Duration: got %v, want %v", res.Duration, dt)
	}

	expected := game.Tick(state, dt)
	if res.State.Munitions != expected.Munitions {
		t.Errorf("Munitions: got %v, want %v", res.State.Munitions, expected.Munitions)
	}

	expectedGain := expected.Munitions - state.Munitions
	if res.MunitionsGained != expectedGain {
		t.Errorf("MunitionsGained: got %v, want %v", res.MunitionsGained, expectedGain)
	}
	if res.MunitionsGained <= 0 {
		t.Errorf("expected positive munitions gained, got %v", res.MunitionsGained)
	}
}

// ---- Zero MunitionsRate → no earnings ------------------------------------

func TestAwayProgress_ZeroRate_NoEarnings(t *testing.T) {
	state := baseState()
	state.MunitionsRate = 0
	savedAt := epoch
	clk := newClock(epoch.Add(time.Hour))

	res := ApplyAwayProgress(state, savedAt, clk)
	if res.MunitionsGained != 0 {
		t.Errorf("expected no munitions with rate=0, got %v", res.MunitionsGained)
	}
}

// ---- Exactly at cap edge -------------------------------------------------

func TestAwayProgress_ExactlyAtCap(t *testing.T) {
	state := baseState()
	savedAt := epoch
	clk := newClock(epoch.Add(MaxAwayDuration))

	res := ApplyAwayProgress(state, savedAt, clk)
	if res.Duration != MaxAwayDuration {
		t.Errorf("expected MaxAwayDuration, got %v", res.Duration)
	}
}

// ---- One second: sanity check against manual arithmetic ------------------

func TestAwayProgress_OneSecond(t *testing.T) {
	state := baseState() // MunitionsRate = 3
	savedAt := epoch
	clk := newClock(epoch.Add(time.Second))

	res := ApplyAwayProgress(state, savedAt, clk)
	// 3 munitions/sec * 1s = +3
	if res.MunitionsGained != 3.0 {
		t.Errorf("expected 3.0 munitions gained in 1s with rate=3, got %v", res.MunitionsGained)
	}
}
