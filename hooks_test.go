package captainhook

import (
	"testing"
)

func TestCommandIdentity(t *testing.T) {
	isWard := CommandIdentity("ward", "ward.exe")

	tests := []struct {
		cmd  string
		want bool
	}{
		{"ward eval", true},
		{"ward.exe eval", true},
		{"C:/code/ward/ward.exe eval --verbose", true},
		{"C:/Program Files/Ward/ward.exe", true},
		{`"C:/Program Files/Ward/ward.exe" eval`, true},
		{`"ward.exe" eval`, true},
		{"claudio.exe", false},
		{"node something", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := isWard(tt.cmd); got != tt.want {
			t.Errorf("isWard(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestInstallSerializesExecArgsAndWindowsCommand(t *testing.T) {
	settings := make(SettingsMap)
	specs := []HookSpec{
		{
			Event:          "PreToolUse",
			Matcher:        "*",
			Command:        "/opt/Ward Tools/ward",
			CommandWindows: `C:/Program Files/Ward/ward.exe eval`,
			Args:           []string{"eval"},
			Timeout:        5,
		},
	}

	if err := Install(&settings, specs, CommandIdentity("ward", "ward.exe")); err != nil {
		t.Fatal(err)
	}

	hooks := settings["hooks"].(map[string]interface{})
	groups := hooks["PreToolUse"].([]interface{})
	group := groups[0].(map[string]interface{})
	entry := group["hooks"].([]interface{})[0].(map[string]interface{})
	if entry["command"] != "/opt/Ward Tools/ward" {
		t.Fatalf("command = %#v", entry["command"])
	}
	if entry["commandWindows"] != `C:/Program Files/Ward/ward.exe eval` {
		t.Fatalf("commandWindows = %#v", entry["commandWindows"])
	}
	args, ok := entry["args"].([]string)
	if !ok || len(args) != 1 || args[0] != "eval" {
		t.Fatalf("args = %#v, want [eval]", entry["args"])
	}
	if group["matcher"] != "*" {
		t.Fatalf("matcher = %#v, want *", group["matcher"])
	}
}

func TestInstallFreshSettings(t *testing.T) {
	settings := make(SettingsMap)
	specs := []HookSpec{
		{Event: "PreToolUse", Matcher: "Bash|Edit", Command: "ward eval", Timeout: 5},
		{Event: "SessionEnd", Command: "ward end-session"},
	}

	err := Install(&settings, specs, CommandIdentity("ward", "ward.exe"))
	if err != nil {
		t.Fatal(err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("hooks section missing")
	}

	if _, ok := hooks["PreToolUse"]; !ok {
		t.Error("PreToolUse hook missing")
	}
	if _, ok := hooks["SessionEnd"]; !ok {
		t.Error("SessionEnd hook missing")
	}
}

func TestInstallIdempotent(t *testing.T) {
	settings := make(SettingsMap)
	specs := []HookSpec{
		{Event: "PreToolUse", Matcher: "Bash|Edit", Command: "ward eval", Timeout: 5},
	}
	isWard := CommandIdentity("ward", "ward.exe")

	Install(&settings, specs, isWard)
	Install(&settings, specs, isWard)

	hooks := settings["hooks"].(map[string]interface{})
	groups := hooks["PreToolUse"].([]interface{})
	if len(groups) != 1 {
		t.Errorf("expected 1 group after double install, got %d", len(groups))
	}
}

func TestInstallPreservesOtherHooks(t *testing.T) {
	settings := SettingsMap{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": ".*",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "claudio.exe",
						},
					},
				},
			},
		},
	}

	specs := []HookSpec{
		{Event: "PreToolUse", Matcher: "Bash|Edit", Command: "ward eval", Timeout: 5},
	}
	isWard := CommandIdentity("ward", "ward.exe")

	Install(&settings, specs, isWard)

	hooks := settings["hooks"].(map[string]interface{})
	groups := hooks["PreToolUse"].([]interface{})
	if len(groups) != 2 {
		t.Errorf("expected 2 groups (claudio + ward), got %d", len(groups))
	}
}

func TestInstallPreservesNonOwnedSiblingsWithinOwnedGroup(t *testing.T) {
	tests := []struct {
		name     string
		commands []interface{}
	}{
		{
			name: "owned command first",
			commands: []interface{}{
				map[string]interface{}{"type": "command", "command": "ward eval --old"},
				map[string]interface{}{"type": "command", "command": "user-lint"},
			},
		},
		{
			name: "owned command after sibling",
			commands: []interface{}{
				map[string]interface{}{"type": "command", "command": "user-lint"},
				map[string]interface{}{"type": "command", "command": "ward eval --old"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := SettingsMap{
				"hooks": map[string]interface{}{
					"PreToolUse": []interface{}{
						map[string]interface{}{
							"matcher": "existing",
							"hooks":   tt.commands,
						},
					},
				},
			}

			err := Install(&settings, []HookSpec{
				{Event: "PreToolUse", Matcher: "new", Command: "ward eval"},
			}, CommandIdentity("ward", "ward.exe"))
			if err != nil {
				t.Fatal(err)
			}

			hooks := settings["hooks"].(map[string]interface{})
			groups := hooks["PreToolUse"].([]interface{})
			if len(groups) != 2 {
				t.Fatalf("groups = %d, want preserved sibling group + new Ward group: %#v", len(groups), groups)
			}
			preserved := groups[0].(map[string]interface{})
			preservedCommands := preserved["hooks"].([]interface{})
			if len(preservedCommands) != 1 {
				t.Fatalf("preserved commands = %d, want 1: %#v", len(preservedCommands), preservedCommands)
			}
			preservedEntry := preservedCommands[0].(map[string]interface{})
			if preservedEntry["command"] != "user-lint" {
				t.Fatalf("preserved command = %#v, want user-lint", preservedEntry["command"])
			}
			installed := groups[1].(map[string]interface{})
			installedEntry := installed["hooks"].([]interface{})[0].(map[string]interface{})
			if installedEntry["command"] != "ward eval" {
				t.Fatalf("installed command = %#v, want ward eval", installedEntry["command"])
			}
		})
	}
}

func TestUninstall(t *testing.T) {
	settings := make(SettingsMap)
	specs := []HookSpec{
		{Event: "PreToolUse", Matcher: "Bash|Edit", Command: "ward eval"},
		{Event: "SessionEnd", Command: "ward end-session"},
	}
	isWard := CommandIdentity("ward", "ward.exe")

	Install(&settings, specs, isWard)
	Uninstall(&settings, isWard)

	if _, ok := settings["hooks"]; ok {
		t.Error("hooks section should be removed after uninstalling all hooks")
	}
}

func TestUninstallPreservesOthers(t *testing.T) {
	settings := SettingsMap{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": ".*",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "claudio.exe",
						},
					},
				},
				map[string]interface{}{
					"matcher": "Bash|Edit",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "ward eval",
						},
					},
				},
			},
		},
	}

	isWard := CommandIdentity("ward", "ward.exe")
	Uninstall(&settings, isWard)

	hooks := settings["hooks"].(map[string]interface{})
	groups := hooks["PreToolUse"].([]interface{})
	if len(groups) != 1 {
		t.Errorf("expected 1 group (claudio only), got %d", len(groups))
	}
}

func TestUninstallPreservesNonOwnedSiblingsWithinOwnedGroup(t *testing.T) {
	tests := []struct {
		name     string
		commands []interface{}
	}{
		{
			name: "owned command first",
			commands: []interface{}{
				map[string]interface{}{"type": "command", "command": "ward eval"},
				map[string]interface{}{"type": "command", "command": "user-lint"},
			},
		},
		{
			name: "owned command after sibling",
			commands: []interface{}{
				map[string]interface{}{"type": "command", "command": "user-lint"},
				map[string]interface{}{"type": "command", "command": "ward eval"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := SettingsMap{
				"hooks": map[string]interface{}{
					"PreToolUse": []interface{}{
						map[string]interface{}{
							"matcher": "existing",
							"hooks":   tt.commands,
						},
					},
				},
			}

			Uninstall(&settings, CommandIdentity("ward", "ward.exe"))

			hooks := settings["hooks"].(map[string]interface{})
			groups := hooks["PreToolUse"].([]interface{})
			if len(groups) != 1 {
				t.Fatalf("groups = %d, want preserved sibling group: %#v", len(groups), groups)
			}
			preserved := groups[0].(map[string]interface{})
			preservedCommands := preserved["hooks"].([]interface{})
			if len(preservedCommands) != 1 {
				t.Fatalf("preserved commands = %d, want 1: %#v", len(preservedCommands), preservedCommands)
			}
			preservedEntry := preservedCommands[0].(map[string]interface{})
			if preservedEntry["command"] != "user-lint" {
				t.Fatalf("preserved command = %#v, want user-lint", preservedEntry["command"])
			}
		})
	}
}
