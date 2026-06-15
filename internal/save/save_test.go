package save

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/casualghoul/idle-glory/internal/game"
)

// fakeClock is an injectable Clock that returns a fixed time.
type fakeClock struct{ t time.Time }

func (f fakeClock) Now() time.Time { return f.t }

// epoch is a fixed reference timestamp for tests.
var epoch = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

func newClock(t time.Time) Clock { return fakeClock{t} }

// sampleState returns a non-zero State suitable for round-trip tests.
func sampleState() game.State {
	return game.State{
		Munitions:     100.5,
		MunitionsRate: 2.0,
		ArmyPower:     10.0,
		EnemyPower:    8.0,
		LinePosition:  1.5,
		OwnedCounts:   map[string]int{"supply_lines": 1},
	}
}

// ---- Round-trip ----------------------------------------------------------

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	clk := newClock(epoch)
	state := sampleState()

	if err := Save(dir, clk, state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out, err := Load(dir, clk)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Result != LoadOK {
		t.Fatalf("expected LoadOK, got %v", out.Result)
	}
	if out.State.Munitions != state.Munitions {
		t.Errorf("Munitions: got %v, want %v", out.State.Munitions, state.Munitions)
	}
	if out.State.MunitionsRate != state.MunitionsRate {
		t.Errorf("MunitionsRate: got %v, want %v", out.State.MunitionsRate, state.MunitionsRate)
	}
	if out.State.OwnedCounts["supply_lines"] != 1 {
		t.Errorf("OwnedCounts: got %v", out.State.OwnedCounts)
	}
	if !out.SavedAt.Equal(epoch.UTC()) {
		t.Errorf("SavedAt: got %v, want %v", out.SavedAt, epoch)
	}
}

// ---- Atomic write --------------------------------------------------------

func TestAtomicWrite_NoLeftoverTmp(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, newClock(epoch), sampleState()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "save.json.tmp")); !os.IsNotExist(err) {
		t.Error("expected .tmp file to be absent after Save")
	}
}

func TestAtomicWrite_SaveFilePresent(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, newClock(epoch), sampleState()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(SavePath(dir)); err != nil {
		t.Errorf("save.json missing after Save: %v", err)
	}
}

func TestAtomicWrite_ExistingSavePreservedOnSecondWrite(t *testing.T) {
	dir := t.TempDir()

	// First save with munitions=100.
	s1 := sampleState()
	s1.Munitions = 100
	if err := Save(dir, newClock(epoch), s1); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	// Second save with munitions=200.
	s2 := sampleState()
	s2.Munitions = 200
	if err := Save(dir, newClock(epoch.Add(time.Minute)), s2); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	out, err := Load(dir, newClock(epoch.Add(time.Minute)))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.State.Munitions != 200 {
		t.Errorf("expected munitions=200 after second save, got %v", out.State.Munitions)
	}
}

// ---- Missing file --------------------------------------------------------

func TestLoad_MissingFile_StartFresh(t *testing.T) {
	dir := t.TempDir()
	out, err := Load(dir, newClock(epoch))
	if err != nil {
		t.Fatalf("unexpected error for missing save: %v", err)
	}
	if out.Result != LoadMissing {
		t.Errorf("expected LoadMissing, got %v", out.Result)
	}
	if out.Message != "" {
		t.Errorf("unexpected message: %q", out.Message)
	}
	// No backup file should have been created.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected empty dir, got %v entries", len(entries))
	}
}

// ---- Corrupt file --------------------------------------------------------

func TestLoad_CorruptFile_BackedUp(t *testing.T) {
	dir := t.TempDir()
	ts := epoch
	clk := newClock(ts)

	// Write garbage bytes to save.json.
	garbage := []byte("not-valid-json{{{")
	if err := os.WriteFile(SavePath(dir), garbage, 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	out, err := Load(dir, clk)
	if err != nil {
		t.Fatalf("unexpected OS error: %v", err)
	}

	// Must signal LoadCorrupt, not start fresh silently.
	if out.Result != LoadCorrupt {
		t.Errorf("expected LoadCorrupt, got %v", out.Result)
	}
	if out.Message == "" {
		t.Error("expected non-empty message for corrupt save")
	}

	// Original save.json must NOT exist (it was renamed away).
	if _, err := os.Stat(SavePath(dir)); !os.IsNotExist(err) {
		t.Error("save.json should not exist after corrupt-backup")
	}

	// A backup file named save.json.corrupt-<unix>[-(n)] must exist with the
	// garbage bytes. Locate it robustly via glob rather than reconstructing the
	// exact name, which is fragile if the collision-suffix logic fires.
	pattern := filepath.Join(dir, fmt.Sprintf("save.json.corrupt-%d*", ts.Unix()))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly 1 backup file matching %s, got %v", pattern, matches)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("backup file missing (%s): %v", matches[0], err)
	}
	if string(data) != string(garbage) {
		t.Errorf("backup contents differ: got %q", data)
	}
}

func TestLoad_CorruptFile_MessageContainsBackupPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(SavePath(dir), []byte("{bad}"), 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	out, err := Load(dir, newClock(epoch))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !strings.Contains(out.Message, "corrupt") {
		t.Errorf("message should mention 'corrupt', got %q", out.Message)
	}
}

// ---- Version: older → migrate --------------------------------------------

func TestLoad_OlderVersion_MigratesSuccessfully(t *testing.T) {
	dir := t.TempDir()

	// Manually write a v0 record (simulate a past schema that had no version field,
	// which json.Unmarshal decodes as Version=0).
	rec := saveRecord{
		Version: 0,
		SavedAt: epoch,
		State:   sampleState(),
	}
	data, _ := json.Marshal(rec)
	if err := os.WriteFile(SavePath(dir), data, 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	out, err := Load(dir, newClock(epoch))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Result != LoadOK {
		t.Errorf("expected LoadOK after migration, got %v (msg=%q)", out.Result, out.Message)
	}
	// State should be intact after migration.
	if out.State.Munitions != rec.State.Munitions {
		t.Errorf("Munitions changed during migration: got %v", out.State.Munitions)
	}
}

// ---- Version: newer → backup + fresh ------------------------------------

func TestLoad_NewerVersion_BackupAndFresh(t *testing.T) {
	dir := t.TempDir()

	// Simulate a save from a future binary (version 999).
	rec := saveRecord{
		Version: 999,
		SavedAt: epoch,
		State:   sampleState(),
	}
	data, _ := json.Marshal(rec)
	if err := os.WriteFile(SavePath(dir), data, 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	out, err := Load(dir, newClock(epoch))
	if err != nil {
		t.Fatalf("Load returned unexpected OS error: %v", err)
	}

	if out.Result != LoadCorrupt {
		t.Errorf("expected LoadCorrupt for newer version, got %v", out.Result)
	}
	if out.Message == "" {
		t.Error("expected message for newer-version scenario")
	}

	// Original save.json must be gone (backed up).
	if _, err := os.Stat(SavePath(dir)); !os.IsNotExist(err) {
		t.Error("save.json should have been renamed away")
	}

	// No panic — the test itself proves that.
}

// ---- SavePath / XDGConfigDir --------------------------------------------

func TestSavePath(t *testing.T) {
	p := SavePath("/tmp/glory")
	if p != "/tmp/glory/save.json" {
		t.Errorf("unexpected SavePath: %v", p)
	}
}

func TestXDGConfigDir_Default(t *testing.T) {
	// Clear XDG_CONFIG_HOME so we exercise the ~/.config branch.
	t.Setenv("XDG_CONFIG_HOME", "")
	dir, err := XDGConfigDir()
	if err != nil {
		t.Fatalf("XDGConfigDir: %v", err)
	}
	if !strings.HasSuffix(dir, "/glory") {
		t.Errorf("expected dir to end in /glory, got %q", dir)
	}
	if !strings.Contains(dir, ".config") {
		t.Errorf("expected .config in path, got %q", dir)
	}
}

func TestXDGConfigDir_EnvOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/cfg")
	dir, err := XDGConfigDir()
	if err != nil {
		t.Fatalf("XDGConfigDir: %v", err)
	}
	if dir != "/custom/cfg/glory" {
		t.Errorf("expected /custom/cfg/glory, got %q", dir)
	}
}

// ---- Version field in saved file ----------------------------------------

func TestSave_VersionFieldIsSet(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, newClock(epoch), sampleState()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(SavePath(dir))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	v, ok := raw["version"]
	if !ok {
		t.Fatal("version field missing from saved JSON")
	}
	if v.(float64) != float64(currentVersion) {
		t.Errorf("version: got %v, want %v", v, currentVersion)
	}
}
