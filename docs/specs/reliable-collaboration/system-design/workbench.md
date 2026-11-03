---
status: current
system: reliable-collaboration
created: 2026-10-26
owners:
  - local-distribution
requirements:
  - REQ-RELIABLE-WORKBENCH-001
  - REQ-RELIABLE-WORKBENCH-002
  - REQ-RELIABLE-WORKBENCH-003
  - REQ-RELIABLE-WORKBENCH-004
  - REQ-RELIABLE-WORKBENCH-005
---

# Long Project Workbench Design

## Ownership and compatibility

`internal/reliable` remains the controller for this local Windows extension.
The embedded HTML/CSS and small JavaScript modules share the existing controller
and task store; the workbench does not introduce another frontend framework,
agent backend, credential store, or evidence authority. The original
[project ledger and recovery design](long-projects.md) continues to own batch
continuation and acceptance.

Codex uses the existing ChatGPT login. DeepSeek Harness uses its existing account,
home, model catalog, skills, and extensions. Frontend/backend implementation is
sequential in one candidate. New independent reviews still use fresh Codex
sessions. Configuration availability is not proof of login or model service
availability. Settings saved during execution apply at the next eligible dispatch.

## Read projections and navigation

`workbench_http.go` adds explicit lightweight views while preserving legacy full
task endpoints for existing consumers:

| Read | Contract |
| --- | --- |
| `GET /api/v1/reliable/tasks?view=summary&offset=N&limit=N` | `tasks`, `total`, `next_offset`; default page size 200, maximum 200 |
| `GET /api/v1/reliable/tasks/{id}?view=workspace` | Selected task, snapshot identities without the full baseline map, no previous-context blob, latest 25 messages, history counts, recovery guidance |
| `GET /api/v1/reliable/tasks/{id}/messages?before=N&limit=N` | Older chronological messages, previous cursor, total and task update identity; default page size 25, maximum 200 |

Summary records whitelist current navigation fields, project identity, shortened
goal, and an active dispatch. They omit full baseline files, historical messages,
audit/review reports, and previous context. Canonical project-root identity comes
from the existing project ledger key, not a display name.

Conditional responses hash the redacted serialized projection into an `ETag`.
Matching `If-None-Match` returns 304 without a body. The client separately owns
list and selected-task request generations, retains valid cached data on 304,
and ignores responses from replaced selections. Older conversations load on
demand. This avoids repeated full-history transfers; it does not establish a
particular model-execution speed improvement.

The desktop composition has project navigation, a central workspace with a
compact state header and fixed composer, and a collapsible Agent panel. The
phone composition uses navigation and settings drawers with the same central
tabs. Background/avatar preferences remain separate from task data. Public
execution events and actual tools remain visible; private model reasoning is
not synthesized or inferred from elapsed time.

## Project edits and handoff

The existing `reliable-project-ledgers-v1` Bbolt bucket receives optional fields;
old records remain readable. `ProjectItem` retains `title` as the full requirement
and adds `short_title`, `milestone`, `depends_on`, and `status_reason`.

`POST /api/v1/reliable/tasks/{id}/project` retains its existing edit fields and
accepts:

- `item_edit`: an item ID and optional title, short title, acceptance condition,
  milestone, dependency IDs, disposition, and reason.
- `item_order`: every persisted item ID exactly once, excluding derived
  historical `batch-*` rows.
- The existing `expected_revision`, checked in the same transaction as edits.

Text limits are 3,000 bytes for a full backlog title, 4,000 for acceptance,
240 each for a short title/milestone, and 1,000 for a defer/cancel reason. Each
item admits at most 32 distinct saved-item dependencies. Graph validation rejects
unknown IDs, self-edges, duplicates, and cycles. Unassigned items may be planned,
deferred, or cancelled; assigned records cannot be rewritten through this editor.
User edits cannot claim verified/accepted status or fabricate evidence.

Dispatch checks dependencies before submission and again in the linked-batch
creation transaction. Readiness uses actual linked task state `accepted`, not a
model claim or a `verified` label. Auto-continuation selects an eligible confirmed
item after human acceptance and does not accept a prerequisite itself.

Editors build lazily to avoid a quadratic checkbox DOM. Dirty editor state keeps
its original revision and survives errors and refreshes. An operator must inspect
updated records before explicitly rebasing a conflicting save or discarding the
draft. Display labels never replace the requirement sent to an implementation.

`ProjectRecord.handoff` is a derived read model, not a new memory file. It carries
task, project/requirement revisions, saved source identities, batch state, bounded
goal and run excerpts, claim categories, evidence references, and warnings. Reported
facts and inferences remain distinct from controller-backed verification.
`requirements_current` means revision equality only, not a fresh source check.
The complete card is bounded to 14,000 serialized bytes and marks omitted or
shortened content. Original records remain retained. A completed acceptance
checkpoint supplies the reference commit for new user notes and the current
handoff, avoiding false stale-memory warnings against the old base.

## Recovery and notifications

`RecoveryGuidanceFor` is a read-only projection of `code`, localized `summary`,
`next_action`, and raw `detail`. Suggested actions are `resume`, `replan`,
`new_batch`, `login`, `checks`, or `none`; normal task eligibility remains the
authorization boundary. Source/scope drift leads to inspection and explicit
planning, never automatic scope enlargement.

Planning invocation retries share a 30-minute context and admit at most three
operational attempts. A failed auditor dispatch must retain evidence and confirmed
owned-process cleanup; requirements, plan revision, source, Git history, and
resource identity are rechecked before retry. The same retained source boundary
must be checked on a later explicit planning continuation. Invalid reports,
read-only mutations, and scope failures are not network retry cases. Writer-stage
recovery remains governed by its existing separate journal. Neither path grants
a fourth quality implementation attempt.

Optional desktop notifications use a browser-stored preference and explicit
Notification permission. Initial snapshots establish state without replaying old
alerts. Relevant subsequent state changes can notify; notification text never
declares acceptance or modifies a task.

## Result files and preview lifecycle

`GET /api/v1/reliable/tasks/{id}/files` pages changed candidate paths, sizes,
status, and source hash; default page size is 100 and maximum is 200. Supplying
`path` requests a changed file as JSON text. The reader checks ownership and idle
state, safe/protected path rules, file identity, UTF-8 content, a 512 KiB limit,
and content hash. It does not edit files or substitute successful reads for
unsupported formats. Immutable patches and original check/review logs remain
available separately.

`GET /api/v1/reliable/tasks/{id}/preview` discovers available entries and status.
`POST` explicitly selects `action: start` or `action: stop`, with an optional
discovered `entry` for start. Discovery never allocates a process. Candidates
must be task-owned, idle, and consistent with retained worktree identity.

Static mode serves available HTML and admitted asset types through a loopback
listener rooted at the selected entry's directory. Discovery includes common
root/build entries and bounded changed/generated HTML candidates. `dist`/`build`
entries carry `build_unverified`; they have not been rebuilt by discovery.

Basic Vite mode requires Windows, local Node, a bare `dev: vite` script, and an
installed Vite module inside that candidate's `node_modules`. Available source
mode takes priority over an old build. It imports the existing module directly
with project configuration/plugins disabled, HMR off, and strict filesystem
boundaries; it does not invoke npm scripts or install dependencies. Unsupported
scripts/configuration requirements remain explicit limitations. A known existing
build may still be selected with its unverified-build warning.

The manager owns at most two preview registrations. Stop retains a stopping
registration until cleanup resolves; duplicate stops join that operation. A
cleanup failure retains an unavailable registration rather than allow a new
process to replace uncertain ownership. Task reservation and engine shutdown stop
owned previews before continuing. Loopback/same-origin checks, safe path rules,
read-only HTTP methods, and existing capability checks continue to apply.

Status distinguishes the start source hash from current candidate source and
marks drift. A running preview is a live view, not an immutable snapshot; its
start hash is not proof that every later browser response was verified. The
operator can stop/restart after changes. Acceptance still independently checks
the retained candidate and evidence. Vite logs use the data-drive cache, retain
a redacted 16 KiB tail, and stop the runtime after 2 MiB of output.

## Storage admission and retention

`GET/POST /api/v1/reliable/storage/budget` reads or saves revisioned policy in
`reliable-storage-budget-v1` in the existing database. Defaults are 8 GiB cache,
2 GiB temporary storage, and 2 GiB minimum free data-volume space; each setting
admits 64 MiB through 1 TiB.

Explicit budget inspection uses the existing bounded, link-safe scan for cache
and temporary categories. Reports identify partial scans, actual available
bytes, unknown free space, and `limit_mode: advisory`. These category limits are
reminders, not hard producer quotas. Saving policy never deletes data.

Allocation admission probes data-volume free space without scanning retained
history. Insufficient or unknown space blocks a new worktree or preview; evidence
reads and policy edits remain possible. Existing idle, marker-owned, at-least-
seven-day temporary cleanup is still the sole deletion path and requires preview
and confirmation. Evidence, worktrees, attachments, native homes, and arbitrary
caches are not reclaimed by quota handling. Application-controlled caches and
temporary files remain on the deployment drive; independent native tools retain
their own storage policies.

## Requirement mapping

| Requirement | Design section |
| --- | --- |
| REQ-RELIABLE-WORKBENCH-001 | Read projections and navigation |
| REQ-RELIABLE-WORKBENCH-002 | Project edits and handoff |
| REQ-RELIABLE-WORKBENCH-003 | Recovery and notifications |
| REQ-RELIABLE-WORKBENCH-004 | Result files and preview lifecycle |
| REQ-RELIABLE-WORKBENCH-005 | Storage admission and retention |
