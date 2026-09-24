package captainhook

import (
	"reflect"
	"testing"
)

func TestOwnedEventsListsEveryEventHoldingOurCommands(t *testing.T) {
	settings := settingsFromJSON(t, `{"hooks": {
		"Stop": "claudio",
		"sessionStart": [{"type": "command", "command": "claudio --hook-agent copilot"}],
		"PreToolUse": [{"matcher": ".*", "hooks": [
			{"type": "command", "command": "other"},
			{"type": "command", "command": "\"C:\\Program Files\\claudio.exe\""}
		]}],
		"Notification": [{"hooks": [{"type": "command", "command": "other"}]}],
		"Weird": 42
	}}`)
	got := OwnedEvents(&settings, isClaudio)

	want := []string{"PreToolUse", "Stop", "sessionStart"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("OwnedEvents = %v, want %v", got, want)
	}
}

func TestOwnedEventsWithoutHooksIsEmpty(t *testing.T) {
	for _, doc := range []string{`{}`, `{"hooks": "nope"}`, `{"hooks": {}}`} {
		settings := settingsFromJSON(t, doc)
		if got := OwnedEvents(&settings, isClaudio); len(got) != 0 {
			t.Errorf("OwnedEvents(%s) = %v, want none", doc, got)
		}
	}
}
