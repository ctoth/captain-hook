# Command Code 1.73.4

**Built from source, not captured.** No live run was possible (no account),
so these payloads were assembled by hand from `dist/cli.mjs` in the
`command-code@1.73.4` npm package. The field names and the text layout come
from the functions below; the command, its output and the ids are made up.

- `buildHookPayload`: every event has `session_id`, `transcript_path`,
  `cwd`, `hook_event_name` and `permission_mode` (each `""` when unset).
  Tool events add `tool_use_id`, `tool_name`, `tool_display_name` and
  `tool_input`; `PostToolUse` adds `tool_response`, a string.
- The tool-name table (`shell_command` displays as `SHELL`, `read_file` as
  `READ`, and so on) is the object passed to `toToolDisplayName`.
- `formatResult2`: a shell result starts with the line `Exit code: N` unless
  the exit counts as a success for that command, then the output.

Not established: whether `PostToolUse` fires for a tool that fails outright.

Replace these with a live capture when one is available.
