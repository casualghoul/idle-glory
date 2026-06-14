package game_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/andrewhorton/glory/internal/game"
)

func TestTick_ZeroDt_NoChange(t *testing.T) {
	initial := game.State{
		Munitions:     100.0,
		MunitionsRate: 5.0,
		ArmyPower:     10.0,
		EnemyPower:    8.0,
		LinePosition:  0.0,
		OwnedCounts:   map[string]int{"artillery": 2},
	}
	result := game.Tick(initial, 0)
	if !reflect.DeepEqual(result, initial) {
		t.Errorf("Tick with dt=0 should return identical state, got %+v", result)
	}
}

func TestTick_AccumulationIsRateTimesDt(t *testing.T) {
	initial := game.State{
		Munitions:     50.0,
		MunitionsRate: 10.0,
	}
	dt := 3 * time.Second
	result := game.Tick(initial, dt)
	expected := 50.0 + 10.0*3.0
	if result.Munitions != expected {
		t.Errorf("expected munitions=%.2f, got %.2f", expected, result.Munitions)
	}
}

func TestTick_CatchUp_TwoTicksEqualsOneLargeTick(t *testing.T) {
	// One Tick(2*d) must equal two Tick(d) calls for accumulation.
	initial := game.State{
		Munitions:     0.0,
		MunitionsRate: 7.5,
	}
	d := 2 * time.Second

	// Two ticks of d
	mid := game.Tick(initial, d)
	twoTicks := game.Tick(mid, d)

	// One tick of 2*d
	oneLarge := game.Tick(initial, 2*d)

	if twoTicks.Munitions != oneLarge.Munitions {
		t.Errorf("two Tick(%v) = %.6f, but one Tick(%v) = %.6f; should be equal",
			d, twoTicks.Munitions, 2*d, oneLarge.Munitions)
	}
}

func TestTick_DoesNotCallTimeNow(t *testing.T) {
	// Tick is pure — same inputs, same outputs. Verified by calling twice.
	s := game.State{Munitions: 1.0, MunitionsRate: 2.0}
	dt := time.Second
	r1 := game.Tick(s, dt)
	r2 := game.Tick(s, dt)
	if !reflect.DeepEqual(r1, r2) {
		t.Errorf("Tick must be pure: same inputs must produce same outputs, got %+v vs %+v", r1, r2)
	}
}

func TestTick_LargeDt(t *testing.T) {
	// Large catch-up tick, e.g. 1 hour
	initial := game.State{
		Munitions:     0.0,
		MunitionsRate: 1.0, // 1 per second
	}
	dt := time.Hour
	result := game.Tick(initial, dt)
	expected := 3600.0
	if result.Munitions != expected {
		t.Errorf("expected munitions=%.2f after 1-hour tick, got %.2f", expected, result.Munitions)
	}
}
