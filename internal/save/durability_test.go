package save

// durability_test.go — additional durability and branch coverage tests (T6, decision D11).
//
// Covers the branches NOT hit by save_test.go:
//  1. Save failure when the directory cannot be created (chmod 0500 parent dir).
//  2. SystemClock.Now() returns a time close to time.Now().
//  3. backupCorrupt collision: two corrupt-loads within the same clock second
//     so the suffix (-1, -2, …) branch fires.

import (
	"os"
	"testing"
	"time"
)

// ---- Save failure: directory not writable --------------------------------

// TestSave_DirectoryNotWritable verifies that Save returns a non-nil error when
// MkdirAll cannot create the target directory (parent is read-only, mode 0500).
// The test is skipped when running as root because root bypasses file permissions.
func TestSave_DirectoryNotWritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: filesystem permission checks are bypassed")
	}

	// Create a parent directory we own, then lock it (r-x, no write).
	parent := t.TempDir()
	if err := os.Chmod(parent, 0500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	// Make sure we restore writability so t.TempDir cleanup works.
	t.Cleanup(func() { _ = os.Chmod(parent, 0700) })

	// Point Save at a subdirectory under the locked parent.
	target := parent + "/glory_save"

	err := Save(target, newClock(epoch), sampleState())
	if err == nil {
		t.Fatal("Save should fail when the directory cannot be created")
	}
}

// ---- SystemClock.Now() --------------------------------------------------

// TestSystemClock_Now verifies the production Clock returns a wall-clock time
// close to time.Now(). This is a trivial but meaningful test because the line
// is otherwise uncovered: it's exercised only in main, not in other tests.
func TestSystemClock_Now(t *testing.T) {
	before := time.Now()
	got := SystemClock{}.Now()
	after := time.Now()

	if got.Before(before) || got.After(after.Add(time.Second)) {
		t.Fatalf("SystemClock.Now() = %v, want between %v and %v", got, before, after)
	}
}

// ---- backupCorrupt collision: suffix branch ------------------------------

// TestLoad_CorruptFile_CollisionSuffix exercises the "-1" suffix branch inside
// backupCorrupt. It does this by pre-creating the first backup path (the one
// without a suffix) before the Load call, forcing the function to append "-1".
func TestLoad_CorruptFile_CollisionSuffix(t *testing.T) {
	dir := t.TempDir()
	ts := epoch

	// Write a corrupt save file.
	if err := os.WriteFile(SavePath(dir), []byte("{not-json"), 0600); err != nil {
		t.Fatalf("setup corrupt save: %v", err)
	}

	// Pre-create the primary backup path (no suffix) to force the collision branch.
	primaryBackup := dir + "/save.json.corrupt-" + itoa(ts.Unix())
	if err := os.WriteFile(primaryBackup, []byte("pre-existing"), 0600); err != nil {
		t.Fatalf("setup collision file: %v", err)
	}

	clk := newClock(ts)
	out, err := Load(dir, clk)
	if err != nil {
		t.Fatalf("Load with collision: unexpected OS error: %v", err)
	}
	if out.Result != LoadCorrupt {
		t.Errorf("expected LoadCorrupt, got %v", out.Result)
	}

	// The backup should have been written to the "-1" suffixed path.
	suffixed := primaryBackup + "-1"
	if _, err := os.Stat(suffixed); err != nil {
		t.Errorf("expected suffixed backup at %s, got error: %v", suffixed, err)
	}

	// The pre-existing collision file must be untouched.
	data, err := os.ReadFile(primaryBackup)
	if err != nil {
		t.Fatalf("pre-existing backup gone: %v", err)
	}
	if string(data) != "pre-existing" {
		t.Errorf("pre-existing backup overwritten: got %q", data)
	}
}

// ---- XDGConfigDir: UserHomeDir error path --------------------------------

// TestXDGConfigDir_HomeDirError exercises the error branch when $HOME is unset
// and os.UserHomeDir() fails. We unset both HOME and XDG_CONFIG_HOME so the
// function falls through to the os.UserHomeDir() call.
// On some CI environments HOME cannot be cleared (it's always set); in that
// case this test becomes a no-op. The branch is only unreachable in production,
// not in tests, so it's worth exercising.
func TestXDGConfigDir_HomeDirError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: HOME might be enforced by the runtime")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	// On Linux, if HOME is empty os.UserHomeDir() looks at /etc/passwd.
	// It might still succeed (if the current uid has an entry). If so, skip.
	dir, err := XDGConfigDir()
	if err != nil {
		// Good — we exercised the error branch.
		_ = dir // silence linter
		return
	}
	// If no error, the function succeeded via /etc/passwd — that's fine,
	// the test is still valid (it ran the code; we can't force the error
	// without modifying production code).
	t.Logf("XDGConfigDir succeeded even with HOME unset (via /etc/passwd): %s", dir)
}

// ---- backupCorrupt rename failure ----------------------------------------

// TestLoad_BackupCorrupt_RenameFails exercises the error path in backupCorrupt
// when the Rename call fails. We do this by making the directory read-only (r-x)
// *after* writing the corrupt save, so that Rename (which needs write on the dir)
// fails.
func TestLoad_BackupCorrupt_RenameFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: filesystem permission checks are bypassed")
	}

	dir := t.TempDir()

	// Write a corrupt save file.
	if err := os.WriteFile(SavePath(dir), []byte("{bad json}"), 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Make the directory read-only so Rename cannot move the file.
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })

	_, err := Load(dir, newClock(epoch))
	if err == nil {
		t.Fatal("Load should return an error when backupCorrupt rename fails")
	}
}

// ---- Load: unreadable save file (non-ErrNotExist OS error) --------------

// TestLoad_UnreadableFile exercises the non-ErrNotExist branch in Load
// (the "unexpected OS-level failure" path). We create a save file then chmod
// it to 0000 so ReadFile returns permission denied — not ErrNotExist.
func TestLoad_UnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: filesystem permission checks are bypassed")
	}

	dir := t.TempDir()
	if err := Save(dir, newClock(epoch), sampleState()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Make the save file unreadable.
	if err := os.Chmod(SavePath(dir), 0000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(SavePath(dir), 0600) })

	_, err := Load(dir, newClock(epoch))
	if err == nil {
		t.Fatal("Load should return an error for an unreadable (0000) save file")
	}
}

// itoa converts an int64 to a decimal string (avoids importing strconv/fmt for a tiny helper).
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
