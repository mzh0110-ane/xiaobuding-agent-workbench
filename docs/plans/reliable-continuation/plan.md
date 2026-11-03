---
status: completed
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-LOOP-002
  - REQ-RELIABLE-PROJECT-001
  - REQ-RELIABLE-PROJECT-002
  - REQ-RELIABLE-PROJECT-003
system_design:
  - ../../specs/reliable-collaboration/system-design/local-loop.md
  - ../../specs/reliable-collaboration/system-design/long-projects.md
---

# Persistent project continuation

The user explicitly requested implementation of continuous project follow-up in the existing local distribution. Preserve native accounts, model selection, evidence gates and the three-round modification budget.

## Scope

1. Persist task-scoped questions and requirement supplements, with idempotent submission and visible delivery states.
2. Run questions read-only and sequentially after an active workflow. Carry prior reports and bounded conversation context, and retain source citations. Keep conversation artifacts outside sealed modification evidence.
3. Freeze requirement revisions at audit boundaries. New requirements invalidate an older passing candidate and must reach fresh audit, implementation, real checks and independent review without resetting rounds.
4. Allow a completed inspection or accepted task to start an explicitly requested linked modification task, retaining project context and native assignments. Never reset an exhausted task automatically.
5. Add an accessible task composer, conversation history, failure feedback and per-task drafts. Verify real browser behavior and deploy the checked local binary with rollback artifacts.
6. After explicit acceptance, record exactly the approved result as a local Git version. Recover eligible legacy accepted parents before linked-child preparation, preserve external repository ownership and user staging, and expose checkpoint completion or retry without consuming rounds or creating duplicate project folders.
7. Add a project ledger with provenance-bearing memory and confirmed backlog items, eligible same-round writer recovery, bounded patch inspection and deployment-drive transient storage. Keep model proposals unconfirmed, human acceptance mandatory and evidence retained.

## Work order

- [x] [Task 01: conversation and controlled requirement continuation](task-01-continuation.md) owns the original continuous-conversation delivery.
- [x] [Task 02: accepted local versions and next-batch preparation](task-02-accepted-checkpoints.md) owns the acceptance-to-next-batch completion in 1.13.0.
- [x] [Task 03: durable long-project collaboration](task-03-long-projects.md) owns the 1.14.0 project view, confirmed backlog, writer recovery and storage boundaries. Implemented and deployed locally on 2026-10-21 with targeted checks and production read-only validation.

## Result

Implemented task conversation, immutable requirement revisions, linked work batches and lifecycle-safe delivery. Final owning-package race run passed 109 top-level tests with seven opt-in skips in 222.354 seconds. All 29 desktop/mobile browser scenarios passed. Real HTTP/database/native-protocol integration retained two answers across reload with zero modification rounds and unchanged source; no paid model call was made. Go vet and owning lint passed. Global specification lint retains unrelated baseline findings; this system has zero findings.

The preceding result describes Task 01. Task 02 was completed and deployed locally as 1.13.0 on 2026-10-21. All 222 current top-level reliable tests were accounted for across race-run output: 209 passed and 13 opt-in/platform cases skipped, with zero missing tests and zero individual test failures. The first monolithic run exceeded its 10-minute package timeout; two exact-name shards of the remaining 16 tests each passed in 86.630 and 87.081 seconds. This is complete inventory coverage across runs, not a claim that the initial monolithic command passed.

The route race test, 19 Node logic checks and 78 desktop/phone browser scenarios passed. Go vet and incremental repository lint passed with zero reported issues. A forced full lint reported 13 global complexity/constant findings; an independent unchanged-baseline comparison was not performed for those findings. Documentation catalog validation covered 352 decisions and 1366 specifications; public checks passed 62 tests and validated 47 pages. Full specification lint retains 107 existing size findings, with zero in reliable-collaboration.

Candidate and production HTTP/browser checks were GET-only. The deployed executable restored the existing 253-table SQLite database with FTS5 support and preserved all 10 task records byte-for-byte at the deployment boundary, connection settings and the desktop shortcut. Executable SHA-256: `f541969704a16d70dbdd306e72f5c1b94851559bcf4911117f6315e59c400429`. As separately authorized project work after deployment, the accepted legacy result was checkpointed as `14b05fb4f571c31d2f3356ef0f941f2025d389c8` with exact tree `36af49778398944d39dc86929eccdb345e60fe96`; the original was clean and no temporary acceptance-index files remained. The same linked child resumed at zero implementation rounds on that base. This operational continuation is distinct from the read-only release validation.

Task 03 completed local deployment as 1.14.0. Its final validation passed nine critical race regressions, one route race test, 26 page logic tests and 87 browser scenarios, with separate project/storage/stage logs and zero incremental lint findings. These are targeted checks, not a claim of full current-backend-suite coverage. GET-only candidate/production checks preserved all ten task records, native connection settings and the desktop shortcut. The installed executable SHA-256 is `c3dd6df6caf36a00cff6c4af3e8464abc274c2f4c5b375a5d21bc44bf2ce5c11`; deployment and runtime receipts are retained in the local distribution's `data/validation/upgrade-1.14.0` directory. The pre-existing business task still requires its separate authorized backend continuation and final acceptance.

## ASCII UI preview

UI-CHECKPOINT-01: Existing task summary after explicit acceptance, above every workspace tab.

```text
Accepted and applied
Local version: <short commit identity>
[ Continue asking / Next modification batch ]
```

UI-CHECKPOINT-02: Existing blocked-task action during an incomplete local version.

```text
Needs attention: <localized source/index/history reason>
Accepted version storage is incomplete. Files are retained.
[ Resolve and continue ]
Automatic attempts remain <used>/3.
```

Desktop and phone use the same existing summary and results components, with wrapping status text and no new modal or folder workflow. The completion identity, specific failure reason and eligibility at 3/3 are required by AC-RELIABLE-LOOP-002.5/.7; spacing and the translated English sketch labels are illustrative.

