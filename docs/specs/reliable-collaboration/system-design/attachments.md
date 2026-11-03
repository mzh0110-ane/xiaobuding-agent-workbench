---
status: current
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-ATTACHMENTS-001
---

# Reliable Task Attachment Design

## Ownership and existing boundaries

The standalone `internal/reliable` workspace owns its Bolt task/message records,
same-origin local mutation token, native runners and acceptance seals. Its
pre-attachment `createRequest`, `messageRequest`, `Task`, `TaskMessage` and
`RoleRequest` carried text only. The existing mutation guard accepts JSON and
`requestJSON` is limited to 64 KiB, so binary payloads must not be embedded in
existing JSON.

Reuse the product limits and admission principles from ordinary Kandev
[prompt attachments](../../tasks/system-design/prompt-attachments.md). That
system's tenant/session/agentctl services remain independent; this local slice
does not invent another native skill or plugin installation registry.

| Acceptance | Design sections |
| --- | --- |
| .1/.2/.7 | Composer state and responsive history |
| .3/.4 | Streaming storage, admission and retention |
| .5 | Native delivery and truthful capability |
| .6 | Evidence and lifecycle integration |
| .8 | Compatibility and local delivery |

## Streaming storage, admission and retention

Expose these routes under `/api/v1/reliable`:

| Method and route | Contract |
| --- | --- |
| `POST /attachments` | Token-protected multipart field `file`; stream one file and return 201 canonical `Attachment` |
| `GET /attachments/{id}` | Verify and download the registered file with safe attachment disposition |
| `GET /attachments/{id}/preview` | Verify and serve only a supported safe raster image; other formats have no inline preview |

`Attachment` contains `id`, `name`, `size`, `media_type`, `sha256`, `created` and
`image`. Physical storage names and paths are server-derived and never supplied
by the browser. Original filenames are display/download metadata, not path
components. Use containment-aware filesystem access, immutable publication and
private files below the existing controller data root, outside worktrees and
original projects. Validate a registered regular file's actual size and hash
before using it. A partial upload never returns a ready descriptor.

Only the exact upload route permits multipart under the existing local-request
and mutation-token guards. Other mutations retain their JSON content-type and
64 KiB parser limits. Read paths retain same-origin protections, `nosniff` and
non-cacheable behavior. Do not render SVG/HTML or unknown formats as active
content. Format classification must not trust an extension or browser MIME alone.

The browser submits only `attachment_ids` alongside existing create/message/
`exhausted-next-batch` JSON. Resolve the complete set to canonical metadata and
verify every referenced blob before persisting the task/message or scheduling
work. Enforce ten files and 100 MiB per file and per submission using raw byte
counts, rejecting duplicate submitted IDs. Invalid or conflicting
references must leave the task/revision and source unchanged.

Optional `Task.attachments` records initial/inherited inputs; optional
`TaskMessage.attachments` records that submitted turn's files. Resolve IDs to
canonical immutable references before ordinary requirement evidence is written.
Optional `TaskMessage.context_attachments` freezes prior read-only question
references when the user explicitly submits a requirement. This snapshot is
part of that requirement receipt. The same message-ID retry reuses its frozen
context rather than collecting questions submitted since its first admission.
An existing idempotent message ID must compare canonical attachments as well as
kind/content; a different file set cannot reuse the same message identity.

Ready references survive draft restoration and service restart. Unready browser
`File` objects are not falsely restored from metadata; show reselection/removal
when their bytes are unavailable. Removing a draft chip does not delete a file
already referenced by saved history. Cancel/archive operations preserve history.
Accepted/inspection continuation and exhausted renewal inherit authorized input
references and prior question references without copying raw bytes to source
or modifying ancestor records. An explicit child request such as "apply the
previous file" can therefore use that file without another upload.
Retain history separately from the bounded current dispatch input set.

## Native delivery and truthful capability

Extend `RoleRequest` with canonical attachment input data resolved by the
controller. Initial investigation, domain writers, fresh independent review,
read-only inspection and queued questions must use the same resolution rules.
Each dispatch saves the actual reference/hash/delivery-kind inventory with its
existing handoff evidence. Prompt text identifies attachments as untrusted user
inputs and distinguishes them from source-relative mandatory `read_files`.

The Codex adapter passes text through stdin to `codex exec` and adds native
image arguments for verified supported images, retaining stdin for
text and avoiding shell-built argument strings. Other files are identified by
verified controller-owned local references that the native tools can read.

Harness sends a text prompt to its headless profile. Its image path uses the
installed native attachment/image APIs through the existing
per-run overlay/bridge, preserving the account, desktop profile and installed
extension reuse. It must not merely add image paths to prose or install another
copy of skills/plugins. The installed `dsh-attachment-local` and headless bundle
are capability references, not proof that a given model accepted an image.

Native transport verification must inspect real argument/request parts with a
deterministic fake executable or native API fixture. No paid call is needed.
PNG, JPEG, WebP and GIF are the adapter's direct raster image candidates. The
installed Harness attachment service applies its own lower format/count/byte
limits, and the resolved model route must advertise image input. Retain all
cumulative canonical references and hand off all verified file paths;
submission limits do not apply
again to project history. Select the latest supported images for direct native
input, bounded to eight images and 20 MiB in aggregate per native call, or a
lower actual native limit. Overflow, larger or unsupported images remain file
references with explicit delivery classification and instructions to use an
available native image-reading tool before claiming visual comprehension. They
must not be silently dropped or described as understood.

Native model selection and effort, the Codex domain-review policy and existing
installed file-reading tools/skills remain unchanged. Native format support is
not a universal parser guarantee. Do not automatically open executables, unpack
archives, download converters or turn unknown file content into instructions.

## Evidence and lifecycle integration

Attachment metadata and bytes are model-independent inputs. Record uniquely
named append-only attachment manifests/receipts with the task/revision and bind
them to dispatch inputs and the final seal when nonempty. Include actual content
hash verification, not just the client filename or a cached metadata hash.
Optional fields and digest inputs remain absent for legacy tasks.

New requirement attachments use the current requirement revision boundary and
invalidate old current acceptance eligibility. They are not injected into an
already frozen writer turn. Read-only question attachments enter that question's
native request and conversation evidence without authorizing source edits.
`workAttachments` includes initial task inputs and requirement inputs plus their
frozen context only once the requirement revision is covered by
`AuditedRevision`. Later question inputs do not enter an active writer or its
seal until another explicit audited requirement or child request admits them.
Preserve source snapshots as source snapshots; uploads live outside Git and
must not be mistaken for original/candidate changes or project resources.

Check attached content before dispatch and at affected recovery/acceptance
boundaries. A missing or modified blob/manifest blocks reuse and acceptance
while retaining source/history. Child renewal provenance must bind inherited
attachments, including ancestors whose review never passed. Do not reseal old
pending candidates or promote a model's attachment-reading claim into proof.

The persisted `Audit.read_attachments` field is optional for compatibility.
Only attached read-only calls extend their native response schema with the
attachment citation list; ordinary writer/reviewer schemas remain compatible.
An answer may cite exact supplied attachment IDs with `read_files: []`. If no
source or attachment was readable, an explicit nonempty `unknowns` limitation
permits an empty read list. When project sources are cited, validate them even
if attachment citations are also valid; unknown attachment IDs, absolute paths
and traversal must fail without a panic or invented project citation.

## Composer state and responsive history

Use one small reusable attachment controller for the existing new-task composer
and `TaskConversation`. Each draft has its own identity and generation. Track
per-file pending/uploading/ready/failed states, progress, removal and retry.
Disable start/send/next-batch handlers synchronously until the selected set is
ready, not just visually through a button. Late callbacks must check their draft
owner and never select a new task or clear a replacement draft.

File picker uses `multiple` without an extension filter. Drop/paste is scoped
to the intended composer; preserve ordinary text paste and Chinese composition.
Show raster thumbnails and escaped file chips with name, size and status. Image
preview is a safe server URL; download history is a server-owned opaque-ID URL.
The separate appearance background picker keeps its original behavior.

Desktop retains the current compact composer. Phone wraps chips, stacks progress
and errors, keeps controls at least 44 px high and avoids whole-page overflow.
History and already submitted inputs remain inspectable after restart or archive
without automatically replaying uploads or dispatching work. The plan contains
the new-task and follow-up ASCII views.

## Compatibility and local delivery

Deterministic storage, API, native transport, message, evidence and browser
fixtures prove the new behavior. Existing 1.10.0 task JSON and sealed candidates
must remain valid with empty attachment fields. Local deployment is idle-only
with backup, exact installed-byte verification and preserved native settings and
desktop shortcut. Production validation is read-only and cannot submit the real
task, call paid native agents or apply source.

## Implementation Plans

- [Reliable attachments](../../../plans/reliable-attachments/plan.md)
