---
id: task-03-long-projects
title: Strengthen durable long-project collaboration
status: completed
wave: 3
depends_on:
  - task-02-accepted-checkpoints
plan: plan.md
requirements:
  - REQ-RELIABLE-PROJECT-001
  - REQ-RELIABLE-PROJECT-002
  - REQ-RELIABLE-PROJECT-003
acceptance_criteria:
  - AC-RELIABLE-PROJECT-001.1
  - AC-RELIABLE-PROJECT-001.2
  - AC-RELIABLE-PROJECT-001.3
  - AC-RELIABLE-PROJECT-002.1
  - AC-RELIABLE-PROJECT-002.2
  - AC-RELIABLE-PROJECT-003.1
  - AC-RELIABLE-PROJECT-003.2
  - AC-RELIABLE-PROJECT-003.3
system_design:
  - ../../specs/reliable-collaboration/system-design/long-projects.md
---

# Long Project Continuity

## Outcome and scope

Add durable project requirements and memory, confirmed backlog continuation, validated writer-stage recovery, a compact project view and deployment-drive temporary storage. Preserve current source and all native accounts. Do not automatically accept candidates, install extensions, delete retained work, or claim raw model reasoning is available.

## UI preview

```text
Task summary: actual stage | round | recoverable failure
[Conversation] [Activity] [Results] [Project]
Project: accepted items / total recorded items
Requirements: status | acceptance condition | evidence | start
Add requirement and acceptance condition
Project memories: provenance | version | needs revalidation
Batch history and storage analysis (explicit actions)
Fixed existing composer
```

## Acceptance

1. A project retains requirements and provenance across batches, resists stale edits, and advances only explicitly confirmed backlog items after human acceptance.
2. A failed backend can resume the same round after validating retained frontend evidence, then reaches real checks and fresh review. Drift and unsafe scope block recovery.
3. Browser and runtime checks validate the project view and deployment-drive temporary roots; cleanup preserves active work and all evidence.

## Verification

- `go test -race -trimpath ./internal/reliable -run 'Test(Project|Stage|Storage|Runtime)' -count=1`
- `node --test apps/backend/internal/reliable/page-project.test.cjs`
- `node scripts/reliable/desktop-browser-check.cjs`
- Existing acceptance, continuation, resource and route regression groups.

## Results

Completed and deployed locally as 1.14.0 on 2026-10-21. Release validation used targeted regression groups; it does not establish a complete current-backend-suite run.

Writer recovery and existing domain/handoff regressions passed after bounded-retry and complexity refactors (`go test -trimpath -run '^Test(StageResume|BothDomains|DualDomain|FrontendScope|BackendHandoff)' ./internal/reliable -count=1`, 259.498 seconds). The suite covers transient retry, bounded pause, legacy journal reconstruction, tampered source/report rejection, partial backend scope, final-round recovery and unchanged quality-rework exhaustion.

An explicitly enabled read-only probe of the existing blocked first-round task passed in 103.38 seconds. It reconstructed the completed frontend from retained dispatch/report evidence, identified backend as the next writer, and verified unchanged candidate hash before and after inspection. The five retained partial backend files were within the audited backend scope. No native model call, task-store write, source change or evidence write occurred. Its deployment-drive receipt is `data/validation/upgrade-1.14.0/stage-legacy-readonly.log` in the local distribution.

Final targeted validation passed nine critical race regressions, one route race regression, 26 page logic tests and 87 browser scenarios. Project, storage and stage-specific execution records remain separate from those totals. Incremental lint reported zero issues. Existing-database startup and final candidate/production browser checks passed with FTS5 enabled and exact embedded-source matching.

Deployment preserved all ten existing task records byte-for-byte, native connection settings and the desktop shortcut. Release checks issued no task mutations, attachment uploads, recovery starts or paid model calls. The existing business task was not declared complete; operational backend continuation remains a separate authorized action. The executable SHA-256 is `c3dd6df6caf36a00cff6c4af3e8464abc274c2f4c5b375a5d21bc44bf2ce5c11`. The local distribution retains `data/validation/upgrade-1.14.0/deployment.json`, `production-runtime.json` and individual check logs, with rollback backup `backups/upgrade-1.14.0-20261021-210010`.

This work does not add same-round checking/reviewer recovery, an application preview server, a per-file editor, a hard cache quota or deletion of retained worktrees/evidence.

Public documentation checks passed all 62 tests and validated 47 published pages. Catalog validation covered 352 decisions and 1,368 specifications. Full specification lint retained 107 pre-existing size findings, with no reliable-collaboration finding. The public task/workflow page remains a how-to guide; the root README and screenshot catalog have no changed local-extension entry to update. Documentation logs are retained under the same deployment-drive validation directory.
