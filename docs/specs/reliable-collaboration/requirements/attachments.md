---
status: active
system: reliable-collaboration
created: 2026-10-20
owners:
  - local-distribution
---

# Reliable Task Attachment Requirements

## Overview

Users can give the local collaboration workspace images, documents, archives,
source files and other file formats when starting a project or following it up.
An upload is retained input, not proof that a model understood its contents.
Reliable collaboration owns its independent local task/message store, native
dispatch and evidence seal. Ordinary Kandev
[prompt attachments](../../tasks/requirements/prompt-attachments.md) keep their
existing ownership and are not replaced by this extension.

## Requirements

### REQ-RELIABLE-ATTACHMENTS-001: File-backed project inputs

**Intent:** Let a novice attach relevant material throughout a project using the
same familiar composer controls, without moving files into source directories.

#### Acceptance criteria

- **AC-RELIABLE-ATTACHMENTS-001.1:** New-task and ongoing-conversation composers
  shall offer multiple-file selection, drag-and-drop and pasted images. They
  shall accept arbitrary file formats as file inputs, including unknown or
  extensionless formats. One submission shall allow at most ten files, each at
  most 100 MiB, with at most 100 MiB of raw bytes in aggregate; exact boundaries
  shall be accepted and one byte above rejected. Attachment-only follow-ups
  shall retain the selected question/requirement intent without inventing a
  completed interpretation. Existing project/goal and modification guards shall
  remain authoritative.
- **AC-RELIABLE-ATTACHMENTS-001.2:** File transfer shall show per-file progress,
  uploading/ready/failed states, retry and removal before submission. A task,
  message or next-batch action carrying files shall remain blocked until all
  selected references are ready or removed. Duplicate sends and late upload
  responses shall not enter another task's draft, clear its content, or create
  duplicate work. Ordinary text paste and input-method composition shall keep
  working.
- **AC-RELIABLE-ATTACHMENTS-001.3:** The server shall retain the original bytes
  outside the original project and candidate source, using server-owned opaque
  identities, bounded streaming, byte count, content hash and canonical metadata.
  Clients shall not choose storage paths or trusted hashes. Invalid references,
  oversize input, interrupted upload and unavailable storage shall fail before
  dispatch without inserting bytes into Git, executing files or extracting
  archives. Uploading, previewing or downloading alone shall not start an agent.
- **AC-RELIABLE-ATTACHMENTS-001.4:** Submitted task/message references and their
  downloadable history shall survive service restart, task changes, cancellation,
  archive restoration and linked child batches. Inheritance shall preserve
  provenance without rewriting old messages or duplicating files into source.
  Submission limits shall not silently become a lifetime project-history limit.
  A draft whose ready reference no longer exists shall show a recoverable error
  rather than silently sending missing bytes or inline fallback data.
- **AC-RELIABLE-ATTACHMENTS-001.5:** Each native dispatch shall receive verified
  attachment inputs appropriate to its effective task, requirement revision and
  selected model. Supported images shall reach the native image input of Codex
  and DeepSeek Harness, not just a textual filename. Other files shall reach the
  existing native tools as verified local references. Capability or image-input
  limits shall produce a truthful delivery classification or an actionable
  pre-dispatch block, never a false claim of visual or document comprehension.
  File retention and file-format understanding shall be distinguished.
  Read-only attachment answers shall cite only supplied canonical attachment
  IDs. An attachment-only answer may use an empty project-source list; if no
  input was readable, it shall explicitly state the limitation rather than
  inventing source citations. Unknown IDs and unsafe project paths shall fail.
- **AC-RELIABLE-ATTACHMENTS-001.6:** Initial inputs and explicit modification
  requirements shall participate in the frozen task/revision evidence, dispatch
  receipts, recovery and final acceptance integrity checks. Missing or changed
  bytes/metadata shall prevent using or accepting affected evidence. Read-only
  question attachments shall remain separate from authorization to modify.
  An explicit modification request shall freeze the prior question attachments
  it can reference, and activate them only with its audited requirement
  revision. Retrying that request shall not collect later question inputs.
  Linked batches shall retain those inputs; a later read-only question shall
  not change an already running work input or its acceptance seal.
  New optional fields shall preserve the existing serialization and seals of
  tasks without attachments. Fresh checks, independent review and human
  acceptance shall remain required for source changes.
- **AC-RELIABLE-ATTACHMENTS-001.7:** Desktop and phone shall expose accessible
  attachment controls, removable draft chips, safe raster previews, filenames,
  sizes and downloadable submitted history. Unknown or active-content formats
  shall remain downloadable files; their preview shall not execute scripts or
  open arbitrary local paths. Errors shall wrap, focus remain usable and the
  390 px page remain within its viewport. Selecting files for task input shall
  not replace the separately configured background image.
- **AC-RELIABLE-ATTACHMENTS-001.8:** Installation shall preserve existing tasks,
  sealed candidates, native logins, model/effort choices, installed skills/plugins
  and desktop entry points. Native transport and file-integrity verification
  shall be tested using deterministic local fixtures; installation and validation
  shall not start the user's stopped task, call paid models or apply source.

## Out of scope

Unlimited file sizes, guaranteed parsing of every format, automatic archive or
executable processing, automatic converter/plugin installation, remote storage,
arbitrary local-path download, background upload execution and guarantees of
zero model error are excluded. Existing native skills and tools may interpret
files they support; unreadable or unsupported content remains explicit.

## Implementation Plans

- [Reliable attachments](../../../plans/reliable-attachments/plan.md)
