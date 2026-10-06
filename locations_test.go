package captainhook

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestAgentSettingsLocations(t *testing.T) {
	type location struct {
		homeEnv, configDir, settingsFile string
		projectFiles                     []string
		matcher                          string
		powerShell                       bool
	}
	want := map[Agent]location{
		AgentClaude:      {"CLAUDE_CONFIG_DIR", ".claude", "settings.json", []string{".claude/settings.json"}, ".*", false},
		AgentCodex:       {"CODEX_HOME", ".codex", "hooks.json", []string{".codex/hooks.json"}, "*", true},
		AgentGemini:      {"", ".gemini", "settings.json", []string{".gemini/settings.json"}, "", false},
		AgentQwen:        {"", ".qwen", "settings.json", []string{".qwen/settings.json"}, ".*", false},
		AgentCopilot:     {"COPILOT_HOME", ".copilot", "settings.json", []string{".github/copilot/settings.local.json", ".github/copilot/settings.json"}, "", false},
		AgentCommandCode: {"", ".commandcode", "settings.json", []string{".commandcode/settings.json"}, "", false},
	}
	for _, hooks := range Agents() {
		w, ok := want[hooks.Agent]
		if !ok {
			t.Errorf("%s: no expected settings location", hooks.Agent)
			continue
		}
		got := location{hooks.HomeEnv, hooks.ConfigDir, hooks.SettingsFile, hooks.ProjectFiles, hooks.Matcher, hooks.PowerShell}
		if !reflect.DeepEqual(got, w) {
			t.Errorf("%s: location = %+v, want %+v", hooks.Agent, got, w)
		}
	}
}

// setHome points the home directory at dir on every OS.
func setHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
}

func TestGlobalSettingsPath(t *testing.T) {
	home := t.TempDir()
	override := t.TempDir()
	codex, _ := Lookup(AgentCodex)
	gemini, _ := Lookup(AgentGemini)

	t.Run("under the home directory", func(t *testing.T) {
		setHome(t, home)
		t.Setenv("CODEX_HOME", "")
		got, err := codex.GlobalSettingsPath()
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(home, ".codex", "hooks.json"); got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	})

	t.Run("the config-home variable replaces the directory", func(t *testing.T) {
		setHome(t, home)
		t.Setenv("CODEX_HOME", "  "+override+"  ")
		got, err := codex.GlobalSettingsPath()
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(override, "hooks.json"); got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	})

	t.Run("a blank config-home variable is unset", func(t *testing.T) {
		setHome(t, home)
		t.Setenv("CODEX_HOME", "   ")
		got, err := codex.GlobalSettingsPath()
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(home, ".codex", "hooks.json"); got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	})

	t.Run("an agent with no config-home variable", func(t *testing.T) {
		setHome(t, home)
		got, err := gemini.GlobalSettingsPath()
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(home, ".gemini", "settings.json"); got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	})

	t.Run("no home directory is an error, not a relative path", func(t *testing.T) {
		setHome(t, "")
		if got, err := gemini.GlobalSettingsPath(); err == nil {
			t.Errorf("path = %q, want an error", got)
		}
	})

	t.Run("the config-home variable works without a home directory", func(t *testing.T) {
		setHome(t, "")
		t.Setenv("CODEX_HOME", override)
		got, err := codex.GlobalSettingsPath()
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(override, "hooks.json"); got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
	})
}

func TestProjectSettingsPaths(t *testing.T) {
	copilot, _ := Lookup(AgentCopilot)
	root := t.TempDir()
	got := copilot.ProjectSettingsPaths(root)
	want := []string{
		filepath.Join(root, ".github", "copilot", "settings.local.json"),
		filepath.Join(root, ".github", "copilot", "settings.json"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("paths = %q, want %q", got, want)
	}
	if runtime.GOOS == "windows" && filepath.ToSlash(got[0]) == got[0] {
		t.Errorf("path %q should use the host separator", got[0])
	}
}

func TestProjectFilesAreCopied(t *testing.T) {
	Agents()[0].ProjectFiles[0] = "mutated"
	if Agents()[0].ProjectFiles[0] == "mutated" {
		t.Fatal("Agents exposed the catalog's ProjectFiles")
	}
}
