---
created: 2026-10-20
status: implemented
requirements:
  - REQ-RELIABLE-RENEWAL-001
system_design:
  - ../../specs/reliable-collaboration/system-design/exhausted-renewal.md
legacy_specs: []
---

# Implementation Plan: Exhausted Batch Renewal

## Overview

Complete continued work on one project after a batch reaches its three-attempt
limit. The explicit continuation request creates a separate, independently
owned child from verified retained source, then requires fresh audit, checks,
Codex review and human acceptance. The user explicitly requested implementation
after the previous explanation of this missing continuation; this package
records the implementation contract and does not introduce another approval
checkpoint.

## Scope

### In scope

- Atomic explicit renewal, stable request/child linkage and one project owner.
- Preserved parent history, verified independent source seed, resource-safe
  transfer, durable restart/cancel guards and cumulative original-base patch.
- Native assignment/context continuity, new sealed handoff evidence and
  localized desktop/phone composer/action states.
- Deterministic regression/browser coverage and idle-only local deployment.

### Out of scope

Fourth automatic attempts, auto-renewal, destructive repair/merge, paid agent
test calls, original user-project edits, automatic acceptance, native plugin
installation, acceleration/result-card work and remote publication.

## Technical approach

Expose the protected `exhausted-next-batch` action through
`StartExhaustedBatch`, returning a linked `TaskMessage`. Add optional
`SeedPending`, `SeedEvidence` and `SuccessorTaskID` task fields;
`prepareExhaustedSeed` guards every child invocation before ordinary
preparation/audit. Add cohesive renewal helpers below
`apps/backend/internal/reliable` for transactional admission, frozen seed
validation/transfer and evidence. Reuse `cancelled_recovery_source.go`,
`cancelled_recovery_evidence.go`, quiescence proof, original-base worktree and
resource-profile boundaries without calling the cancelled recovery lifecycle.
Optional `Task` lineage/pending/evidence fields integrate with `store.go`,
`engine.go`, `requirements.go`, acceptance/evidence digest and archived
actions.

Add fail-first Go coverage before implementation and existing Edge fixture
coverage in `scripts/reliable/desktop-browser-check.cjs` before changing
`page.js`, `page-conversation.js` and scoped phone CSS. Keep server eligibility
authoritative. The public task guide describes project lifetime versus one
batch's limit. Independent source review checks the irreversible boundaries
before an idle-only deployment to `E:/小布丁` with the existing desktop entry.

| Context | Identity/capability | Intended behavior | Verification/fallback |
| --- | --- | --- | --- |
| Codex ChatGPT and DeepSeek Harness | Existing role model/effort plans and future selections | Same native assignments, fresh domain Codex review | Deterministic runner; no login changes or paid calls |
| Ordinary source project | Original base + parent owned candidate + child owned candidate | Seed adds/deletes/untracked source independently | Snapshot/patch equality; altered source blocks |
| Frozen-resource project | Manifest/base/digest and child-owned metadata | Transfer bounded source only, retain resource partition | Resource delta/marker tests; no full hydration |
| Existing accepted/inspection continuations | Current message contract | Behavior unchanged | Existing conversation regressions |
| Resting exhausted/blocked/cancelled/interrupted parent | Round three, no linked child/active owner | Explicit new linked budget | Conflict/idempotency/restart fixtures |
| Awaiting acceptance, active, archived or read-only | Existing lifecycle | Keep existing action, no seeded renewal | Negative API/browser scenarios |

## ASCII UI preview

Use existing tokens/actions/drawer. These diagrams specify action order,
truthful budgets and explicit request submission; pixel spacing is
illustrative. The task document scrolls, existing sidebar and header keep their
behavior.

### UI-RENEW-01: Exhausted task and requirement composer

Entry: select a resting exhausted modification task; desktop. Maps to
AC-RELIABLE-RENEWAL-001.1/.2/.7.

```text
This batch used 3/3 attempts. The project can continue in another batch.
Partial changes and unresolved findings are retained.
[Preserve changes, start next batch] [View working copy]

Anything else for the next batch? (optional)
+--------------------------------------------------------+
| Optional next requirements ...                         |
+--------------------------------------------------------+
[Start next batch]       Each batch has up to 3 attempts.
```

### UI-RENEW-02: Preparing, failure and linked history

Maps to AC-RELIABLE-RENEWAL-001.3/.4/.6/.7.

```text
Preparing next batch. Checking and preserving previous changes.
[Preparing ... (disabled)]
New batch: 0/3 attempts. [Previous batch]

Cannot continue: original project changed since the saved base. Both working
copies are retained. Your requirements remain entered. [Retry when resolved]
[View working copy]

Previous batch: 3/3 attempts. [Open next batch]
```

### UI-RENEW-03: Phone

Entry: same task/composer at 390 px. Maps to AC-RELIABLE-RENEWAL-001.1/.7;
stack actions, wrap text and preserve focus.

```text
+------------------------------+
| This batch used 3/3.          |
| Keep improving in next batch. |
| [Preserve and start next    ] |
| [View working copy          ] |
| Optional next requirements   |
| +--------------------------+ |
| | ...                      | |
| +--------------------------+ |
| [Start next batch           ]|
| Errors wrap below.           |
+------------------------------+
```

## Tests

The focused suite uses real Git/source fixtures with deterministic agent and
check boundaries. Actual results remain separate from this coverage mapping.

| Acceptance | Deterministic evidence |
| --- | --- |
| .1/.4 | `TestExhaustedBatchExplicitContinuation`, `TestExhaustedBatchAdmissionIsAtomicAndIdempotent`, `TestExhaustedBatchRejectsQuestionKindWithoutCreatingSuccessor` |
| .2/.6 | `TestExhaustedBatchAcceptanceAppliesCumulativeCandidate`: inherited plus new source reach the original only after fresh review and human acceptance |
| .3 | `TestExhaustedBatchRejectsInvalidParentBeforeAgentCall`, `TestExhaustedBatchPreservesFrozenResourcesAndChildGuard`, pending child branch/junction tests, five `TestExhaustedSeedCopy*` cases and `TestSnapshotHandlesFileDirectoryTransitions` |
| .4/.6 | `TestExhaustedBatchRestartResumesPartiallyCopiedSource`, `TestExhaustedBatchRestartRejectsUnexpectedCandidateEdits`, `TestExhaustedSeedReceiptRemainsAnExecutionAndAcceptanceGate`, ancestry/archive and changed-ancestor-acceptance tests |
| .4/.5 | Three `TestExhaustedParent*` cases: obsolete cancellation/settings rejection with read-only answer stop preserved; `TestRenewalPendingRejectsNewRequirements` |
| .5/.6 | `TestExhaustedBatchPreservesMeaningfulAutomaticVerification`, existing future-plan/native-domain tests, failed review retained as context rather than current authority |
| .1/.7 | Existing Edge fixture: explicit action/composer, optional requirements, single-flight/retry identity, errors, selection retention and linked navigation at desktop 1440/phone 390 |

## E2E tests

Extend `scripts/reliable/desktop-browser-check.cjs` using disposable fake tasks
and blocked/deterministic API fixtures. Cover exhausted selection without POST,
explicit submission exactly once, with and without additional requirements,
response-driven child selection, checking/error/content preservation, old child
link, active/archived/read-only ineligibility and no phone overflow. Exercise
fixture end-to-end from the actual protected renewal route through fresh
audit/check/review/human acceptance in Go. Production browser checks may GET
only; do not renew the real stopped task.

## Work orders

- [x] [Task 01: Continue exhausted work in a preserved child](task-01-exhausted-renewal.md)

## Verification results

Implementation, corrected checks and idle-only local delivery are complete as
of 2026-10-20. The paired requirement is active, design current, this plan
implemented and its work order done.

The primary owner read the complete design package and confirmed the dedicated
API, original-base source identity, atomic ownership and desktop/phone structure.
Independent source review identified and corrected archive/lineage, ancestry
receipt, automatic-check selection, source shape/staging, parent cancellation,
obsolete current/future model updates and pending-child ownership boundaries.
The installed program has not been inferred from source or fixture success.

- Corrected targeted race evidence comprises **47 distinct top-level passes and
  one host-privilege skip** across recorded batches. The first combined run
  recorded 18 passes, one manually constructed lifecycle-fixture failure and
  one symlink skip before a redundant 30,000-resource fixture exceeded its
  timeout (`1206.743s`). Those failure/timeout logs are retained, not described
  as a successful run. The manual fixture was corrected to clear the pending
  continuation flag that the real lifecycle clears; the resource bridge uses
  two real frozen inputs while existing large-resource tests retain scale
  coverage. The corrected completion batch passed 9 tests (`107.781s`), and
  the shared lifecycle/model/source regression passed 22 tests (`190.426s`).
  Their latest results resolve every earlier test failure. This is segmented
  regression evidence, not a successful single whole-package run.
- The source symlink fixture was skipped because this Windows host lacks the
  required privilege. Task-directory junction rejection, wrong branch,
  resource tampering, retained partial changes, cumulative human application,
  ancestor receipt changes and old-parent action guards passed.
- `scripts/reliable/desktop-browser-check.cjs`: **52/52** passed on desktop and
  390 px phone. Page-state checks: **7/7**. Public-doc tests: **62/62**; public
  validation checked **47 pages**. The final browser result includes the actual
  project-conflict code and the superseded-parent model drawer.
- `go vet ./internal/reliable` passed; reliable-package golangci-lint reported
  **0 issues**. The local `fts5` Windows build passed with the existing toolchain.
- The actual exhausted project passed the read-only deployment preflight in
  `12.22s`, retaining all eight task records at the original state and budget.
  It created no child, dispatched no agent and applied no source changes.
- Catalog validation covered **352 decisions and 1364 specifications**. The
  package has four valid UTF-8/no-BOM artifacts, 13 resolved local links and a
  successful local PR-documentation coverage result (`covered`, zero errors).
  Package diff checks passed. Full specification lint still returns **107
  unrelated pre-existing file-size findings**, zero reliable-collaboration
  findings and no new owning-system failure.
- Local deployment/runtime result: Deployed **1.10.0** to `E:/小布丁`. The stopped-service database copy
  preserved all eight task histories, validated the existing pending sealed
  candidate without applying it (`1.02s`), and established stopped-owner cleanup
  proof for the recorded histories. Production task JSON is byte-identical;
  native settings and desktop shortcut hashes are unchanged. Candidate and
  production desktop/390 px runtime checks passed with zero mutation requests.
  The real task remains exhausted at 3/3 with no child; its explicit renewal
  action is visible and enabled, and was never clicked. Backup:
  `E:/小布丁/backups/upgrade-1.10.0-20261020-211920`. Installed executable SHA-256:
  `e636050d9a57b159b2712536bd38dcd92d2cfd33948197fdd708d0c45419e8e0`.
  Candidate launch/runtime/deployment proofs bind those exact bytes. No paid
  call, recovery, renewal or human acceptance ran during installation.

Logs and delivery receipts live under
`E:/小布丁/data/validation/upgrade-1.10.0`. Validation and installation never
click renewal/recovery or acceptance on the real project, call paid models,
install native skills/plugins, modify original source, or create Git commits.

## Risks

- Generic project locks currently treat exhausted parents as unfinished; a
  non-atomic release would admit competing writers.
- Reusing ordinary linked continuation would start from the original and lose
  unaccepted seed changes; adopting the seed as baseline would omit them from
  final application.
- Partial transfer, parent reparse substitution, mutable original, changed
  resource metadata or unknown old processes must block without repair/reset.
- Receipt omission after restart or archive/ancestor actions could bypass the
  lineage and validation obligation; all mutation paths need bounded checks.
