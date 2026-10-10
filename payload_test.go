package captainhook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// fixture is one recorded hook invocation: the settings key the hook was
// registered under and the JSON the agent wrote to its stdin.
type fixture struct {
	Name    string          `json:"name"`
	Key     string          `json:"key"`
	Payload json.RawMessage `json:"payload"`
}

// fixtureSets maps each testdata/<agent>/<version> directory to its agent.
// Each directory's SOURCE.md says where its payloads came from.
var fixtureSets = map[string]Agent{
	"claude/2.1.290":        AgentClaude,
	"codex/0.161.0-alpha.2": AgentCodex,
	"copilot/1.0.91":        AgentCopilot,
	"qwen/0.25.0":           AgentQwen,
	"gemini/0.62.0":         AgentGemini,
	"commandcode/1.73.4":    AgentCommandCode,
}

func loadFixtures(t *testing.T, set string) []fixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filepath.FromSlash(set), "payloads.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []fixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("%s: %v", set, err)
	}
	return fixtures
}

// wantPayload is what Parse must make of one fixture. exit is the expected
// ExitCode, or noExit when the payload carries none.
type wantPayload struct {
	event   string
	tool    string
	rawTool string
	phase   Phase
	outcome Outcome
	exit    int
	errText string
	prompt  string
	message string
}

const noExit = -1 << 31

var wantPayloads = map[string]wantPayload{
	// Claude Code reports a failed tool as PostToolUseFailure, so its
	// PostToolUse is a success whatever the response looks like.
	"claude/2.1.290/PreToolUse-powershell":               {event: "PreToolUse", tool: "Bash", rawTool: "PowerShell", phase: PhaseBefore, exit: noExit},
	"claude/2.1.290/PostToolUse-powershell":              {event: "PostToolUse", tool: "Bash", rawTool: "PowerShell", phase: PhaseAfter, outcome: OutcomeSuccess, exit: noExit},
	"claude/2.1.290/PostToolUseFailure-powershell-exit3": {event: "PostToolUseFailure", tool: "Bash", rawTool: "PowerShell", phase: PhaseAfter, outcome: OutcomeFailure, exit: 3, errText: "Exit code 3"},
	"claude/2.1.290/PostToolUse-read":                    {event: "PostToolUse", tool: "Read", rawTool: "Read", phase: PhaseAfter, outcome: OutcomeSuccess, exit: noExit},
	"claude/2.1.290/UserPromptSubmit":                    {event: "UserPromptSubmit", exit: noExit, prompt: "Do the steps."},
	"claude/2.1.290/Stop":                                {event: "Stop", exit: noExit},
	"claude/2.1.290/SessionStart":                        {event: "SessionStart", exit: noExit},

	// Codex gives hooks a shell command's output but not its exit code, so
	// the outcome of a shell command is unknown, even for one that exited 3.
	"codex/0.161.0-alpha.2/PreToolUse-bash":         {event: "PreToolUse", tool: "Bash", rawTool: "Bash", phase: PhaseBefore, exit: noExit},
	"codex/0.161.0-alpha.2/PostToolUse-bash-output": {event: "PostToolUse", tool: "Bash", rawTool: "Bash", phase: PhaseAfter, outcome: OutcomeUnknown, exit: noExit},
	"codex/0.161.0-alpha.2/PostToolUse-bash-exit3":  {event: "PostToolUse", tool: "Bash", rawTool: "Bash", phase: PhaseAfter, outcome: OutcomeUnknown, exit: noExit},
	"codex/0.161.0-alpha.2/Stop":                    {event: "Stop", exit: noExit},
	"codex/0.161.0-alpha.2/SessionStart":            {event: "SessionStart", exit: noExit},

	// Copilot: Event is the catalog key whichever spelling the hook was
	// registered under. A shell command that exits nonzero still reports
	// result_type "success"; the exit code is in the result text.
	"copilot/1.0.91/PascalCase-PreToolUse-bash":          {event: "PreToolUse", tool: "Bash", rawTool: "Bash", phase: PhaseBefore, exit: noExit},
	"copilot/1.0.91/PascalCase-PostToolUse-bash-exit0":   {event: "PostToolUse", tool: "Bash", rawTool: "Bash", phase: PhaseAfter, outcome: OutcomeSuccess, exit: 0},
	"copilot/1.0.91/PascalCase-PostToolUse-bash-exit3":   {event: "PostToolUse", tool: "Bash", rawTool: "Bash", phase: PhaseAfter, outcome: OutcomeFailure, exit: 3},
	"copilot/1.0.91/PascalCase-PostToolUse-read":         {event: "PostToolUse", tool: "Read", rawTool: "Read", phase: PhaseAfter, outcome: OutcomeSuccess, exit: noExit},
	"copilot/1.0.91/PascalCase-PostToolUseFailure-write": {event: "PostToolUseFailure", tool: "Write", rawTool: "Write", phase: PhaseAfter, outcome: OutcomeFailure, exit: noExit, errText: "Parent directory does not exist"},
	"copilot/1.0.91/PascalCase-Stop":                     {event: "Stop", exit: noExit},
	"copilot/1.0.91/PascalCase-PermissionRequest":        {event: "PermissionRequest", tool: "Bash", rawTool: "powershell", exit: noExit},
	"copilot/1.0.91/camelCase-preToolUse-powershell":     {event: "PreToolUse", tool: "Bash", rawTool: "powershell", phase: PhaseBefore, exit: noExit},
	"copilot/1.0.91/camelCase-postToolUse-exit3":         {event: "PostToolUse", tool: "Bash", rawTool: "powershell", phase: PhaseAfter, outcome: OutcomeFailure, exit: 3},
	"copilot/1.0.91/camelCase-postToolUse-view":          {event: "PostToolUse", tool: "Read", rawTool: "view", phase: PhaseAfter, outcome: OutcomeSuccess, exit: noExit},
	"copilot/1.0.91/camelCase-postToolUseFailure-create": {event: "PostToolUseFailure", tool: "Write", rawTool: "create", phase: PhaseAfter, outcome: OutcomeFailure, exit: noExit, errText: "Parent directory does not exist"},
	"copilot/1.0.91/camelCase-agentStop":                 {event: "Stop", exit: noExit},
	"copilot/1.0.91/camelCase-sessionStart":              {event: "SessionStart", exit: noExit},
	"copilot/1.0.91/camelCase-userPromptTransformed":     {event: "userPromptTransformed", exit: noExit, prompt: "Do the steps."},

	"qwen/0.25.0/SessionStart":              {event: "SessionStart", exit: noExit},
	"qwen/0.25.0/Notification-auth-success": {event: "Notification", exit: noExit, message: "Successfully authenticated with openai"},
	"qwen/0.25.0/UserPromptSubmit":          {event: "UserPromptSubmit", exit: noExit, prompt: "Do the steps."},

	// Gemini reports a nonzero shell exit only as an "Exit Code: N" line in
	// llmContent; error is set only when the shell could not start.
	"gemini/0.62.0/BeforeTool-shell":                {event: "BeforeTool", tool: "Bash", rawTool: "run_shell_command", phase: PhaseBefore, exit: noExit},
	"gemini/0.62.0/AfterTool-shell-exit0":           {event: "AfterTool", tool: "Bash", rawTool: "run_shell_command", phase: PhaseAfter, outcome: OutcomeSuccess, exit: noExit},
	"gemini/0.62.0/AfterTool-shell-exit128":         {event: "AfterTool", tool: "Bash", rawTool: "run_shell_command", phase: PhaseAfter, outcome: OutcomeFailure, exit: 128},
	"gemini/0.62.0/AfterTool-shell-failed-to-start": {event: "AfterTool", tool: "Bash", rawTool: "run_shell_command", phase: PhaseAfter, outcome: OutcomeFailure, exit: noExit, errText: "spawn ENOENT"},

	// Command Code starts a shell result with "Exit code: N" unless the
	// exit counts as a success, so a result without that line succeeded.
	"commandcode/1.73.4/PreToolUse-shell":          {event: "PreToolUse", tool: "Bash", rawTool: "shell_command", phase: PhaseBefore, exit: noExit},
	"commandcode/1.73.4/PostToolUse-shell-output":  {event: "PostToolUse", tool: "Bash", rawTool: "shell_command", phase: PhaseAfter, outcome: OutcomeSuccess, exit: noExit},
	"commandcode/1.73.4/PostToolUse-shell-exit128": {event: "PostToolUse", tool: "Bash", rawTool: "shell_command", phase: PhaseAfter, outcome: OutcomeFailure, exit: 128},
}

func checkPayload(t *testing.T, got *Payload, want wantPayload) {
	t.Helper()
	if got.Event != want.event {
		t.Errorf("Event = %q, want %q", got.Event, want.event)
	}
	if got.Prompt != want.prompt {
		t.Errorf("Prompt = %q, want %q", got.Prompt, want.prompt)
	}
	if got.Message != want.message {
		t.Errorf("Message = %q, want %q", got.Message, want.message)
	}
	if want.tool == "" {
		if got.Tool != nil {
			t.Errorf("Tool = %+v, want none", *got.Tool)
		}
		return
	}
	if got.Tool == nil {
		t.Fatalf("Tool = nil, want %s", want.tool)
	}
	tool := got.Tool
	if tool.Name != want.tool || tool.RawName != want.rawTool {
		t.Errorf("tool Name, RawName = %q, %q, want %q, %q", tool.Name, tool.RawName, want.tool, want.rawTool)
	}
	if tool.Phase != want.phase {
		t.Errorf("Phase = %s, want %s", tool.Phase, want.phase)
	}
	if tool.Outcome != want.outcome {
		t.Errorf("Outcome = %s, want %s", tool.Outcome, want.outcome)
	}
	switch {
	case want.exit == noExit && tool.ExitCode != nil:
		t.Errorf("ExitCode = %d, want none", *tool.ExitCode)
	case want.exit != noExit && tool.ExitCode == nil:
		t.Errorf("ExitCode = none, want %d", want.exit)
	case want.exit != noExit && *tool.ExitCode != want.exit:
		t.Errorf("ExitCode = %d, want %d", *tool.ExitCode, want.exit)
	}
	if tool.Error != want.errText {
		t.Errorf("Error = %q, want %q", tool.Error, want.errText)
	}
}

func TestParseFixtures(t *testing.T) {
	seen := map[string]bool{}
	for set, agent := range fixtureSets {
		for _, fx := range loadFixtures(t, set) {
			id := set + "/" + fx.Name
			seen[id] = true
			t.Run(id, func(t *testing.T) {
				want, ok := wantPayloads[id]
				if !ok {
					t.Fatalf("fixture %s has no expectation in wantPayloads", id)
				}
				got, err := Parse(agent, fx.Key, fx.Payload)
				if err != nil {
					t.Fatalf("Parse: %v", err)
				}
				if got.Agent != agent {
					t.Errorf("Agent = %q, want %q", got.Agent, agent)
				}
				if got.SessionID != "00000000-0000-4000-8000-000000000000" {
					t.Errorf("SessionID = %q", got.SessionID)
				}
				if got.CWD != `C:\proj` {
					t.Errorf("CWD = %q", got.CWD)
				}
				checkPayload(t, got, want)
			})
		}
	}
	for id := range wantPayloads {
		if !seen[id] {
			t.Errorf("wantPayloads names %s, which is not a fixture", id)
		}
	}
}

// TestParseFindsTheAgent covers payloads that arrive with no agent named:
// Parse works the agent out when the payload gives it away, and otherwise
// leaves Agent empty and reads the payload by its shape alone.
func TestParseFindsTheAgent(t *testing.T) {
	tests := []struct {
		id    string
		agent Agent
		want  Outcome
	}{
		// turn_id is Codex's; its shell outcome stays unknown.
		{"codex/0.161.0-alpha.2/PostToolUse-bash-exit3", AgentCodex, OutcomeUnknown},
		// camelCase fields and tool_result are Copilot's.
		{"copilot/1.0.91/camelCase-postToolUse-exit3", AgentCopilot, OutcomeFailure},
		{"copilot/1.0.91/PascalCase-PostToolUse-bash-exit3", AgentCopilot, OutcomeFailure},
		{"copilot/1.0.91/PascalCase-PermissionRequest", AgentCopilot, OutcomeNone},
		// tool_display_name is Command Code's.
		{"commandcode/1.73.4/PostToolUse-shell-output", AgentCommandCode, OutcomeSuccess},
		// AfterTool is an event only Gemini fires.
		{"gemini/0.62.0/AfterTool-shell-exit128", AgentGemini, OutcomeFailure},
		// Nothing in these says which agent sent them.
		{"claude/2.1.290/PostToolUse-read", "", OutcomeSuccess},
		{"claude/2.1.290/PostToolUseFailure-powershell-exit3", "", OutcomeFailure},
	}
	byID := map[string]fixture{}
	for set := range fixtureSets {
		for _, fx := range loadFixtures(t, set) {
			byID[set+"/"+fx.Name] = fx
		}
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			fx, ok := byID[tt.id]
			if !ok {
				t.Fatalf("no fixture %s", tt.id)
			}
			got, err := Parse("", fx.Key, fx.Payload)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got.Agent != tt.agent {
				t.Errorf("Agent = %q, want %q", got.Agent, tt.agent)
			}
			if got.Tool == nil {
				t.Fatal("Tool = nil")
			}
			if got.Tool.Outcome != tt.want {
				t.Errorf("Outcome = %s, want %s", got.Tool.Outcome, tt.want)
			}
		})
	}
}

// TestParseNamesEveryCatalogEvent checks that an event arrives as its
// catalog key under either spelling, whether the payload names the event
// or only the settings key does.
func TestParseNamesEveryCatalogEvent(t *testing.T) {
	for _, hooks := range Agents() {
		for _, event := range hooks.Events {
			for _, spelling := range []string{event.Name, event.PascalName} {
				if spelling == "" {
					continue
				}
				t.Run(string(hooks.Agent)+"/"+spelling, func(t *testing.T) {
					named, err := Parse(hooks.Agent, "", []byte(`{"hook_event_name":"`+spelling+`"}`))
					if err != nil {
						t.Fatalf("Parse with the event in the payload: %v", err)
					}
					if named.Event != event.Key() {
						t.Errorf("payload event %q: Event = %q, want %q", spelling, named.Event, event.Key())
					}
					keyed, err := Parse(hooks.Agent, spelling, []byte(`{}`))
					if err != nil {
						t.Fatalf("Parse with the event as the settings key: %v", err)
					}
					if keyed.Event != event.Key() {
						t.Errorf("settings key %q: Event = %q, want %q", spelling, keyed.Event, event.Key())
					}
				})
			}
		}
	}
}

func TestParseRejectsWhatItCannotRead(t *testing.T) {
	for name, data := range map[string]string{
		"empty":         ``,
		"not JSON":      `hello`,
		"not an object": `["PreToolUse"]`,
		"no event name": `{"session_id":"s","cwd":"/c"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := Parse(AgentClaude, "", []byte(data)); err == nil {
				t.Errorf("Parse = %+v, want an error", got)
			}
		})
	}
}

func TestParseKeepsUnknownEventNames(t *testing.T) {
	got, err := Parse(AgentClaude, "", []byte(`{"hook_event_name":"SomethingNew"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Event != "SomethingNew" {
		t.Errorf("Event = %q, want SomethingNew", got.Event)
	}
}

func TestParseToolInput(t *testing.T) {
	tests := []struct {
		name, payload, want string
	}{
		{"tool_input object", `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"ls"}}`, `{"command":"ls"}`},
		{"Copilot toolArgs", `{"toolName":"powershell","toolArgs":{"command":"ls"}}`, `{"command":"ls"}`},
		{"Copilot toolInput", `{"hookName":"permissionRequest","toolName":"powershell","toolInput":{"command":"ls"}}`, `{"command":"ls"}`},
		{"JSON object sent as a string", `{"toolName":"powershell","toolArgs":"{\"command\":\"ls\"}"}`, `{"command":"ls"}`},
		{"a string that is not JSON stays a string", `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":"ls -la"}`, `"ls -la"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse("", "preToolUse", []byte(tt.payload))
			if err != nil {
				t.Fatal(err)
			}
			if got.Tool == nil {
				t.Fatal("Tool = nil")
			}
			if string(got.Tool.Input) != tt.want {
				t.Errorf("Input = %s, want %s", got.Tool.Input, tt.want)
			}
		})
	}
}

// TestParseToolNames: a tool is named by its Claude Code tool name when one
// exists, following the table GitHub publishes for Copilot's own tools; a
// tool with no Claude equivalent keeps the name its agent gave it.
func TestParseToolNames(t *testing.T) {
	tests := []struct {
		raw, want, server, tool string
	}{
		{"Bash", "Bash", "", ""},
		{"PowerShell", "Bash", "", ""},
		{"run_shell_command", "Bash", "", ""}, // Gemini, Qwen
		{"shell_command", "Bash", "", ""},     // Command Code
		{"view", "Read", "", ""},              // Copilot
		{"read_file", "Read", "", ""},
		{"create", "Write", "", ""},
		{"write_file", "Write", "", ""},
		{"replace", "Edit", "", ""},            // Gemini
		{"apply_patch", "Edit", "", ""},        // Codex, Copilot
		{"str_replace_editor", "Edit", "", ""}, // Copilot
		{"edit_file", "Edit", "", ""},          // Command Code
		{"rg", "Grep", "", ""},
		{"grep_search", "Grep", "", ""},
		{"glob", "Glob", "", ""},
		{"web_fetch", "WebFetch", "", ""},
		{"google_web_search", "WebSearch", "", ""},
		{"update_todo", "TodoWrite", "", ""},
		{"write_todos", "TodoWrite", "", ""},
		{"ask_user", "AskUserQuestion", "", ""},
		{"task", "Agent", "", ""},
		{"spawn_agent", "Agent", "", ""}, // Codex
		{"NotebookEdit", "NotebookEdit", "", ""},
		{"list_directory", "list_directory", "", ""},
		{"mcp__github__create_issue", "MCP", "github", "create_issue"},
		// Gemini joins server and tool with one underscore, which cannot be
		// split again: either name may contain underscores.
		{"mcp_filesystem_read_file", "MCP", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := Parse("", "", []byte(`{"hook_event_name":"PreToolUse","tool_name":"`+tt.raw+`"}`))
			if err != nil {
				t.Fatal(err)
			}
			if got.Tool == nil {
				t.Fatal("Tool = nil")
			}
			if got.Tool.Name != tt.want || got.Tool.RawName != tt.raw {
				t.Errorf("Name, RawName = %q, %q, want %q, %q", got.Tool.Name, got.Tool.RawName, tt.want, tt.raw)
			}
			var server, tool string
			if got.Tool.MCP != nil {
				server, tool = got.Tool.MCP.Server, got.Tool.MCP.Tool
			}
			if server != tt.server || tool != tt.tool {
				t.Errorf("MCP server, tool = %q, %q, want %q, %q", server, tool, tt.server, tt.tool)
			}
		})
	}
}

// TestParseToolResponse: Response is the tool's result exactly as the agent
// sent it, under whichever field name that agent uses.
func TestParseToolResponse(t *testing.T) {
	tests := []struct {
		name, payload, want string
	}{
		{"tool_response object", `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_response":{"stdout":"hi"}}`, `{"stdout":"hi"}`},
		{"tool_response string", `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_response":"hi\n"}`, `"hi\n"`},
		{"Copilot tool_result", `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_result":{"result_type":"success"}}`, `{"result_type":"success"}`},
		{"Copilot toolResult", `{"toolName":"powershell","toolResult":{"resultType":"success"}}`, `{"resultType":"success"}`},
		{"no response", `{"hook_event_name":"PreToolUse","tool_name":"Bash"}`, ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse("", "postToolUse", []byte(tt.payload))
			if err != nil {
				t.Fatal(err)
			}
			if got.Tool == nil {
				t.Fatal("Tool = nil")
			}
			if string(got.Tool.Response) != tt.want {
				t.Errorf("Response = %s, want %s", got.Tool.Response, tt.want)
			}
		})
	}
}

func TestParseInterruptedTools(t *testing.T) {
	tests := []struct {
		name, payload string
	}{
		{"failure event with is_interrupt", `{"hook_event_name":"PostToolUseFailure","tool_name":"Bash","error":"interrupted","is_interrupt":true}`},
		{"response with interrupted", `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_response":{"stdout":"","stderr":"","interrupted":true}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(AgentClaude, "", []byte(tt.payload))
			if err != nil {
				t.Fatal(err)
			}
			if got.Tool == nil || got.Tool.Outcome != OutcomeInterrupted {
				t.Errorf("Tool = %+v, want an interrupted outcome", got.Tool)
			}
		})
	}
}
