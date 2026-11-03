---
id: "01-exhausted-renewal"
title: "Continue exhausted work in a preserved child"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-RELIABLE-RENEWAL-001
acceptance_criteria:
  - AC-RELIABLE-RENEWAL-001.1
  - AC-RELIABLE-RENEWAL-001.2
  - AC-RELIABLE-RENEWAL-001.3
  - AC-RELIABLE-RENEWAL-001.4
  - AC-RELIABLE-RENEWAL-001.5
  - AC-RELIABLE-RENEWAL-001.6
  - AC-RELIABLE-RENEWAL-001.7
system_design:
  - ../../specs/reliable-collaboration/system-design/exhausted-renewal.md
---

# Task 01: Continue Exhausted Work in a Preserved Child

## Summary

Deliver explicit exhausted-batch renewal from the optional requirement composer
through atomic project admission and independently preserved source to fresh
verification. Keep all previous rounds/evidence and final human acceptance of
cumulative work.

## In scope

- Fail-first protected renewal API and preserved-seed regression tests.
- Atomic idempotent parent/child lineage, independent source/resource transfer,
  immutable evidence, per-invocation restart/cancel guards and cumulative patch.
- Native model/effort/context continuity, Chinese desktop/phone actions and
  composer, linked history and deterministic fixture checks.
- Public/local instructions, read-only live preflight and idle-only backed-up
  deployment preserving old states, seals, settings and desktop entry.

## Out of scope

Resetting old round counts, fourth automatic attempts, automatic renewal,
unverified review adoption, destructive candidate repair, original-project test
edits, paid native calls, automatic acceptance, remote publication and the
separate pending speed/results implementation.

## Acceptance

1. One explicit continuation request creates one linked child with the preserved source
   and new zero-of-three budget; parent remains three-of-three and unchanged in
   source/history. Conflicts, drift, unknown cleanup and tampering block safely.
2. Restart/cancel cannot bypass seed validation or admit competing writers;
   fresh audit/checks/Codex review seal inherited plus new changes for explicit
   human acceptance, including repeated sequential batches.
3. Desktop/phone support explicit composer submission, duplicate disable,
   failure/retry/content preservation and parent/child links. Local deployment
   retains all prior task records, sealed candidates, native settings and entry.

## ASCII UI preview

[UI-RENEW-01/02/03](plan.md#ascii-ui-preview) maps .1/.3/.4/.7:

```text
Desktop: This batch used 3/3. Project work can continue.
[Preserve changes, start next batch] [View working copy]
Optional next requirements: [ ...                         ]
[Start next batch]

Phone:   This batch used 3/3. [Preserve and start next                        ]
[View working copy                              ] Optional next requirements: [
...                    ] [Start next batch                               ]

Preparing: [Preparing ... (disabled)] New batch 0/3.
Error: specific reason, retained copies/content, retry when resolved.
Parent: 3/3. [Open next batch]  Child: 0/3. [Previous batch]
```

## Verification

Use the existing local Go/CGO toolchain environment documented in [the
speed/results
plan](../reliable-speed-results/plan.md#verification-commands-and-environment),
from repository root. Native calls and production mutation are prohibited
during validation. Scope any long race run to the changed package; do not
overlap full suites or claim segmented evidence is a single full-suite pass.

```powershell
go -C apps/backend test -trimpath -race ./internal/reliable -run 'TestExhausted|TestRenewal|TestSnapshotHandlesFileDirectoryTransitions|TestMessageGuardsRetainTheWorkBatchLimit|TestInspectionStartsLinkedModificationBatchWithoutResettingHistory|TestCancelFollowupPreservesCompletedInspection|TestConcurrentRoundAdmissionHasOneWinner|TestThreeRoundsSurviveRestart|TestAcceptanceRejectsMissingOrAlteredEvidence' -count=1 -v
node scripts/reliable/desktop-browser-check.cjs
go -C apps/backend vet ./internal/reliable
Push-Location apps/backend
try { & 'E:\小布丁\tools\golangci-lint-2.9.0-windows-amd64\golangci-lint.exe' run ./internal/reliable --timeout 4m } finally { Pop-Location }
node --test scripts/validate-public-docs.test.mjs
node scripts/validate-public-docs.mjs
python -B scripts/list-docs.py validate
python -B scripts/lint-spec-files.py --all
git diff --check
git status --short -- docs/plans/reliable-exhausted-renewal
```

After focused checks pass, run package lint and one bounded common-lifecycle
race regression because shared admission/evidence paths changed. Record exact
commands, runtime versions, results and owning-spec findings; complete the
deployment receipt only after read-only runtime verification of the installed
binary.

## Files likely touched

- `apps/backend/internal/reliable/continuation.go`, `conversation.go`,
  `requirements.go`, `store.go`, `engine.go`, `evidence.go` and new renewal helpers.
- `apps/backend/internal/reliable/page.js`, `page-conversation.js`, `page.css`.
- New renewal Go tests and `scripts/reliable/desktop-browser-check.cjs`.
- `docs/public/tasks-and-workflows.md` and this requirement/design/plan package.

## Dependencies

Existing local conversation, retained-candidate validation, resource workspace,
owned process cleanup, native assignment and human acceptance contracts. No
dependency on the four pending speed/results work orders.

## Risks

Original-base versus seed-baseline confusion, atomic lock transfer, partial
resource transfer, ancestor resumption and stale publication after cancellation
can lose work or bypass verification. Preserve files and reject ambiguity.

## Parallelism

Sequential vertical slice. Bounded disjoint UI/document work can run against a
fixed server contract; shared task schema/admission and evidence are integrated
before final verification.

## Inputs

- Paired renewal requirement/design and linked plan UI previews.
- Existing conversation, recovery, resource, round-limit and acceptance tests.
- User authorization to implement and deploy the completed continuation flow.

## Results

Implementation, corrected targeted verification and local delivery are
complete. The requirement is active, its design current, the plan implemented
and this work order done. The linked plan records the review receipt, exact
failure/correction history, backup and installed executable hash.

- Fail-first API/copy/parent-action/ancestor-proof cases reproduced the missing
  continuation and changed boundaries before their corrections. The final
  recorded race batches establish 47 distinct top-level passes and one source
  symlink host-privilege skip. A prior manual lifecycle fixture and oversized
  repeated resource fixture failed/timed out; the corrected 9-test completion
  batch and 22-test common-lifecycle batch passed (`107.781s`, `190.426s`). This
  is segmented evidence, not a single passing whole-package invocation.
- Browser scenarios passed 52/52; page-state checks 7/7; public-doc tests 62/62
  and validation of 47 pages. Go vet passed, reliable-package lint reported
  zero issues, and the local Windows `fts5` build passed.
- Actual retained-candidate preflight passed in `12.22s` without child creation,
  agent dispatch or application. All eight existing task records were retained.
- Catalog, local-link/no-BOM checks, local PR coverage and package diff checks
  passed. Full spec lint retains 107 unrelated existing size findings and zero
  owning-system findings. Detailed mappings and logs are in the linked plan.
- Local deployment/runtime result: Deployed **1.10.0** to `E:/小布丁` with
  backup `E:/小布丁/backups/upgrade-1.10.0-20261020-211920`. The closed database
  preserved eight records and the old pending seal (`1.02s`), without applying
  it. Candidate and production desktop/390 px checks passed with zero mutation
  requests. Production task JSON is byte-identical; native settings and shortcut
  hashes match. The real task remains exhausted at 3/3 with no successor; the
  explicit renewal button is visible/enabled and was never clicked. Exact
  installed bytes and operational receipts are recorded in the linked plan.

No original user-project mutation, paid native call, automatic continuation or
acceptance, plugin/skill installation, Git commit or remote publication is part
of this validation.
