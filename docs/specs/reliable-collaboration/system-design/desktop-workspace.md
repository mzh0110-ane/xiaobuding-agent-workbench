---
status: current
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-DESKTOP-001
  - REQ-RELIABLE-ARCHIVE-001
  - REQ-RELIABLE-ACTIVITY-001
  - REQ-RELIABLE-INSPECT-001
  - REQ-RELIABLE-CONTINUE-001
  - REQ-RELIABLE-AGENT-SETTINGS-001
---

# Desktop Workspace Design

## Boundary and mapping

`apps/backend/internal/reliable` owns the embedded `/reliable` workspace, task records, local task actions, and their security boundary. Ordinary upstream Kandev tasks and native tool account configuration remain outside this change.

| Requirement | Design sections |
| --- | --- |
| REQ-RELIABLE-DESKTOP-001 | Workspace projection; Local folder actions; Appearance and interaction |
| REQ-RELIABLE-ARCHIVE-001 | Archive metadata and lifecycle; Persistence and compatibility; Workspace projection |
| REQ-RELIABLE-ACTIVITY-001 | Live execution projection |
| REQ-RELIABLE-INSPECT-001 | Direct read-only inspection |
| REQ-RELIABLE-CONTINUE-001 | Persistent conversation and revision boundaries |
| REQ-RELIABLE-AGENT-SETTINGS-001 | Future plan isolation and settings drawer |

## Future plan isolation and settings drawer

Resting tasks expose versioned `future_plan` metadata through a separate task action. Keep original `model_plan`, plan evidence, dispatch records and seals unchanged. Question dispatch and linked batch creation use the future selection when present; new modification requirements promote it into ordinary immutable execution-plan evidence before rework. Archived, changing-stage and stale-version requests fail visibly.

The UI projects the appropriate main/future plan into the existing controlled draft reconciler. A side/header toggle opens a complementary right panel on desktop and a modal drawer on phones. Move existing assignment DOM nodes into the panel, preserving field identity and draft values. Role tabs disclose one labeled tool/model/effort group at a time. Polling and catalog refresh keep focus, selected role, drawer state and unsaved choices. Mobile background inertness belongs to this drawer while open and is released before returning focus.

## Persistent conversation and revision boundaries

Task metadata contains bounded, idempotent user messages. Read-only question workers share the engine reservation and lifecycle with project work, drain after execution, and never append to sealed dispatch history. Their native logs, reports and citation fingerprints live under a separate conversation root. Restart marks uncompleted deliveries interrupted rather than replaying paid calls automatically. Previous reports and dialogue are unverified context; each reply must read current source.

Modification requirements receive monotonic revisions and immutable evidence records. Audit freezes an effective revision; all role handoffs carry that snapshot. A supplement arriving during implementation or review forces another audit/implementation round, and the final seal transaction checks the reviewed revision. Acceptance and submission serialize through the engine lock. A read-only question never clears a passing seal. A requirement for an inspected/accepted task creates a linked new task only after the explicit user send action, with the previous source-based context retained.

## Archive metadata and lifecycle

Extend `Task` with an optional string timestamp serialized as `archived_at,omitempty`. Use UTC timestamps consistent with the existing store. Empty means current; nonempty means archived. Archive is an organization dimension, not a new execution state.

Task-scoped `archive` and `unarchive` POST actions use the existing local-request, same-origin, JSON, and browser-token checks. The engine holds its execution-reservation lock across the active-run guard and the store's atomic state-predicate/metadata update. Only `accepted`, `cancelled`, `exhausted`, `blocked`, `interrupted`, and `awaiting_acceptance` are eligible for archive. Queued, preparing, auditing, implementing, checking, reviewing, cancelling, applying, and unknown states are rejected without partially writing the record. Repeated archive retains its timestamp; restore clears it. Return the authoritative task for the UI to reconcile.

`awaiting_acceptance` is eligible for archive. Retain the original state and the unfinished-project reservation: `Store.Create` must continue to reject the same or overlapping project based on execution state, regardless of archive metadata. Archive is not a substitute for cancellation or acceptance. Archived tasks require restoration before resume, assignment edits, or acceptance; HTTP resume rejects synchronously instead of acknowledging a background attempt that will fail. Existing execution-state, source, evidence, cancellation, recovery, and three-round policies retain their authority.

## Persistence and compatibility

Keep the archive field omitted for existing unarchived tasks. Organization metadata must not enter or rewrite the frozen task contract, patch, source manifests, verification evidence, or seal inputs. Archive/restore changes neither round count nor saved results and leaves all local files in place. Validate a previously sealed awaiting-acceptance candidate before and after archive/restore using the existing read-only acceptance validation path.

Service recovery must preserve archive metadata without treating it as an execution transition. Do not derive whether a project is unfinished from which sidebar displays its task. Failed mutations leave the prior record and UI projection intact.

## Local folder actions

Resolve opening targets on the server from the loaded task: `open-project` uses `Project`, the compatible `open` action uses `Worktree`, and `open-evidence` uses the controller-owned evidence root plus task ID. Canonical working-copy and evidence paths must remain within their controller storage boundaries. Actions select this closed set of task-bound targets rather than accepting arbitrary paths or commands.

Require an existing directory before dispatch to the Windows shell through an argument vector. Do not interpolate a shell command, create a missing folder, modify source, or execute project files. Treat opening a folder as an explicit local browser action under the existing mutation guard. Missing targets and platform errors return a localized visible failure without a success indication.

## Workspace projection

Project selection opens an owned, topmost Windows folder dialog on the interactive desktop. Suppress its console with `CREATE_NO_WINDOW` rather than hiding the GUI. Dispose both dialog and owner after selection/cancellation; cancel the owned subprocess when the request ends. Return the exact UTF-8 folder path, or an empty path on user cancellation so the current draft remains intact. Extend only this request's write deadline beyond the three-minute human-selection timeout; the server's shorter ordinary API deadline must not discard a successful selection. Picker failures/timeouts give localized feedback and retain the manual path input. Windows opt-in tests exercise visible/topmost presentation, an actual Unicode folder selection, cancellation, and subprocess cleanup.

The embedded HTML/CSS/JavaScript presents a compact navigation sidebar and a main task surface. Current/archive selection and search are presentation state; task rows derive archive and execution badges from server records. Keep pending-acceptance status visible even for archived tasks. The same menu model drives row overflow, right-click, and keyboard entry so action availability cannot diverge across entry points.

Task detail puts real status, checks, review outcome, and the next permitted action before model controls. Evidence and original/candidate/evidence folder access remain reachable. Model arrangements move to a disclosure or drawer and retain the existing native catalog, validation, safe subsequent-dispatch updates, and default assignment contract. New-task entry starts with the project and outcome and exposes a compact work summary with expandable complete settings. UI expansion must not reset selected values.

`page-workspace.js` owns the conversation, activity and results tab projection. Each task has a browser-tab-local selected panel and in-memory scroll positions; polling does not replace this presentation state. The panels share an independently scrolling content region, while the existing conversation form is docked below it. Moving the form preserves its inputs, attachments, send handling and task-bound drafts. Tabs expose `tablist`, `tab` and `tabpanel` relationships, selected state, roving focus and arrow/Home/End navigation. Task switches invalidate asynchronous output through the existing task guards.

`page-state.js` derives a compact title without modifying `Task.Goal`. The Conversation tab discloses the full goal and localizes the internal next-batch delimiter for display. `page-proof.js` projects the saved current round, source fingerprint, checks and review findings. A passing check alone does not establish acceptance or whole-project completion; a newer requirement revision prevents a passing result label. Check links use each recorded `check.log`, and patch/review shortcuts are resolved against the task evidence inventory. Retain explicit loading, missing and failure states rather than creating artifact URLs from assumed success. Raw files and dispatch history remain available behind disclosures; this projection does not add website preview, implementation summaries or per-file diff rendering.

Archive success moves a row between views without deleting the task or manufacturing another selected task. Restore reverses visibility. Refresh or failed requests reconcile against server data. Do not render generated conversation history as if it were an execution log.

## Live execution projection

`activity.go` stores capped, credential-redacted JSONL records in the controller's `activity` directory, outside sealed task evidence. The task-scoped GET activity endpoint accepts an incremental byte cursor, returns at most 100 complete records, and reports whether the task currently owns an execution reservation. The UI polls at one second during work, retains bounded recent rows, follows only when requested, preserves disclosure state, uses text nodes for untrusted output and invalidates pending requests on task change. Existing tasks without records keep an honest empty state.

Native stdout is projected from complete JSON lines during execution; stderr and real check output have separate labels. Codex and Harness adapters retain their original complete evidence streams. Only allowlisted native tool/message/status fields enter the live projection. Raw `reasoning`/`thinking` text becomes a status; an explicit public summary event can supply summary text. The bounded output writer owns its buffer instead of promoting `bytes.Buffer.ReadFrom`, which would bypass both output limits and live callbacks during `io.Copy`.

All Git subprocess paths share the owned-process runner with Windows job-object cleanup and a bounded pipe wait. Cancellation ends descendants, including the checkout process spawned by the Git launcher. Git carriage-return progress is split into actual local file-count records; no estimated overall percentage is generated.

## Direct read-only inspection

The explicit `read_only` task flag bypasses local initialization, worktree creation and modification rounds. It uses the selected investigation assignment and native read-only sandbox. The original directory can be a plain folder or a separately prepared linked worktree. A fresh identified session returns the normal structured audit report; citations must resolve to existing regular files inside the selected project, with existing per-file/total byte bounds and at most 100 citations. Retain the source fingerprints and report under inspection evidence. The `inspected` terminal state can be archived and releases project admission but cannot apply a patch or receive future-stage edits. The UI explicitly distinguishes this report from tested or independently reviewed changes.

Modification tasks retain the full workflow and mutable-source limits. Preflight checks the existing Git index before checkout. Oversized resource-heavy existing repositories can use the bounded [resource-aware profile](large-resource-projects.md); large all-source repositories still fail before copying. No profile silently removes tracked data or rewrites the user's Git ignore rules. Read-only tasks require only the selected investigation connection, not unused frontend/backend connections.

## Appearance and interaction

Use the existing embedded page architecture and visual tokens. Persist only `light`, `dark`, or `system` as a browser-local preference. System mode uses the browser's operating-system appearance query and follows subsequent appearance changes. This preference is separate from server configuration and native account settings.

Task detail defaults to a wide working surface. The width toggle stores only `wide` or `portrait` in browser-local storage and reserves the existing right-hand background area in portrait mode. New-task setup keeps its existing portrait layout. Preserve the avatar, background choice and transparency controls independently. At narrow widths, allow the tab content to scroll while keeping the composer and current task actions reachable; width preference does not create horizontal page overflow.

An overflow button provides a discoverable alternative to right-click. Keyboard menu activation, menu focus, Escape dismissal, and focus restoration follow the same action model. Narrow layouts replace the persistent sidebar with a drawer; opening and closing it preserves selection and form drafts. Search, menu actions, result inspection, and full assignment settings remain accessible at phone width. Rendered checks cover desktop, phone, themes, disabled actions, and the awaiting-acceptance archive warning.
