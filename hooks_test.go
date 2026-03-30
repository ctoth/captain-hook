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
