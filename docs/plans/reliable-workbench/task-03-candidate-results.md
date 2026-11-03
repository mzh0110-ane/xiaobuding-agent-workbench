---
id: task-03-candidate-results
title: Inspect candidate results and validate local preview
status: completed
wave: 3
depends_on:
  - task-01-navigation
  - task-02-project-continuity
plan: plan.md
requirements:
  - REQ-RELIABLE-WORKBENCH-004
  - REQ-RELIABLE-WORKBENCH-005
acceptance_criteria:
  - AC-RELIABLE-WORKBENCH-004.1
  - AC-RELIABLE-WORKBENCH-004.2
  - AC-RELIABLE-WORKBENCH-004.3
  - AC-RELIABLE-WORKBENCH-004.4
  - AC-RELIABLE-WORKBENCH-005.2
system_design:
  - ../../specs/reliable-collaboration/system-design/workbench.md
---

# Candidate Results and Preview

## Outcome and scope

Connect retained checks/review to changed-file inspection and an explicitly
started candidate preview. Validate ownership, source/build clarity, process
shutdown, and low-space admission before local installation. Keep original
project application and acceptance in the existing controller.

Exclude arbitrary project command execution, dependency installation, project
Vite config/plugin loading, a per-file editor, automatic acceptance, and claims
that a live preview is immutable verification evidence.

## Acceptance

1. File reads are paged and bounded; protected/redirected/non-UTF-8 or changed
   content is rejected. Original patch and check/review evidence remain available.
2. Preview starts only explicitly on an idle owned candidate, distinguishes
   supported static/generated HTML, basic installed Vite, unsupported modes,
   old builds, and source drift; uncertain cleanup blocks replacement.
3. Final real Windows shortcut, candidate/production startup, old database and
   task preservation, desktop/phone rendering, documentation, and independent
   review pass before claiming deployment complete.

## ASCII UI preview

UI-WORKBENCH-03, Results excerpt from [the combined preview](plan.md#ascii-ui-preview):

```text
Results: checks | independent review | unresolved requirements
[Changed files] [Read text] [Raw patch / evidence]
Preview: selected entry | start/current source | old-build warning
[Start preview] [Open candidate URL] [Stop]
Busy / unsupported / stale / cleanup unknown: reason and available action
[Human accept and apply] OR [Continue requirements]
```

The same ordering applies on phones; links/buttons must wrap without hiding
failure states. This maps to REQ-RELIABLE-WORKBENCH-004 and allocation admission.

## Files and dependencies

`result_files.go`, `result_files_policy.go`, preview manager/HTTP/Vite/log modules,
workbench routes, results page integration, engine reservation/shutdown, and the
storage admission policy. Reuse the original startup script and desktop shortcut;
the Windows PowerShell 5.1 UTF-8 BOM compatibility fix must remain intact.

## Verification

```sh
# From apps/backend.
go test -race -trimpath -p 1 ./internal/reliable -run '^TestResult(Preview|Files)' -count=1
go test -race -trimpath -p 1 ./internal/backendapp -run '^TestReliableRoutes' -count=1
# From the repository root after complete-checkout integration.
node scripts/reliable/desktop-browser-check.cjs
node --test scripts/validate-public-docs.test.mjs
node scripts/validate-public-docs.mjs
python3 scripts/list-docs.py validate
python3 scripts/lint-spec-files.py --all
```

Use the existing E-drive cache environment and `fts5` build tag for the release
candidate. If a listed validator has unrelated baseline findings, record its
actual output and distinguish changed-system findings instead of claiming a
globally clean run. Do not run browser mutations against a user's live task.

## Results

The final 11-test candidate file/preview race group passed in 68.316 seconds,
including generated nested HTML discovery, bare Vite source priority, bounded
reads, cross-origin/path rejection, and stopping-registration ownership.
`validation/preview-validation.json` contains the targeted command and hashes.

Results-page integration passed in the combined 27-check isolated browser group,
including start/stop ordering, stale preview responses, and ordered file reads.
All 54 combined page-logic checks passed. Public docs passed 62 tests/47 pages;
the catalog validated 352 decisions and 1370 specifications. Global spec lint
retains 107 existing size findings and none for this changed system.

A Windows candidate was built with `fts5` and passed startup using an existing
database read-only backup (253 tables, one FTS5 table, integrity check passed).
Final embedded sources matched candidate and production pages. Production
desktop/phone read-only checks passed with zero errors and zero write requests.
All 13 raw task records remained identical, native connection settings and
shortcut bytes were preserved, and two closed-copy legacy evidence/quiescence
checks passed in 2.542 seconds.

1.15.0 deployment completed to `E:\小布丁`, with rollback backup
`E:\小布丁\backups\upgrade-1.15.0-20261026-204601`. No user business task was
resumed or accepted. The original desktop shortcut was actually launched and
reused the same backend PID without a duplicate service. Windows PowerShell 5.1
target, launcher BOM, and parse checks passed. The storage budget read also
confirmed known free space. See `validation/desktop-shortcut-verification.json`.

The real backend route check passed in 3.213 seconds; reliable-package Go vet
passed and final incremental lint reported zero issues. The final candidate
SHA-256 is `f57f007f65760b7f62a7330e4bf8a6994eb041469d135e9dc5aa955289deec83`.
