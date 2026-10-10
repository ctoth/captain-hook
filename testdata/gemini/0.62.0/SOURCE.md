# Gemini CLI 0.62.0

**Built from source, not captured.** No live run was possible (the test
account could not sign in), so these payloads were assembled by hand from
the code at tag v0.62.0. The field names and the text layout come from the
files below; the command, its output and the ids are made up.

- Common fields `session_id`, `transcript_path`, `cwd`, `hook_event_name`,
  `timestamp`, and the tool fields `tool_name`, `tool_input`,
  `tool_response`: `packages/core/src/hooks/types.ts` (`HookInput`,
  `BeforeToolInput`, `AfterToolInput`).
- `tool_response` is `{llmContent, returnDisplay, error}`:
  `packages/core/src/core/coreToolHookTriggers.ts`.
- Shell `llmContent` is `Output: ...`, then `Error: ...` when the process
  could not start, then `Exit Code: N` only when N is nonzero, then
  `Process Group PGID: N`: `packages/core/src/tools/shell.ts`. A nonzero exit
  does not set `error`.
- `error` is `{message, type}` with `type: "shell_execute_error"` when the
  process could not start: `packages/core/src/tools/shell.ts`,
  `packages/core/src/tools/tool-error.ts`.
- The `<untrusted_context>` wrapper, tags on their own lines:
  `packages/core/src/utils/textUtils.ts` (`wrapUntrusted`).
- Tool names: `packages/core/src/tools/definitions/base-declarations.ts`.

Replace these with a live capture when one is available.
