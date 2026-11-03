---
id: "02-preparation-reuse"
title: "Remove duplicate preparation and expose timing"
status: pending
wave: 2
depends_on:
  - 01-native-completion
plan: "plan.md"
requirements:
  - REQ-RELIABLE-EFFICIENCY-001
acceptance_criteria:
  - AC-RELIABLE-EFFICIENCY-001.3
  - AC-RELIABLE-EFFICIENCY-001.4
  - AC-RELIABLE-EFFICIENCY-001.5
  - AC-RELIABLE-EFFICIENCY-001.6
  - AC-RELIABLE-EFFICIENCY-001.7
system_design:
  - ../../specs/reliable-collaboration/system-design/execution-efficiency.md
---

# Task 02: Remove Duplicate Preparation and Expose Timing

## Summary

Measure preparation and stage overhead, reuse one verified admission inventory
and same-boundary snapshots, and show the actual operation/elapsed duration.
Preserve every actor, resource and acceptance integrity boundary.

## In scope

- Attempt-scoped prepared descriptors, boundary-scoped immutable snapshot input
  and digest-verified parsed-resource-profile reuse.
- Invalidations for source/index drift, hydration, actors, cancellation/restart
  and verification; no mtime-only trust or persistent stale snapshot cache.
- Bounded diagnostic sidecars/GET and compact current-operation status.

## Out of scope

Lower model efforts, concurrent domain writers, skipping tests, persistent
resource-result migration, whole-project scans on polling and preview staging.

## Acceptance

1. Counted fixtures perform one admission inventory per unchanged attempt and
   no duplicate equivalent snapshot inside one actor-free boundary, while all
   actor/check/review/application boundary validations still occur.
2. Same-size/same-time source/resource edits, original drift and changed evidence
   invalidate reuse and retain existing failure behavior and legacy seals.
3. Timing records expose real cold/warm phase spans and actual native cleanup;
   the UI current-operation row is truthful, responsive and free of side effects.

## ASCII UI preview

Task status row supplement to [UI-01](plan.md#ui-01-desktop-task-result-candidate-awaiting-acceptance):

```text
+-----------------------------------------------------+
| Independent review | elapsed 10:05                   |
| Native report returned; finishing owned cleanup ... |
| > Phase durations (measured, not estimated)           |
+-----------------------------------------------------+
```

Map to AC-RELIABLE-EFFICIENCY-001.3. Same stacked row on phone; no additional
fixed overlay or whole-workflow percentage. Preserve the background and drawer.

## Verification

From repository root after the plan's environment setup; new test names planned:

```powershell
go -C apps/backend test -trimpath -race ./internal/reliable -run 'TestReliableTimingProjection|TestHandoffSnapshotReuse|TestPreparationInventoryReuse|TestReuseDetectsTampering|TestLargeResourcePreparation|TestSparsePreparationRecovery|TestResourceRequestDispatch|TestAcceptanceRejectsMissingOrAlteredEvidence' -count=1 -v
go -C apps/backend test -trimpath ./internal/reliable -run '^$' -bench 'BenchmarkReliablePreparation' -benchtime=3x -count=1
node scripts/reliable/desktop-browser-check.cjs
```

## Files likely touched

Under `apps/backend/internal/reliable`: `engine.go`, `workspace.go`,
`resource_inventory.go`, `resource_workspace.go`, `resource_snapshot.go`,
`resource_dispatch.go`, `dispatch.go`, `stages.go`, `domain_execution.go`,
`page-activity.js`, `page.js`, proposed timing/reuse files and focused tests;
`scripts/reliable/desktop-browser-check.cjs` for the status projection.

## Dependencies

Task 01 completion diagnostics. Do not merge reuse across the new cleanup
boundary until all owned processes have stopped.

## Risks

Repeated manifest parsing is only part of cost. Combining snapshots across an
actor or trusting index/mtime would hide real source/resource changes.

## Parallelism

Sequential; snapshot/dispatch/evidence consumers share invariants.

## Inputs

Execution-efficiency design, resource preparation design, current snapshot,
domain handoff, acceptance and activity tests.

## Results

Pending.
