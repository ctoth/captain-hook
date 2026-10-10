# Qwen Code 0.25.0

Live capture, Windows 11, 2026-10-06. A project `.qwen/settings.json`
registered a logging command for each event; `qwen -p` started a session.

Changed after capture: UUIDs, paths and the prompt text. Nothing else.

The run stopped at the first model call (the API key was rejected), so there
are no tool payloads here. Qwen's tool payloads are known only from its
source (`packages/core/src/hooks/types.ts`, tag v0.24.7): failures arrive as
`PostToolUseFailure` with a string `error` and an optional `is_interrupt`.
