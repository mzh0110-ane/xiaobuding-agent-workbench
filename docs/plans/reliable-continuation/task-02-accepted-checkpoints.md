---
id: "02-accepted-checkpoints"
title: "Accepted local versions and next-batch preparation"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-RELIABLE-LOOP-002
acceptance_criteria:
  - AC-RELIABLE-LOOP-002.1
  - AC-RELIABLE-LOOP-002.2
  - AC-RELIABLE-LOOP-002.3
  - AC-RELIABLE-LOOP-002.4
  - AC-RELIABLE-LOOP-002.5
  - AC-RELIABLE-LOOP-002.6
  - AC-RELIABLE-LOOP-002.7
system_design:
  - ../../specs/reliable-collaboration/system-design/local-loop.md
---

# Task 02: Accepted local versions and next-batch preparation

## Summary

Complete human acceptance with a local version of the exact approved source. Let explicitly requested next batches reuse that version, including eligible legacy accepted results, while preserving manual staging, unrelated edits, evidence and per-batch round limits.

## In scope

- Optional durable acceptance intent/result metadata and exact-tree Git checkpointing.
- Idempotent retry after application or history interruption without Agent calls or duplicate commits.
- Legacy accepted-parent reconciliation before linked-child dirty-root admission.
- Existing summary/result copy, local commit identity and retry at a spent implementation budget.
- Regression coverage for external roots, staged/manual edits, evidence tampering, crash windows and file-count stability.

## Out of scope

Remote operations, external repository adoption, blanket staging, whole-folder backups, new skills/plugins, automatic acceptance and unrelated UI redesign.

## Acceptance

- Acceptance and a linked next batch work end to end in managed and external temporary projects, with exactly the accepted source recorded locally and fresh child verification.
- Interruptions and eligible legacy tasks reconcile against proven source/history while staged or unrelated changes remain protected. Repeated completed retries create no duplicate receipt or commit.
- Existing desktop and phone task summaries expose the local version or actionable failure above every workspace tab. Checkpoint retry works at 3/3 without another model or implementation round.

## ASCII UI preview

Use [UI-CHECKPOINT-01 and UI-CHECKPOINT-02](plan.md#ascii-ui-preview) in the existing task summary, which remains visible above the results tab.

```text
Completed: Local version <short commit identity>
Blocked:   <specific reason> [ Resolve and continue ]
```

Shared desktop/phone composition wraps the existing status and actions. These outcomes map to AC-RELIABLE-LOOP-002.5/.7.

## Verification

From the repository root:

```powershell
go -C apps/backend test -trimpath -race ./internal/reliable -count=1
go -C apps/backend vet ./internal/reliable
node --test apps/backend/internal/reliable/page-state.test.cjs apps/backend/internal/reliable/page-proof.test.cjs apps/backend/internal/reliable/page-attachment-state.test.cjs apps/backend/internal/reliable/page-workspace-state.test.cjs
node scripts/reliable/desktop-browser-check.cjs
python scripts/list-docs.py validate
python scripts/lint-spec-files.py --all
node --test scripts/validate-public-docs.test.mjs
node scripts/validate-public-docs.mjs
git diff --check -- docs/specs/reliable-collaboration docs/plans/reliable-continuation docs/public/tasks-and-workflows.md
git status --short -- docs/plans/reliable-continuation
```

Fixtures use temporary projects and native protocol stubs rather than paid model calls. Production checks are read-only and never accept, apply or resume the real project merely to test the release.

| Criteria | Executable coverage |
| --- | --- |
| LOOP-002.1/.3 | `accepted_checkpoint_test.go`: `TestAcceptedExternalProjectIsCheckpointedBeforeTheNextBatch`; existing managed-project continuation coverage |
| LOOP-002.2 | `accepted_git_test.go`: `TestAcceptedGitPreservesManualChangesAndStaging`, `TestAcceptedGitReusesUserCommitAndRejectsForeignLock`, `TestAcceptedGitNoOpReusesBaseAndGitOperationBlocks` |
| LOOP-002.4 | `accepted_checkpoint_test.go`: `TestLegacyAcceptedPatchIsCheckpointedWhenTheNextBatchStarts`, `TestLegacyAcceptedCheckpointPreservesManualChangesAndInvalidReceipts`; `accepted_git_test.go`: `TestAcceptedGitLegacyRequiresExactAppliedBytes` |
| LOOP-002.5 | `accepted_checkpoint_test.go`: `TestPendingAcceptedCheckpointResumesAtTheThreeRoundLimit`, `TestCompletedCheckpointBeforeTerminalStateResumesWithoutAnotherCommit`; `accepted_git_test.go`: `TestAcceptedGitRecoversAfterCommitAndBeforeIndexInstall` |
| LOOP-002.6 | `TestAcceptanceCheckpointIsIdempotentAndDoesNotCreateEmptyCommits`, `TestAcceptedGitAppliesAndCheckpointsOnlyReviewedPatch`; both check bounded history/source-file effects |
| LOOP-002.7 | Seven checkpoint scenarios in `scripts/reliable/desktop-browser-check.cjs` cover desktop/phone, stored drafts, explicit retry, localized failure, completed-but-interrupted acceptance, accepted-version next-batch entry and future-model settings. `page-state.test.cjs` covers pending checkpoint scope without rewriting sealed assignments. |

## Files likely touched

- `apps/backend/internal/reliable/engine.go`
- `apps/backend/internal/reliable/workspace.go`
- `apps/backend/internal/reliable/continuation.go`
- `apps/backend/internal/reliable/accepted_checkpoint*.go`
- `apps/backend/internal/reliable/accepted_git*.go`
- `apps/backend/internal/reliable/page-proof.js`
- `apps/backend/internal/reliable/page*.js`
- `scripts/reliable/desktop-browser-check.cjs`
- `docs/specs/reliable-collaboration/{requirements,system-design}/local-loop.md`
- `docs/specs/reliable-collaboration/{requirements,system-design}/local-snapshots.md`
- `docs/public/tasks-and-workflows.md`

## Dependencies

The shipped Task 01 continuation behavior and existing source-bound acceptance gates. No pending work-order dependency.

## Risks

- A filesystem or Git mutation can complete before its result is persisted; replay must inspect exact source/history rather than assume failure.
- A legacy accepted source mixed with later edits cannot be safely adopted as the old accepted result.
- Optional task fields and post-acceptance receipts must preserve historical evidence serialization.

## Parallelism

`sequential` for source/history state integration. Independent documentation and adversarial review can proceed against the settled contract.

## Inputs

- [Local loop requirements](../../specs/reliable-collaboration/requirements/local-loop.md).
- [Accepted local versions design](../../specs/reliable-collaboration/system-design/local-loop.md#accepted-local-versions).
- Existing acceptance, managed local snapshot and linked-continuation tests.

## Results

Completed in local version 1.13.0 on 2026-10-21.

- Race regression accounted for all 222 current top-level reliable tests: 209 pass, 13 opt-in/platform skips, no missing test and no individual test failure. The initial 10-minute monolithic package command timed out after 604.234 seconds; the remaining two exact-name shards, 16 tests each with a 15-minute budget, passed in 86.630 and 87.081 seconds. The union and initial timeout are explicitly recorded in `upgrade-1.13.0/go-race-union-summary.json`; this is not a passing single-run claim.
- `TestReliableRoutesAreOptInAndRejectUngatedAcceptance` passed under race detection. Go vet passed. Incremental repository lint reported zero issues; forced full lint reported 13 global complexity/constant findings, without an independent unchanged-baseline comparison.
- All 19 Node logic tests and 78 mock browser scenarios passed, including the seven new checkpoint cases. Desktop and phone checkpoint screenshots were inspected; actions remain reachable without horizontal overflow. These checks use temporary fixtures and do not call paid models.
- `list-docs.py validate` passed for 352 decisions and 1366 specifications. Public documentation validation passed 62 tests and 47 pages. PR documentation coverage accepted the new work order, requirement IDs and paired design. Full specification lint retains 107 existing size findings and zero reliable-collaboration findings. Scoped whitespace and work-order status checks passed.
- Candidate and production runtime checks were GET-only, with no validation-triggered upload, model call, acceptance or recovery. The deployed FTS5-enabled runtime restored the existing 253-table database, preserving the 10 task JSON records, native connection settings and desktop shortcut at the upgrade boundary. Binary SHA-256: `f541969704a16d70dbdd306e72f5c1b94851559bcf4911117f6315e59c400429`.
- Separately authorized continuation then recovered the real accepted parent to local commit `14b05fb4f571c31d2f3356ef0f941f2025d389c8`, exact tree `36af49778398944d39dc86929eccdb345e60fe96`. Original source was clean and temporary acceptance-index files were absent. The existing child, `next-1df466043664a81bd7a663becc`, resumed investigation at 0/3 on that base instead of creating another task. This project action is separate from release verification.

Detailed local validation artifacts are retained under the installed distribution's `data/validation/upgrade-1.13.0/`; the implementation does not add those files to user project source.
