package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const (
	preferredBaseDir = "/mnt/config"
	fileName         = "config.yaml"
)

// File describes the resolved config location for a binary.
type File struct {
	Dir  string
	Path string
}

// Load ensures a YAML config exists for the given binary, fills in missing
// defaults, and decodes the result into target. Defaults must use YAML tags to
// match desired key names.
func Load(binary string, defaults any, target any) (File, error) {
	if binary == "" {
		return File{}, errors.New("binary name is required")
	}
	if target == nil {
		return File{}, errors.New("target config cannot be nil")
	}

	dir, err := resolveDir(binary)
	if err != nil {
		return File{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return File{}, err
	}
	path := filepath.Join(dir, fileName)

	defaultMap, err := asMap(defaults)
	if err != nil {
		return File{}, err
	}

	data, created, err := read(path)
	if err != nil {
		return File{}, err
	}

	updated := created
	for key, value := range defaultMap {
		if _, ok := data[key]; ok {
			continue
		}
		data[key] = value
		updated = true
	}
	if updated {
		if err := write(path, data); err != nil {
			return File{}, err
		}
	}
	if err := decode(data, target); err != nil {
		return File{}, err
	}
	return File{Dir: dir, Path: path}, nil
}

func resolveDir(binary string) (string, error) {
	binary = filepath.Base(binary)
	if binary == "" || binary == "." || binary == string(filepath.Separator) {
		return "", fmt.Errorf("invalid binary name %q", binary)
	}
	if info, err := os.Stat(preferredBaseDir); err == nil && info.IsDir() {
		return filepath.Join(preferredBaseDir, fmt.Sprintf(".%s", binary)), nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, fmt.Sprintf(".%s", binary)), nil
}

func asMap(defaults any) (map[string]any, error) {
	raw, err := yaml.Marshal(defaults)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := yaml.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if result == nil {
		result = map[string]any{}
	}
	return result, nil
}

func read(path string) (map[string]any, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{}, true, nil
		}
		return nil, false, err
	}
	var result map[string]any
	if err := yaml.Unmarshal(content, &result); err != nil {
		return nil, false, err
	}
	if result == nil {
		result = map[string]any{}
	}
	return result, false, nil
}

func write(path string, data map[string]any) error {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, key := range keys {
		node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key})
		var value yaml.Node
		if err := value.Encode(data[key]); err != nil {
			return err
		}
		node.Content = append(node.Content, &value)
	}
	content, err := yaml.Marshal(node)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func decode(data map[string]any, target any) error {
	content, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(content, target)
}
