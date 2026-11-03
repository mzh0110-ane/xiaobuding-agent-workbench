---
created: 2026-10-19
status: implemented
requirements:
  - REQ-RELIABLE-RESOURCES-001
system_design:
  - ../../specs/reliable-collaboration/system-design/large-resource-projects.md
legacy_specs: []
---

# Implementation Plan: Large Question-Bank Projects

## Overview

Fix resource-heavy project admission without turning off candidate isolation or evidence verification. Implement bounded resource-aware preparation first, materialization and evidence second, and user-facing compatibility/deployment last. Execute the work orders sequentially in the main conversation after the design-package handoff.

## Confirmed intent and observations

- The user requested a fix for their blocked teaching project, continued Codex/Harness collaboration, and existing audit/review/three-round protections.
- Read-only inventory found 153344 tracked entries. The proposed binary classification accounts for 149752 entries and leaves 3592 source/text entries, including 1892 SVG files. Actual size inspection then identified large JSON/JSONL question data and EPUB/GZip resources. The final profile leaves 3263 source/text files and defers 150081 inputs. It did not read every resource or verify application correctness.
- The current blocked task has no candidate, round zero, and no active model call. Its assignments were changed by the user and must be retained.
- The design handoff was followed by the user's explicit implementation request. This package is now implemented and locally deployed as 1.9.0; the current development task is not automatically resumed.

## Scope

### In scope

Clean existing Git repositories, bounded sparse preparation, immutable resource inventory, independent materialization, truthful source/read evidence, cancellation/retry, legacy evidence, and a localized continuation path.

### Out of scope

Deleting/moving the question bank, rewriting user Git history, uploading, original-source links, changing native login/skills/plugins, automatic acceptance, arbitrary resource edits, and automatically upgrading ordinary-folder snapshot policy.

## Technical approach

- `workspace.go`, proposed `resource_workspace.go` and `resource_snapshot.go`: bounded inventory, candidate sparse setup and explicit snapshot-version branch.
- `engine.go`, `store.go`, `local_project.go`: persisted resource-contract descriptor and recovery intent, while retaining external repository admission and project reservation.
- `dispatch.go`, `stages.go`, `domain_execution.go`: bounded resource requests, per-invocation availability and unchanged implementation round accounting.
- `evidence.go`, patch export/application: bind resource manifest and request records, reject resource mutations and preserve old seals.
- `page.js` and existing activity producer: localized reasons and counts only. No layout redesign or model-selection change.
- Reconcile AC-RELIABLE-LOOP-001.7, AC-RELIABLE-INSPECT-001.3 and the corresponding current designs when this new exception is verified; leave this draft unimplemented until then.

Provider compatibility follows the matrix in the paired design. Both native transports use the same controller resource-request contract. Missing/malformed resource support remains an explicit failed or preparation response, never an implied passing implementation.

## ASCII UI preview

UI-01: Existing activity/status region, shared desktop and phone composition:

```text
Preparing independent candidate
Source: 3,263 files | Deferred input resources: 150,081
Current activity: verifying requested input resources
[Cancel]

Paused: requested resource is missing or exceeds the input limit.
[Continue after resolving] [Stop task]
```

Keep this region inside the existing task flow; it is not a new modal or form. Counts are actual observed inventory counts, and phone rows wrap without horizontal overflow. Chinese product labels are required; English above describes ordering. No new user configuration is necessary.

## Tests

| Contract | Planned primary regression owner |
| --- | --- |
| Admission, accounting and sparse isolation | `resource_workspace_test.go`: `TestLargeResourcePreparation`, `TestWorktreeRejectsOversizedIndexBeforeCopy`, `TestSparsePreparationRecovery` |
| Hydration, original isolation and unsafe destinations | `resource_materialize_test.go`: `TestResourceMaterialization`, `TestResourceDestinationGuard`, `TestMaterializationCancellation` |
| Citations, resource drift and bounded requests | `resource_evidence_test.go`: `TestDeferredResourceReadRejected`, `TestResourceRequestDispatch`, `TestResourceDriftBlocksAcceptance` |
| Existing seals and real app admission | Existing `workspace_limits_test.go` opt-in plus new resource-profile opt-in; legacy sealed-candidate validator |

These names are implementation targets, not existing passing tests. TDD must first reproduce the 30000-entry failure at the real preparation boundary, then prove each new contract.

## E2E tests

Extend the existing `scripts/reliable/desktop-browser-check.cjs` to verify resource preparation/count/error messages and continuation in desktop and mobile viewports. Use a real local HTTP/store fixture with a fake external model boundary, verify that missing resources never enable acceptance, and retain the model drawer scenarios. Map UI-01 to AC-RELIABLE-RESOURCES-001.7.

## Work orders

- [x] [Task 01: Resource-aware preparation](task-01-resource-preparation.md)
- [x] [Task 02: Materialization and evidence](task-02-resource-evidence.md)
- [x] [Task 03: Compatibility and local delivery](task-03-compatibility-delivery.md)

## Verification results

- The full reliable-package race suite passed 124 top-level tests with eight opt-in skips (479.440 s). After the final resource patch correction, the focused resource/ordinary-patch race suite passed all 12 cases (214.657 s). A separate schema/request protocol race check passed. No skipped paid native test is counted as a live model verification.
- Actual project preparation admitted 3263 source/text entries and 150081 deferred resources. The final opt-in run also hydrated an actual WMF and `data/questions_phy.json`, preserved logical source identity and the original Git index/HEAD/contents, and passed in 120.12 s.
- The final 36 browser scenarios passed, including old-limit retry, desktop/mobile resource errors, the model drawer and existing conversation flows. Final-binary production checks retained eight tasks, native choices and original background; no task-setting writes, browser errors or paid model calls occurred.
- Go vet and owning golangci-lint passed (zero issues). Public-doc checks passed 62 tests and validated 47 published pages. The documentation catalog validates 351 decisions and 1356 specifications. Global specification lint retains unrelated baseline findings; this owner has zero findings. A pre-existing linter test's Windows symlink fixture needs an unavailable privilege; no privilege or test was changed.
- Installed 1.9.0 binary SHA-256: `e76af2c11b6a3afbf8d2a2fac8b6025c1b81c482dad9fc3f145c1fb9befd5469`. Backup: `E:\小布丁\backups\upgrade-1.9.0-20261020-124858`. Eight stopped tasks and one pending seal were verified against the closed DB copy without applying changes.
- Production remains paused on the original blocked task at round zero; explicit user continuation starts the next workflow action. Complete logs and runtime receipts are in `E:\小布丁\data\validation\upgrade-1.9.0`.
- Source edits remain local and uncommitted; no push, PR, native settings change or new skill/plugin installation was performed. The source bundle/patch receipt retains the complete changed and new files for review.

## Risks

- Removing the file ceiling alone would still repeat a large copy/scan.
- Sparse missing resources can cause misleading integration failures unless explicitly hydrated and verified.
- Resource availability must not make source identity change or create invented-read evidence.
- Existing seals must remain verifiable through their original snapshot algorithm.
- Per-worktree Git configuration and interrupted initialization must not alter original sparse state or discard candidate work.
- No performance promise is made before measuring actual source preparation on the user's repository.
