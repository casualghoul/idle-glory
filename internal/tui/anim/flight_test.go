package anim_test

import (
	"math"
	"testing"
	"time"

	"github.com/andrewhorton/glory/internal/tui/anim"
)

func TestShellFlight_Endpoints(t *testing.T) {
	f := anim.ShellFlight{
		StartX:    0,
		StartY:    10,
		EndX:      80,
		EndY:      10,
		ArcHeight: 5,
		Duration:  time.Second,
	}

	// At t=0: should be at start
	x, y, done := f.At(0)
	if x != f.StartX {
		t.Errorf("At(0) x = %v, want %v", x, f.StartX)
	}
	if y != f.StartY {
		t.Errorf("At(0) y = %v, want %v", y, f.StartY)
	}
	if done {
		t.Error("At(0) done = true, want false")
	}

	// At Duration: should be at end
	x, y, done = f.At(f.Duration)
	if math.Abs(x-f.EndX) > 1e-9 {
		t.Errorf("At(Duration) x = %v, want %v", x, f.EndX)
	}
	if math.Abs(y-f.EndY) > 1e-9 {
		t.Errorf("At(Duration) y = %v, want %v", y, f.EndY)
	}
	if !done {
		t.Error("At(Duration) done = false, want true")
	}

	// Beyond Duration: still done, clamped
	x2, y2, done2 := f.At(2 * f.Duration)
	if x2 != x {
		t.Errorf("At(2*Duration) x = %v, want %v (clamped)", x2, x)
	}
	if y2 != y {
		t.Errorf("At(2*Duration) y = %v, want %v (clamped)", y2, y)
	}
	if !done2 {
		t.Error("At(2*Duration) done = false, want true")
	}
}

func TestShellFlight_NegativeElapsed(t *testing.T) {
	f := anim.ShellFlight{
		StartX: 0, StartY: 5, EndX: 10, EndY: 5,
		ArcHeight: 3, Duration: time.Second,
	}
	// Negative elapsed: clamp to p=0
	x, y, done := f.At(-time.Second)
	if x != f.StartX {
		t.Errorf("At(-Duration) x = %v, want %v", x, f.StartX)
	}
	if y != f.StartY {
		t.Errorf("At(-Duration) y = %v, want %v", y, f.StartY)
	}
	if done {
		t.Error("At(-Duration) done = true, want false")
	}
}

func TestShellFlight_ArcShape(t *testing.T) {
	// y-axis convention: smaller y = higher on screen (terminal rows go down).
	// ArcHeight lifts the shell UP, so y decreases at the apex.
	f := anim.ShellFlight{
		StartX:    0,
		StartY:    10,
		EndX:      80,
		EndY:      10,
		ArcHeight: 5,
		Duration:  time.Second,
	}

	// At midpoint, y should be higher (lower row number) than the chord.
	mid := f.Duration / 2
	_, yMid, _ := f.At(mid)

	// Chord value at p=0.5 is lerp(StartY, EndY, 0.5) = 10
	chordMid := (f.StartY + f.EndY) / 2
	// Arc lifts up: yMid should be chordMid - ArcHeight (approx, at p=0.5 it's exactly ArcHeight)
	expectedApex := chordMid - f.ArcHeight
	if math.Abs(yMid-expectedApex) > 1e-9 {
		t.Errorf("At(mid) y = %v, want %v (apex at midpoint)", yMid, expectedApex)
	}
	// yMid should be strictly less than chordMid (higher on screen)
	if yMid >= chordMid {
		t.Errorf("At(mid) y = %v, not above chord y = %v", yMid, chordMid)
	}
}

func TestShellFlight_XMonotonic(t *testing.T) {
	f := anim.ShellFlight{
		StartX: 5, StartY: 10, EndX: 75, EndY: 8,
		ArcHeight: 4, Duration: time.Second,
	}

	// Sample many x values and assert they're monotonically increasing.
	prevX := math.Inf(-1)
	for i := 0; i <= 100; i++ {
		p := time.Duration(i) * f.Duration / 100
		x, _, _ := f.At(p)
		if x < prevX {
			t.Errorf("x not monotonic at step %d: x=%v, prevX=%v", i, x, prevX)
		}
		prevX = x
	}
}

func TestShellFlight_NoNaNOrInf(t *testing.T) {
	f := anim.ShellFlight{
		StartX: 0, StartY: 0, EndX: 100, EndY: 20,
		ArcHeight: 10, Duration: time.Second,
	}

	for i := 0; i <= 1000; i++ {
		elapsed := time.Duration(i) * f.Duration / 1000
		x, y, _ := f.At(elapsed)
		if math.IsNaN(x) || math.IsInf(x, 0) {
			t.Errorf("NaN/Inf x at step %d", i)
		}
		if math.IsNaN(y) || math.IsInf(y, 0) {
			t.Errorf("NaN/Inf y at step %d", i)
		}
	}
}

func TestShellFlight_AsymmetricEndpoints(t *testing.T) {
	// Different start/end Y to test the lerp+arc formula handles the slope.
	f := anim.ShellFlight{
		StartX: 0, StartY: 20, EndX: 60, EndY: 5,
		ArcHeight: 8, Duration: 2 * time.Second,
	}

	x0, y0, _ := f.At(0)
	if x0 != 0 || y0 != 20 {
		t.Errorf("start mismatch: got (%v,%v)", x0, y0)
	}
	x1, y1, done := f.At(2 * time.Second)
	if math.Abs(x1-60) > 1e-9 || math.Abs(y1-5) > 1e-9 {
		t.Errorf("end mismatch: got (%v,%v) done=%v", x1, y1, done)
	}
	if !done {
		t.Error("done should be true at Duration")
	}

	// At midpoint the arc should lift above the chord.
	_, yMid, _ := f.At(time.Second)
	// chord at p=0.5: lerp(20,5,0.5) = 12.5
	chordMid := 12.5
	if yMid >= chordMid {
		t.Errorf("midpoint y %v should be above chord %v", yMid, chordMid)
	}
}
