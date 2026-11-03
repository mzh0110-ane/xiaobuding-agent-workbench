---
status: active
system: reliable-collaboration
created: 2026-10-19
owners:
  - local-distribution
---

# Desktop Workspace Requirements

## Overview

A novice can find a collaboration task, inspect its actual result, open its local folders, and put inactive work away without confusing presentation with execution. An archived task remains a persisted task with its original execution state and evidence.

### REQ-RELIABLE-DESKTOP-001: Focused local task workspace

- **AC-RELIABLE-DESKTOP-001.1:** The workspace shall provide a compact task sidebar with search and separate current-task and archived-task views. Search shall filter the selected view without changing persisted task data.
- **AC-RELIABLE-DESKTOP-001.2:** Every task row shall expose the same available actions through a visible overflow button, right-click, and keyboard context-menu activation. Escape shall close the menu and return focus to its invoking control; unavailable actions shall expose their reason.
- **AC-RELIABLE-DESKTOP-001.3:** Task actions shall open the original project, isolated working copy, or evidence folder associated with that task. Missing folders shall produce a visible error; an arbitrary client-supplied path shall not select an opening target.
- **AC-RELIABLE-DESKTOP-001.4:** A task detail view shall prioritize its actual execution status, verification and review results, remaining action, and evidence. Tool/model arrangements shall remain available in an expandable section or drawer without occupying the initial result area.
- **AC-RELIABLE-DESKTOP-001.5:** New-task entry shall prioritize the project and requested outcome, summarize enabled work assignments compactly, and allow expansion of every supported tool, model, and reasoning-effort control. Collapsing controls shall preserve their values and existing default assignments and models.
- **AC-RELIABLE-DESKTOP-001.6:** The user shall select light, dark, or system theme as a local preference that survives a page reload. System mode shall follow the operating-system appearance without changing task or account settings.
- **AC-RELIABLE-DESKTOP-001.7:** On narrow screens, task navigation shall use a dismissible drawer and keep task actions, search, creation, and results reachable without horizontal page overflow. Menu and drawer controls shall support keyboard focus and accessible names.
- **AC-RELIABLE-DESKTOP-001.8:** The workspace shall display persisted task facts and real execution evidence without fabricating chat messages, progress, successful verification, or model availability.
- **AC-RELIABLE-DESKTOP-001.9:** Task detail shall separate conversation, execution activity and results into keyboard-accessible tabs. The selected tab and its scroll position shall survive task polling and task switches; the message composer shall remain reachable below the tab content without scrolling through the full history.
- **AC-RELIABLE-DESKTOP-001.10:** Task titles shall remain compact while the complete original goal and appended batch requirements remain available in the conversation. Presentation shall not rewrite persisted requirements or expose internal requirement delimiters as user-facing instructions.
- **AC-RELIABLE-DESKTOP-001.11:** A local width control shall switch between a wider task workspace and a layout that reserves space for the portrait background. The preference shall survive reload without changing task data, native assignments, the selected background or its transparency.
- **AC-RELIABLE-DESKTOP-001.12:** Results shall identify their recorded batch round and available source fingerprint, distinguish batch verification from whole-project completion, and expose unresolved review findings. Each available check log shall open the recorded file for that check; patch and review links shall require an actual retained artifact. Missing results shall remain explicit rather than being presented as a pass.

### REQ-RELIABLE-ACTIVITY-001: Observable local execution

- **AC-RELIABLE-ACTIVITY-001.1:** The task workspace shall show real preparation progress, native tool calls, command output and public agent messages while execution is still running. A Git checkout percentage shall describe local file preparation, never upload or overall task completion.
- **AC-RELIABLE-ACTIVITY-001.2:** Incremental records shall survive reload and service restart, remain separate from sealed verification evidence, apply native credential redaction and bounded storage, and never synthesize history for older tasks. Raw private reasoning shall not be projected into the UI; exposed summary events and reasoning status are distinct.
- **AC-RELIABLE-ACTIVITY-001.3:** Users shall filter records and stop following the latest entry without losing task state. Task switches and failed requests shall not publish stale output into another task. Windows cancellation shall terminate owned Git descendants and settle without hanging on inherited output pipes.

### REQ-RELIABLE-INSPECT-001: Fast explicit project inspection

- **AC-RELIABLE-INSPECT-001.1:** An explicit read-only task shall read the selected directory or existing linked Git worktree directly without Git initialization, another worktree, code changes, verification commands or an acceptance/apply action. Native runners shall request their read-only sandbox rather than silently falling back to full access.
- **AC-RELIABLE-INSPECT-001.2:** Investigation shall use the selected planning connection/model/effort, identify its session and persist its report plus fingerprints of cited in-project regular files. Completed inspection shall have a distinct terminal state and shall not claim independent verification of code changes.
- **AC-RELIABLE-INSPECT-001.3:** Modification tasks shall check the applicable full or resource-aware admission profile before copying a worktree, explain oversized-source rejection, and retain their existing evidence audit, independent review, human acceptance and three-round limit.

### REQ-RELIABLE-ARCHIVE-001: Reversible task organization

- **AC-RELIABLE-ARCHIVE-001.1:** Archiving shall persist an archive timestamp separately from execution state. Restoring shall remove that timestamp; both operations shall survive service restart and preserve the task identity.
- **AC-RELIABLE-ARCHIVE-001.2:** Archive requests for queued, preparing, running, cancelling, or applying tasks shall be rejected by the server without changing task data. Running includes the `auditing`, `implementing`, `checking`, and `reviewing` stages and a reserved execution that has not yet changed its saved state. UI availability shall follow the same state rule.
- **AC-RELIABLE-ARCHIVE-001.3:** An `awaiting_acceptance` task may be archived, but its row and detail shall visibly retain that it still needs acceptance, including in the archived view. Archiving shall not apply its candidate or release its unfinished-project reservation.
- **AC-RELIABLE-ARCHIVE-001.4:** Archive and restore shall preserve execution state, round count, source and candidate identity, evidence, checks, review, and independent acceptance gates. Existing sealed candidates shall remain valid under the same verification rules.
- **AC-RELIABLE-ARCHIVE-001.5:** Archive and restore shall not cancel, resume, execute, accept, delete, or clean up a task or its files. Archived tasks shall require restoration before continuation, assignment changes, or acceptance. Cancellation, continuation, and acceptance shall remain distinct actions with their execution eligibility rules; task menus shall contain no destructive deletion action.
- **AC-RELIABLE-ARCHIVE-001.6:** Current-task navigation shall exclude archived tasks and archived navigation shall include them. Either view shall allow inspecting the task and restoring its current-task visibility; a failed archive or restore shall leave the visible state consistent with the server.

### REQ-RELIABLE-AGENT-SETTINGS-001: Accessible agent choices across continuation

- **AC-RELIABLE-AGENT-SETTINGS-001.1:** Users shall open/close agent settings from a persistent side/header control without scrolling the task page. A role tab shall expose that role's native tool, model and reasoning effort while retaining other role choices.
- **AC-RELIABLE-AGENT-SETTINGS-001.2:** Completed inspection and other resting tasks shall allow future model/effort choices for subsequent questions and work batches. Such choices shall not rewrite sealed execution plans, dispatch history or acceptance evidence; active calls shall retain their frozen selection.
- **AC-RELIABLE-AGENT-SETTINGS-001.3:** Saved future choices shall survive reload, use version guards across windows, feed native question dispatch and linked-batch creation, and enter new execution evidence before resumed modification. Archived tasks remain read-only and the three-round/review/acceptance policies remain authoritative.
- **AC-RELIABLE-AGENT-SETTINGS-001.4:** Drawer controls shall preserve unsaved choices through polling, role/context changes and catalog refresh. Mobile modality shall trap focus, restore focus on close, support Escape/backdrop dismissal and avoid viewport overflow; desktop controls shall remain reachable alongside conversation.



### REQ-RELIABLE-CONTINUE-001: Persistent project follow-up

- **AC-RELIABLE-CONTINUE-001.1:** Users shall send multiple task-scoped questions and explicit modification requirements. Persist actual messages, delivery status and replies; preserve drafts across task switches and reject duplicate identifiers with different content.
- **AC-RELIABLE-CONTINUE-001.2:** Questions shall use the planning model's native read-only mode, carry existing reports and bounded prior dialogue, cite verified project files, and serialize after active project work. Question artifacts shall not modify sealed candidate evidence or consume implementation rounds.
- **AC-RELIABLE-CONTINUE-001.3:** Supplements to unfinished modification tasks shall enter the next audit boundary. A candidate reviewed against an older requirement revision shall not be accepted. All enabled writers and fresh Codex review shall receive the effective requirements under the existing total three-round budget.
- **AC-RELIABLE-CONTINUE-001.4:** An explicit modification request after inspection or acceptance shall create a linked task with project context and native assignments; exhausted tasks shall not silently receive a fresh budget. Archived tasks shall require restoration. Cancellation, failure and restart shall retain truthful message delivery states.

## Exclusions

Changing native login or model routing, changing default work assignments, synchronizing private Agent conversations, deleting task history or working copies, weakening evidence verification, extending the three-round budget, and unattended acceptance are excluded.
