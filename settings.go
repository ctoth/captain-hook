// Package captainhook provides shared Claude Code settings.json management
// for tools that register hooks (ward, claudio, etc).
package captainhook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// SettingsMap represents a Claude Code settings JSON object.
type SettingsMap map[string]interface{}

// FindSettingsPath returns the path to ~/.claude/settings.json.
// On Windows, prefers USERPROFILE over HOME (HOME may be MSYS-style).
// Returns the first existing file, or the primary path for creation.
func FindSettingsPath() (string, error) {
	paths := settingsPaths()
	if len(paths) == 0 {
		return "", fmt.Errorf("cannot determine home directory")
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return paths[0], nil
}

func settingsPaths() []string {
	home := homeDir()
	if home == "" {
		return nil
	}
	paths := []string{filepath.Join(home, ".claude", "settings.json")}

	if runtime.GOOS == "windows" {
		userProfile := os.Getenv("USERPROFILE")
		if userProfile != "" && userProfile != home {
			paths = append(paths, filepath.Join(userProfile, ".claude", "settings.json"))
		}
	}
	return paths
}

func homeDir() string {
	if runtime.GOOS == "windows" {
		if p := os.Getenv("USERPROFILE"); p != "" {
			return p
		}
		if p := os.Getenv("HOME"); p != "" {
			return normalizeMSYSPath(p)
		}
		drive := os.Getenv("HOMEDRIVE")
		path := os.Getenv("HOMEPATH")
		if drive != "" && path != "" {
			return drive + path
		}
		return ""
	}
	return os.Getenv("HOME")
}

func normalizeMSYSPath(path string) string {
	if len(path) >= 3 && path[0] == '/' && path[2] == '/' &&
		((path[1] >= 'a' && path[1] <= 'z') || (path[1] >= 'A' && path[1] <= 'Z')) {
		drive := strings.ToUpper(string(path[1]))
		return drive + ":" + filepath.FromSlash(path[2:])
	}
	return path
}

// ReadSettings reads and parses a Claude Code settings.json file.
// Returns empty settings if the file doesn't exist or is empty.
func ReadSettings(path string) (*SettingsMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s := make(SettingsMap)
			return &s, nil
		}
		return nil, fmt.Errorf("read settings: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" || content == "null" {
		s := make(SettingsMap)
		return &s, nil
	}

	var s SettingsMap
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse settings %s: %w", path, err)
	}
	if s == nil {
		s = make(SettingsMap)
	}
	return &s, nil
}

// WriteSettings writes settings to a file atomically.
// Creates the directory if needed. Preserves existing file permissions.
func WriteSettings(path string, settings *SettingsMap) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}

	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode() & os.ModePerm
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	// Atomic write: temp file + rename
	tmp, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp: %w", err)
	}
	tmp.Close()

	if err := os.Chmod(tmpName, mode); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("chmod temp: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename temp: %w", err)
	}
	return nil
}
