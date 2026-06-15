package anim_test

import (
	"math"
	"testing"

	"github.com/casualghoul/idle-glory/internal/tui/anim"
)

// --- ValueSpring tests ---

func TestValueSpring_ConvergesToTarget(t *testing.T) {
	vs := anim.NewValueSpring(60)
	target := 100.0

	// Step many frames; should converge.
	for i := 0; i < 300; i++ {
		vs.Step(target)
	}
	if math.Abs(vs.Value()-target) > 0.1 {
		t.Errorf("ValueSpring did not converge: got %v, want ~%v", vs.Value(), target)
	}
}

func TestValueSpring_Deterministic(t *testing.T) {
	// Two springs with same seed should give identical values.
	a := anim.NewValueSpring(60)
	b := anim.NewValueSpring(60)
	target := 50.0

	for i := 0; i < 100; i++ {
		a.Step(target)
		b.Step(target)
	}
	if a.Value() != b.Value() {
		t.Errorf("ValueSpring not deterministic: a=%v b=%v", a.Value(), b.Value())
	}
}

func TestValueSpring_StartEqualsTarget_StaysPut(t *testing.T) {
	vs := anim.NewValueSpring(60)
	// Default value is 0; step toward 0 — should stay at 0.
	for i := 0; i < 50; i++ {
		vs.Step(0)
	}
	if vs.Value() != 0 {
		t.Errorf("ValueSpring at equilibrium moved: got %v", vs.Value())
	}
}

func TestValueSpring_OvershootsBeforeSettling(t *testing.T) {
	// An under-damped spring should overshoot its target at least once.
	// Default params should be under-damped for juice.
	vs := anim.NewValueSpring(60)
	target := 100.0

	maxVal := 0.0
	for i := 0; i < 200; i++ {
		vs.Step(target)
		if vs.Value() > maxVal {
			maxVal = vs.Value()
		}
	}
	// Overshoot means we exceeded the target at some point.
	if maxVal <= target {
		t.Errorf("ValueSpring did not overshoot: max=%v, target=%v", maxVal, target)
	}
}

func TestValueSpring_SetValue(t *testing.T) {
	vs := anim.NewValueSpring(60)
	vs.SetValue(50)
	if vs.Value() != 50 {
		t.Errorf("SetValue: got %v want 50", vs.Value())
	}
}

// --- ScreenShake tests ---

func TestScreenShake_DecaysToZero(t *testing.T) {
	ss := anim.NewScreenShake(60)
	ss.Trigger(10)

	maxSteps := 300
	for i := 0; i < maxSteps; i++ {
		ss.Step()
		if math.Abs(float64(ss.Offset())) < 1 {
			return // decayed to ~0
		}
	}
	t.Errorf("ScreenShake did not decay to ~0 within %d steps; final offset=%v", maxSteps, ss.Offset())
}

func TestScreenShake_Deterministic(t *testing.T) {
	a := anim.NewScreenShake(60)
	b := anim.NewScreenShake(60)

	a.Trigger(8)
	b.Trigger(8)

	allSame := true
	for i := 0; i < 100; i++ {
		a.Step()
		b.Step()
		if a.Offset() != b.Offset() {
			allSame = false
			break
		}
	}
	if !allSame {
		t.Error("ScreenShake is not deterministic")
	}
}

func TestScreenShake_ZeroBeforeTrigger(t *testing.T) {
	ss := anim.NewScreenShake(60)
	if ss.Offset() != 0 {
		t.Errorf("ScreenShake initial offset = %v, want 0", ss.Offset())
	}
	ss.Step()
	if ss.Offset() != 0 {
		t.Errorf("ScreenShake offset after step (no trigger) = %v, want 0", ss.Offset())
	}
}

func TestScreenShake_LargerMagnitudeMoreDisplacement(t *testing.T) {
	// A bigger trigger should produce a bigger initial displacement.
	ssSmall := anim.NewScreenShake(60)
	ssLarge := anim.NewScreenShake(60)

	ssSmall.Trigger(1)
	ssLarge.Trigger(20)

	// After the first step, the large one should have greater |offset|.
	ssSmall.Step()
	ssLarge.Step()

	if math.Abs(float64(ssLarge.Offset())) < math.Abs(float64(ssSmall.Offset())) {
		t.Errorf("larger magnitude did not produce larger initial displacement: small=%v large=%v",
			ssSmall.Offset(), ssLarge.Offset())
	}
}

func TestScreenShake_NeverDiverges(t *testing.T) {
	ss := anim.NewScreenShake(60)
	ss.Trigger(100)

	for i := 0; i < 500; i++ {
		ss.Step()
		if math.Abs(float64(ss.Offset())) > 1000 {
			t.Errorf("ScreenShake diverged at step %d: offset=%v", i, ss.Offset())
			return
		}
	}
}
