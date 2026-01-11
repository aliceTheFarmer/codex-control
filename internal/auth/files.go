package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"codex-control/internal/fsx"
)

const usageStateFile = ".codex-auth-last-used.json"

// File represents an auth file candidate.
type File struct {
	Name    string
	Path    string
	Size    int64
	ModTime time.Time
}

// CopyResult describes the installed auth file.
type CopyResult struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Bytes       int64  `json:"bytes"`
}

// ListFiles scans the provided directory for regular files.
func ListFiles(root string) ([]File, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	files := make([]File, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Name() == usageStateFile {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		files = append(files, File{
			Name:    entry.Name(),
			Path:    filepath.Join(root, entry.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})
	return files, nil
}

// Install copies the auth file into ~/.codex/auth.json.
func Install(src string) (CopyResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return CopyResult{}, err
	}
	destDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return CopyResult{}, err
	}
	dest := filepath.Join(destDir, "auth.json")
	info, err := os.Stat(src)
	if err != nil {
		return CopyResult{}, err
	}
	if err := fsx.CopyFile(src, dest, 0o600); err != nil {
		return CopyResult{}, err
	}
	return CopyResult{Source: src, Destination: dest, Bytes: info.Size()}, nil
}

// ValidateRoot ensures the directory exists and returns the folder that actually stores auth files.
// If the provided path contains an "auths" subdirectory with files, that subdirectory is preferred.
func ValidateRoot(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("auths path is not set; set --auths-path or update the config file")
	}
	cleaned := filepath.Clean(path)
	info, err := os.Stat(cleaned)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", cleaned)
	}
	candidates := []string{cleaned}
	if filepath.Base(cleaned) != "auths" {
		candidates = append([]string{filepath.Join(cleaned, "auths")}, candidates...)
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || !info.IsDir() {
			continue
		}
		hasFiles, err := dirHasFiles(candidate)
		if err != nil {
			return "", err
		}
		if hasFiles {
			return filepath.Clean(candidate), nil
		}
	}
	return "", fmt.Errorf("no auth files found inside %s", cleaned)
}

func dirHasFiles(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Name() == usageStateFile {
			continue
		}
		return true, nil
	}
	return false, nil
}
