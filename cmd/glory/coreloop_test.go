package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andrewhorton/glory/internal/game"
	"github.com/andrewhorton/glory/internal/save"
)

// fakeClock is a controllable Clock for tests.
type fakeClock struct {
	t time.Time
}

func (f *fakeClock) Now() time.Time          { return f.t }
func (f *fakeClock) Advance(d time.Duration) { f.t = f.t.Add(d) }

// newTestClock returns a fake clock set to a fixed reference time.
func newTestClock() *fakeClock {
	ref := time.Date(2024, 7, 28, 8, 0, 0, 0, time.UTC)
	return &fakeClock{t: ref}
}

// newTestState returns a starting state with observable munitions rate.
func newTestState() game.State {
	return newState()
}

// TestQuitSavesProgressAndRelaunchRestoresIt verifies that Save + Load round-trips
// the game state exactly (munitions, rate, owned counts).
func TestQuitSavesProgressAndRelaunchRestoresIt(t *testing.T) {
	dir := t.TempDir()
	clk := newTestClock()

	// Build up some state.
	s := newTestState()
	// Tick forward 60 seconds to accumulate munitions.
	s = game.Tick(s, 60*time.Second)

	if err := save.Save(dir, clk, s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload without advancing clock (same instant).
	out, err := save.Load(dir, clk)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Result != save.LoadOK {
		t.Fatalf("want LoadOK, got %v", out.Result)
	}

	loaded := out.State
	if loaded.Munitions != s.Munitions {
		t.Errorf("Munitions: want %v, got %v", s.Munitions, loaded.Munitions)
	}
	if loaded.MunitionsRate != s.MunitionsRate {
		t.Errorf("MunitionsRate: want %v, got %v", s.MunitionsRate, loaded.MunitionsRate)
	}
	for k, want := range s.OwnedCounts {
		if got := loaded.OwnedCounts[k]; got != want {
			t.Errorf("OwnedCounts[%q]: want %d, got %d", k, want, got)
		}
	}
}

// TestAwayProgressOnRelaunch verifies that reloading after a known duration
// applies exactly rate*duration munitions (same as game.Tick).
func TestAwayProgressOnRelaunch(t *testing.T) {
	dir := t.TempDir()
	clk := newTestClock()

	s := newTestState()
	savedAt := clk.Now()

	if err := save.Save(dir, clk, s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Advance clock by 30 minutes.
	awayDuration := 30 * time.Minute
	clk.Advance(awayDuration)

	out, err := save.Load(dir, clk)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Result != save.LoadOK {
		t.Fatalf("want LoadOK, got %v", out.Result)
	}

	away := save.ApplyAwayProgress(out.State, savedAt, clk)

	// Munitions gained must match game.Tick directly.
	expected := game.Tick(s, awayDuration).Munitions - s.Munitions
	if away.MunitionsGained != expected {
		t.Errorf("MunitionsGained: want %.4f, got %.4f", expected, away.MunitionsGained)
	}
	if away.Duration != awayDuration {
		t.Errorf("Duration: want %v, got %v", awayDuration, away.Duration)
	}
}

// TestBackwardClockClamp verifies that a clock going backward never causes
// negative or unexpected munitions gain.
func TestBackwardClockClamp(t *testing.T) {
	dir := t.TempDir()
	clk := newTestClock()

	s := newTestState()
	savedAt := clk.Now()

	if err := save.Save(dir, clk, s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Clock goes backward by 1 hour.
	clk.Advance(-1 * time.Hour)

	out, err := save.Load(dir, clk)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Result != save.LoadOK {
		t.Fatalf("want LoadOK, got %v", out.Result)
	}

	away := save.ApplyAwayProgress(out.State, savedAt, clk)

	if away.MunitionsGained != 0 {
		t.Errorf("backward clock: want 0 MunitionsGained, got %v", away.MunitionsGained)
	}
	if away.Duration != 0 {
		t.Errorf("backward clock: want 0 Duration, got %v", away.Duration)
	}
	if away.State.Munitions != s.Munitions {
		t.Errorf("backward clock: munitions should not change; want %v, got %v",
			s.Munitions, away.State.Munitions)
	}
}

// TestHugeSleepClamp verifies that an extremely long away period is capped at
// MaxAwayDuration, preventing instant-win overflow.
func TestHugeSleepClamp(t *testing.T) {
	dir := t.TempDir()
	clk := newTestClock()

	s := newTestState()
	savedAt := clk.Now()

	if err := save.Save(dir, clk, s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Advance clock by 30 days — far beyond the 24h cap.
	clk.Advance(30 * 24 * time.Hour)

	out, err := save.Load(dir, clk)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Result != save.LoadOK {
		t.Fatalf("want LoadOK, got %v", out.Result)
	}

	away := save.ApplyAwayProgress(out.State, savedAt, clk)

	// Earnings must equal exactly MaxAwayDuration of ticking.
	cappedGain := game.Tick(s, save.MaxAwayDuration).Munitions - s.Munitions
	if away.MunitionsGained != cappedGain {
		t.Errorf("huge sleep: want capped gain %.4f, got %.4f", cappedGain, away.MunitionsGained)
	}
	if away.Duration != save.MaxAwayDuration {
		t.Errorf("huge sleep: want Duration=%v, got %v", save.MaxAwayDuration, away.Duration)
	}
}

// TestBuyChangesAccumulationRate verifies that purchasing a MunitionsRate upgrade
// causes subsequent ticks to accumulate faster.
func TestBuyChangesAccumulationRate(t *testing.T) {
	s := newTestState()

	// Tick before buying.
	dt := 10 * time.Second
	beforeTick := game.Tick(s, dt)
	gainBefore := beforeTick.Munitions - s.Munitions

	// Afford the cheapest MunitionsRate upgrade.
	// supply_lines costs 50 at BaseCost; give the player enough munitions.
	s.Munitions = 1000

	upgraded, ok := game.Buy(s, "supply_lines")
	if !ok {
		t.Fatal("Buy supply_lines: expected success")
	}

	afterTick := game.Tick(upgraded, dt)
	gainAfter := afterTick.Munitions - upgraded.Munitions

	if gainAfter <= gainBefore {
		t.Errorf("expected higher gain after upgrade; before=%.4f after=%.4f", gainBefore, gainAfter)
	}
}

// TestCorruptSaveHandling verifies that writing garbage to the save path results
// in LoadCorrupt, a backup file on disk, and a non-empty Message.
func TestCorruptSaveHandling(t *testing.T) {
	dir := t.TempDir()
	clk := newTestClock()

	savePath := save.SavePath(dir)
	if err := os.WriteFile(savePath, []byte("not valid json {{{{"), 0600); err != nil {
		t.Fatalf("write garbage: %v", err)
	}

	out, err := save.Load(dir, clk)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if out.Result != save.LoadCorrupt {
		t.Fatalf("want LoadCorrupt, got %v", out.Result)
	}
	if out.Message == "" {
		t.Error("want non-empty Message for corrupt save")
	}

	// The original save.json must no longer exist.
	if _, statErr := os.Stat(savePath); !os.IsNotExist(statErr) {
		t.Error("original save.json should be gone after backup")
	}

	// A backup file should exist in the dir.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	found := false
	for _, e := range entries {
		if len(e.Name()) > len("save.json.corrupt-") &&
			e.Name()[:len("save.json.corrupt-")] == "save.json.corrupt-" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a backup file matching save.json.corrupt-*")
	}
}

// TestRunHeadlessProducesValidState verifies that runHeadless returns a state
// with accumulated munitions and optionally a purchased upgrade, given
// deterministic inputs.
func TestRunHeadlessProducesValidState(t *testing.T) {
	clk := newTestClock()
	start := newTestState()

	result := runHeadless(start, clk, headlessConfig{
		ticks:  10,
		tickDt: time.Second,
	})

	// After 10 ticks of 1 second each, munitions must have grown.
	expectedMin := start.Munitions + start.MunitionsRate*10
	if result.Munitions < expectedMin {
		t.Errorf("runHeadless: want munitions >= %.4f, got %.4f", expectedMin, result.Munitions)
	}
}

// TestRunHeadlessFilepath verifies the full save-load cycle via runHeadless:
// run once, save, then simulate a relaunch with away-progress.
func TestRunHeadlessFilepath(t *testing.T) {
	dir := t.TempDir()
	clk := newTestClock()

	s := newState()
	afterFirst := runHeadless(s, clk, headlessConfig{ticks: 5, tickDt: time.Second})

	if err := save.Save(dir, clk, afterFirst); err != nil {
		t.Fatalf("Save: %v", err)
	}

	savedAt := clk.Now()
	clk.Advance(10 * time.Minute)

	out, err := save.Load(dir, clk)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Result != save.LoadOK {
		t.Fatalf("want LoadOK, got %v", out.Result)
	}

	away := save.ApplyAwayProgress(out.State, savedAt, clk)
	if away.MunitionsGained <= 0 {
		t.Error("expected positive munitions gain on second launch")
	}

	// Verify the output path for save exists
	savePath := save.SavePath(dir)
	if _, err := os.Stat(savePath); err != nil {
		t.Errorf("save file should exist at %s: %v", savePath, err)
	}

	_ = filepath.Join(dir, "save.json") // keep import used
}
