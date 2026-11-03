---
status: active
system: reliable-collaboration
created: 2026-10-26
owners:
  - local-distribution
---

# Long Project Workbench Requirements

## Overview

The local reliable-collaboration workbench helps an operator continue a project
across work batches, identify the current execution or blocker, and inspect a
candidate before accepting it. This extends the existing
[project continuity contract](long-projects.md); it does not replace its evidence,
independent-review, native-account, or human-acceptance boundaries.

## Terms

- **Batch:** One admitted goal with at most three quality attempts, including the
  initial implementation. An explicitly requested successor can continue the project.
- **Operational retry:** A bounded attempt to recover a failed native invocation;
  it does not grant an additional quality attempt.
- **Candidate:** The task-owned worktree. Its live preview is distinct from the
  original project and from immutable verification evidence.
- **Handoff card:** A bounded view of saved project and task records with source
  identities, claim categories, and remaining uncertainties.

## Requirements

### REQ-RELIABLE-WORKBENCH-001: Inspectable project navigation

- **AC-RELIABLE-WORKBENCH-001.1:** Task navigation shall group batches by canonical
  project identity and retain accessible links to current, historical, archived,
  blocked, and awaiting-acceptance records without merging unrelated namesakes.
- **AC-RELIABLE-WORKBENCH-001.2:** List reads shall return bounded summaries without
  message bodies, full snapshots, or raw reports. Selected-task reads shall load
  the latest 25 messages by default, support older pages, and permit unchanged
  list/detail reads to return no body through conditional requests.
- **AC-RELIABLE-WORKBENCH-001.3:** Desktop and phone views shall keep conversation,
  execution, results, project plans, and a fixed composer reachable without
  traversing an entire history. Background refresh shall preserve unsaved input.
- **AC-RELIABLE-WORKBENCH-001.4:** Agent settings shall identify future-dispatch
  changes separately from actual running assignments. The desktop side panel
  shall collapse; the phone composition shall retain a dismissible settings
  overlay, keyboard focus, readable text, and non-overflowing controls.

### REQ-RELIABLE-WORKBENCH-002: Editable backlog and attributable handoff

- **AC-RELIABLE-WORKBENCH-002.1:** An unassigned backlog item shall support a short
  display label, full requirement, acceptance condition, milestone, stable order,
  and explicit defer/cancel reason. An assigned item's requirement or acceptance
  history shall not be rewritten through the backlog editor.
- **AC-RELIABLE-WORKBENCH-002.2:** Dependencies shall name distinct saved backlog
  items, reject self-reference, missing items, and cycles, and require actual
  human acceptance before dependent dispatch, including automatic continuation.
  Stale edits shall fail without overwriting current records.
- **AC-RELIABLE-WORKBENCH-002.3:** A handoff card shall separate reported facts,
  inferences, unresolved issues, and verified-batch status; retain task/version
  provenance; identify obsolete or truncated context; and serialize within
  14,000 bytes. A passing batch shall not establish completion of every project
  requirement or verify historical model claims.
- **AC-RELIABLE-WORKBENCH-002.4:** Saved notes and handoff views after acceptance
  shall use the completed accepted checkpoint when available. Reads shall not
  create per-message memory files, duplicate project directories, or reset rounds.

### REQ-RELIABLE-WORKBENCH-003: Actionable operational recovery

- **AC-RELIABLE-WORKBENCH-003.1:** A paused task shall expose a localized recovery
  category, suggested next action, and available raw diagnostic. Guidance shall
  not authorize a mutation, start work, broaden scope, or accept a result.
- **AC-RELIABLE-WORKBENCH-003.2:** A failed read-only planning invocation may use
  at most three operational attempts in one shared 30-minute window after
  validating process cleanup, source, resource, requirement, and model-plan
  identity. Source mutation, invalid reports, and scope failures shall not be
  treated as transient network errors.
- **AC-RELIABLE-WORKBENCH-003.3:** A later explicit planning continuation shall
  revalidate the retained planning source identity rather than trust source
  changed by a failed or invalid read-only invocation. Runtime recovery shall
  preserve the batch's quality-attempt counter and evidence requirements.
- **AC-RELIABLE-WORKBENCH-003.4:** Optional desktop notifications shall require an
  explicit browser preference and permission, and shall announce meaningful
  state transitions rather than replay all historical states on initial load.

### REQ-RELIABLE-WORKBENCH-004: Candidate results and bounded preview

- **AC-RELIABLE-WORKBENCH-004.1:** An idle owned candidate shall expose paged changed
  files and a read-only UTF-8 text reader bounded to 512 KiB. Protected paths,
  redirected files, stale reads, unsupported formats, and unavailable files shall
  fail explicitly without being presented as read successfully.
- **AC-RELIABLE-WORKBENCH-004.2:** Preview discovery shall not start a service.
  Explicit start shall bind a local preview to the selected candidate, display
  its start/current source identities, and distinguish stale output from current
  state. A preview shall not constitute passing verification or human acceptance.
- **AC-RELIABLE-WORKBENCH-004.3:** Supported preview modes shall include available
  static/generated HTML and a basic candidate-local Vite runtime with existing
  dependencies. The basic runtime shall not install dependencies, execute npm
  lifecycle hooks, or load project Vite configuration/plugins. An existing
  build fallback shall be labeled as not rebuilt or verified against source.
- **AC-RELIABLE-WORKBENCH-004.4:** Preview stop shall retain process ownership until
  cleanup is confirmed. Concurrent replacement, resumed task execution, or
  shutdown shall not silently discard uncertain cleanup. File and preview
  access shall remain local and constrained to admitted candidate content.

### REQ-RELIABLE-WORKBENCH-005: Explicit storage pressure controls

- **AC-RELIABLE-WORKBENCH-005.1:** Operators shall be able to inspect and save
  revision-protected cache, temporary-storage, and minimum-free-space settings.
  Cache and temporary limits shall be explicitly advisory rather than claimed
  as enforced producer quotas.
- **AC-RELIABLE-WORKBENCH-005.2:** Insufficient or unknown free space on the data
  volume shall block new worktree or preview allocation while preserving access
  to retained results and settings. Status polling shall not recursively scan
  all historical storage.
- **AC-RELIABLE-WORKBENCH-005.3:** Storage handling shall retain source, active and
  historical candidates, audit evidence, attachments, native homes, and installed
  extensions. Deletion shall remain the existing explicit preview/confirmation
  path for eligible old inactive owned temporary directories.

## Exclusions

This delivery does not provide a general IDE, arbitrary application hosting,
parallel writers in one candidate, automatic acceptance, unlimited automatic
rework, credential migration, a second skill/plugin installation, a hard cache
quota, automated deletion of retained evidence/worktrees, or error-free models.
Existing ordinary Kandev workflows and remote execution remain outside this local
extension's ownership.
