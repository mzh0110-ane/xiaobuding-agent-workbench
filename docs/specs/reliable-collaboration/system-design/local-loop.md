---
status: current
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-LOOP-001
  - REQ-RELIABLE-LOOP-002
---

# Local Collaboration System Design

## Purpose and boundaries

`internal/reliable` implements the controller, CLI role runner, worktree/snapshot operations, durable state, HTTP authority boundary and Chinese local panel. `backendapp/reliable_routes.go` owns opt-in startup and shutdown wiring.

## Requirement mapping

| Requirement | Design section |
| --- | --- |
| REQ-RELIABLE-LOOP-001 | [Control flow](#control-flow) |
| REQ-RELIABLE-LOOP-002 | [Accepted local versions](#accepted-local-versions); [Failure and recovery](#failure-and-recovery) |

## Data and contracts

`<Kandev home>/reliable-settings.json` selects absolute data, CLI and configuration paths, the model and an optional demo project. The file stores no credential. Absence disables the extension. Controller-owned tasks reside in bbolt; round admission uses a write transaction. Evidence files are organized by task and round, and role execution attempts have unique directories.

`/api/v1/reliable` exposes task creation, inspection, evidence reading, resume, cancellation and acceptance. Creation accepts project, goal, commands and allowed paths, never workflow state. Mutations require a per-start browser capability and local same-origin access. Ordinary Kandev APIs do not access these records.

## Control flow

Audit scope validation canonicalizes one terminal `/` in `allowed_paths`, `frontend_paths` and `backend_paths` before checking authorization or overlap. Persist and dispatch these canonical scopes, while retaining the original native response as immutable execution evidence. Do not use filesystem cleaning to resolve traversal, normalize evidence filenames, or change the general `safeRelative` contract. Invalid scope errors include the field, index and original path. An explicit resume reruns the interrupted audit without admitting an implementation round until its plan is valid.

Automatic-enabled tasks complete [local project preparation](local-snapshots.md) before the worktree flow below. Managed recovery points establish the original-source baseline; they do not commit unaccepted Agent-produced results or relax subsequent drift checks. Accepted-result versions are a separate operation bound to the human-approved patch.

A task first creates a clean Git worktree. The auditor proposes a scope within user authorization. The controller persists the next round before starting a new implementer. Scope and source history are checked, verification commands execute, and a fresh reviewer receives the actual patch and logs. Source fingerprints surround read-only and verification stages. Only a passing candidate reaches human acceptance. A digest binds the frozen contract, audit, implementation report, check results and raw logs, source snapshots, review, decision and patch. Acceptance revalidates that digest and current source before applying the patch to an unchanged clean original.

## Accepted local versions

Human acceptance authorizes both application and a local Git version of the exact verified patch. The original Git root can be managed or external; this operation never writes the managed-root marker to an external repository. It does not authorize preparation to commit unrelated external changes, alter remote configuration or bypass the clean-original requirement before application.

Persist an acceptance-bound intent in optional `Task.LocalCheckpoint` metadata before mutating source/history. Bind it to task, canonical original root, verified base, patch identity and expected accepted source. Compare the real index and repository operation state before any history update. Keep command-local identity, disabled Git hooks/helpers and the sanitized Git environment used by controller Git operations. Commit only the expected accepted tree; do not blanket-stage the original directory or consume pre-existing user staging.

Retain the completed local commit identity as controller-owned task metadata and a bounded result receipt. These post-acceptance records do not rewrite historical reports, checks, frozen contracts or evidence seals. There is one acceptance operation per accepted task, not one receipt per page refresh or continuation attempt. Git stores the local version in the existing repository; no full-folder backup, new worktree, per-file backup or new project-root source file is required for this checkpoint.

The next explicitly requested linked batch verifies and uses the accepted version before ordinary preparation. The child retains the canonical original, native assignments, attachments and project dialogue through the existing continuation contract, with its own zero-of-three starting budget. Its fresh candidate and evidence remain subject to the normal worktree lifecycle; this checkpoint feature does not remove that existing candidate or promise zero disk usage.

For an already accepted task from an older release, derive a missing checkpoint only from retained acceptance authority. Revalidate its saved acceptance receipt, evidence digest and patch identity, reconstruct the accepted tree from its verified base and patch, and require the original to match that exact state. This recovery does not depend on the old candidate still being mutable-source-clean after acceptance. Additional source edits, manual staging, a changed base or missing evidence block this synthesis rather than becoming part of an accepted commit. A clean original already committed by the user can enter normal next-batch preparation directly; that path does not manufacture a historical acceptance receipt or assert that later user changes passed the old review.

Successful application and checkpoint completion transition the task to `accepted`. A checkpoint failure transitions it to `blocked` with its intent retained. `Resume` finishes that acceptance operation without dispatching an Agent or consuming an implementation round, including when the batch already used three rounds. The task summary shows the completed local commit identity above all workspace tabs, including Results. A failure retains the operation identity and localized cause, and an explicit retry rechecks source/history before continuing. Displaying the status or refreshing the page performs no Git mutation.

Legacy accepted tasks stay accepted while child preparation attempts their missing checkpoint before ordinary dirty-root admission. Failure blocks the child's preparation and preserves its zero-round budget. Recovery does not rewrite the parent's historic acceptance seal or claim that unrelated current changes were accepted.

## Failure and recovery

Failed checks and serious findings request rework within the three-round budget. Missing or ambiguous evidence blocks. Cancel and timeout terminate owned Windows process jobs. Startup marks interrupted executions as interrupted without replaying them or resetting rounds.

An acceptance interruption is reconciled against its persisted intent, actual source and Git history on explicit retry. An untouched original can still require application; an exact applied result can require only its local version; an already-created matching commit requires only completion of the retained receipt. Reconciliation must prove which boundary completed before advancing, never reapply a patch or duplicate a commit by assuming that a missing response means failure. Unknown or conflicting source/index/history remains blocked for inspection. No recovery path reruns an Agent or resets the implementation count.

## Security and limits

Snapshots include at most 30000 source entries, 256 MiB in total and 16 MiB per file. These bounded limits admit the current local Kandev source tree without excluding tracked files or weakening source coverage. The fingerprint format, ordering and evidence digest are unchanged. Admission failure never truncates the manifest. Private model conversations are not synchronized; the project, source hashes and retained evidence are the shared state.

Role subprocesses use their native Codex or Harness sandbox. See [native tool connections](model-handoff.md) for account separation, dispatch identity and sealing. Verification processes run project code under the current user, with API-key/token/secret environment variables removed. This is a workflow integrity boundary, not isolation from malicious local administrators or malicious project code. Known configured credentials are redacted from exposed reports and logs. Model read acknowledgements are not proof of semantic comprehension.

## Related decisions

- [Controller-owned collaboration](../../../decisions/2026-10-18-local-reliable-collaboration.md)
