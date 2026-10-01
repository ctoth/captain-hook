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

## Releasing

Push a `vX.Y.Z` tag. The Go module proxy serves it right away. The Release
workflow re-runs the tests and publishes a GitHub release with generated
notes.
