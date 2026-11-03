---
created: 2026-10-20
status: implemented
requirements:
  - REQ-RELIABLE-RECOVERY-001
system_design:
  - ../../specs/reliable-collaboration/system-design/cancelled-recovery.md
legacy_specs: []
---

# Implementation Plan: Continue a Cancelled Collaboration Task

## Overview

Explicit recovery of a stopped modification task uses its existing partial
candidate and remaining rounds. One vertical work order owns transactional
admission, source/evidence validation, UI and regression coverage. This is an
independent priority package; the earlier
[speed/results package](../reliable-speed-results/plan.md) remains pending and
does not automatically recover a task.

## Scope and confirmed assumptions

Confirmed: the user wants the previously explained missing recovery capability.
Preserve partial changes, native accounts/skills/plugins and chosen roles/models,
evidence audit, independent review, three rounds per batch and human acceptance.
No automatic paid call or resume of the live cancelled project is part of tests
or installation. No unresolved preference blocks design.

The former HTTP/engine rejection is replaced by dedicated atomic recovery
admission. Cancel still preserves files and releases its project reservation;
recovery reclaims that reservation before validating or dispatching work.

### In scope

- Dedicated explicit recover-cancelled action with atomic project reservation.
- Verification of established partial candidates and bounded owned preparation
  recovery, original drift/resource checks and restart-safe pending intent.
- Remaining-round UI, fresh audit and acceptance invalidation/provenance.
- Focused tests and idle-only local delivery after implementation authorization.

### Out of scope

Cancelled read-only retry, a fourth round, merge across competing tasks,
destructive candidate repair, auto-resume, raw report adoption, new model presets,
parallel writers, acceleration/result preview implementation and remote release.

## Technical approach

`http.go` exposes the protected POST. `cancelled_recovery.go` orchestrates
reserved execution and receipt publication; `_store.go`, `_source.go` and
`_evidence.go` separate atomic admission, retained-source ownership and strict
history/resource validation. `runTask` validates on every new invocation when
pending or previous recovery evidence exists, including after pending-marker
clear and restart. One uninterrupted execution retains its existing round gate.

`process.go`, `process_windows.go` and `process_quiescence.go` establish owned
Windows Job cleanup and separate operational receipts. The explicit installer
migration holds the verified stopped service's process handle through receipt
publication against a closed database copy; uncertain PID/owner proof blocks.
Operational cleanup receipts do not alter sealed source evidence or prove review.
`evidence.go` binds recovery validation receipts only for recovered candidates,
preserving legacy digests. Future-plan promotion, source/resource checks and
`Store.StartRound` remain authoritative. The existing page, conversation and CSS
primitives supply explicit recovery, pending requirements and phone stacking.

| Context | Identity/capability | Intended behavior | Evidence/fallback |
| --- | --- | --- | --- |
| Codex and Harness writer plans | Existing native role selections | Fresh audit, next admitted round and independent review | Deterministic AgentRunner fixtures; no login/skill changes |
| Established ordinary/resource candidate | Task/base/common Git directory + contract + receipts | Preserve partial source, validate before execution | Existing snapshot/resource tests; ambiguous ownership blocks |
| Cancelled incomplete preparation | Owned intent and clean recoverable initialization | Finish verified preparation at round zero | No reset/replacement of dirty ambiguous candidates |
| Read-only, archived, stopping or exhausted | Saved task lifecycle | No modification recovery; explain next step | Report/folder access preserved |

## ASCII UI preview

UI labels are illustrative product-localized structure, not pixel geometry.
Reuse current avatar/background/Agent controls. The action and truthful round
hint are requirements; no new modal or permanent side panel is required.

### UI-RECOVER-01: Task action row before and after

Entry: select a cancelled modification task, desktop.

```text
Before:
Task stopped. Working copy and evidence retained.
[View working copy]

After:
Task stopped. Partial changes retained.
Used 1/3 rounds. 2 implementation rounds remain.
[Check copy and continue] [View working copy]
```

Maps to AC-RELIABLE-RECOVERY-001.1/.5/.7. The main task document scrolls;
existing sidebar/header remain unchanged.

### UI-RECOVER-02: Validation, conflict and budget states

```text
Checking retained candidate ...
[Checking ... (disabled)] [View working copy]

Cannot continue: another unfinished task owns this project.
No changes were discarded. [Retry] [View working copy]

Used 3/3 rounds. This batch cannot continue.
[View working copy] [Evidence]
```

Maps to AC-RELIABLE-RECOVERY-001.3-.7. No successful state is shown until
validation passes. Original drift, missing receipt, resource tampering and
unknown process cleanup get their own cause and a nondestructive next step.

### UI-RECOVER-03: Phone composition

```text
+------------------------------+
| Task stopped                 |
| Partial changes retained.    |
| Used 1/3. 2 rounds remain.    |
| [Check copy and continue    ]|
| [View working copy          ]|
| Reason wraps below on error. |
+------------------------------+
```

Maps to AC-RELIABLE-RECOVERY-001.7. Stack the action/hint at 390 px, retain
keyboard focus, and prevent whole-page horizontal overflow. No desktop split
pane is squeezed into the phone view.

## Tests

Targeted pre-sign-off recovery batches passed 16 top-level Go tests; the browser
fixture adds seven recovery scenarios while retaining all 36 existing scenarios.
Final review-correction coverage remains tracked in the results below.

| Acceptance | Targeted tests |
| --- | --- |
| .1/.4 | TestCancelledRecoveryAdmission, TestCancelledRecoveryConflicts, TestCancelledRecoverySingleOwnerAndCancellation, TestCancelledRecoveryAdmissionRacesNewProjectOwner |
| .2/.3 | TestCancelledRecoveryPreservesPartialSource, TestCancelledRecoveryRejectsDrift, TestCancelledRecoveryKnownGitCleanupFailure, TestCancelledRecoveryUnknownLegacyProcesses, TestCancelledRecoveryResourceReferences, TestCancelledRecoveryBeforePreparation |
| .5 | TestCancelledRecoveryKeepsRoundBudget; 0/1/2/3 consumed rounds and restart |
| .6 | TestCancelledRecoveryPendingGuard, TestCancelledRecoveryEvidence, TestCancelledRecoveryKeepsModelAndRequirementHistory, TestCancelledRecoveryRejectsRequirementWhilePending, TestCancelledRecoveryReadHasNoSideEffects |
| .1/.7 | Existing Edge browser fixture plus new eligibility/checking/error/remaining-round cases |

## E2E tests

`scripts/reliable/desktop-browser-check.cjs` covers desktop 1440 px and phone
390 px. Cover explicit click only, pending duplicate disabled, validation error,
server rejection, archive restore without resume, round-three/read-only
ineligibility, polling state/selection retention and no model/start request on
GET. Use a deterministic backend fixture for POST-to-fresh-audit flow, never
the user's live task or paid native agents.

## Work orders

- [x] [Task 01: Recover retained cancelled candidates](task-01-cancelled-recovery.md)

This can be implemented before speed/results. If previous native cleanup cannot
be proved, recovery must block; the separate speed fix is not a permission to
ignore lingering actors.

## Verification

Use the toolchain environment documented in
[the existing local plan](../reliable-speed-results/plan.md#verification-commands-and-environment),
from repository root. Exact implementation commands are in the work order.
Design-package gates are catalog, owning-system spec lint, local document
coverage, reference/encoding validation and `git diff --check`.

## Verification results

Implementation and candidate checks recorded on 2026-10-20:

- The 17 top-level `TestCancelledRecovery*` tests passed in targeted race batches,
  covering preserved partial source, admission, round budgets, restart guards,
  resource/evidence changes and uncertain cleanup. Browser RED first proved the
  missing action; a second RED exposed the interrupted-pending display trap.
- `node scripts/reliable/desktop-browser-check.cjs`: 43/43 passed, including
  desktop/390 px checking, error, retry, explicit continuation and no passive work.
- Public-doc tests: 62/62 passed; the public validator checked 47 pages.
- `go -C apps/backend vet ./internal/reliable`: passed in the implementation run.
  `golangci-lint run ./internal/reliable --timeout 4m`: zero issues.
- The isolated 1.9.1 candidate runtime passed with zero mutation requests,
  unchanged empty task storage, native model defaults, original assets, reachable
  assignment drawer and no 390 px horizontal overflow or browser errors.
- The first full race run exposed an existing 20-second test-helper deadline.
  It was raised to 60 seconds; the affected automatic-check regression then
  passed in isolation (`19.464s`). This does not establish a full-suite pass.
- Independent implementation review identified candidate junction ownership
  and stopped-service owner/PID proof gaps. Final correction verification must
  be included with the final race result; missing-owner proof must fail closed.
- Full reliable race suite: **The first complete-package attempt ended at its 15-minute total limit after 137 top-level passes, one old 20-second helper timeout and six opt-in skips. The corrected helper and every previously unfinished test passed in a 41-pass/two-skip completion race batch (204.791s); the junction boundary and process cleanup tests were included. The final 13-test review-correction race batch passed in 19.466s, including durable cleanup failure invalidation, retry/restart guards and junction rejection. Across the recorded race batches, 152 distinct top-level tests passed and eight opt-in checks were skipped; deployment separately enabled the legacy seal and retained-candidate checks. This is segmented regression evidence, not a successful single full-suite run.**.
- Local deployment: **Deployed 1.9.1 to E:/小布丁 with an idle-only service replacement and backup. The closed database copy preserved all eight task records and verified one old sealed candidate without applying it. The actual stopped task passed retained-source/history preflight without recovery or dispatch. Production desktop/phone runtime checks passed with zero mutation requests, identical task records, native settings and shortcut hashes. The real task remains cancelled at 1/3 rounds; two rounds remain.**. All eight production tasks remain resting and preserved.
  No production task recovery, native paid call or automatic acceptance is
  authorized by validation or installation.

Package checks passed: catalog validation covered 352 decisions and 1362
specifications; the four statuses, 12 local document links, UTF-8 without BOM
and local work-order coverage were valid. Full specification lint still reports
107 existing unrelated file-size findings, with zero reliable-collaboration
findings and no other rules. The package diff check passed. The separate
speed/results package remains pending.

## Risks

Project-ownership races, restarting during admission, partial sparse
initialization, scope/evidence drift and unknown legacy subprocess cleanup are
the main failure boundaries. Preserve files and block instead of silently
resetting or accepting them. A partial implementation can contain bugs; renewed
checks and fresh independent review remain required.
