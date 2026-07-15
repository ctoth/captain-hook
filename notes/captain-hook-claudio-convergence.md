# Captain Hook / Claudio Convergence Fixed-Point Log - 2026-07-14

Target architecture:
- `captainhook` owns shared Codex hook-group mutation semantics: install,
  replacement, uninstall, caller identity, and preservation of unrelated hook
  commands.
- Claudio supplies its executable identity and desired hook definitions, then
  uses the shared owner directly for Codex hook persistence.
- Claudio retains product-specific hook registry, agent semantics, path
  selection, locking, verification, and non-Codex formats.

Forbidden surfaces:
- Captain Hook replacement or uninstall that drops unrelated commands from a
  mixed hook group.
- A Claudio-local duplicate of Codex hook-group mutation semantics after
  migration.
- A wrapper, adapter, compatibility branch, or parallel old/new Codex merge
  path introduced only to preserve the former ownership split.

Search gates:
- Claudio production references to superseded Codex hook-group mutation code.
- Captain Hook first-entry-only ownership checks.
- Parallel Captain Hook and Claudio-local Codex mutation paths.

Runtime gates:
- Captain Hook focused preservation and idempotency tests.
- Captain Hook full `go test ./...`.
- Claudio focused install, merge, uninstall, and Codex end-to-end tests.
- Claudio full `go test ./...`.
- `go mod tidy` and `go mod verify` for any dependency change.

## Iteration 0 - ownership diagnosis

Slice read:
- `captain-hook/hooks.go`
- `captain-hook/settings.go`
- `claudio/internal/install/hooks.go`
- `claudio/internal/cli/install_command.go`
- `claudio/internal/install/hook_registry.go`
- `claudio/internal/install/codex_settings.go`
- `claudio/internal/install/hooks_merge_test.go`
- `claudio/CLAUDE.md`

Surfaces:
- `captainhook.Install` / `captainhook.Uninstall`
  - Disposition: rewrite
  - Owner after cleanup: `captainhook`
  - Evidence: current `groupBelongsTo` examines only the first command and
    replacement/uninstall act on the whole group.
- Claudio mixed-group preservation contract
  - Disposition: move
  - Owner after cleanup: Captain Hook tests and implementation for the shared
    Codex representation.
  - Evidence: Claudio regression tests cover `[claudio, custom]` arrays and
    Claudio/non-Claudio siblings within one matcher group.
- Claudio Codex hook-group mutation
  - Disposition: consolidate
  - Owner after cleanup: `captainhook`
  - Evidence: Captain Hook declares itself shared settings infrastructure for
    Ward and Claudio and already models `commandWindows`.

Gate results:
- Pass: direct live inspection established the ownership boundary.
- Pending: branch and tracked-file state in both repositories.
- Pending: failing Captain Hook regression tests.
- Pending: implementation, consumer migration, full gates, commits, and push.

Current blocker:
- None external. The next required action is Git branch/tracked-state
  verification before the first test edit.
- The published dependency/version path must be read from current repository
  state before Claudio's module can be changed; no local `replace` will be
  committed.

Commit:
- Recorded with the Iteration 1 closeout.

Next slice:
- Captain Hook mixed-group preservation, test first.

## Iteration 1 - Captain Hook mixed-group preservation

Slice read:
- Complete Captain Hook production slice: `hooks.go`, `settings.go`, and
  `go.mod`.
- Complete existing test slice: `hooks_test.go`.

Surfaces:
- `groupBelongsTo`
  - Disposition: delete
  - Owner after cleanup: entry-level mutation in `stripOwnedCommands`.
  - Action: removed the whole-group boolean predicate.
  - Evidence: it inspected only `hooks[0]`, so it either missed later owned
    commands or caused callers to delete an entire mixed group.
- `stripOwnedCommands`
  - Disposition: keep
  - Owner after cleanup: Captain Hook hook-group mutation core.
  - Action: install and uninstall both remove only commands recognized by the
    caller identity, preserve unrelated and malformed siblings, preserve group
    metadata, and drop a group only when no commands remain.
  - Evidence: both mutation paths require exactly the same ownership operation;
    this is the shared owner operation, not a compatibility wrapper.

Gate results:
- Expected fail: focused install regression deleted the custom sibling when the
  owned command was first and retained the stale owned command when it was
  later.
- Expected fail: focused uninstall regression removed the entire hooks section
  when the owned command was first.
- Pass: `go test ./... -run
  'TestInstallPreservesNonOwnedSiblingsWithinOwnedGroup|TestUninstallPreservesNonOwnedSiblingsWithinOwnedGroup'
  -count=1`.
- Pass: `go test ./...`.
- Pass: `go vet ./...`.
- Pass: `rg -n "groupBelongsTo|Check the first hook entry" --glob '*.go'`
  returned zero hits.
- Pass: `git diff --check`.

Current blocker:
- None. The kept source slice is committed. The next action is to publish the
  record and release `v0.1.1`.

Commit:
- `9c33119 fix: preserve mixed hook groups`.

Next slice:
- Publish the corrected Captain Hook version, then migrate Claudio against that
  released owner.
