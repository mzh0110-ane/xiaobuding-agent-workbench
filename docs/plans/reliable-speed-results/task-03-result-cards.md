---
id: "03-result-cards"
title: "Show evidence-bound result cards and files"
status: pending
wave: 3
depends_on:
  - 02-preparation-reuse
plan: "plan.md"
requirements:
  - REQ-RELIABLE-RESULTS-001
acceptance_criteria:
  - AC-RELIABLE-RESULTS-001.1
  - AC-RELIABLE-RESULTS-001.2
  - AC-RELIABLE-RESULTS-001.3
  - AC-RELIABLE-RESULTS-001.4
  - AC-RELIABLE-RESULTS-001.5
  - AC-RELIABLE-RESULTS-001.6
system_design:
  - ../../specs/reliable-collaboration/system-design/task-results.md
---

# Task 03: Show Evidence-Bound Result Cards and Files

## Summary

Deliver a read-only result projection, compact result card and lazy file/diff
viewer. Preserve exact task/round/source provenance and expose partial artifacts
without claiming that they passed independent review or were applied.

## In scope

- Versioned result IDs and bounded projection/index, legacy single-writer and
  domain report adapters, check/review/patch provenance and tamper fallbacks.
- Exact artifact references; all individual log links; patch access independent
  of review success; source fingerprint matching on explicit file reads.
- Desktop result drawer, phone dialog, keyboard/focus, loading/error/retry,
  polling generation guards, archive/reload and existing folder actions.

## Out of scope

Starting servers, rendering active HTML on the reliable origin, automatic
acceptance, live file editing, invented outputs and historical report adoption.

## Acceptance

1. Result card/viewer truthfully handles awaiting acceptance, applied,
   inspection, cancelled/partial and previous-round evidence; old passes and
   orphan raw reports never become current success or application eligibility.
2. Each patch/file/check/review link resolves the exact recorded generation;
   missing, tampered, oversized or unavailable historical data yields a reason.
   Reads do not scan projects, run work or mutate seals/task state.
3. Desktop/phone and keyboard flows retain state through polling, show every
   check log, and prevent stale task responses, inaccessible close controls,
   injection and whole-page overflow.

## ASCII UI preview

Excerpt of [UI-01 through UI-03](plan.md#ascii-ui-preview), with the same labels:

```text
UI-01 desktop:
Task result [Candidate]                 [Changes -> drawer]
Summary | actual checks | accepted review | [Open copy]

UI-02 partial:
Task result [Cancelled / Partial]
Recorded checks; review not established. [Changes] [Raw evidence]

UI-03 phone:
Task result [Candidate]       -> [Back] File selector [Close]
Summary (wrapped)               [Diff] [Source]
[Changes] [Open copy]            Bounded read-only code pane
```

UI-01/02 map to AC-RELIABLE-RESULTS-001.1-.4/.6; UI-03 maps to .5.
Preserve existing portrait and Agent drawer. No preview start action until
Task 04 supplies a supported recipe; show the truthful unavailable/setup state.

## Verification

From repository root after plan environment setup. New test names are planned:

```powershell
go -C apps/backend test -trimpath -race ./internal/reliable -run 'TestResultProjection|TestResultRoundProvenance|TestResultLegacyDomainReports|TestResultArtifactBounds|TestResultReadHasNoSideEffects|TestAcceptanceRejectsMissingOrAlteredEvidence' -count=1 -v
node scripts/reliable/desktop-browser-check.cjs
```

## Files likely touched

Under `apps/backend/internal/reliable`: proposed `result_projection.go`,
`result_http.go`, `result_projection_test.go`, `result_http_test.go`,
`page-results.js`; existing `stages.go`, `domain_execution.go`, `http.go`,
`page.js`, `page-proof.js`, `page.html`, `page.css` and embedding fixtures.
`scripts/reliable/desktop-browser-check.cjs` owns rendered state coverage.

## Dependencies

Task 02 diagnostic/result boundary compatibility. Read current evidence and
domain report shapes before selecting a legacy adapter; no production migration.

## Risks

Raw report existence does not establish stage validity. Historical selectors
need explicit artifact identities despite StartRound clearing task-level results.
Live candidate bytes may no longer match a historical result.

## Parallelism

Sequential; touches the shared HTTP/embed/page composition used by Task 04.

## Inputs

Task-results design through presentation; plan UI-01/02/03; existing result
and evidence-browser tests; read-only current cancelled-task artifact shapes.

## Results

Pending.
