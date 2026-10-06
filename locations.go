package captainhook

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoHomeDirectory is returned when a path under the user's home is needed
// and no home directory can be found. Falling back to a literal "~" would
// create a "~" directory under the working directory.
var ErrNoHomeDirectory = errors.New("cannot determine home directory: set HOME (or USERPROFILE on Windows)")

// HomeDir returns the user's home directory: on Windows USERPROFILE, then an
// MSYS-style HOME, then HOMEDRIVE+HOMEPATH; elsewhere HOME.
func HomeDir() (string, error) {
	if home := homeDir(); home != "" {
		return home, nil
	}
	return "", ErrNoHomeDirectory
}

// ConfigDirPath returns the agent's config directory. The agent's
// config-home variable, when set, replaces <home>/<ConfigDir> entirely: it
// is the only place the agent then looks. A blank variable counts as unset.
func (h AgentHooks) ConfigDirPath() (string, error) {
	if h.HomeEnv != "" {
		if dir := strings.TrimSpace(os.Getenv(h.HomeEnv)); dir != "" {
			return dir, nil
		}
	}
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, h.ConfigDir), nil
}

// GlobalSettingsPath returns the user-wide file that holds the agent's hooks.
func (h AgentHooks) GlobalSettingsPath() (string, error) {
	dir, err := h.ConfigDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, h.SettingsFile), nil
}

// ProjectSettingsPaths returns the project-scope files that hold the agent's
// hooks under root, in the order the agent prefers.
func (h AgentHooks) ProjectSettingsPaths(root string) []string {
	paths := make([]string, len(h.ProjectFiles))
	for i, file := range h.ProjectFiles {
		paths[i] = filepath.Join(root, filepath.FromSlash(file))
	}
	return paths
}
