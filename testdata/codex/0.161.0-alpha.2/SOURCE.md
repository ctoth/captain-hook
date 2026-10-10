# Codex CLI 0.161.0-alpha.2

Live capture, Windows 11, 2026-10-05. A project `.codex/hooks.json`
registered a logging command for each event; `codex exec -s workspace-write
--dangerously-bypass-hook-trust` ran a scripted prompt (a passing shell
command, then `exit 3`).

Changed after capture: UUIDs, paths, the prompt text and
`last_assistant_message`. Nothing else.

`PostToolUse-bash-exit3` is the payload for a command that exited 3. Codex
does not send the exit code to hooks: the shell tool always counts as
successful for hook purposes and the hook receives only the output text
(`codex-rs/core/src/tools/context.rs`, `ExecCommandToolOutput`, tag
rust-v0.160.0). The outcome of a Codex shell command is therefore unknown.
