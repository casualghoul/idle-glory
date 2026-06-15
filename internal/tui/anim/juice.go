package anim

import (
	"math"

	"github.com/charmbracelet/harmonica"
)

// --- ValueSpring ---

// ValueSpring is a spring-backed scalar animator suitable for number-pop
// effects, resource/charge bar springs, and impact recoil. It uses an
// under-damped Harmonica spring so the value overshoots and oscillates
// before settling — that juicy feel is intentional.
//
// Default fps is supplied at construction; the spring advances one frame
// per Step call.
type ValueSpring struct {
	spring harmonica.Spring
	pos    float64
	vel    float64
}

// Spring tuning constants.
// Under-damped (dampingRatio < 1) gives the overshoot that makes number-pops
// and bar springs feel juicy.
const (
	defaultAngularFrequency = 8.0  // oscillation speed
	defaultDampingRatio     = 0.35 // under-damped; produces visible overshoot
)

// NewValueSpring creates a ValueSpring that ticks at fps frames per second.
//
// Tick-rate contract: the caller MUST call Step at the same rate as fps.
// The spring advances one internal fixed dt = 1/fps per Step call; a
// mismatch causes the animation to run at the wrong speed. For a 30 fps
// tick loop, pass 30.
//
// Panics if fps <= 0.
func NewValueSpring(fps int) *ValueSpring {
	if fps <= 0 {
		panic("anim: fps must be > 0")
	}
	dt := harmonica.FPS(fps)
	return &ValueSpring{
		spring: harmonica.NewSpring(dt, defaultAngularFrequency, defaultDampingRatio),
	}
}

// Step advances the spring one frame toward target.
func (v *ValueSpring) Step(target float64) {
	v.pos, v.vel = v.spring.Update(v.pos, v.vel, target)
}

// Value returns the current spring position.
func (v *ValueSpring) Value() float64 { return v.pos }

// SetValue teleports the spring to a new position with zero velocity.
// Useful for initialising the spring without animation.
func (v *ValueSpring) SetValue(pos float64) {
	v.pos = pos
	v.vel = 0
}

// --- ScreenShake ---

// ScreenShake produces a decaying oscillating column offset for whole-view
// shake on shell impact. It uses an under-damped Harmonica spring returning
// toward zero from an initial displacement, so the shake overshoots and
// rings down naturally.
//
// Call Trigger to start a shake and Step on every frame. Offset returns
// the current integer column displacement the renderer should apply.
type ScreenShake struct {
	spring harmonica.Spring
	pos    float64
	vel    float64
}

// Shake spring constants: high frequency, low damping = snappy ringdown.
const (
	shakeAngularFrequency = 20.0 // fast oscillation
	shakeDampingRatio     = 0.25 // under-damped: rings before settling
)

// NewScreenShake creates a ScreenShake that ticks at fps frames per second.
//
// Tick-rate contract: the caller MUST call Step at the same rate as fps.
// The spring advances one internal fixed dt = 1/fps per Step call; a
// mismatch causes the shake to ring down at the wrong speed. For a 30 fps
// tick loop, pass 30.
//
// Panics if fps <= 0.
func NewScreenShake(fps int) *ScreenShake {
	if fps <= 0 {
		panic("anim: fps must be > 0")
	}
	dt := harmonica.FPS(fps)
	return &ScreenShake{
		spring: harmonica.NewSpring(dt, shakeAngularFrequency, shakeDampingRatio),
	}
}

// Trigger starts the shake by displacing the spring by magnitude columns.
// A magnitude of ~5–10 is appropriate for a heavy shell impact.
//
// Sign semantics: a negative magnitude shakes in the opposite direction
// (both positive and negative are valid). Pass math.Abs(magnitude) if you
// always want the same initial direction regardless of the caller's sign.
func (s *ScreenShake) Trigger(magnitude float64) {
	s.pos = magnitude
	s.vel = 0
}

// Step advances the shake one frame, springing back toward zero.
func (s *ScreenShake) Step() {
	s.pos, s.vel = s.spring.Update(s.pos, s.vel, 0)
}

// Offset returns the current column offset the renderer should apply.
// It rounds to the nearest integer so the terminal grid stays on whole columns.
func (s *ScreenShake) Offset() int {
	return int(math.Round(s.pos))
}
