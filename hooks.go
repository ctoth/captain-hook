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
		if command[0] == '"' || command[0] == '\'' {
			if end := strings.IndexByte(command[1:], command[0]); end >= 0 {
				candidates = append(candidates, command[1:end+1])
			}
		} else if parts := strings.Fields(command); len(parts) > 0 {
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
			if group, keep := stripOwnedCommands(g, isOurs); keep {
				kept = append(kept, group)
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

	kept := make([]interface{}, 0, len(groups)+1)
	for _, g := range groups {
		if group, keep := stripOwnedCommands(g, isOurs); keep {
			kept = append(kept, group)
		}
	}
	kept = append(kept, newGroup)
	hooks[spec.Event] = kept
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

func stripOwnedCommands(groupRaw interface{}, isOurs IdentityFunc) (interface{}, bool) {
	group, ok := groupRaw.(map[string]interface{})
	if !ok {
		return groupRaw, true
	}
	hooksRaw, ok := group["hooks"].([]interface{})
	if !ok {
		return groupRaw, true
	}

	kept := make([]interface{}, 0, len(hooksRaw))
	removed := false
	for _, hookRaw := range hooksRaw {
		entry, ok := hookRaw.(map[string]interface{})
		if !ok {
			kept = append(kept, hookRaw)
			continue
		}
		cmd, ok := entry["command"].(string)
		if ok && isOurs(cmd) {
			removed = true
			continue
		}
		kept = append(kept, hookRaw)
	}

	if !removed {
		return groupRaw, true
	}
	if len(kept) == 0 {
		return nil, false
	}

	updated := make(map[string]interface{}, len(group))
	for key, value := range group {
		updated[key] = value
	}
	updated["hooks"] = kept
	return updated, true
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
