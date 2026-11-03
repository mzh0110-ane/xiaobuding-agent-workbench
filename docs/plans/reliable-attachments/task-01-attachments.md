---
id: "01-attachments"
title: "Add file-backed inputs throughout collaboration"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-RELIABLE-ATTACHMENTS-001
acceptance_criteria:
  - AC-RELIABLE-ATTACHMENTS-001.1
  - AC-RELIABLE-ATTACHMENTS-001.2
  - AC-RELIABLE-ATTACHMENTS-001.3
  - AC-RELIABLE-ATTACHMENTS-001.4
  - AC-RELIABLE-ATTACHMENTS-001.5
  - AC-RELIABLE-ATTACHMENTS-001.6
  - AC-RELIABLE-ATTACHMENTS-001.7
  - AC-RELIABLE-ATTACHMENTS-001.8
system_design:
  - ../../specs/reliable-collaboration/system-design/attachments.md
---

# Task 01: Add File-Backed Inputs Throughout Collaboration

## Summary

Deliver protected file upload and truthful native image/file inputs in both
new-task and ongoing-conversation composers. Preserve attachment history and
evidence through revisions, restart, recovery and linked project batches.

## In scope

- Fail-first storage/API admission, content integrity and native-transport tests.
- Canonical optional task/message references, inheritance and sealed evidence;
  no raw upload bytes or user-selected storage paths in source or JSON prompts.
- Shared multiple-file/drop/paste composer, progress, preview/download, removal,
  retry, draft ownership and desktop/phone parity.
- Existing native profile/skill/plugin reuse, public/local guidance and idle-only
  deployment with legacy records/seals and native preferences preserved.

## Out of scope

Universal file parsing guarantees, unlimited bytes, automatic execution/archive
extraction, new converters/plugins, remote storage, automatic task start on
upload, original-project validation edits and paid model calls.

## Acceptance

1. Both composers upload arbitrary formats with consistent ten-file/100-MiB
   limits, explicit ready/failure states and verified server references; no
   incomplete selection can start task/message/next-batch work.
2. Actual native image arguments/parts and file references reach the appropriate
   roles and questions. Canonical content evidence survives child/restart paths;
   missing/tampered input blocks affected dispatch or acceptance, preserving old
   seals and source safety.
   Attachment-only read-only answers cite supplied IDs without invented project
   paths; explicit requirements freeze prior question context, same-ID retries
   keep it, and later questions do not change running work or its seal.
3. Desktop/phone controls, draft ownership and submitted preview/download history
   pass bounded browser tests; local delivery preserves existing data, profiles
   and desktop entry without executing real tasks or applying source.

## ASCII UI preview

[UI-ATTACH-01/02/03/04](plan.md#ascii-ui-preview) maps .1/.2/.4/.5/.7:

```text
New task: [Goal ...] [image x] [file.docx Ready x]
          [Attach files] [Start]
Follow-up: [Question / Modification] [Text ...]
           [image x] [data.zip Uploading 45% x] [Attach files]
           [Send (blocked until ready)]
Phone: chips and errors wrap; Attach and Send remain separate reachable rows.
History: [image Preview / Download] [file Download]
```

## Verification

From repository root using the existing local Go/CGO toolchain environment in
[the speed/results plan](../reliable-speed-results/plan.md#verification-commands-and-environment).
Keep native account calls and production mutation outside all fixtures.

```powershell
go -C apps/backend test -p 1 -trimpath -race ./internal/reliable -run '^(TestAttachment|TestNative|TestMessageGuardsRetainTheWorkBatchLimit|TestExhaustedBatchAcceptanceAppliesCumulativeCandidate|TestExhaustedBatchAcceptanceRejectsChangedAncestorReceipt|TestAcceptanceRejectsMissingOrAlteredEvidence|TestCancelledRecoveryPreservesPartialSource|TestInspectionStartsLinkedModificationBatchWithoutResettingHistory|TestQuestionPreservesSealedCandidateAndIdempotentSubmission|TestLegacyDispatchSerializationOmitsNewFields|TestIndependentReviewRequestsReworkThenHumanAcceptance)' -count=1 -v -timeout 6m
node scripts/reliable/desktop-browser-check.cjs
node --test apps/backend/internal/reliable/page-state.test.cjs
node --test apps/backend/internal/reliable/page-attachment-state.test.cjs
go -C apps/backend vet ./internal/reliable
node --test scripts/validate-public-docs.test.mjs
node scripts/validate-public-docs.mjs
node E:/小布丁/data/validation/upgrade-1.11.0/native-image-installed-check.mjs
python -B scripts/list-docs.py validate
python -B scripts/lint-spec-files.py --all
git diff --check -- docs/plans/reliable-attachments docs/specs/reliable-collaboration/requirements/attachments.md docs/specs/reliable-collaboration/system-design/attachments.md
git status --short -- docs/plans/reliable-attachments
```

Use the installed-native offline proof and exact tests recorded in the plan,
then run every changed suite after the final edit. Require no new owning
spec findings; record unrelated existing global lint debt separately. Run the
changed package lint and bounded shared admission/evidence regressions once
focused checks pass. Do not infer a whole-suite pass from segmented results.
The native offline command is a retained local validation artifact, not a
required new native install or persistent plugin registration. The release
build used `go build -p 1 -trimpath -tags fts5 -ldflags '-s -w'` with the existing
local toolchain; candidate/runtime and deployment scripts are retained beside
the receipts rather than invoked against production by this work order.

## Files likely touched

- `apps/backend/internal/reliable` attachment storage/HTTP/evidence/native helpers
  and new `*_test.go` files.
- `http.go`, `setup_actions.go`, `store.go`, `conversation.go`, `requirements.go`,
  `dispatch.go`, question/inspection runners, continuation/renewal and sealing.
- `process.go`, `harness.go` and a per-run native image bridge.
- `page-attachments.js`, `page.html`, `page.js`, `page-conversation.js`, `page.css`,
  browser/native fixture scripts and public task guidance.

## Dependencies

Existing local token authorization, native profiles/model catalogs, task/message
revision guards, recovery/renewal provenance and final human acceptance.

## Risks

Unsupported native image claims, incomplete byte transfer, hidden history caps,
cross-draft late results, mutable referenced bytes and legacy seal changes.

## Parallelism

One vertical work order. Storage/native/UI implementation may proceed against
the agreed API in disjoint files; the shared task schema, evidence and admission
integration remain serialized by their owner before final verification.

## Inputs

- Paired attachment requirement/design and all four plan UI previews.
- Existing upstream attachment limits and local native runner implementations.
- Explicit user authorization for implementation and local deployment.

## Results

Implemented and deployed as 1.11.0. Final focused race run:
32 top-level PASS in 177.597 s, including all 18 new attachment tests; one
existing interactive folder-picker SKIP. Native segment: 17 PASS, including four
new image cases; storage segment: seven PASS. Final browser: 61/61 PASS; state:
10/10; public-document tests: 62/62 and 47 pages validated. Package lint has zero
issues; vet/build PASS. The isolated candidate's actual browser multipart upload
and hash/preview/reload checks PASS for PNG and an unknown binary, with zero
task mutations. Installed Harness offline image delivery also PASS without paid
calls. The plan records receipts and the test-only browser wait correction.
Closed-copy compatibility preserved all nine task records and verified two old
sealed candidates without applying patches. Actual deployment preserved those
nine raw records, native settings and shortcut hashes. Production runtime PASS
on desktop/390 px with GET only, no task mutations, recovery or paid calls; the
retained child remains awaiting acceptance at round 1. Backup:
`E:/小布丁/backups/upgrade-1.11.0-20261020-231937`. The plan links the installed
binary hash and deployment/runtime receipts. Do not infer a whole-package suite
result or model comprehension from these focused checks.
