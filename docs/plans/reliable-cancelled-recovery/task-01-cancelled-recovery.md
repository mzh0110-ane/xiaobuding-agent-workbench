---
id: "01-cancelled-recovery"
title: "Recover retained cancelled candidates"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-RELIABLE-RECOVERY-001
acceptance_criteria:
  - AC-RELIABLE-RECOVERY-001.1
  - AC-RELIABLE-RECOVERY-001.2
  - AC-RELIABLE-RECOVERY-001.3
  - AC-RELIABLE-RECOVERY-001.4
  - AC-RELIABLE-RECOVERY-001.5
  - AC-RELIABLE-RECOVERY-001.6
  - AC-RELIABLE-RECOVERY-001.7
system_design:
  - ../../specs/reliable-collaboration/system-design/cancelled-recovery.md
---

# Task 01: Recover Retained Cancelled Candidates

## Summary

Add the explicit stopped-task recovery action from HTTP/UI through atomic
admission, preserved-source validation and fresh audit. Keep the original task,
partial changes, remaining budget and independent acceptance gates.

## In scope

- Fail-first regression for current cancelled rejection and retained dirty
  candidate preservation; transactional overlap checks and synchronous reserve.
- Pending recovery guard across restart, source/resource ownership validation,
  original drift failures and uniquely bound recovery provenance.
- Optional legacy-compatible task fields, new receipt sealing, native assignment
  preservation, remaining-round UI and targeted desktop/phone coverage.
- Public/local instructions and idle-only local deployment with backup/receipt
  after all implementation checks, without executing the user's stopped task.

## Out of scope

New budgets, read-only retry, candidate reset/merge, automatic acceptance,
automatic project resume, acceleration/preview changes and remote publication.

## Acceptance

1. One explicit claim preserves a valid retained candidate and starts fresh
   audit/next round; conflicts, drift, wrong ownership, tampering, stopping and
   exhausted/read-only/archived states block without source loss or paid calls.
2. Cancel/restart cannot bypass pending validation; receipts bind the next seal,
   old failed reports remain historical, and 0/1/2/3 round fixtures retain budget.
3. Desktop/phone show the same explicit action, accurate checking/error/round
   states and no passive execution. Local delivery retains tasks/seals/settings.

## ASCII UI preview

[UI-RECOVER-01/02/03](plan.md#ascii-ui-preview), covering .1/.3/.5/.7:

```text
Desktop: Used 1/3. 2 rounds remain.
[Check copy and continue] [View working copy]

Phone:  Used 1/3. 2 rounds remain.
[Check copy and continue                   ]
[View working copy                         ]

Pending: [Checking ... (disabled)]
Error: specific cause, retain files, retry when resolved.
```

## Verification

From repository root after the linked plan's local environment setup. The
targeted commands exercise the implemented recovery and common lifecycle gates.

```powershell
go -C apps/backend test -trimpath -race ./internal/reliable -run 'TestCancelledRecovery|TestThreeRoundsSurviveRestart|TestConcurrentRoundAdmissionHasOneWinner|TestRestartFinishesPendingCancellationWithoutResettingRounds|TestPreparingRecoveryAndCancellationKeepZeroImplementationRounds|TestInterruptedOwnedInitializationResumesWithoutAdoptingExternalGit|TestSparsePreparationRecovery|TestAcceptanceRejectsMissingOrAlteredEvidence' -count=1 -v
node scripts/reliable/desktop-browser-check.cjs
go -C apps/backend vet ./internal/reliable
node --test scripts/validate-public-docs.test.mjs
node scripts/validate-public-docs.mjs
python -B scripts/list-docs.py validate
python -B scripts/lint-spec-files.py --all
git diff --check
```

Record actual results, including existing global spec-size debt, and require
zero owning-system/new findings. After targeted checks pass, run the reliable
package race suite once because recovery affects common admission/sealing. No
native model call or original-project mutation is needed for fixture proof.

## Implementation

Under `apps/backend/internal/reliable`, `cancelled_recovery.go` orchestrates the
reserved execution; `_store.go`, `_source.go` and `_evidence.go` own admission,
retained candidate/original checks and strict history/resource validation.
`engine.go` enforces validation per invocation even after receipt publication;
`evidence.go` binds new recovery provenance without changing legacy digests.
Process helpers establish owned Windows Job cleanup and separate operational
quiescence receipts. Stopped-service migration uses a copied closed database
while the installer holds the verified owner's process handle through receipt
publication; it cannot manufacture source verification or reseal old candidates.

The page, composer and scoped phone CSS expose explicit recovery and block
requirements during validation. The existing browser fixture and public task
guide cover the user flow. Disposable original/candidate fixtures exercise
partial edits, drift and resource tampering; the live original and candidate
are not test setup.

## Dependencies

None on the four pending speed/results work orders. Read current ownership,
initialization, resource and evidence tests; fail closed on unknown cleanup.

## Risks

Calling a clean-initialization helper on an established dirty candidate can
discard or reject legitimate partial work. Queuing without an atomic project
claim can race new task creation. Pending validation must survive restart.

## Parallelism

Sequential; this vertical slice changes shared task admission and sealing.

## Inputs

Paired recovery requirement/design, plan UI previews, existing cancellation,
round, initialization, archive, resource and sealed-evidence tests.

## Results

Implementation is complete; final corrected-suite sign-off and local rollout
remain explicit delivery gates:

- Targeted race batches passed all 16 top-level recovery tests. Browser tests
  passed 43/43 after fail-first missing-action and interrupted-guard proofs.
- Public-doc tests passed 62/62; validation covered 47 public pages. Backend vet
  passed and the reliable-package golangci-lint run reported zero issues.
- The isolated 1.9.1 candidate runtime passed: zero mutation requests, no
  browser errors, preserved native defaults/assets and no 390 px overflow.
- An existing full-suite 20-second helper timeout was raised to 60 seconds;
  the affected regression passed in isolation in `19.464s`. Full-suite and
  independent-review correction results are not inferred from this check.
- Full reliable race suite: **The first complete-package attempt ended at its 15-minute total limit after 137 top-level passes, one old 20-second helper timeout and six opt-in skips. The corrected helper and every previously unfinished test passed in a 41-pass/two-skip completion race batch (204.791s); the junction boundary and process cleanup tests were included. The final 13-test review-correction race batch passed in 19.466s, including durable cleanup failure invalidation, retry/restart guards and junction rejection. Across the recorded race batches, 152 distinct top-level tests passed and eight opt-in checks were skipped; deployment separately enabled the legacy seal and retained-candidate checks. This is segmented regression evidence, not a successful single full-suite run.**.
- Local deployment: **Deployed 1.9.1 to E:/小布丁 with an idle-only service replacement and backup. The closed database copy preserved all eight task records and verified one old sealed candidate without applying it. The actual stopped task passed retained-source/history preflight without recovery or dispatch. Production desktop/phone runtime checks passed with zero mutation requests, identical task records, native settings and shortcut hashes. The real task remains cancelled at 1/3 rounds; two rounds remain.**. Eight production
  task histories remain resting and must be preserved; no live recovery or
  paid native invocation was performed by these checks.
- Package checks passed: catalog validation covered 352 decisions and 1362
  specifications; all four statuses, 12 local document links, UTF-8 without BOM
  and one-work-order local coverage were valid. The package diff check passed.
  Full specification lint reports 107 existing unrelated file-size findings,
  zero reliable-collaboration findings and no other rule violations.
