---
status: current
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-RECOVERY-001
---

# Cancelled Task Recovery Design

## Ownership and implementation

`internal/reliable` owns this vertical task-state transition, admission checks,
native dispatch and Chinese workspace control. Existing [local loop](local-loop.md),
[resource workspaces](large-resource-projects.md) and
[desktop workspace](desktop-workspace.md) remain authoritative.

| Requirement | Design sections |
| --- | --- |
| REQ-RELIABLE-RECOVERY-001 | Admission, validation, execution, evidence, UI |

The dedicated recovery action admits cancelled tasks; ordinary resume still
accepts blocked/interrupted states. `Cancel` retains worktree/evidence and
cancels pending messages. Cancelled tasks release project admission, so recovery
reclaims the project transactionally before validation or dispatch.

| Implementation | Responsibility |
| --- | --- |
| `cancelled_recovery.go` | Reserved execution, validation orchestration and recovery receipt publication |
| `cancelled_recovery_store.go` | Atomic eligibility/project admission, pending intent and failed-guard state |
| `cancelled_recovery_source.go` | Candidate/original ownership, frozen baseline, retained scope and bounded preparation recovery |
| `cancelled_recovery_evidence.go` | Strict bounded history, model/requirement/dispatch and resource-reference validation |
| `process_quiescence.go`, `process.go`, `process_windows.go` | Owned cleanup observation and separate operational quiescence receipts |
| `engine.go`, `http.go`, `evidence.go` | Per-invocation guard, protected HTTP action and recovered-candidate sealing |
| `page.js`, `page-conversation.js`, `page.css` | Explicit action, pending composer guard and phone stacking |

The currently cancelled `next-9d81684e55eb5320fba4e40ec0` task is a useful
read-only format example at round 1. Do not recover it, dispatch its agents,
adopt its raw review or apply its changes during tests or deployment.

## Admission and API

`POST /api/v1/reliable/tasks/{id}/recover-cancelled` accepts an empty JSON body with
existing `localRequest` and `authorizedMutation` protection. The server resolves
all paths and task facts; a client cannot supply state, a round count or a new
candidate directory. Return 202 with the current task after successful claim;
return a stable conflict/error code when ineligible. The response means checks
were scheduled, not that recovery validation or native work succeeded.

`Engine.RecoverCancelled` takes `e.mu`, calls `reserveLocked` and a
single Bolt admission transaction in the established mutex-to-store order.
Inside that transaction, recheck cancelled, unarchived, modification task,
remaining rounds and absence of another unfinished overlapping project using
the same `projectsOverlap` and terminal-state policy as `createTaskTx`.
The task becomes queued with a unique `RecoveryPending` intent. Do not perform
long filesystem operations while holding the mutex/store transaction.

Release the reservation on every admission failure. After admission, the
background execution uses the already reserved context and `runTask`; it must
not call `RunTask` and reserve twice. Closing, cancellation and archive races
follow current engine ownership. Persisting queued plus pending intent claims
the project before another task creation can succeed. A second click gets an
already-claimed response without a second audit or additional round.

GET may advertise state-level eligibility and round arithmetic only. It must
not perform validation, Git commands, resource hydration or `TakeSnapshot`:
resource snapshots can update materialization bookkeeping.

## Retained-source validation

The recovery validators distinguish three
cases rather than routing every case through worktree creation:

| Persisted state | Validation and behavior |
| --- | --- |
| No candidate, no implementation admitted | Resume only controller-owned initialization/preparation; honor captured original base and initialization intent |
| Initial candidate exists but baseline not durably published | Use existing owned initialization recovery only if task branch/common directory/base/preparation identity all match; incomplete dirty/ambiguous initialization blocks |
| Established candidate plus baseline | Keep its dirty partial source; validate it without clean-worktree reset or re-creation |

For an established candidate verify canonical task-owned path under
`Root/worktrees/{id}`, no path/reparse substitution, Git common directory matching
the original, branch `xiaobuding/{id}`, `HEAD == Base`, baseline and frozen
contract identity, and original `requireClean`/expected-base/content policy.
Verify existing model-plan and requirement evidence chains rather than demanding
the initial model choice equal a legitimately updated selection. Retain
cumulative `ApprovedPaths`; check partial modifications remain within the
authorized cumulative scope and user limits. Do not run `AutoSnapshot` to
commit original edits made while the task was stopped.

In resource mode additionally verify manifest digest/base, marker/readiness,
resource path index, staged frozen-resource guard and acknowledged physical
resource identities. Do not reset sparse checkout, rewrite the frozen manifest,
hydrate every input or use a clean-initialization helper to erase partial work.
Retain the current resource-aware snapshot semantics and bounded source limits.

Validation occurs only after the explicit POST owns execution. It may update
controller-owned availability bookkeeping according to the existing resource
contract, but must not overwrite/delete original or candidate source. Preserve
the pre-validation partial-source fingerprint in the recovery receipt.

Validate previous process quiescence before reading and advancing the source.
An active/cancelling owner blocks. Honor known native/Git cleanup failures;
never treat mere absence of a current `e.active` entry as proof that an
ambiguously detached legacy process stopped. If ownership/cleanup cannot be
established from the service instance and owned process lifecycle, report
`recovery_process_unknown` and retain the candidate. Do not kill arbitrary
processes by guessed PID or executable name. The separate
[execution efficiency work](execution-efficiency.md) improves native cleanup
evidence, but is not permission to manufacture completion for this task.

### Operational process quiescence

Windows commands start suspended and join the owned kill-on-close Job Object
before execution. Cancellation and normal completion terminate/wait for the
owned job and close it. The invocation tracks every owned cleanup; a failed or
unconfirmed cleanup cannot publish a successful operational receipt.

`process_quiescence.go` writes exclusive bounded receipts below
`Root/quiescence/{task-id}/{runs-hash}.json`, binding the task, base and exact
dispatch history. The normal method is `owned-process-cleanup`. These files
are operational prerequisites, outside the task's sealed source evidence;
they are not recovery validation receipts, source verification, review results
or acceptance authority. Legacy seals do not change when a cleanup receipt is
created. Unknown native histories fail with `recovery_process_unknown`.

An explicit Windows deployment migration can attest a stopped legacy service
using `quiescence_deployment_windows_test.go` against a copy of its closed
database. The installer first verifies the recorded executable and holds its
actual process handle before stopping it; the handle remains held until all
receipts are published, preventing PID reuse during the proof. The helper
verifies that exact owner has exited and publishes `windows-service-exit`
operational receipts bound to the copied histories. Missing, inaccessible or
unconfirmed owner identity must fail closed. A guessed PID, an empty active
map or a raw failed report cannot authorize migration.

Migration does not execute a task, edit its source, change stored task rows or
reseal old evidence. It is an explicit stopped-service installation operation;
startup, GET and ordinary recovery requests never manufacture this proof.

## Execution and restart

`Task` has optional `RecoveryPending` and `RecoveryEvidence` fields,
omitted when absent. The pending descriptor contains attempt identity, original
round, intent reference and prior reason. Old tasks and seals keep their
serialization and behavior.

`runTask` processes the recovery guard before normal preparation or audit
whenever either `RecoveryPending` or nonempty `RecoveryEvidence` exists.
`Store.Recover` can turn an interrupted queued task into interrupted, but must
preserve that recovery obligation. A successful receipt only satisfies it for
the current reserved execution in memory, never for a later invocation. Thus
the existing resume path revalidates even after a crash following receipt
publication/pending-marker clear, during audit, or during later implementation.
No dispatch is allowed until this invocation's validation receipt is durable.
Startup never auto-runs recovery. Repeat validation with uniquely linked
attempt evidence, never overwrite history or reset the round; one uninterrupted
execution need not re-run this guard for each automatic rework round.

On successful validation, promote any valid saved future plan through the
existing `promoteFuturePlan`/evidence path. Append the receipt reference, clear
the pending guard and clear stale current `EvidenceHash`, `Snapshot`, `PatchHash`,
`Review`, `Checks` and required-read projection. Preserve baseline, original
contract, requirements, previous `Audit` needed for cumulative scope,
`ApprovedPaths`, sessions, dispatches and all prior-round files.

Then run the existing audit -> `Store.StartRound` -> implementation -> actual
checks -> fresh review sequence on the retained candidate. A cancelled admitted
round remains spent. New work starts at the next round; audit alone adds none.
Do not use failed dispatch report existence as trusted audit/review completion.

Failure before validation completes returns to cancelled with a visible
recovery reason, retaining pending/attempt history as needed. Cancellation wins
over stale success. Never leave a failed guard as an ordinary blocked task that
can bypass recovery validation. Reconcile only this target's orphan dispatch
bookkeeping through the existing recovery semantics; do not globally invoke
`Store.Recover` or invent successful end reports.

## Evidence compatibility

Write uniquely named intent and validated receipt files under this task's
evidence root; failed guards retain their task reason and activity. A validated
receipt binds original/candidate/base identities, admitted
round, requirement/model revisions, pre-audit source and resource fingerprints,
previous state and recovered attempt. Keep prior artifacts immutable.

Append nonempty `RecoveryEvidence` references in `evidenceDigestSnapshot` for
new recovered candidates only. Tasks without recovery references produce the
same hash as before. A missing/altered receipt prevents the next candidate's
acceptance. Preserve failed/raw reviewer logs as history; do not promote them.
`acceptanceEvidence`, current revision checks and explicit application remain
unchanged. Do not regenerate or reseal the user's old pending demos.

## UI and interaction

Use the current task action row in `page.js`, with the existing primary button
and `perform` error pattern. Label the action `检查副本并继续`; show `已用 1/3
轮，还可修改 2 轮` for the observed case. While claimed/validating, display
`正在检查保留的副本` and disable duplicate action. Full eligibility remains
server-authoritative; missing files, original drift or project conflicts produce
specific Chinese recovery instructions without destructive reset advice.
An interrupted pending guard exposes the existing explicit continuation action
instead of pretending that validation is still running. It retains the pending
obligation and blocks requirement submission until validation succeeds.

Archived tasks first use existing restore visibility, which does not execute.
Read-only tasks retain report/question behavior and no modification recovery
button. At round three show the exhausted reason and file/evidence access.
Do not enable requirement submission on cancelled tasks until recovery succeeds;
after recovery the existing supplement/next-audit behavior applies. Previously
cancelled queued questions are not silently replayed.

Desktop keeps the compact action row; phone wraps buttons and explanation
vertically. Reuse current theme, avatar, portrait reservation, Agent drawer,
keyboard/focus and error accessibility. No overall redesign or new dialog is
required. Keep task selection and messages stable through polling and failure.

### Process proof invalidation

A failed owned cleanup persists a nonempty optional `ProcessCleanupEpoch` on
that task. Operational receipts bind this epoch as well as the canonical run
history, so a prior Git-only receipt cannot validate a newer cleanup failure.
Zero epochs keep legacy receipt paths and serialization unchanged. Repeated
recovery claims carry an unresolved prior cleanup reason through cancellation
and restart; a failed check cannot be cleared by clicking again. Installer proof
requires opening the retained, signalled owner handle successfully; a missing PID
is insufficient. Read-only deployment preflight checks the actual retained
candidate using a closed database copy, without recovery or native dispatch.

## Implementation Plans

- [Cancelled task recovery](../../../plans/reliable-cancelled-recovery/plan.md)
