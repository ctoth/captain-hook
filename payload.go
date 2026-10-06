package captainhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Phase says where in a tool call a tool event fired.
type Phase int

const (
	// PhaseNone is an event about a tool that is neither its start nor its
	// end, such as a permission request.
	PhaseNone Phase = iota
	PhaseBefore
	PhaseAfter
)

func (p Phase) String() string {
	switch p {
	case PhaseBefore:
		return "before"
	case PhaseAfter:
		return "after"
	default:
		return "none"
	}
}

// Outcome is how a tool call ended, as far as the payload says.
type Outcome int

const (
	// OutcomeNone means the event is not the end of a tool call.
	OutcomeNone Outcome = iota
	OutcomeSuccess
	OutcomeFailure
	OutcomeInterrupted
	// OutcomeUnknown means the tool call ended and the payload does not
	// say how. Codex, for one, gives hooks a shell command's output but
	// not its exit code.
	OutcomeUnknown
)

func (o Outcome) String() string {
	switch o {
	case OutcomeSuccess:
		return "success"
	case OutcomeFailure:
		return "failure"
	case OutcomeInterrupted:
		return "interrupted"
	case OutcomeUnknown:
		return "unknown"
	default:
		return "none"
	}
}

// Payload is one hook invocation, read from what the agent wrote to the
// hook's stdin.
type Payload struct {
	// Agent is the agent Parse was told sent the payload or, when it was
	// told nothing, the agent it recognized. Empty when neither.
	Agent Agent
	// Event is the event's key in the agent's catalog (Event.Key), whichever
	// spelling the payload or the settings key used. An event the catalog
	// does not list keeps the name it arrived with.
	Event          string
	SessionID      string
	TranscriptPath string
	CWD            string
	// Prompt is the user's prompt, on prompt events.
	Prompt string
	// Message is the notification text, on notification events.
	Message string
	// Tool is the tool the event is about, or nil.
	Tool *ToolCall
	// Raw is the payload as it arrived.
	Raw json.RawMessage
}

// ToolCall is the tool a tool event is about.
type ToolCall struct {
	// Name is the Claude Code name for the tool when it has one ("Bash" for
	// any shell, "Read", "Edit", ...), "MCP" for an MCP tool, and otherwise
	// RawName.
	Name string
	// RawName is the tool's name as the agent sent it.
	RawName string
	// MCP holds the server and tool of an MCP tool whose name can be split.
	MCP *MCPTool
	// Input is the tool's arguments. An agent that sends them as a string
	// of JSON gets them decoded, so Input is an object whenever the agent
	// sent one in either form.
	Input   json.RawMessage
	Phase   Phase
	Outcome Outcome
	// ExitCode is the shell exit code, when the payload carries one.
	ExitCode *int
	// Error is the agent's error text for a failed call.
	Error string
}

// MCPTool names an MCP tool by its server and its name on that server.
type MCPTool struct {
	Server string
	Tool   string
}

// Parse reads one hook payload.
//
// agent is the agent that sent it. Pass "" when that is not known: Parse
// then recognizes the agent where the payload gives it away and otherwise
// reads the payload by its shape alone.
//
// eventKey is the settings key the hook was registered under. It names the
// event for payloads that do not: Copilot CLI's camelCase payloads carry no
// event name. Pass "" when the payload is known to name its event.
func Parse(agent Agent, eventKey string, data []byte) (*Payload, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("hook payload is not a JSON object: %w", err)
	}
	if fields == nil {
		return nil, errors.New("hook payload is not a JSON object")
	}

	named := stringField(fields, "hook_event_name", "hookName")
	if agent == "" {
		agent = recognizeAgent(fields, named)
	}
	name := named
	if name == "" {
		name = eventKey
	}
	if name == "" {
		return nil, errors.New("hook payload does not name its event and no settings key was given")
	}

	payload := &Payload{
		Agent:          agent,
		Event:          catalogKey(agent, name),
		SessionID:      stringField(fields, "session_id", "sessionId"),
		TranscriptPath: stringField(fields, "transcript_path", "transcriptPath"),
		CWD:            stringField(fields, "cwd"),
		Prompt:         stringField(fields, "prompt"),
		Message:        stringField(fields, "message"),
		Raw:            append(json.RawMessage(nil), data...),
	}
	payload.Tool = toolCall(agent, payload.Event, fields)
	return payload, nil
}

// recognizeAgent names the agent a payload came from when the payload has
// a mark only that agent leaves, and returns "" otherwise. named is the
// event name the payload carries, if any.
func recognizeAgent(fields map[string]json.RawMessage, named string) Agent {
	for _, key := range []string{"sessionId", "toolName", "hookName", "tool_result"} {
		if _, ok := fields[key]; ok {
			return AgentCopilot
		}
	}
	if _, ok := fields["turn_id"]; ok {
		return AgentCodex
	}
	if _, ok := fields["tool_display_name"]; ok {
		return AgentCommandCode
	}
	if named == "" {
		return ""
	}
	// An event only one agent fires gives that agent away.
	var only Agent
	for _, hooks := range catalog {
		if hooks.Supports(named) {
			if only != "" {
				return ""
			}
			only = hooks.Agent
		}
	}
	return only
}

// catalogKey returns the catalog key of the agent's event called name under
// either spelling, or name when the agent or the event is not in the catalog.
func catalogKey(agent Agent, name string) string {
	for _, hooks := range catalog {
		if hooks.Agent != agent {
			continue
		}
		for _, event := range hooks.Events {
			if event.Name == name || (event.PascalName != "" && event.PascalName == name) {
				return event.Key()
			}
		}
	}
	return name
}

// toolPhases is the phase of each tool event. The camelCase names are
// Copilot CLI's, for payloads whose agent was not recognized.
var toolPhases = map[string]Phase{
	"PreToolUse":         PhaseBefore,
	"preToolUse":         PhaseBefore,
	"BeforeTool":         PhaseBefore, // Gemini
	"PostToolUse":        PhaseAfter,
	"postToolUse":        PhaseAfter,
	"AfterTool":          PhaseAfter, // Gemini
	"PostToolUseFailure": PhaseAfter,
	"postToolUseFailure": PhaseAfter,
}

// toolCall returns the tool the event is about, or nil when the payload
// names none.
func toolCall(agent Agent, event string, fields map[string]json.RawMessage) *ToolCall {
	rawName := stringField(fields, "tool_name", "toolName")
	if rawName == "" {
		return nil
	}
	name, mcp := canonicalTool(rawName)
	call := &ToolCall{
		Name:    name,
		RawName: rawName,
		MCP:     mcp,
		Input:   toolInput(fields),
		Phase:   toolPhases[event],
	}
	if call.Phase == PhaseAfter {
		failed := event == "PostToolUseFailure" || event == "postToolUseFailure"
		call.Outcome, call.ExitCode, call.Error = toolOutcome(agent, name == "Bash", failed, fields)
	}
	return call
}

// toolInput returns the tool's arguments. Copilot CLI sends them as
// toolArgs (toolInput on a permission request), documented as a string of
// JSON and sent by 1.0.91 as an object; both become the object.
func toolInput(fields map[string]json.RawMessage) json.RawMessage {
	for _, key := range []string{"tool_input", "toolArgs", "toolInput"} {
		raw, ok := fields[key]
		if !ok {
			continue
		}
		var text string
		if json.Unmarshal(raw, &text) == nil {
			if inner := strings.TrimSpace(text); strings.HasPrefix(inner, "{") && json.Valid([]byte(inner)) {
				return json.RawMessage(inner)
			}
		}
		return raw
	}
	return nil
}

// reportsFailuresSeparately is true for the agents that send a failed tool
// call as its own event (PostToolUseFailure), so that their PostToolUse
// means the tool worked.
func reportsFailuresSeparately(agent Agent) bool {
	return agent == AgentClaude || agent == AgentQwen || agent == AgentCopilot
}

// toolOutcome works out how a finished tool call ended. shell says the tool
// is the shell; failed says the event is the agent's failure event.
func toolOutcome(agent Agent, shell, failed bool, fields map[string]json.RawMessage) (Outcome, *int, string) {
	if failed {
		errText := stringField(fields, "error")
		exit := firstLineExitCode(errText)
		if boolField(fields, "is_interrupt") {
			return OutcomeInterrupted, exit, errText
		}
		return OutcomeFailure, exit, errText
	}

	var response json.RawMessage
	for _, key := range []string{"tool_response", "tool_result", "toolResult"} {
		if raw, ok := fields[key]; ok {
			response = raw
			break
		}
	}
	var text string
	var object map[string]json.RawMessage
	switch {
	case len(response) == 0 || string(response) == "null":
		// No response to read.
	case json.Unmarshal(response, &text) == nil:
		return textOutcome(agent, shell, text)
	case json.Unmarshal(response, &object) == nil:
		return objectOutcome(object)
	}
	// Nothing readable (no response, or an array): the event itself is the
	// only evidence.
	if reportsFailuresSeparately(agent) {
		return OutcomeSuccess, nil, ""
	}
	return OutcomeUnknown, nil, ""
}

// textOutcome reads a tool response that is plain text.
func textOutcome(agent Agent, shell bool, text string) (Outcome, *int, string) {
	if agent == AgentCodex {
		// Codex fires PostToolUse only for a call it counts as successful,
		// and it counts every shell command as successful: the hook gets
		// the command's output and never its exit code. The output is not
		// searched for an exit line, since any such line is the command's
		// own.
		if shell {
			return OutcomeUnknown, nil, ""
		}
		return OutcomeSuccess, nil, ""
	}
	if exit := firstLineExitCode(text); exit != nil {
		if *exit != 0 {
			return OutcomeFailure, exit, ""
		}
		return OutcomeSuccess, exit, ""
	}
	switch {
	case agent == AgentCommandCode && shell:
		// Command Code leaves the exit line off only when the exit counts
		// as a success for that command.
		return OutcomeSuccess, nil, ""
	case reportsFailuresSeparately(agent):
		return OutcomeSuccess, nil, ""
	default:
		return OutcomeUnknown, nil, ""
	}
}

// copilotShellExit matches the line Copilot CLI ends a shell result with,
// such as "<shellId: 1 completed with exit code 3>".
var copilotShellExit = regexp.MustCompile(`<shellId: \S+ completed with exit code (-?\d+)>`)

// geminiExitLine matches the "Exit Code: N" line Gemini CLI puts in a shell
// result's llmContent when N is not zero.
var geminiExitLine = regexp.MustCompile(`(?m)^Exit Code: (-?\d+)\s*$`)

// objectOutcome reads a tool response that is a JSON object. An object with
// no sign of failure is a success.
func objectOutcome(response map[string]json.RawMessage) (Outcome, *int, string) {
	if boolField(response, "interrupted") {
		return OutcomeInterrupted, nil, ""
	}

	// Copilot CLI: {result_type, text_result_for_llm}, camelCase under a
	// camelCase settings key. A shell command that exits nonzero still has
	// result_type "success"; its exit code is in the text.
	if resultType := stringField(response, "result_type", "resultType"); resultType != "" {
		exit := submatchInt(copilotShellExit.FindStringSubmatch(stringField(response, "text_result_for_llm", "textResultForLlm")))
		if resultType != "success" || (exit != nil && *exit != 0) {
			return OutcomeFailure, exit, ""
		}
		return OutcomeSuccess, exit, ""
	}

	// Gemini CLI sets error to {message, type} when a tool fails; other
	// agents use a plain string.
	if raw, ok := response["error"]; ok && string(raw) != "null" {
		var message string
		var detail map[string]json.RawMessage
		switch {
		case json.Unmarshal(raw, &message) == nil:
			if message != "" {
				return OutcomeFailure, nil, message
			}
		case json.Unmarshal(raw, &detail) == nil:
			return OutcomeFailure, nil, stringField(detail, "message")
		default:
			return OutcomeFailure, nil, ""
		}
	}

	// An MCP tool result marks its own failure.
	if boolField(response, "isError") {
		return OutcomeFailure, nil, ""
	}

	// Gemini CLI reports a nonzero shell exit only as a line of llmContent,
	// after the command's output: the last such line is Gemini's.
	if matches := geminiExitLine.FindAllStringSubmatch(stringField(response, "llmContent"), -1); len(matches) > 0 {
		exit := submatchInt(matches[len(matches)-1])
		if exit != nil && *exit != 0 {
			return OutcomeFailure, exit, ""
		}
		return OutcomeSuccess, exit, ""
	}
	return OutcomeSuccess, nil, ""
}

// exitLine matches a first line that gives an exit code: "Exit code 3"
// (Claude Code's error text) or "Exit code: 3" (Command Code).
var exitLine = regexp.MustCompile(`(?i)^exit code:? (-?\d+)`)

// firstLineExitCode returns the exit code text's first line gives, or nil.
// Only the first line counts: later lines are the command's own output.
func firstLineExitCode(text string) *int {
	first, _, _ := strings.Cut(text, "\n")
	return submatchInt(exitLine.FindStringSubmatch(strings.TrimSpace(first)))
}

// submatchInt returns the integer in a regexp match's first group, or nil
// when there was no match.
func submatchInt(match []string) *int {
	if len(match) < 2 {
		return nil
	}
	n, err := strconv.Atoi(match[1])
	if err != nil {
		return nil
	}
	return &n
}

// stringField returns the first of keys that holds a non-empty JSON string.
func stringField(fields map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		var s string
		if raw, ok := fields[key]; ok && json.Unmarshal(raw, &s) == nil && s != "" {
			return s
		}
	}
	return ""
}

// boolField reports whether key holds the JSON value true.
func boolField(fields map[string]json.RawMessage, key string) bool {
	var b bool
	raw, ok := fields[key]
	return ok && json.Unmarshal(raw, &b) == nil && b
}
