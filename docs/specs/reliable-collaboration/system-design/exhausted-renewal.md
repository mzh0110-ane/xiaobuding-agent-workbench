---
status: current
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-RENEWAL-001
---

# Exhausted Batch Renewal Design

## Ownership and context

`apps/backend/internal/reliable` owns this vertical flow. Existing [desktop
workspace](desktop-workspace.md), [local loop](local-loop.md), [cancelled
recovery](cancelled-recovery.md), [resource
workspaces](large-resource-projects.md) and [native
assignments](domain-assignment.md) remain authoritative.

The existing `startLinkedBatch` admits accepted or inspected parents and starts
from the original project. Merely widening its eligibility would discard an
exhausted candidate's unaccepted edits. Renewal therefore adds an explicit
seeded-child path while preserving the existing accepted/inspection path.

| Acceptance | Design sections |
| --- | --- |
| AC-RELIABLE-RENEWAL-001.1/.4 | Explicit admission, lifecycle and restart |
| AC-RELIABLE-RENEWAL-001.2/.3 | Independent candidate and frozen inputs |
| AC-RELIABLE-RENEWAL-001.5 | Context and native assignment continuity |
| AC-RELIABLE-RENEWAL-001.6 | Evidence and cumulative acceptance |
| AC-RELIABLE-RENEWAL-001.7 | Workspace interaction and local delivery |

## Explicit admission

Expose protected `POST /api/v1/reliable/tasks/{id}/exhausted-next-batch` with
`messageRequest {id, kind: "requirement", content}` and a stable request ID.
Content is optional; empty content continues the existing goal and unresolved
findings. `Engine.StartExhaustedBatch` is a distinct admission branch. The
client cannot provide a source directory, base, seed digest, prior review
decision or round reset. Return the linked message and child ID after durable
admission; this means scheduled validation, not that validation passed or
implementation ran.

Under the established engine-mutex-to-Bolt order, recheck the parent is
unarchived, a modification task, has three admitted rounds and is resting in
exhausted, blocked, interrupted or cancelled state with retained source and an
established baseline. Reject active execution/questions and
acceptance/application. Source, cleanup and ownership proof runs before
dispatch, outside the admission transaction. A single Bolt transaction checks
all unfinished overlapping projects, claims the original project for the new
child, records `SuccessorTaskID` on the parent plus `ParentTaskID` on the
child, and saves the immutable requested requirement. The parent round and
source are unchanged. `supersededProjectTask` exempts an ancestor only after
checking that its existing same-project successor points back to it. Every live
descendant still blocks unrelated creation; do not weaken generic project
overlap checks.

Idempotency binds parent ID, message ID, kind and exact content. The same
request returns the same child. A different payload under the same ID fails. A
second request for an already linked exhausted parent resolves to that existing
child or a specific conflict; it cannot branch obsolete source. Descendant
batches are renewed from their current child, never from an ancestor. Parent
recovery, resume, apply or further mutation must honor the supersession/link
guard. Cancelling an obsolete parent is rejected, including while it is
reserved for seed validation; actual read-only answer cancellation remains
available. Future model settings on an obsolete parent direct the user to its
existing child. Archive visibility does not change verified lineage identity or
release an active descendant's project ownership.

## Independent candidate and frozen inputs

Reserve the child before background work. Optional `SeedPending *ExhaustedSeed`
and `SeedEvidence []string` preserve old serialization. Durable pending seed
metadata records the parent/child identities, expected original base, frozen
parent lifecycle and source evidence, requested requirement and attempt
identity. Long Git/file operations occur outside the mutex/transaction.
`kickQuestions` dispatches the child through the existing `ContinuePending`
path; `prepareExhaustedSeed` runs before ordinary prepare/audit. Reserve the
parent during source validation; an active parent question blocks before any
paid invocation. Prior owned process cleanup must be trusted through the
existing quiescence contract; no guessed process kill, active-map absence or
raw failed review establishes this proof.

Validate the parent using the retained-source ownership/history/resource
boundaries from cancelled recovery without changing its budget or current
state. Require the original to remain clean at its frozen base, including
physical source fingerprints, not Git status alone. Bind a pre-copy source
snapshot and resource digest. Ensure inherited differences are authorized by
the parent's cumulative approved scope and user limits.

Prepare a distinct task-owned child worktree at the original base with
`AutoSnapshot` disabled. Keep `Project` equal to the original project and
`Base` equal to the original base; the parent worktree must not become a new
original. Preserve the frozen original baseline as `Baseline`, separately from
the seed snapshot. Transfer only the verified source delta, including deletions
and previously untracked source. Use `os.Root` for bounded source access and
never stage the parent. Enforce existing path/link/size boundaries and confirm
the resulting child snapshot matches the seed. Avoid hard links or writable
sharing between the parent and child. Never reset an ambiguous dirty child on
retry. Apply authorized file deletions deepest-first before creating new files,
so file/directory shape changes remain valid. Remove only empty ancestor
directories of authorized deletions, without recursive deletion; ignored or
unexpected contents must be retained and block replacement. Source staging
files live under controller-owned `evidence/{child}/seed-staging`, outside the
worktree, and are atomically renamed through the controller `os.Root`. An
interrupted staging file therefore cannot pollute a source snapshot on retry.

Resource mode retains the original frozen manifest identity and source/resource
partition. The child gets its own validated manifest/index/preparation marker;
the marker must name the child evidence path. Validate frozen resource bytes
and staged/resource guards as applicable. Do not duplicate the resource
library, hydrate all inputs, silently recategorize frozen files or promote
materialized resources into writable source. Copying a parent metadata
directory verbatim cannot establish child ownership.

The pending descriptor survives each partial operation. On restart, ordinary
resume revalidates the admitted seed and child before dispatch. Retry may write
only paths still equal to the original baseline or already equal to the
intent-bound desired bytes. Unknown partial drift blocks while preserving both
copies. Revalidate the source chain on every new child execution even after
successful transfer; clearing the pending field cannot bypass the obligation.
Validate both the child canonical task-owned path/branch and its original-base
baseline before every pending copy, rather than trusting an existing worktree
path. Bounded read-only ancestry validation rejects cycles, lost receipts and
altered ancestor source while allowing archive visibility changes.
Resource availability can grow without changing the frozen source identity;
intent comparison separates availability bookkeeping from source hashes.
Cancellation wins over stale transfer publication. No startup or GET execution
is permitted. Completing transfer alone consumes no implementation round.

## Context and native assignment continuity

Use `continuationPlan` and the existing saved-future-plan semantics, validating
catalog compatibility and review-role restrictions. Preserve the frontend,
backend, planner and independent Codex reviewer choices, model and effort.
Reuse installed native Codex/Harness settings, skills and plugins without new
authentication or parallel configuration. Domain assignments keep the existing
Codex-only independent-review policy; legacy role assignments retain their
existing compatibility rules.

Generate a bounded linked context containing previous task/goal, user
requirement history, source seed identity, reports and unresolved findings.
Mark prior failed review findings as unresolved historical input. Do not copy a
raw review as the child's current `Review`, checks, dispatch success or passing
`EvidenceHash`. Keep complete old history available through parent links,
rather than recursively embedding the full ancestor chain. Preserve authorized
cumulative changed paths for the inherited delta, then extend them only through
the child's fresh audit.

## Evidence and cumulative acceptance

Write exclusive child-owned handoff intent and validated receipt files. Bind
parent and child IDs, original project/base, parent round and contract
identity, captured seed snapshot, resource digest, requirement/model revisions
and transfer result. Optional renewal fields keep their zero-value
serialization absent. Append `SeedEvidence` references to the child's evidence
digest only when nonempty so old task seals remain byte-compatible. Missing or
modified provenance prevents child acceptance; operational cleanup proof
remains separate from source seals. The seed intent also freezes an
`AncestryEvidenceHash` over raw ancestor contract, assignment, requirement,
dispatch, check-log, recovery and seed evidence. Later executions and
acceptance compare it through a bounded metadata-only walk. Acceptance does not
snapshot or materialize ancestor resource libraries. Newline or otherwise
byte-altered ancestor receipts cannot be silently adopted after child review.

After verified transfer, clear only the child's stale current acceptance fields
and run the existing fresh audit, `Store.StartRound`, implementation, actual
checks and independent review sequence. The child starts at round zero and has
at most three admitted implementation rounds. A later user-initiated renewal
creates another child with another bounded budget; no old batch round resets.

`changes.patch`, snapshot/scope checks and final human application remain based
on the original baseline. Acceptance must therefore apply the inherited plus
new delta, not just edits made after seed transfer. The parent failed decision
is never treated as a pass. Require the original still matches its expected
base before application, using existing rollback-safe apply behavior. No test
or installation may accept or execute the real user task.

## Workspace interaction and local delivery

Reuse `page.js`, `page-conversation.js` and existing task actions/composer. For
an exhausted resting modification show the used batch budget and explicit
`保留修改，开始下一批` action. That explicit action starts continuation directly,
using the current modification requirement draft if present; otherwise it uses
the existing goal and unresolved findings. The separate composer accepts optional
additional requirements before its explicit submit action. An unsent read-only
question draft remains preserved on the old task. Existing read-only
questions remain separate. Preserve inherited automatic-check semantics,
including zero-test detection, and generate child-owned check-plan evidence. Submitting switches to the linked child only after a
successful response. Show validation as preparation, truthful zero-of-three
budget, errors with retained-file reassurance and a parent link. The old task
keeps its three-of-three history and a link to the child. Its Agent drawer
becomes read-only and directs later model changes to the linked child; both
current and future model-update APIs reject obsolete-parent mutations.

Disable duplicate submission synchronously, preserve content on failure and
never start work because the page is opened. Phone controls stack with wrapping
text, visible focus and no horizontal overflow. Preserve the established theme,
avatar, background and collapsible Agent drawer. ASCII structure is in the
plan.

Local deployment is idle-only with a closed-database backup and rollback.
Verify all prior task rows, sealed evidence, native settings and shortcut
hashes are unchanged. Use disposable deterministic fixtures for full
renewal-through-human-acceptance proof; production preflight and browser checks
remain read-only.

## Implementation Plans

- [Exhausted batch renewal](../../../plans/reliable-exhausted-renewal/plan.md)
