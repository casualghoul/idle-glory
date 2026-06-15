// Package anim provides deterministic, unit-testable animation primitives for
// the glory TUI. It has no Bubble Tea dependency and does not import internal/game;
// all computations work on plain floats and ints.
//
// # Y-axis convention
//
// Smaller y values are HIGHER on screen (terminal rows increase downward).
// ArcHeight is therefore subtracted from the baseline y so that the shell
// visually arcs upward in the viewport.
package anim

import "time"

// ShellFlight describes one shell's ballistic trajectory across no-man's-land.
//
// The flight uses lerp(x) + parabola(y) over a fixed timer — NOT a spring.
// Springs decelerate, settle and overshoot on both axes, which is wrong for
// a ballistic projectile. The parabolic arc IS the visual money shot.
type ShellFlight struct {
	StartX, StartY float64       // launch point
	EndX, EndY     float64       // target point
	ArcHeight      float64       // peak height above the chord (in terminal rows)
	Duration       time.Duration // total flight time
}

// At returns the shell position at the given elapsed time and whether the
// flight has completed (elapsed >= Duration).
//
// x is linearly interpolated from StartX to EndX.
// y follows a parabola: at p=0 it equals StartY, at p=1 it equals EndY,
// with an apex lifted by ArcHeight at p=0.5. Because y increases downward
// in the terminal, ArcHeight is subtracted so the shell arcs upward visually.
//
// elapsed is clamped to [0, Duration]; calling At beyond Duration is safe.
// If Duration <= 0, the flight is considered already completed and (EndX, EndY, true)
// is returned immediately.
func (f ShellFlight) At(elapsed time.Duration) (x, y float64, done bool) {
	if f.Duration <= 0 {
		return f.EndX, f.EndY, true // zero-duration flight has already landed
	}
	p := float64(elapsed) / float64(f.Duration)
	if p < 0 {
		p = 0
	}
	if p >= 1 {
		p = 1
		done = true
	}

	x = lerp(f.StartX, f.EndX, p)

	// base: straight-line chord between start and end y
	base := lerp(f.StartY, f.EndY, p)
	// arc: parabola 4*p*(1-p) peaks at p=0.5 with value 1.0
	// subtract because smaller y = higher on screen
	arc := f.ArcHeight * 4 * p * (1 - p)
	y = base - arc

	return x, y, done
}

// lerp linearly interpolates between a and b by fraction t.
func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}
