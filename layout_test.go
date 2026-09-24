package captainhook

import "testing"

var isClaudio = CommandIdentity("claudio", "claudio.exe")

func TestInstallFlatLayoutIsIdempotentAndKeepsOthers(t *testing.T) {
	settings := settingsFromJSON(t, `{"hooks": {"sessionStart": [
		{"type": "command", "command": "other-tool"},
		{"type": "command", "command": "claudio --old"}
	]}}`)
	specs := []HookSpec{{
		Event:   "sessionStart",
		Command: "claudio --hook-agent copilot",
		Flat:    true,
		Extra:   map[string]interface{}{"timeoutSec": 30},
	}}

	for i := 0; i < 2; i++ {
		if err := Install(&settings, specs, isClaudio); err != nil {
			t.Fatal(err)
		}
	}

	assertSettingsJSON(t, settings, `{"hooks": {"sessionStart": [
		{"type": "command", "command": "other-tool"},
		{"type": "command", "command": "claudio --hook-agent copilot", "timeoutSec": 30}
	]}}`)
}

func TestInstallFlatLayoutConvertsLegacyStringToACommandEntry(t *testing.T) {
	settings := settingsFromJSON(t, `{"hooks": {"sessionStart": "other-tool"}}`)
	specs := []HookSpec{{Event: "sessionStart", Command: "claudio", Flat: true}}

	if err := Install(&settings, specs, isClaudio); err != nil {
		t.Fatal(err)
	}

	assertSettingsJSON(t, settings, `{"hooks": {"sessionStart": [
		{"type": "command", "command": "other-tool"},
		{"type": "command", "command": "claudio"}
	]}}`)
}

func TestUninstallRemovesFlatCommandEntries(t *testing.T) {
	settings := settingsFromJSON(t, `{"hooks": {
		"sessionStart": [
			{"type": "command", "command": "other-tool"},
			{"type": "command", "command": "claudio --hook-agent copilot"}
		],
		"sessionEnd": [{"type": "command", "command": "C:\\bin\\claudio.exe"}]
	}}`)

	Uninstall(&settings, isClaudio)

	assertSettingsJSON(t, settings, `{"hooks": {"sessionStart": [
		{"type": "command", "command": "other-tool"}
	]}}`)
}

func TestInstallWritesExtraEntryFields(t *testing.T) {
	settings := make(SettingsMap)
	specs := []HookSpec{{
		Event:   "BeforeTool",
		Command: "claudio --hook-agent gemini",
		Extra:   map[string]interface{}{"name": "claudio"},
	}}

	if err := Install(&settings, specs, isClaudio); err != nil {
		t.Fatal(err)
	}

	assertSettingsJSON(t, settings, `{"hooks": {"BeforeTool": [
		{"hooks": [{"type": "command", "command": "claudio --hook-agent gemini", "name": "claudio"}]}
	]}}`)
}

func TestInstallRejectsExtraFieldsThatOverrideOwnFields(t *testing.T) {
	for _, key := range []string{"type", "command", "commandWindows", "args", "timeout"} {
		t.Run(key, func(t *testing.T) {
			doc := `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "other"}]}]}}`
			settings := settingsFromJSON(t, doc)
			specs := []HookSpec{{
				Event:   "Stop",
				Command: "claudio",
				Extra:   map[string]interface{}{key: "x"},
			}}

			if err := Install(&settings, specs, isClaudio); err == nil {
				t.Fatalf("Install accepted Extra[%q]; want an error", key)
			}
			assertSettingsJSON(t, settings, doc)
		})
	}
}
