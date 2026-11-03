---
created: 2026-10-20
status: implemented
requirements:
  - REQ-RELIABLE-ATTACHMENTS-001
system_design:
  - ../../specs/reliable-collaboration/system-design/attachments.md
legacy_specs: []
---

# Implementation Plan: Reliable Attachments

## Overview

Complete file/image inputs for both starting a reliable task and continuing its
conversation. The user explicitly requested implementation and local deployment;
the package records the concrete contract without another planning-only stop.
Ship bounded streaming and reference admission first, wire native delivery and
evidence, then integrate the same composer controls and verify local delivery.

## Scope

### In scope

- Ten files/100 MiB per file and aggregate per submission, arbitrary formats,
  protected streaming upload and verified safe preview/download.
- Ready-reference drafts, multi-file picker/drop/pasted images, progress/removal/
  retry, new-task and follow-up history at desktop and phone sizes.
- Canonical task/message attachment references, revision/idempotency guards,
  child inheritance, restart and evidence compatibility.
- Actual Codex/Harness image transport, honest file/capability status and
  deterministic no-paid-call tests; idle-only backed-up local delivery.

### Out of scope

Unlimited bytes, guaranteed universal parsing, converter/plugin installation,
automatic archive/executable processing, remote storage, original-source upload
copies, automatic task execution/acceptance and changes to ordinary Kandev.

## Technical approach

Add cohesive storage/HTTP/native input helpers to `internal/reliable`. Extend
the existing local token guard only for the exact multipart upload endpoint.
Extend create/message/renewal requests with `attachment_ids`; persist canonical
optional attachment references before scheduling execution. Root-owned shared
integration covers `setup_actions.go`, `conversation.go`, `requirements.go`,
`dispatch.go`, inspection/question runners, linked continuation/renewal,
`evidence.go` and recovery validation. Native adapters extend `RoleRequest`,
Codex arguments and the existing Harness per-run bridge, without new installs.

Use optional `Audit.read_attachments` only in attached read-only native schemas.
Allow an attachment-only answer with canonical IDs and an empty project list;
unreadable inputs need explicit `unknowns`. Validate any cited project path even
when valid attachment citations are present. Freeze prior question references
in optional `TaskMessage.context_attachments` at explicit requirement admission;
same-ID retries keep that snapshot. Activate its inputs with the audited
revision, and inherit prior question references into explicit linked/renewed
batches. A later question cannot alter work already running or its seal.

Use a scoped `page-attachments.js` helper embedded by the existing page loader
for both form and conversation composers. Reuse visual tokens, focus handling,
draft identity and server error projection. Add deterministic storage/runner
fixtures and bounded browser cases before changing production logic.

| Provider/input | Identity and capability | Intended behavior | Evidence/fallback |
| --- | --- | --- | --- |
| Codex ChatGPT images | Verified raster, native image arguments, selected model | Real image input with text stdin | Fake executable captures exact argv; unsupported is explicit |
| DeepSeek Harness images | Installed native attachment APIs and existing profile | Native structured image parts | Local bridge fixture; no installed-plugin duplication |
| Any other file format | Opaque ID, raw size and SHA-256 | Retain/download and verified local file reference | Unknown readability is stated, never invented |
| Task/requirement/question inputs | Canonical refs and effective revision | Correct fresh stage/question handoff | Real controller/runner fixture and mismatch guards |
| Legacy task or sealed candidate | No optional attachment fields | Existing JSON/seals unchanged | Existing compatibility test on closed copy |

## ASCII UI preview

These views specify structure, explicit submission and honest status. Pixel
spacing is illustrative; existing appearance, Agent drawer and project layout
stay intact. The main task view scrolls; attachment chips wrap within it.

### UI-ATTACH-01: New task, desktop

Entry: new task composer. Maps to AC-RELIABLE-ATTACHMENTS-001.1/.2/.7.

```text
Project folder: [E:/MyProject                          ] [Choose]
+--------------------------------------------------------------+
| Describe what should be done ...                             |
|                                                              |
| [image thumbnail  x] [requirements.docx  1.2 MB  Ready  x]    |
| [data.zip  Uploading 45%  -----------------          x]       |
| [Attach files]  Drop files or paste images here.              |
| Frontend/backend choices                    [Start (blocked)]|
+--------------------------------------------------------------+
Up to 10 files. 100 MiB each and in total per submission.
```

### UI-ATTACH-02: Ongoing project, desktop

Entry: existing follow-up composer/history. Maps to .1/.2/.4/.5/.7.

```text
Previous user requirement:
  [screenshot.png  Preview / Download] [spec.pdf  Download]

What should happen? [Question (read-only) / Modification]
+--------------------------------------------------------------+
| Please compare this new screenshot with the current page ... |
| [new-image thumbnail  x] [notes.txt  Ready  x]                |
| [Attach files]                                  [Send]       |
+--------------------------------------------------------------+
Images: native image input. Other files: available to native tools.
Successful upload does not certify file comprehension or task success.
```

### UI-ATTACH-03: Phone new-task and follow-up composition

Entry: either composer at 390 px. Maps to .1/.2/.7. The same controls wrap;
the file button remains reachable without drag or clipboard APIs.

```text
+------------------------------+
| Goal / follow-up text ...    |
|                              |
| [image thumbnail] [remove]   |
| document.docx  1.2 MB        |
| Ready              [remove] |
| data.zip  Uploading 45%      |
| [Attach files              ]|
| [Start / Send              ]|
| Errors wrap below.           |
+------------------------------+
History: [Preview] [Download]
```

### UI-ATTACH-04: Failure, removal and restored draft

Maps to .2/.3/.4/.5/.7. Keep text and unaffected chips while reporting failures.

```text
report.bin  Upload failed: exceeds 100 MiB. [Choose again] [Remove]
image.png   Ready. Native image support depends on the selected model.
old-file    Saved draft reference unavailable. [Choose again] [Remove]
[Send (blocked until failed selections are ready or removed)]
```

## Tests

| Acceptance | Focused evidence |
| --- | --- |
| .1/.3 | `TestAttachmentUpload*`: multipart protection, exact limits, arbitrary bytes/name, stream cancellation, immutable hash and missing/changed download |
| .2/.7 | Browser attachment cases for picker/drop/paste, progress, remove/retry, double send, delayed owner change, IME/text paste and mobile layout |
| .4/.6 | `TestAttachmentFlow*`, `TestAttachmentQuestionReferencesFreezeWhenWorkIsRequested`, `TestAttachmentQuestionReferencesSurviveInspectionToModification`: task/message ID resolution, mismatch/partial set, frozen question context, audited revision, restart, child paths and legacy digests |
| .5 | `TestNativeCodexReceivesActualImageArgumentsAndBoundedDelivery`, `TestNativeHarnessReceivesActualImageBlocksAndHonestFallback`: exact native arguments/parts, lower native limits and honest unread references |
| .5/.6 | `TestAttachmentOnlyQuestionCanCiteUploadedInputs`, `TestAttachmentReportCitationsCannotReplaceInvalidProjectSources`: supplied-ID citations, unread inputs and independently checked project paths |
| .6/.8 | Recovery/acceptance content tamper and closed-copy legacy seal checks; runtime GET-only task/settings/shortcut preservation |

Additional exact integrity and byte-budget cases are recorded below. A focused
run does not imply a whole-package test pass.

## E2E tests

`scripts/reliable/desktop-browser-check.cjs` includes disposable upload/API
fixtures covering both entry points, attachment-only follow-up, preview/download
history, upload failure/content retention, unfinished-upload gates, task-switch
and reload ownership, renewal headers and desktop/phone behavior. An isolated
candidate runtime additionally uploads actual PNG/unknown-binary files through
the real browser XHR endpoint and verifies bytes, thumbnails and restored ready
references. Native transport uses deterministic local fixtures, never paid
account calls. Production runtime checks are GET-only and preserve records.

## Work orders

- [x] [Task 01: Add file-backed inputs throughout collaboration](task-01-attachments.md)

## Verification results

Implemented and deployed as local release 1.11.0. Results from saved receipts:

- Storage race segment: seven top-level tests PASS in 6.348 s, covering protected
  multipart upload, truncation/cancellation, preview/download, restart/tamper,
  exact raw-byte/history limits and redirected directories. Evidence:
  `E:/小布丁/data/validation/upgrade-1.11.0/attachment-storage.log`.
- Native race segment: 17 top-level tests PASS in 6.866 s, comprising four new
  attachment cases plus 13 existing native/policy/cleanup regressions. The new
  cases prove actual Codex image arguments, Harness image blocks and honest
  text-only/lower-limit fallback, latest-image/20-MiB budgeting and changed-byte
  rejection before either native process starts. Evidence:
  `E:/小布丁/data/validation/upgrade-1.11.0/native-final-race.log`.
- Installed Harness offline proof used its actual Cordis plugin mount,
  `createUserMessage` and `LocalAttachmentStore`, verified one immutable native
  image block and a durable readable reference. It used an isolated home and
  zero paid calls. Evidence:
  `E:/小布丁/data/validation/upgrade-1.11.0/native-image-installed/result.json`.
- Final source race run: 32 top-level tests PASS in 177.597 s, including all 18
  new attachment tests and shared legacy serialization, question/requirement
  revision, partial-source recovery, cumulative-child acceptance, evidence-tamper
  and native regressions. One existing opt-in interactive Windows folder-picker
  test SKIP because the interactive desktop was not enabled. This is a focused
  run, not a whole-package test pass. Evidence:
  `E:/小布丁/data/validation/upgrade-1.11.0/attachment-final-race.log`.
- Final browser suite: 61/61 PASS, including nine new attachment cases. A first
  run exposed a readiness wait missing from the reload test; waiting for its
  visible restored chip fixed that test, with no product change. State tests:
  10/10 PASS. Public-document tests: 62/62 PASS; 47 pages validated. Evidence:
  `browser-final/results.json`, `state-final.log`, `public-docs-tests.log` and
  `public-docs-final.log` under the same validation directory.
- Final package lint: zero issues; `go vet` and release build PASS. Binary
  SHA-256: `c779656ff9fc01dd320613b6d600747f10f35d73595336a60f1a1a59063ba906`.
  Evidence: `static-build-results.json` and corresponding final logs.
- Actual isolated candidate 1.11.0 runtime PASS, bound to those executable bytes:
  two real XHR multipart uploads (PNG and unknown binary), exact download hashes
  and PNG preview hash, completed thumbnail and two ready references restored
  after reload. Zero task rows and zero task mutation requests; exactly two
  upload-only mutations. Evidence: `candidate-runtime.json`.
- Independent bounded read-only review of storage/input resolution and native
  transport found no concrete release blocker. This review is not paid-model
  verification or proof of universal file comprehension.
- Closed-database compatibility: all nine retained task records matched the
  stopped online snapshot; two existing sealed candidates remained valid
  without applying either patch (92.52 s). Stopped-process ownership proof
  covered all nine histories without dispatching or editing a task (0.14 s).
  Evidence: `closed-database-verification.log`.
- Actual deployment to `E:/小布丁` PASS. Backup:
  `E:/小布丁/backups/upgrade-1.11.0-20261020-231937`. Installed executable hash
  matches the final candidate. All nine raw task records, native connection
  settings and desktop shortcut hashes are unchanged; zero paid calls, task
  mutations, recoveries or production attachment uploads. Evidence:
  `deployment.json`.
- Production runtime PASS on desktop and 390 px: new-task and actual retained
  child's follow-up attachment controls are visible/enabled and were not
  clicked. Nine records remain identical before/after; model catalogs and image
  assets are unchanged. Child `next-e6d4b6c66129c771878edd23ae` remains
  `awaiting_acceptance`, round 1. Runtime validation used GET only, with zero
  mutations. Evidence: `production-runtime.json`.

The offline native and isolated upload proofs do not claim paid-model file
comprehension. Requirements/design are active/current, this plan implemented
and its work order done. Final document validation records 352 decisions and
1366 specifications; no owning-system spec lint findings. The 107 pre-existing
global size findings remain unrelated to this package.

## Risks

- Relaxing the global JSON-only guard could expose unrelated mutations.
- Treating browser names/hashes as trusted paths or inserting bytes into source
  could corrupt projects or weaken seals.
- Passing only image filenames would falsely imply native vision delivery.
- Cumulative history can exceed one native turn's image budget; retain history
  and expose delivery limits rather than silently losing inputs.
- Restart, task switch, idempotent retry and child handoff can attach the wrong
  draft or omit content from the evidence chain.
