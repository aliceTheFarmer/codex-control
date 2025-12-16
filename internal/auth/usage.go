package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// UsageTracker records last-used timestamps for auth files.
type UsageTracker struct {
	path string
	data map[string]time.Time
}

// LoadUsageTracker loads metadata stored inside the auth directory.
func LoadUsageTracker(dir string) (*UsageTracker, error) {
	tracker := &UsageTracker{
		path: filepath.Join(dir, usageStateFile),
		data: map[string]time.Time{},
	}
	raw, err := os.ReadFile(tracker.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return tracker, nil
		}
		return nil, err
	}
	var payload map[string]string
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	for name, value := range payload {
		ts, err := time.Parse(time.RFC3339, value)
		if err != nil {
			continue
		}
		tracker.data[name] = ts
	}
	return tracker, nil
}

// LastUsed returns the tracked timestamp for the given file.
func (t *UsageTracker) LastUsed(name string) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.data[name]
}

// Touch records the current time for the provided file name.
func (t *UsageTracker) Touch(name string, ts time.Time) error {
	if t == nil {
		return nil
	}
	if t.data == nil {
		t.data = map[string]time.Time{}
	}
	t.data[name] = ts
	return t.save()
}

func (t *UsageTracker) save() error {
	tmp, err := os.CreateTemp(filepath.Dir(t.path), "codex-auth-usage-*.json")
	if err != nil {
		return err
	}
	enc := make(map[string]string, len(t.data))
	for name, ts := range t.data {
		enc[name] = ts.UTC().Format(time.RFC3339)
	}
	if err := json.NewEncoder(tmp).Encode(enc); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), t.path)
}
