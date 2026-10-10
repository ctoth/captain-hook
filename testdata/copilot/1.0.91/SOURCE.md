# GitHub Copilot CLI 1.0.91

Live capture, Windows 11, 2026-10-01. A user-level `settings.json` (under a
scratch `COPILOT_HOME`) registered one logging command per settings key, in
both spellings; `copilot -p` ran a scripted prompt (a passing shell command,
`exit 3`, then create, view and edit a file). `key` is the settings key each
payload arrived under.

Changed after capture: UUIDs, paths and the prompt text. Nothing else.

What the capture shows:

- The settings key picks the payload format. camelCase keys get camelCase
  fields and no event name; PascalCase keys get snake_case fields and
  `hook_event_name`.
- `PermissionRequest` is the exception: under either key it arrives
  camelCase, with the event in `hookName` and the arguments in `toolInput`.
- PascalCase payloads use Claude tool names (`Bash`, `Read`, `Write`);
  camelCase payloads use runtime names (`powershell`, `view`, `create`).
- A shell command that exits nonzero is still a `PostToolUse` with
  `result_type: "success"`. Its exit code is only in the result text.
- A tool that fails outright arrives as `PostToolUseFailure`.

Reference: https://docs.github.com/en/copilot/reference/hooks-reference
