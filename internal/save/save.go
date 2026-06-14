package save

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/andrewhorton/glory/internal/game"
)

// currentVersion is the save-format schema version. Increment this when the
// on-disk format changes in a backwards-incompatible way.
const currentVersion = 1

// saveRecord is the on-disk JSON envelope. It carries versioning metadata
// separate from game.State so the pure game package never needs version logic.
type saveRecord struct {
	Version int        `json:"version"`
	SavedAt time.Time  `json:"saved_at"`
	State   game.State `json:"state"`
}

// LoadResult describes the outcome of a Load call.
type LoadResult int

const (
	// LoadOK means a valid save was found and returned.
	LoadOK LoadResult = iota
	// LoadMissing means no save file exists; the caller should start fresh.
	LoadMissing
	// LoadCorrupt means the file existed but could not be parsed or had a
	// future version; it has been backed up and the caller should start fresh.
	LoadCorrupt
)

// LoadOut is returned by Load. Check Result to decide how to proceed.
type LoadOut struct {
	// Result indicates the outcome.
	Result LoadResult
	// State is the loaded (and migrated) game state. Valid only when Result == LoadOK.
	State game.State
	// SavedAt is the wall-clock time the save was written. Valid only when Result == LoadOK.
	SavedAt time.Time
	// Message is a human-readable note to surface to the player.
	// Non-empty when Result == LoadCorrupt (identifies the backup path).
	Message string
}

// SavePath returns the canonical path for the save file under baseDir.
// baseDir is normally XDGConfigDir().
func SavePath(baseDir string) string {
	return filepath.Join(baseDir, "save.json")
}

// XDGConfigDir returns the Glory config directory following XDG conventions:
// $XDG_CONFIG_HOME/glory if $XDG_CONFIG_HOME is set, else ~/.config/glory.
func XDGConfigDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("save: locate home dir: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "glory"), nil
}

// Save writes state to <dir>/save.json atomically (write tmp, fsync, rename).
// dir is created (mode 0700) if it does not exist.
// clk is used to stamp SavedAt.
func Save(dir string, clk Clock, state game.State) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("save: mkdir %s: %w", dir, err)
	}

	rec := saveRecord{
		Version: currentVersion,
		SavedAt: clk.Now().UTC(),
		State:   state,
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("save: marshal: %w", err)
	}

	tmpPath := filepath.Join(dir, "save.json.tmp")
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("save: open tmp: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("save: write tmp: %w", err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("save: fsync tmp: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("save: close tmp: %w", err)
	}

	dest := SavePath(dir)
	if err := os.Rename(tmpPath, dest); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("save: rename to %s: %w", dest, err)
	}

	return nil
}

// Load reads <dir>/save.json and returns a LoadOut describing what happened.
// It never returns a non-nil error for expected conditions (missing file,
// corrupt file, future version); those are signalled via LoadOut.Result.
// A non-nil error means an unexpected OS-level failure (e.g. permission denied
// on the directory itself, or backup rename failure).
func Load(dir string, clk Clock) (LoadOut, error) {
	path := SavePath(dir)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LoadOut{Result: LoadMissing}, nil
		}
		return LoadOut{}, fmt.Errorf("save: read %s: %w", path, err)
	}

	var rec saveRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		msg, backupErr := backupCorrupt(dir, path, data, clk)
		if backupErr != nil {
			return LoadOut{}, backupErr
		}
		return LoadOut{Result: LoadCorrupt, Message: msg}, nil
	}

	// Version guard: future version → treat like corrupt (backup + fresh).
	if rec.Version > currentVersion {
		msg, backupErr := backupCorrupt(dir, path, data, clk)
		if backupErr != nil {
			return LoadOut{}, backupErr
		}
		return LoadOut{
			Result:  LoadCorrupt,
			Message: fmt.Sprintf("save written by a newer version (%d > %d); %s", rec.Version, currentVersion, msg),
		}, nil
	}

	// Version guard: older version → migrate.
	if rec.Version < currentVersion {
		rec = migrate(rec, rec.Version)
	}

	return LoadOut{
		Result:  LoadOK,
		State:   rec.State,
		SavedAt: rec.SavedAt,
	}, nil
}

// migrate upgrades rec from fromVersion to currentVersion by applying additive
// defaults for each schema step. For v1 there is nothing to do yet; this
// function is the hook future versions will fill in.
func migrate(rec saveRecord, fromVersion int) saveRecord {
	// Each case falls through to the next, applying changes incrementally.
	// To add a future v2 migration:
	//   case 1:
	//       rec.State.NewField = defaultValue
	//       rec.Version = 2
	//       fallthrough
	//   case 2:
	//       ...
	switch fromVersion {
	// v0 → v1: no structural changes; just stamp the current version.
	// (v0 was a pre-release format with no version field; json decodes it as 0.)
	default:
		rec.Version = currentVersion
	}
	return rec
}

// backupCorrupt moves the live save aside to save.json.corrupt-<unix> and
// removes the original path. Returns a human-readable message and any error.
func backupCorrupt(dir, path string, _ []byte, clk Clock) (string, error) {
	ts := clk.Now().Unix()
	backupPath := filepath.Join(dir, fmt.Sprintf("save.json.corrupt-%d", ts))
	if err := os.Rename(path, backupPath); err != nil {
		return "", fmt.Errorf("save: backup corrupt save to %s: %w", backupPath, err)
	}
	return fmt.Sprintf("corrupt save backed up to %s; starting fresh", backupPath), nil
}
