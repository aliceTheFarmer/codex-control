package env

import (
	"os"
	"path/filepath"
)

const (
	defaultWorkspace = "/tmp/codex-control"
	targetBinary     = "/usr/bin/codex"
)

// PrepareWorkspace ensures the shared workspace directory exists and is empty.
func PrepareWorkspace() (string, error) {
	if err := os.RemoveAll(defaultWorkspace); err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(defaultWorkspace, 0o755); err != nil {
		return "", err
	}
	return defaultWorkspace, nil
}

// CleanupWorkspace deletes the given workspace directory.
func CleanupWorkspace(path string) error {
	if path == "" {
		return nil
	}
	if filepath.Clean(path) != filepath.Clean(defaultWorkspace) {
		return nil
	}
	return os.RemoveAll(path)
}

// TargetBinaryPath returns the final installation path for Codex.
func TargetBinaryPath() string {
	return targetBinary
}
