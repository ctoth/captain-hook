package captainhook

import (
	"encoding/json"
	"fmt"
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
			exe := strings.Trim(portableBase(candidate), `"'`)
			for _, name := range names {
				if strings.EqualFold(exe, name) {
					return true
				}
			}
		}
		return false
	}
}

// portableBase returns the last path element, splitting on both '/' and
// '\'. filepath.Base only honors the host separator, but settings files
// are portable: a Windows path must be recognized on Linux too.
func portableBase(path string) string {
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		return path[i+1:]
	}
	return path
}

// legacyMatcher is the matcher given to a legacy string command when it is
// rewritten as a matcher group.
const legacyMatcher = ".*"

// Install adds or updates hooks in settings for a given tool.
// It's idempotent: running it twice produces the same result.
// Other tools' hooks are preserved, including legacy string commands, which
// are rewritten as one-command entries so ours can be appended next to them.
//
// Install returns an error, leaving settings unchanged, when the hooks
// section or an event it would touch has a shape it does not understand.
func Install(settings *SettingsMap, specs []HookSpec, isOurs IdentityFunc) error {
	hooks, err := hooksSection(settings)
	if err != nil {
		return err
	}

	// Group specs by event, keeping first-seen order, so every spec for an
	// event survives: strip ours once, then append all of them.
	var events []string
	byEvent := make(map[string][]HookSpec)
	for _, spec := range specs {
		if _, seen := byEvent[spec.Event]; !seen {
			events = append(events, spec.Event)
		}
		byEvent[spec.Event] = append(byEvent[spec.Event], spec)
	}

	// Validate every touched event before changing anything.
	updated := make(map[string][]interface{}, len(events))
	for _, event := range events {
		entries, err := eventEntries(hooks[event])
		if err != nil {
			return fmt.Errorf("hooks.%s: %w", event, err)
		}
		kept, _ := stripOwned(entries, isOurs)
		for _, spec := range byEvent[event] {
			kept = append(kept, buildGroup(spec))
		}
		updated[event] = kept
	}

	for event, entries := range updated {
		hooks[event] = entries
	}
	(*settings)["hooks"] = hooks
	return nil
}

// Uninstall removes all hooks belonging to a tool from settings.
// Other tools' hooks are preserved. Events and a hooks section that end up
// empty are deleted; shapes it does not understand are left alone.
func Uninstall(settings *SettingsMap, isOurs IdentityFunc) {
	hooksMap, ok := (*settings)["hooks"].(map[string]interface{})
	if !ok {
		return
	}

	for event, value := range hooksMap {
		entries, err := eventEntries(value)
		if err != nil {
			continue
		}
		kept, removed := stripOwned(entries, isOurs)
		if removed == 0 {
			continue
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

// hooksSection returns settings["hooks"], creating it when absent. A hooks
// value that is not a JSON object is an error.
func hooksSection(settings *SettingsMap) (map[string]interface{}, error) {
	raw, ok := (*settings)["hooks"]
	if !ok || raw == nil {
		return make(map[string]interface{}), nil
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("hooks: expected an object, got %T", raw)
	}
	return m, nil
}

// eventEntries returns an event's value as a hook array. An absent value is
// empty and a legacy string is one command in a matcher group. Any other
// shape is an error.
func eventEntries(value interface{}) ([]interface{}, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case []interface{}:
		return v, nil
	case string:
		return []interface{}{map[string]interface{}{
			"matcher": legacyMatcher,
			"hooks":   []interface{}{commandEntry(v)},
		}}, nil
	default:
		return nil, fmt.Errorf("expected an array or a command string, got %T", value)
	}
}

func commandEntry(command string) map[string]interface{} {
	return map[string]interface{}{"type": "command", "command": command}
}

func buildGroup(spec HookSpec) map[string]interface{} {
	entry := commandEntry(spec.Command)
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

// stripOwned returns the entries left after removing our commands, and how
// many commands it removed. A matcher group is dropped only when removal
// emptied it; groups without our commands are kept as they are. The input
// is not modified.
func stripOwned(entries []interface{}, isOurs IdentityFunc) ([]interface{}, int) {
	kept := make([]interface{}, 0, len(entries)+1)
	removed := 0
	for _, entry := range entries {
		group, n, keep := stripOwnedCommands(entry, isOurs)
		removed += n
		if keep {
			kept = append(kept, group)
		}
	}
	return kept, removed
}

func stripOwnedCommands(groupRaw interface{}, isOurs IdentityFunc) (interface{}, int, bool) {
	group, ok := groupRaw.(map[string]interface{})
	if !ok {
		return groupRaw, 0, true
	}
	hooksRaw, ok := group["hooks"].([]interface{})
	if !ok {
		return groupRaw, 0, true
	}

	kept := make([]interface{}, 0, len(hooksRaw))
	for _, hookRaw := range hooksRaw {
		if entry, ok := hookRaw.(map[string]interface{}); ok && isOwnedCommand(entry, isOurs) {
			continue
		}
		kept = append(kept, hookRaw)
	}

	removed := len(hooksRaw) - len(kept)
	if removed == 0 {
		return groupRaw, 0, true
	}
	if len(kept) == 0 {
		return nil, removed, false
	}

	updated := make(map[string]interface{}, len(group))
	for key, value := range group {
		updated[key] = value
	}
	updated["hooks"] = kept
	return updated, removed, true
}

func isOwnedCommand(entry map[string]interface{}, isOurs IdentityFunc) bool {
	cmd, ok := entry["command"].(string)
	return ok && isOurs(cmd)
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
