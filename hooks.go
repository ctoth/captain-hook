package captainhook

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// HookEntry represents a single hook command within a hook group.
type HookEntry struct {
	Type           string   `json:"type"`
	Command        string   `json:"command"`
	CommandWindows string   `json:"commandWindows,omitempty"`
	Args           []string `json:"args,omitempty"`
	Timeout        int      `json:"timeout,omitempty"`
}

// HookGroup represents a matcher + hooks array pair in the settings.
type HookGroup struct {
	Matcher string      `json:"matcher,omitempty"`
	Hooks   []HookEntry `json:"hooks"`
}

// HookSpec defines a hook that a tool wants to register.
type HookSpec struct {
	Event          string   // "PreToolUse", "SessionEnd", etc.
	Matcher        string   // "Bash|Edit|Write|WebFetch", ".*", etc.
	Command        string   // executable in exec form, shell command otherwise
	CommandWindows string   // optional Windows command override
	Args           []string // optional exec-form argument vector
	Timeout        int      // seconds, 0 = default
}

// IdentityFunc returns true if a command string belongs to a given tool.
// Used to detect existing hooks during merge (for idempotent install).
type IdentityFunc func(command string) bool

// CommandIdentity returns an IdentityFunc that matches commands by executable base name.
// Example: CommandIdentity("ward", "ward.exe") matches "ward eval", "C:/code/ward/ward.exe eval".
func CommandIdentity(names ...string) IdentityFunc {
	return func(command string) bool {
		command = strings.TrimSpace(command)
		if command == "" {
			return false
		}

		// Exec-form commands can be an unquoted path containing spaces because
		// the argument vector is stored separately. Check the whole value first.
		candidates := []string{command}
		if parts := strings.Fields(command); len(parts) > 0 {
			candidates = append(candidates, parts[0])
		}
		for _, candidate := range candidates {
			exe := strings.Trim(filepath.Base(candidate), `"'`)
			for _, name := range names {
				if strings.EqualFold(exe, name) {
					return true
				}
			}
		}
		return false
	}
}

// Install adds or updates hooks in settings for a given tool.
// It's idempotent: running it twice produces the same result.
// Other tools' hooks are preserved.
func Install(settings *SettingsMap, specs []HookSpec, isOurs IdentityFunc) error {
	hooks := getOrCreateHooksSection(settings)

	for _, spec := range specs {
		installOneHook(hooks, spec, isOurs)
	}

	(*settings)["hooks"] = map[string]interface{}(hooks)
	return nil
}

// Uninstall removes all hooks belonging to a tool from settings.
// Other tools' hooks are preserved.
func Uninstall(settings *SettingsMap, isOurs IdentityFunc) {
	hooksRaw, ok := (*settings)["hooks"]
	if !ok {
		return
	}
	hooksMap, ok := hooksRaw.(map[string]interface{})
	if !ok {
		return
	}

	for event, groupsRaw := range hooksMap {
		groups, ok := groupsRaw.([]interface{})
		if !ok {
			continue
		}

		var kept []interface{}
		for _, g := range groups {
			if !groupBelongsTo(g, isOurs) {
				kept = append(kept, g)
			}
		}

		if len(kept) == 0 {
			delete(hooksMap, event)
		} else {
			hooksMap[event] = kept
		}
	}

	if len(hooksMap) == 0 {
		delete(*settings, "hooks")
	}
}

func getOrCreateHooksSection(settings *SettingsMap) map[string]interface{} {
	if hooksRaw, ok := (*settings)["hooks"]; ok {
		if m, ok := hooksRaw.(map[string]interface{}); ok {
			return m
		}
	}
	m := make(map[string]interface{})
	(*settings)["hooks"] = m
	return m
}

func installOneHook(hooks map[string]interface{}, spec HookSpec, isOurs IdentityFunc) {
	newGroup := buildGroup(spec)

	groupsRaw, exists := hooks[spec.Event]
	if !exists {
		hooks[spec.Event] = []interface{}{newGroup}
		return
	}

	groups, ok := groupsRaw.([]interface{})
	if !ok {
		hooks[spec.Event] = []interface{}{newGroup}
		return
	}

	// Find and replace existing hook from this tool, or append
	replaced := false
	for i, g := range groups {
		if groupBelongsTo(g, isOurs) {
			groups[i] = newGroup
			replaced = true
			break
		}
	}

	if !replaced {
		groups = append(groups, newGroup)
	}
	hooks[spec.Event] = groups
}

func buildGroup(spec HookSpec) map[string]interface{} {
	entry := map[string]interface{}{
		"type":    "command",
		"command": spec.Command,
	}
	if spec.CommandWindows != "" {
		entry["commandWindows"] = spec.CommandWindows
	}
	if len(spec.Args) > 0 {
		entry["args"] = append([]string(nil), spec.Args...)
	}
	if spec.Timeout > 0 {
		entry["timeout"] = spec.Timeout
	}

	group := map[string]interface{}{
		"hooks": []interface{}{entry},
	}
	if spec.Matcher != "" {
		group["matcher"] = spec.Matcher
	}
	return group
}

func groupBelongsTo(groupRaw interface{}, isOurs IdentityFunc) bool {
	group, ok := groupRaw.(map[string]interface{})
	if !ok {
		return false
	}
	hooksRaw, ok := group["hooks"].([]interface{})
	if !ok || len(hooksRaw) == 0 {
		return false
	}
	// Check the first hook entry in the group
	entry, ok := hooksRaw[0].(map[string]interface{})
	if !ok {
		return false
	}
	cmd, ok := entry["command"].(string)
	if !ok {
		return false
	}
	return isOurs(cmd)
}

// deepCopy creates a deep copy of settings via JSON round-trip.
func deepCopy(s *SettingsMap) (*SettingsMap, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal for copy: %w", err)
	}
	var copy SettingsMap
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, fmt.Errorf("unmarshal for copy: %w", err)
	}
	return &copy, nil
}
