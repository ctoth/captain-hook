package captainhook

import (
	"encoding/json"
	"reflect"
	"testing"
)

// settingsFromJSON parses a settings document for a test.
func settingsFromJSON(t *testing.T, doc string) SettingsMap {
	t.Helper()
	var s SettingsMap
	if err := json.Unmarshal([]byte(doc), &s); err != nil {
		t.Fatalf("bad test JSON %s: %v", doc, err)
	}
	return s
}

// assertSettingsJSON fails unless settings equal the JSON document.
func assertSettingsJSON(t *testing.T, settings SettingsMap, want string) {
	t.Helper()
	got, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	var gotV, wantV interface{}
	if err := json.Unmarshal(got, &gotV); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &wantV); err != nil {
		t.Fatalf("bad want JSON: %v", err)
	}
	if !reflect.DeepEqual(gotV, wantV) {
		t.Errorf("settings =\n%s\nwant\n%s", got, want)
	}
}

var wardEnd = []HookSpec{{Event: "Stop", Command: "ward end"}}

func TestInstallKeepsForeignLegacyStringCommand(t *testing.T) {
	settings := settingsFromJSON(t, `{"hooks": {"Stop": "other-tool --flag"}}`)

	if err := Install(&settings, wardEnd, CommandIdentity("ward")); err != nil {
		t.Fatal(err)
	}

	assertSettingsJSON(t, settings, `{"hooks": {"Stop": [
		{"matcher": ".*", "hooks": [{"type": "command", "command": "other-tool --flag"}]},
		{"hooks": [{"type": "command", "command": "ward end"}]}
	]}}`)
}

func TestInstallReplacesOwnedLegacyStringCommand(t *testing.T) {
	settings := settingsFromJSON(t, `{"hooks": {"Stop": "ward old-end"}}`)

	if err := Install(&settings, wardEnd, CommandIdentity("ward")); err != nil {
		t.Fatal(err)
	}

	assertSettingsJSON(t, settings, `{"hooks": {"Stop": [
		{"hooks": [{"type": "command", "command": "ward end"}]}
	]}}`)
}

func TestInstallRejectsUnknownShapesWithoutChangingSettings(t *testing.T) {
	cases := map[string]string{
		"number event":     `{"hooks": {"Stop": 42, "PreToolUse": []}}`,
		"object event":     `{"hooks": {"Stop": {"command": "x"}, "PreToolUse": []}}`,
		"string hooks key": `{"hooks": "not an object"}`,
	}
	specs := []HookSpec{
		{Event: "PreToolUse", Command: "ward eval"},
		{Event: "Stop", Command: "ward end"},
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			settings := settingsFromJSON(t, doc)
			if err := Install(&settings, specs, CommandIdentity("ward")); err == nil {
				t.Fatal("Install succeeded; want an error for an unknown hook shape")
			}
			assertSettingsJSON(t, settings, doc)
		})
	}
}

func TestUninstallRemovesOwnedLegacyStringCommand(t *testing.T) {
	settings := settingsFromJSON(t, `{"hooks": {"Stop": "ward end", "Notification": "other-tool"}}`)

	Uninstall(&settings, CommandIdentity("ward"))

	assertSettingsJSON(t, settings, `{"hooks": {"Notification": "other-tool"}}`)
}

func TestUninstallLeavesUnknownShapesAlone(t *testing.T) {
	doc := `{"hooks": {"Stop": 42, "Notification": {"command": "ward end"}}}`
	settings := settingsFromJSON(t, doc)

	Uninstall(&settings, CommandIdentity("ward"))

	assertSettingsJSON(t, settings, doc)
}

func TestInstallKeepsEverySpecForTheSameEvent(t *testing.T) {
	settings := make(SettingsMap)
	specs := []HookSpec{
		{Event: "PreToolUse", Matcher: "Bash", Command: "ward eval-bash"},
		{Event: "PreToolUse", Matcher: "Edit", Command: "ward eval-edit"},
	}
	isWard := CommandIdentity("ward")

	for i := 0; i < 2; i++ {
		if err := Install(&settings, specs, isWard); err != nil {
			t.Fatal(err)
		}
	}

	assertSettingsJSON(t, settings, `{"hooks": {"PreToolUse": [
		{"matcher": "Bash", "hooks": [{"type": "command", "command": "ward eval-bash"}]},
		{"matcher": "Edit", "hooks": [{"type": "command", "command": "ward eval-edit"}]}
	]}}`)
}
