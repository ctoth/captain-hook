package captainhook

// Agent names a coding agent whose settings file holds hooks.
type Agent string

const (
	AgentClaude      Agent = "claude"
	AgentCodex       Agent = "codex"
	AgentGemini      Agent = "gemini"
	AgentQwen        Agent = "qwen"
	AgentCopilot     Agent = "copilot"
	AgentCommandCode Agent = "commandcode"
)

// Event is one hook event an agent fires.
type Event struct {
	// Name is the event key as the agent's docs or source spell it.
	Name string
	// PascalName is a second, PascalCase key the agent also accepts, or ""
	// when there is none. Only GitHub Copilot CLI has one: its PascalCase
	// keys send VS Code-compatible snake_case payloads.
	PascalName string
}

// Key returns the settings key to install under: PascalName when the event
// has one, else Name.
func (e Event) Key() string {
	if e.PascalName != "" {
		return e.PascalName
	}
	return e.Name
}

// AgentHooks is the hook surface of one agent.
type AgentHooks struct {
	Agent Agent
	// Flat is true when the agent's settings put command entries directly
	// in each event's array, without matcher groups (see HookSpec.Flat).
	Flat bool
	// Source is where the event list was checked, with the version.
	Source string
	// Events lists every event the agent fires, in its docs' order.
	Events []Event
}

// Supports reports whether key names one of the agent's events, under
// either spelling.
func (h AgentHooks) Supports(key string) bool {
	for _, event := range h.Events {
		if event.Name == key || (event.PascalName != "" && event.PascalName == key) {
			return true
		}
	}
	return false
}

func events(names ...string) []Event {
	out := make([]Event, len(names))
	for i, name := range names {
		out[i] = Event{Name: name}
	}
	return out
}

// catalog lists every agent's hook events. Each list was checked against
// the Source named beside it; update the Source when re-checking.
var catalog = []AgentHooks{
	{
		Agent:  AgentClaude,
		Source: "https://code.claude.com/docs/en/hooks (Claude Code 2.1.287)",
		Events: events(
			"SessionStart", "Setup", "UserPromptSubmit", "UserPromptExpansion",
			"PreToolUse", "PermissionRequest", "PermissionDenied", "PostToolUse",
			"PostToolUseFailure", "PostToolBatch", "Notification", "MessageDisplay",
			"SubagentStart", "SubagentStop", "TaskCreated", "TaskCompleted",
			"Stop", "StopFailure", "TeammateIdle", "InstructionsLoaded",
			"ConfigChange", "CwdChanged", "DirectoryAdded", "FileChanged",
			"WorktreeCreate", "WorktreeRemove", "PreCompact", "PostCompact",
			"PreModelSwitch", "PostModelSwitch", "Elicitation", "ElicitationResult",
			"SessionEnd",
		),
	},
	{
		Agent:  AgentCodex,
		Source: "github.com/openai/codex codex-rs/config/src/hook_config.rs @ rust-v0.160.0",
		Events: events(
			"PreToolUse", "PermissionRequest", "PostToolUse", "PreCompact",
			"PostCompact", "SessionStart", "SessionEnd", "UserPromptSubmit",
			"SubagentStart", "SubagentStop", "Stop", "Interrupt",
		),
	},
	{
		Agent:  AgentGemini,
		Source: "github.com/google-gemini/gemini-cli packages/core/src/hooks/types.ts @ v0.62.0",
		Events: events(
			"BeforeTool", "AfterTool", "BeforeAgent", "Notification", "AfterAgent",
			"SessionStart", "SessionEnd", "PreCompress", "BeforeModel", "AfterModel",
			"BeforeToolSelection",
		),
	},
	{
		Agent:  AgentQwen,
		Source: "github.com/QwenLM/qwen-code packages/core/src/hooks/types.ts @ v0.24.7",
		Events: events(
			"PreToolUse", "PostToolUse", "PostToolUseFailure", "PostToolBatch",
			"Notification", "UserPromptSubmit", "UserPromptExpansion", "SessionStart",
			"Stop", "MessageDisplay", "SubagentStart", "SubagentStop", "PreCompact",
			"PostCompact", "SessionEnd", "SessionDelete", "PermissionRequest",
			"PermissionDenied", "StopFailure", "TodoCreated", "TodoCompleted",
			"InstructionsLoaded",
		),
	},
	{
		Agent:  AgentCopilot,
		Flat:   true,
		Source: "https://docs.github.com/en/copilot/reference/hooks-reference (Copilot CLI 1.0.91)",
		Events: []Event{
			{Name: "sessionStart", PascalName: "SessionStart"},
			{Name: "sessionEnd", PascalName: "SessionEnd"},
			{Name: "userPromptSubmitted", PascalName: "UserPromptSubmit"},
			{Name: "userPromptTransformed"},
			{Name: "preToolUse", PascalName: "PreToolUse"},
			{Name: "postToolUse", PascalName: "PostToolUse"},
			{Name: "postToolUseFailure", PascalName: "PostToolUseFailure"},
			{Name: "agentStop", PascalName: "Stop"},
			{Name: "subagentStart"},
			{Name: "subagentStop", PascalName: "SubagentStop"},
			{Name: "errorOccurred", PascalName: "ErrorOccurred"},
			{Name: "preCompact", PascalName: "PreCompact"},
			{Name: "permissionRequest", PascalName: "PermissionRequest"},
			{Name: "notification"},
		},
	},
	{
		Agent:  AgentCommandCode,
		Source: "https://commandcode.ai/docs/hooks (command-code 1.73.4)",
		Events: events("PreToolUse", "PostToolUse", "Stop", "SessionStart"),
	},
}

// Agents returns every agent in the catalog. The result is a copy.
func Agents() []AgentHooks {
	out := make([]AgentHooks, len(catalog))
	for i, hooks := range catalog {
		out[i] = hooks.clone()
	}
	return out
}

// Lookup returns the hook surface of agent, or false when the catalog does
// not know it.
func Lookup(agent Agent) (AgentHooks, bool) {
	for _, hooks := range catalog {
		if hooks.Agent == agent {
			return hooks.clone(), true
		}
	}
	return AgentHooks{}, false
}

func (h AgentHooks) clone() AgentHooks {
	h.Events = append([]Event(nil), h.Events...)
	return h
}
