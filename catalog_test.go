package captainhook

import "testing"

func TestLookupKnowsEveryAgent(t *testing.T) {
	for _, agent := range []Agent{AgentClaude, AgentCodex, AgentGemini, AgentQwen, AgentCopilot, AgentCommandCode} {
		hooks, ok := Lookup(agent)
		if !ok {
			t.Fatalf("Lookup(%q) not found", agent)
		}
		if hooks.Agent != agent {
			t.Errorf("Lookup(%q).Agent = %q", agent, hooks.Agent)
		}
		if len(hooks.Events) == 0 {
			t.Errorf("Lookup(%q) has no events", agent)
		}
		if hooks.Source == "" {
			t.Errorf("Lookup(%q) has no Source", agent)
		}
	}
}

func TestLookupRejectsUnknownAgent(t *testing.T) {
	if _, ok := Lookup("opencode"); ok {
		t.Fatal("OpenCode has no settings hooks and must not be in the catalog")
	}
}

func TestOnlyCopilotUsesTheFlatLayout(t *testing.T) {
	for _, hooks := range Agents() {
		if want := hooks.Agent == AgentCopilot; hooks.Flat != want {
			t.Errorf("%s: Flat = %v, want %v", hooks.Agent, hooks.Flat, want)
		}
	}
}

func TestEventKeysAreUniquePerAgent(t *testing.T) {
	for _, hooks := range Agents() {
		seen := map[string]bool{}
		for _, event := range hooks.Events {
			for _, key := range []string{event.Name, event.PascalName} {
				if key == "" {
					continue
				}
				if seen[key] {
					t.Errorf("%s: duplicate event key %q", hooks.Agent, key)
				}
				seen[key] = true
			}
		}
	}
}

func TestKeyPrefersPascalName(t *testing.T) {
	if got := (Event{Name: "agentStop", PascalName: "Stop"}).Key(); got != "Stop" {
		t.Errorf("Key() = %q, want Stop", got)
	}
	if got := (Event{Name: "subagentStart"}).Key(); got != "subagentStart" {
		t.Errorf("Key() = %q, want subagentStart", got)
	}
}

func TestSupportsMatchesEitherSpelling(t *testing.T) {
	copilot, _ := Lookup(AgentCopilot)
	for _, key := range []string{"agentStop", "Stop", "notification", "userPromptTransformed"} {
		if !copilot.Supports(key) {
			t.Errorf("copilot should support %q", key)
		}
	}
	if copilot.Supports("Notification") {
		t.Error("copilot documents only lowercase notification")
	}
}

func TestCatalogHasEventsAddedUpstream(t *testing.T) {
	cases := map[Agent][]string{
		AgentClaude:  {"DirectoryAdded", "PreModelSwitch", "PostModelSwitch"},
		AgentCodex:   {"SessionEnd", "Interrupt"},
		AgentQwen:    {"PostToolBatch", "UserPromptExpansion", "MessageDisplay", "SessionDelete", "PermissionDenied", "InstructionsLoaded"},
		AgentCopilot: {"userPromptTransformed"},
	}
	for agent, keys := range cases {
		hooks, _ := Lookup(agent)
		for _, key := range keys {
			if !hooks.Supports(key) {
				t.Errorf("%s should support %q", agent, key)
			}
		}
	}
}

func TestAgentsReturnsACopy(t *testing.T) {
	Agents()[0].Events[0].Name = "Mutated"
	if Agents()[0].Events[0].Name == "Mutated" {
		t.Fatal("Agents exposed the catalog's backing arrays")
	}
	hooks, _ := Lookup(AgentClaude)
	hooks.Events[0].Name = "Mutated"
	again, _ := Lookup(AgentClaude)
	if again.Events[0].Name == "Mutated" {
		t.Fatal("Lookup exposed the catalog's backing array")
	}
}
