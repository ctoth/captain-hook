package captainhook

import "strings"

// claudeToolNames maps the names agents give their tools to the Claude Code
// tool name for the same job. It follows the table GitHub publishes for
// Copilot CLI's own tools, which also states the rule for everything else:
// a tool with no Claude equivalent keeps its runtime name.
//
// "Bash" is the shell tool, whichever shell it runs.
//
// Sources, by agent:
//   - Copilot CLI: https://docs.github.com/en/copilot/reference/hooks-reference
//     ("Runtime tool" / "Claude tool name" table)
//   - Codex: codex-rs/core/src/tools/hook_names.rs
//   - Gemini CLI: packages/core/src/tools/definitions/base-declarations.ts
//   - Qwen Code: packages/core/src/tools/tool-names.ts
//   - Command Code: the table behind toToolDisplayName in dist/cli.mjs
var claudeToolNames = map[string]string{
	"PowerShell":        "Bash", // Claude Code
	"bash":              "Bash", // Copilot
	"powershell":        "Bash", // Copilot
	"run_shell_command": "Bash", // Gemini, Qwen
	"shell_command":     "Bash", // Command Code

	"view":            "Read", // Copilot
	"read_file":       "Read", // Gemini, Qwen, Command Code
	"read_many_files": "Read", // Gemini

	"create":     "Write", // Copilot
	"write_file": "Write", // Gemini, Qwen, Command Code

	"edit":               "Edit", // Copilot, Qwen
	"str_replace_editor": "Edit", // Copilot
	"apply_patch":        "Edit", // Copilot, Codex
	"replace":            "Edit", // Gemini
	"edit_file":          "Edit", // Command Code

	"grep":        "Grep", // Copilot, Command Code
	"rg":          "Grep", // Copilot
	"grep_search": "Grep", // Gemini, Qwen

	"glob": "Glob", // Copilot, Gemini, Qwen, Command Code

	"web_fetch": "WebFetch", // Copilot, Gemini, Qwen

	"web_search":        "WebSearch", // Copilot, Qwen
	"google_web_search": "WebSearch", // Gemini

	"update_todo": "TodoWrite", // Copilot
	"todo_write":  "TodoWrite", // Qwen
	"write_todos": "TodoWrite", // Gemini

	"ask_user":          "AskUserQuestion", // Copilot
	"ask_user_question": "AskUserQuestion", // Qwen

	"task":        "Agent", // Copilot
	"agent":       "Agent", // Qwen
	"spawn_agent": "Agent", // Codex
}

// mcpToolName is the Name of every MCP tool.
const mcpToolName = "MCP"

// canonicalTool returns the name a tool goes by in a Payload and, for an
// MCP tool whose name can be split, its server and tool.
//
// Claude Code, Codex, Qwen and Command Code name MCP tools
// mcp__<server>__<tool>. Gemini names them mcp_<server>_<tool>, which
// cannot be split again because either part may contain underscores.
func canonicalTool(raw string) (string, *MCPTool) {
	if rest, ok := strings.CutPrefix(raw, "mcp__"); ok {
		if server, tool, found := strings.Cut(rest, "__"); found && server != "" && tool != "" {
			return mcpToolName, &MCPTool{Server: server, Tool: tool}
		}
		return mcpToolName, nil
	}
	if strings.HasPrefix(raw, "mcp_") {
		return mcpToolName, nil
	}
	if name, ok := claudeToolNames[raw]; ok {
		return name, nil
	}
	return raw, nil
}
