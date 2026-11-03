---
status: active
system: reliable-collaboration
created: 2026-10-20
owners:
  - local-distribution
---

# Exhausted Batch Renewal Requirements

## Overview

A project can receive a repeated sequence of user-requested work batches. The
three-attempt limit bounds one automatic batch, not the lifetime of the
project. An exhausted batch can contain useful unaccepted source changes;
renewal preserves those changes without representing them as verified work.
Reliable collaboration owns the source identity, linked batch lifecycle and
verification contract. This extends the [local loop](local-loop.md), [desktop
workspace](desktop-workspace.md) and [cancelled
recovery](cancelled-recovery.md).

## Terms

- **Renewal:** An explicit user request to start a separate batch from the
  retained source of a stopped batch that used its three implementation rounds.
- **Seed:** The retained unaccepted candidate and its immutable provenance.
  A seed is input to fresh investigation, never passing review evidence.
- **Original:** The project directory to which a verified cumulative patch can
  be applied only after explicit human acceptance.

## Requirements

### REQ-RELIABLE-RENEWAL-001: Continue exhausted work in a linked batch

**Intent:** Let a novice keep improving the same project after a bounded
automatic loop stops, without losing partial changes or weakening acceptance.

#### Acceptance criteria

- **AC-RELIABLE-RENEWAL-001.1:** An unarchived stopped modification batch with
  three admitted rounds and a retained candidate shall offer an explicit
  action to preserve changes and explicitly start the next batch. An optional
  additional user request extends the existing goal; leaving it empty retains
  the goal and unresolved findings. Selecting, polling, reloading, restoring
  visibility or restarting the service shall not renew work. Active, applying, awaiting-acceptance and
  read-only batches shall retain their existing separate actions.
- **AC-RELIABLE-RENEWAL-001.2:** A renewal shall preserve the old batch's round
  count, source, messages, reports and evidence. It shall create a linked child
  with a separate three-round budget beginning at zero, the same original
  project, and an independently owned candidate containing the retained partial
  changes. Repeated explicit renewals shall allow further batches; neither the
  original project nor the previous candidate shall be silently overwritten.
- **AC-RELIABLE-RENEWAL-001.3:** Before any model invocation, renewal shall
  verify original cleanliness and base, candidate ownership and source scope,
  applicable frozen resources, prior process quiescence and historical evidence.
  Missing, altered, redirected or ambiguous inputs shall block with a specific
  cause while retaining files. The transferred bytes shall match the verified
  seed, including deletions and untracked source; a changed seed shall not be
  silently recaptured or adopted.
- **AC-RELIABLE-RENEWAL-001.4:** Admission shall persist parent-child linkage
  and one overlapping-project owner atomically. Duplicate requests, competing
  child requests, new task creation, cancellation, archiving and restart shall
  not admit conflicting writers or duplicate children. A previous linked batch
  shall direct the user to its existing child rather than restart obsolete work.
  Interrupted transfer shall retain a durable validation obligation.
- **AC-RELIABLE-RENEWAL-001.5:** The child shall inherit effective native
  assignments and saved future preferences, verification settings, authorized
  cumulative scope, previous user context and unresolved review findings.
  Previously cancelled questions shall not silently run. New requirements and
  model changes shall follow existing safe-boundary semantics.
- **AC-RELIABLE-RENEWAL-001.6:** Renewal shall append immutable provenance
  binding parent, child, original base, seed source and resource identities.
  Its acceptance evidence shall bind that provenance and cumulative changes
  from the original base, including inherited changes. Fresh investigation,
  actual checks, a fresh independent review and explicit human acceptance shall
  still be required. Frontend/backend assignments shall use Codex review under
  the existing native-role policy; legacy assignments retain their contract. Failed or raw parent reviews shall remain historical
  findings, never a pass or acceptance authority.
- **AC-RELIABLE-RENEWAL-001.7:** Desktop and phone shall distinguish per-batch
  limits from continued project work, expose an optional requirement composer,
  visible preparation/error/retry states and links between batches. Duplicate
  submission shall be disabled while pending. Keyboard activation, visible
  focus, wrapping text, stable selection and no phone horizontal overflow shall
  follow the current workspace. Upgrading shall preserve existing task states,
  sealed candidates, native settings and desktop entry points.

## Out of scope

A fourth automatic attempt in one batch, automatic renewal, adopting unverified
review output, merging concurrent candidates, destructive conflict resolution,
remote publication, native login/skill/plugin installation, and guarantees of
zero model error are excluded. Existing accepted/inspection continuations and
cancelled recovery with remaining rounds keep their respective contracts.

## Implementation Plans

- [Exhausted batch renewal](../../../plans/reliable-exhausted-renewal/plan.md)
