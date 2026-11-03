---
id: task-02-project-continuity
title: Editable project continuity and operational recovery
status: completed
wave: 2
depends_on:
  - task-01-navigation
plan: plan.md
requirements:
  - REQ-RELIABLE-WORKBENCH-002
  - REQ-RELIABLE-WORKBENCH-003
  - REQ-RELIABLE-WORKBENCH-005
acceptance_criteria:
  - AC-RELIABLE-WORKBENCH-002.1
  - AC-RELIABLE-WORKBENCH-002.2
  - AC-RELIABLE-WORKBENCH-002.3
  - AC-RELIABLE-WORKBENCH-002.4
  - AC-RELIABLE-WORKBENCH-003.2
  - AC-RELIABLE-WORKBENCH-003.3
  - AC-RELIABLE-WORKBENCH-003.4
  - AC-RELIABLE-WORKBENCH-005.1
  - AC-RELIABLE-WORKBENCH-005.2
  - AC-RELIABLE-WORKBENCH-005.3
system_design:
  - ../../specs/reliable-collaboration/system-design/workbench.md
---

# Project Continuity and Operational Recovery

## Outcome and scope

Extend the existing ledger with editable pending work, human-accepted dependency
gates, bounded attributable handoff, explicit recovery guidance, and advisory
storage settings. Reuse the current database, candidate, and native tools.
Exclude automatic acceptance, arbitrary scope enlargement, hard cache quotas,
history deletion, and extra per-message memory files.

## Acceptance

1. Pending edits are revision-protected; assigned proof cannot be rewritten;
   cyclic/unknown dependencies fail and dispatch atomically requires accepted
   prerequisites. Errors retain editor drafts and original full requirements.
2. Handoff claims remain attributable and bounded; operational planning retries
   retain the frozen source/requirement identity and shared time window, including
   a fresh identity check on later explicit continuation.
3. Optional notification preferences and explicit space settings preserve native
   homes and retained data. Low/unknown free space blocks new allocations while
   evidence reads remain usable; category budgets are labeled advisory.

## ASCII UI preview

UI-WORKBENCH-03, Project excerpt from [the combined preview](plan.md#ascii-ui-preview):

```text
Pending item: short title | status | milestone
[Full requirement / acceptance]
[Edit] [Up] [Down]    Prerequisites: accepted before start
Expanded editor: original draft / revision conflict / explicit retry
Handoff: version | reported facts | inference | unresolved | source links
Space: advisory budgets | actual free space | [Inspect] [Save]
```

Phone rendering uses the same vertical order. Editors/dependency choices build
only when expanded; invalid, stale, or assigned states must remain explicit.
This maps to REQ-RELIABLE-WORKBENCH-002/.003/.005.

## Files and dependencies

Project ledger/actions/edit/dependency/handoff modules, recovery guidance and
audit retry modules, storage policy/budget modules, and the project/budget page
modules. Existing linked-batch transactions and acceptance are the dispatch
authority; guidance and display state cannot replace them.

## Verification

```sh
# From apps/backend, serialized on the local host.
go test -race -trimpath -p 1 ./internal/reliable -run '^TestProject' -count=1
go test -race -trimpath -p 1 ./internal/reliable -run '^Test(Audit|Recovery|StorageBudget)' -count=1
# From the repository root.
node --test apps/backend/internal/reliable/page-project.test.cjs
node --test apps/backend/internal/reliable/page-budget.test.cjs
node scripts/reliable/desktop-browser-check.cjs
```

Browser coverage must include retained failed/revision-conflict edits, lazy
editors, dependencies, real task state gates, readable handoff provenance,
notification opt-in, and saved/failed space settings. Include independent review
of failed read-only planning followed by explicit continuation.

## Results

The final project race group passed in 92.188 seconds after the accepted-commit
and truncation regressions were reproduced and fixed. All 11 project page logic
tests and its syntax check passed. See staged project validation receipts.
This is targeted project coverage, not a complete backend-suite claim.

Recovery/source-drift checks and independent final review passed after receipt
integrity and fixed writer-scope regressions. Explicit replanning requires the
current confirmed hash, preserves quality rounds, and does not grant future
writers a broader scope. Historical confirmation receipts remain sealed across
later rounds and requirements.

The combined 27-check browser group covered retained conflicting project edits,
lazy editors, budget-toggle recovery, and human confirmation before replanning;
54 combined page-logic checks passed. Budget limits are advisory, with a separate
free-space admission floor. Upgrade compatibility and deployment passed; all 13
raw task records and native connection settings were preserved. The existing
user business task has not been resumed or accepted by this work order.

After the final style refactor, 19 targeted race checks passed in 135.976 seconds,
including planning/audit retry and scope checks. Reliable-package Go vet passed
and incremental lint reported zero issues.
