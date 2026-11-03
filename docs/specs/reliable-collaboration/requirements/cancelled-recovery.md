---
status: active
system: reliable-collaboration
created: 2026-10-20
owners:
  - local-distribution
---

# Cancelled Task Recovery Requirements

## Overview

Users can explicitly continue a stopped modification task using its retained
working copy and remaining implementation rounds. Reliable collaboration owns
the task lifecycle, source identity and evidence that make this possible. This
extends the [local loop](local-loop.md) and
[desktop workspace](desktop-workspace.md), without changing ordinary upstream
Kandev tasks or silently restarting work.

## Requirements

### REQ-RELIABLE-RECOVERY-001: Explicit retained-candidate recovery

**Intent:** Let a novice stop work and continue later without losing partial
changes or weakening verification and round limits.

#### Acceptance criteria

- **AC-RELIABLE-RECOVERY-001.1:** An unarchived cancelled modification task with
  fewer than three admitted implementation rounds shall offer an explicit
  recovery action showing used and remaining rounds. Selecting, polling,
  reloading, archiving or restoring visibility shall not recover a task.
  Read-only tasks, stopping/running tasks and tasks with no remaining rounds
  shall not use this modification recovery action.
- **AC-RELIABLE-RECOVERY-001.2:** Recovery shall preserve partial candidate
  source, task identity, original goal and requirement history, effective native
  assignments, cumulative authorized scope, prior sessions and historical
  evidence. It shall not reset/recreate an established working copy, discard
  changes, apply the candidate or adopt a raw failed review as verified work.
- **AC-RELIABLE-RECOVERY-001.3:** Before any model invocation, recovery shall
  verify task ownership, original-project identity/cleanliness, candidate
  history, frozen contract and applicable source/resource integrity. Changed
  originals, missing/mismatched ownership/evidence, unsupported initialization,
  source-scope violations or ambiguous process cleanup shall produce a specific
  failure while retaining files and historical evidence.
  Process-cleanup receipts establish operational quiescence only; they shall
  not substitute for sealed source, actual checks or independent review.
- **AC-RELIABLE-RECOVERY-001.4:** At most one execution may claim recovery.
  Concurrent recovery clicks, cancellation, archiving, shutdown and creation of
  another unfinished overlapping-project task shall not admit conflicting
  writers. Recovery shall refuse an unfinished task that already owns an
  overlapping project, even when the old cancelled task released that project.
- **AC-RELIABLE-RECOVERY-001.5:** Recovery shall retain the admitted round count,
  re-audit current candidate source and admit the next implementation round only
  through the existing round gate. A task stopped at round one has two rounds
  remaining; stopping during preparation or audit shall not itself consume a
  round. Three used rounds shall prevent recovery, including after restart.
- **AC-RELIABLE-RECOVERY-001.6:** Recovery shall append uniquely identified
  provenance and invalidate old current acceptance eligibility. Actual checks,
  a fresh independent review and explicit human acceptance shall still be
  required. Recovery interrupted by cancellation/restart shall revalidate its
  retained inputs before dispatch and never bypass that requirement through the
  existing interrupted-task resume action.
- **AC-RELIABLE-RECOVERY-001.7:** Desktop and phone shall expose the same recovery
  action, visible checking/error/retry states and remaining rounds. Duplicate
  clicks shall be disabled while a request is pending. Accessible names,
  keyboard activation, visible focus, wrapping error text and preserved task
  selection shall follow the current workspace. Read-only questions and new
  requirement messages retain their existing separate eligibility rules.

## Out of scope

Automatic restart, a fresh three-round budget for exhausted work, recovery of
cancelled read-only inspection, merging independent task candidates, accepting
partial work, deleting old evidence, changing logins/skills/plugins and
guarantees of zero model error are excluded. Missing or ambiguous candidates
remain available for manual inspection rather than destructive repair.

## Implementation Plans

- [Cancelled task recovery](../../../plans/reliable-cancelled-recovery/plan.md)
