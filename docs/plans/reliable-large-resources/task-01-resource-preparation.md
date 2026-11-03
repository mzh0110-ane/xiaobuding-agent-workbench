---
id: "01-resource-preparation"
title: "Prepare isolated resource-aware candidates"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-RELIABLE-RESOURCES-001
acceptance_criteria:
  - AC-RELIABLE-RESOURCES-001.1
  - AC-RELIABLE-RESOURCES-001.2
  - AC-RELIABLE-RESOURCES-001.6
system_design:
  - ../../specs/reliable-collaboration/system-design/large-resource-projects.md
---

# Task 01: Resource-Aware Preparation

## Summary

Allow bounded resource-heavy existing Git projects through preparation without copying the entire resource library or weakening ordinary source limits. Freeze all resource identities and preserve existing project/candidate ownership rules.

## In scope

- Failing integration reproduction using a committed tree above 30000 entries, including real WMF/PNG cases.
- Canonical bounded index/HEAD inventory, resource manifest, source accounting, no-checkout sparse worktree initialization and cancellation.
- Verified recovery intent and versioned resource descriptor; preserve existing worktrees and legacy preparation.

## Out of scope

Native calls, role resource requests, evidence sealing, resource editing, ordinary-folder policy expansion, production deployment.

## Acceptance

1. A large-resource fixture prepares with source/SVG/HTML present, deferred resources absent, all entries accounted for and original source/index/history unchanged.
2. Large all-source trees, staged/dirty roots, links/submodules, unsafe names, resource/source budget violations and cancellation fail without unowned cleanup or new model calls.
3. An owned interrupted sparse preparation resumes once; an ambiguous/modified candidate blocks. Existing small preparation remains unchanged.

## Verification

From `apps/backend`, with the documented local Go toolchain:

```powershell
go test -trimpath -race ./internal/reliable -run 'TestLargeResourcePreparation|TestLargeSourceStillRejected|TestSparsePreparationRecovery|TestWorktreeRejectsOversizedIndexBeforeCopy|TestSnapshotAdmitsLargeProjectAndRejectsExcessFiles' -count=1 -v
```

## Files likely touched

`workspace.go`, `engine.go`, `store.go`, proposed `resource_workspace.go`, `resource_workspace_test.go`, and focused existing limit fixtures under `apps/backend/internal/reliable`.

## Dependencies

None. Read the paired design, including the failed temporary hydration probe; do not mistake the draft for production authorization or test evidence.

## Risks

Git non-cone path escaping, case aliases, per-worktree configuration, source byte ceilings, large manifest response size and interruption ownership.

## Parallelism

Sequential; this task changes contracts consumed by Task 02. No subagent delegation is authorized.

## Results

Large-resource admission, immutable inventory, source/data/config classification, source limits, filter refusal, ignored output and clean checkpoint recovery are covered by passing race regressions. The actual original repository prepared 3263 mutable entries with 150081 deferred inputs; original index/HEAD/contents stayed unchanged.
