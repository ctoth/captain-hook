# captain-hook

Shared hook-settings management for tools that register agent hooks (Claude
Code, Codex, Gemini CLI, Qwen Code, GitHub Copilot CLI). Several tools can
install hooks into the same settings file without stepping on each other.

```sh
go get github.com/ctoth/captain-hook
```

## Use

```go
isOurs := captainhook.CommandIdentity("ward", "ward.exe")

settings, err := captainhook.ReadSettings(path)
if err != nil { ... }

err = captainhook.Install(settings, []captainhook.HookSpec{
	{Event: "PreToolUse", Matcher: "Bash|Edit", Command: "ward eval", Timeout: 5},
	{Event: "SessionEnd", Command: "ward end-session"},
}, isOurs)
if err != nil { ... } // an unknown hook shape; settings are unchanged

err = captainhook.WriteSettings(path, settings) // atomic, keeps permissions
```

`Uninstall(settings, isOurs)` removes every command `isOurs` recognizes.
`OwnedEvents(settings, isOurs)` lists the events that hold one.

## Guarantees

- **Idempotent.** Installing twice gives the same file. Our old commands are
  stripped and the new ones appended; other tools' commands, including
  siblings inside a shared matcher group, are left alone.
- **Never clobbers.** A legacy string command is kept (rewritten as a
  one-command entry). Any other shape it does not understand is an error from
  `Install` and left untouched by `Uninstall`. `Install` validates everything
  before changing anything.
- **Portable identity.** `CommandIdentity` matches the executable's base name,
  quoted or not, with `/` or `\` separators on every OS, so a settings file
  written on Windows is still recognized on Linux.

## Layouts

| HookSpec | Writes |
| --- | --- |
| default | `[{"matcher": m, "hooks": [{"type": "command", "command": c}]}]` |
| `Flat: true` | `[{"type": "command", "command": c}]` (GitHub Copilot CLI) |

`CommandWindows`, `Args` and `Timeout` fill the matching command-entry fields.
`Extra` adds any others, such as `"name"` or `"timeoutSec"`; it may not
override a field captain-hook writes itself.

## Event catalog

`Lookup(agent)` returns which hook events an agent fires, so tools don't each
keep their own list:

```go
hooks, ok := captainhook.Lookup(captainhook.AgentCodex)
if ok && hooks.Supports("SessionEnd") { ... }
for _, event := range hooks.Events {
	spec := captainhook.HookSpec{Event: event.Key(), Flat: hooks.Flat, ...}
}
```

`Agents()` lists every agent. Each entry records the `Source` (docs page or
source file, with version) its event list was checked against. GitHub Copilot
CLI events carry both spellings: `Name` is the camelCase key and `PascalName`
the VS Code-compatible key, which gets snake_case payloads. `Key()` prefers
`PascalName`. OpenCode is not listed: it has plugins, not settings hooks.

Each entry also says where the agent keeps its hooks and how it wants them
written:

```go
path, err := hooks.GlobalSettingsPath()       // ~/.codex/hooks.json, or under $CODEX_HOME
candidates := hooks.ProjectSettingsPaths(".") // in the order the agent prefers
spec := captainhook.HookSpec{Event: "PreToolUse", Matcher: hooks.Matcher, ...}
```

`HomeEnv` is the variable that replaces the config directory when set
(`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `COPILOT_HOME`). `Matcher` is the matcher
that selects every tool in the agent's own syntax, empty when none should be
written. `PowerShell` marks an agent that runs hooks through PowerShell on
Windows and so needs `HookSpec.CommandWindows`.

## Reading payloads

`Parse` turns what an agent wrote to a hook's stdin into one `Payload`, so
tools don't each learn every agent's field names, tool names and way of
reporting failure:

```go
payload, err := captainhook.Parse(captainhook.AgentCopilot, "postToolUse", stdin)
if err != nil { ... }

payload.Event // "PostToolUse": the catalog key, whichever spelling arrived
if tool := payload.Tool; tool != nil {
	tool.Name    // "Bash" (Copilot sent "powershell"; that is tool.RawName)
	tool.Outcome // OutcomeFailure: the result text says the command exited 3
}
```

- **Agent.** Pass the agent when you know it; a hook command can carry it as
  a flag. With `""`, `Parse` recognizes the agent where the payload gives it
  away and otherwise reads the payload by its shape alone.
- **Event.** The second argument is the settings key the hook was registered
  under. It names the event for payloads that don't: GitHub Copilot CLI's
  camelCase payloads carry no event name.
- **Tool names.** A tool goes by its Claude Code name when it has one (`Bash`
  for any shell, `Read`, `Write`, `Edit`, `Grep`, ...), following the table
  GitHub publishes for Copilot's own tools. A tool with no Claude equivalent
  keeps its name. MCP tools are `MCP`, with the server and tool split out when
  the name allows it.
- **Outcome.** Agents report a failed tool differently: as a separate event
  (Claude Code, Qwen Code, Copilot), as an exit-code line inside the result
  text (Gemini CLI, Command Code, and Copilot for shell commands), or not at
  all. `OutcomeUnknown` is the honest answer for the last case: Codex gives
  hooks a shell command's output and never its exit code.

`testdata/<agent>/<version>/` holds the payloads the parser is tested
against. Each directory's `SOURCE.md` says whether they were captured from a
live run or built from the agent's source code.

## Releasing

Push a `vX.Y.Z` tag. The Go module proxy serves it right away. The Release
workflow re-runs the tests and publishes a GitHub release with generated
notes.
