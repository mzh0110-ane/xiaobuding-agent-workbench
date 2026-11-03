# Stage model handoff implementation

Status: complete. Deployed to the authorized local distribution on 2026-10-19.

1. Persist versioned stage assignments and reject stale or unavailable selections.
2. Freeze actual dispatch identity and controller-built handoff evidence at each boundary. Preserve active calls, task identity, worktree and round budget.
3. Wire Codex ChatGPT login and the separate native Harness account home without rewriting credentials.
4. Simplify the Chinese UI, add preflight/folder selection and a fresh sample flow.
5. Validate routing races, restart persistence, missing evidence, UI interaction and a mixed-provider real sample. Deploy only after checks pass and retain a rollback binary.

See task-01-model-handoff.md.

Validation: 62 full-package race regression items passed, plus the final default-plan race case; adapter race tests, go vet and package golangci-lint passed. Three client-state cases and five browser scenarios passed. A real Harness/Codex/Harness sample used three distinct native sessions, reached acceptance in round one and passed four checks after application to an isolated project. The actual binary passed desktop/mobile and native SPA checks. Windows PowerShell 5.1 cold start retained all three production history records. Documentation catalogs validated 349 decisions and 1346 specs; the extension added no spec violations, while 107 existing upstream file-size findings remain. Evidence and rollback files are retained in the local distribution.
