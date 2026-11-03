---
created: 2026-10-20
status: draft
requirements:
  - REQ-RELIABLE-EFFICIENCY-001
  - REQ-RELIABLE-RESULTS-001
  - REQ-RELIABLE-PREVIEW-001
system_design:
  - ../../specs/reliable-collaboration/system-design/execution-efficiency.md
  - ../../specs/reliable-collaboration/system-design/task-results.md
legacy_specs: []
---

# Implementation Plan: Execution Efficiency and Task Results

## Overview

First remove waiting after a native turn has completed, then reduce repeated
preparation without removing integrity boundaries. Add an evidence-bound result
card and viewer, followed by explicit local website preview. No production or
permanent test changes are authorized by this design package alone.

The later requested [cancelled-task recovery](../reliable-cancelled-recovery/plan.md)
has its own priority package and can be implemented independently. This package
does not implicitly resume cancelled work or change its budget.

Owner: reliable-collaboration owns dispatch completion and task-result evidence;
upstream task/session and reusable UI systems do not own these local contracts.

## Scope and assumption check

Confirmed: preserve Codex ChatGPT login and DeepSeek Harness native login,
existing skills/plugins, user-selected role models/efforts, independent Codex
review, evidence audit, three rounds per batch and explicit human acceptance.
The user wants shorter waits and clickable task outputs similar to desktop
agent result cards. Local deployment remains `E:\小布丁` with its existing
shortcut. This is a new behavior package, not continuation of the completed
large-resource work orders.

Verified: a native reviewer returned a `rework` report about twenty minutes
before dispatch timed out; the current task is cancelled at 1/3. Its saved
domain reports describe thirteen changed files, but `Task.Review` is absent.
The current UI hides the patch link without a review, displays no implementation
summary, and hardcodes the first log link using the current round. `StartRound`
already clears task-level checks/review; historical selectors and interrupted
stage publication still need explicit artifact identities. Initial preparation
repeats admission three times. An observed eleven-minute snapshot interval is
not yet attributable to one function. Actual controller checks took only about
three seconds in the observed run; removing checks has little value.

Working decisions: preview means an actual supported webpage when available,
otherwise files/reports with an explicit limitation. Preserve the existing
portrait, theme and Agent drawer. Do not add a new login, model preset, skill
collection or private-reasoning stream. No material question blocks planning.

### In scope

- Native completion teardown, cancellation correctness and timing evidence.
- Attempt/boundary-scoped preparation reuse, without mtime-only trust.
- Exact round/source result projection, card, per-file diff and all check logs.
- Static/Flask preview in separate disposable copies with explicit lifecycle.
- Local delivery after the implementation checks, preserving state and seals.

### Out of scope

Parallel writers sharing one candidate, automatic effort reductions, fewer
checks, automatic application/resume of cancelled work, arbitrary framework
launchers, dependency installs, tunnels, remote publishing and malicious-code
OS isolation. No measured speed multiplier is promised before implementation.

## Technical approach

### 01. Native completion

Add a native-only supervisor and protocol validators beside `process.go` and
`harness.go`. Preserve exit-based `runOwnedCommand` for Git/checks. Keep terminal
evidence, controlled cleanup and cancellation separate; consume trailing output
before final validation. Codex/Harness differ in event/session/report shapes.
Use five seconds of natural-exit grace and two seconds of drain/cleanup. Fail
closed if cleanup is ambiguous. Add diagnostics outside seal inputs.

| Provider/transport | Identity | Intended behavior | Evidence and fallback |
| --- | --- | --- | --- |
| Codex CLI JSON | Runtime thread + turn + output report | Bounded post-completion cleanup | Helper executable; incomplete/unknown evidence keeps bounded failure |
| Harness CLI JSON | Project/session + turn_end + final | Same lifecycle, strict native parser | Helper executable; no Codex-event assumption |
| Git/check process | Exit status | Existing behavior | Existing Windows descendant tests |
| Old dispatch | Saved report/error | Display only | Never reinterpret/replay |

### 02. Preparation and timing

Instrument `Engine` preparation, `workspace.go`, `resource_inventory.go`,
`resource_workspace.go`, `TakeSnapshot`, domain boundaries and dispatch before
changing reuse. Transfer one validated inventory through preparation; pass one
immutable snapshot within an actor-free handoff. Cache parsed profiles by
verified content digest within an attempt. Full content checks still surround
actors, verification, review and acceptance. New timing/status projection must
not enter old evidence hashes or perform work on GET.

### 03. Results and file viewer

Add proposed `result_projection.go`, `result_http.go`, `page-results.js` and
focused result fixtures. Integrate them through `http.go`'s embed composition
and `page.js` near the task status. Correct `page-proof.js` links to exact
evidence references. Serve bounded read-only result metadata and lazy file/diff
content; keep evidence plain text. Record new stage result indexes separately
from seals and adapt legacy reports without migration. Rendering claims never
overrides `acceptanceEvidence`.

### 04. Explicit preview and local delivery

Implement typed static/Flask recipes, `PreviewManager`, separate copies/origins,
owned process lifetime and readiness. Extend the result card with one clear
preview action, status and stop/log controls. Existing Flask project needs its
runtime; its Jinja template is not standalone output. No tests import or launch
the real user's app or call paid APIs. After all work-order checks pass, stage
a candidate binary, verify on isolated data, back up source/runtime/closed data,
and replace the installed binary only when production is idle. Preserve tasks,
models, appearance, shortcut and pending seals. Record the exact artifact hash
and results in local delivery documents; do not claim deployment before it.

## ASCII UI preview

UI labels below illustrate structure in English; delivered copy is Chinese.
Spacing is illustrative. Summary placement, truthfulness, lazy detail, distinct
phone composition and labelled preview state are requirements.

### UI-01: Desktop task result, candidate awaiting acceptance

Entry: select a task. Existing sidebar/header and Agent settings remain fixed;
the main task document scrolls. Result drawer is opened explicitly.

```text
+------------------+-----------------------------------------------+
| Tasks / Archive  | Task title                    [Agent settings]|
| Selected task    | Status: awaiting your acceptance              |
|                  +-----------------------------------------------+
|                  | Task result                  [Candidate]      |
|                  | Saved implementation summary                  |
|                  | 13 changed files | Checks: pass | Review: pass |
|                  | [Preview website] [Changes (13)] [Open copy]   |
|                  | [Accept and apply]      > Evidence / limits   |
|                  +-----------------------------------------------+
|                  | Continue this project ...                     |
+------------------+-----------------------------------------------+
                      Opens [Changes (13)] ->
                     +---------------------+------------------------+
                     | Files (exact round) | path/to/file           |
                     | > templates/...    | [Diff] [Source] [Close]|
                     |   app.py           | bounded read-only view |
                     +---------------------+------------------------+
```

UI-01 maps to AC-RELIABLE-RESULTS-001.1-.5 and
AC-RELIABLE-PREVIEW-001.1. If preview is unsupported the control says setup
needed/unavailable with the reason; it does not open a broken template.

### UI-02: Partial, historical and evidence failure states

```text
+-------------------------------------------------------------+
| Task result                         [Cancelled / Partial]    |
| Saved summary (round 1, source generation ...)               |
| Changes: 13 | Checks: recorded pass | Review: not established |
| Raw reviewer output exists; it was not accepted by controller.|
| [Inspect partial result] [Changes] [Open working copy]        |
| No accept action.  > Raw reports / historical evidence        |
+-------------------------------------------------------------+
| [Earlier round v]  Round 1 evidence; current round is 2.      |
| Evidence unavailable: missing or changed report. [Retry]      |
+-------------------------------------------------------------+
```

UI-02 maps to AC-RELIABLE-RESULTS-001.1-.4/.6. Loading reserves card space and
announces busy state. Empty read-only inspection shows its investigation report
and explicitly says no code/check/review result exists. Applied results say
applied, with original folder access. There is no success inferred from prose.

### UI-03: Phone composition and focused viewer

```text
+-------------------------------+  +------------------------------+
| [Menu] Task  [Agent settings] |  | [Back] Changes, round 1 [X]  |
| Task result       [Candidate] |  | [Select file              v] |
| Summary wraps across lines.  |  | [Diff] [Source]               |
| Checks / review / limitations|  | Code pane (local scrolling)   |
| [Preview website           ] |  |                              |
| [Changes (13)] [Open copy]   |  +------------------------------+
| [Accept and apply          ] |
| > Evidence and limitations   |
| Continue project ...         |
+-------------------------------+
```

UI-03 maps to AC-RELIABLE-RESULTS-001.5. No two-column compression, whole-page
horizontal scroll or lost close/focus target at 390 px. Drawer/dialog close
restores focus. Use existing semantic controls and theme/background tokens.

### UI-04: Explicit preview lifecycle

```text
+-------------------------------------------------------------+
| Website preview | Candidate, round 1 | Not a verification     |
| Preparing copy: source verified, requested resources ...      |
| [Stop] > Preparation log                                     |
| Starting: waiting for this app's health response ...          |
| Ready: http://127.0.0.1:<owned-port>/... [Open] [Stop]          |
| Failed: missing Flask dependency. [Retry] [View log]           |
| Stopped / expired: [Start preview]                            |
+-------------------------------------------------------------+
```

UI-04 maps to AC-RELIABLE-PREVIEW-001.1-.6. Only one state row is visible at a
time. Phone stacks actions beneath URL/status. Opening a result never starts
this process. An old generation is labelled; polling never resets the process.

## Tests

All newly named tests below are planned, not already passing evidence.

| Acceptance | Targeted executable evidence |
| --- | --- |
| EFFICIENCY-001.1/.2/.7 | `native_process_test.go`: TestNativeCompletionLifecycle, TestNativeCompletionRejectsInvalidEvidence; Windows ownership tests |
| EFFICIENCY-001.3 | `timing_test.go`: TestReliableTimingProjection; UI current-stage/cleanup state |
| EFFICIENCY-001.4/.5/.6 | `handoff_snapshot_test.go`: TestHandoffSnapshotReuse, TestPreparationInventoryReuse, TestReuseDetectsTampering; cold/warm benchmark and existing resource/acceptance tests |
| RESULTS-001.1/.2/.3/.6 | `result_projection_test.go`: TestResultProjection, TestResultRoundProvenance, TestResultLegacyDomainReports |
| RESULTS-001.3/.4 | `result_http_test.go`: TestResultArtifactBounds, TestResultReadHasNoSideEffects; sealed fixture before/after read |
| RESULTS-001.5 | Browser desktop/phone result drawer, keyboard/focus and overflow scenarios |
| PREVIEW-001.1/.2/.5 | `preview_test.go`: TestPreviewLifecycle, TestPreviewRecipes, TestPreviewReadinessOwnership |
| PREVIEW-001.3/.4/.6 | `preview_test.go`: TestPreviewIsolationAndAuthorization, TestPreviewResourceStaging, TestPreviewAcceptanceSerialization |

## E2E tests

Extend `scripts/reliable/desktop-browser-check.cjs`, its headless Edge desktop
1440 px and phone 390 px contexts. Cover: awaiting acceptance vs applied vs
cancelled/raw review; previous-round pass vs current incomplete round; distinct
log links; missing/tampered artifacts; result loading/error/retry; file viewer
keyboard/focus; rapid task/result changes; archive/reload preservation;
unsupported preview; static/Flask preview preparing/ready/failure/stop/expiry.
Assert result polling makes no mutation/model/check/start request. Add an
isolated static/minimal Flask fixture to the owned-preview integration checks;
never launch the real application. Keep all existing browser scenarios passing.

## Work orders

- [ ] [Task 01: Bound native completion cleanup](task-01-native-completion.md)
- [ ] [Task 02: Remove duplicate preparation and expose timing](task-02-preparation-reuse.md)
- [ ] [Task 03: Show evidence-bound result cards and files](task-03-result-cards.md)
- [ ] [Task 04: Add explicit local website previews and deliver locally](task-04-local-preview.md)

Dependency order: 01 -> 02 -> 03 -> 04. Each is an independently verifiable
slice. Sequential integration avoids concurrent edits to process, HTTP and
page composition. Investigations may be delegated read-only under platform
rules; this package does not assign model tiers or change user selections.

## Verification commands and environment

Run from repository root in PowerShell. Existing local toolchain only:

```powershell
$env:PATH = 'E:\小布丁\tools\go\bin;E:\小布丁\tools\llvm-mingw-20261006-ucrt-x86_64\bin;' + $env:PATH
$env:GOMODCACHE = 'E:\小布丁\tools\gomod'
$env:GOCACHE = 'E:\小布丁\tools\gocache'
$env:CGO_ENABLED = '1'
$env:CC = 'clang'
$env:GOMAXPROCS = '4'
$env:PYTHONUTF8 = '1'
```

Each work order gives its exact Go/browser checks. The complete implemented
package additionally requires these bounded integration/document gates:

```powershell
go -C apps/backend test -trimpath -race ./internal/reliable -count=1
go -C apps/backend test -trimpath ./internal/backendapp -run Reliable -count=1
go -C apps/backend vet ./internal/reliable
$env:XIAOBUDING_VALIDATION_DIR = 'E:\小布丁\data\validation\upgrade-1.10.0'
node scripts/reliable/desktop-browser-check.cjs
node --test scripts/validate-public-docs.test.mjs
node scripts/validate-public-docs.mjs
python -B scripts/list-docs.py validate
python -B scripts/lint-spec-files.test.py
python -B scripts/lint-spec-files.py --all
git diff --check
```

Baseline global spec lint has unrelated failures and Windows spec-linter tests
have one symlink-privilege limitation; record exact fresh results and require
zero newly introduced or owning-system findings. Do not disguise baseline
failures as a full pass or change privileges. Run local PR-documentation
coverage through `.github/scripts/pr-docs.cjs: validateCoverage` with in-memory
changed-file/document inputs; do not call GitHub or publish status from planning.

## Verification results

Implementation remains pending; all four work orders are `pending`. No
production or permanent test source was changed for this package, and no
runtime deployment or paid native call was performed.

Design-package validation on 2026-10-20:

- `python -B scripts/list-docs.py validate`: passed, 352 decisions and 1360
  specifications; all four new draft specification files and the ADR are
  discoverable in their catalogs.
- `python -B scripts/lint-spec-files.py --all`: 107 existing unrelated file-size
  findings, zero reliable-collaboration findings. No new size exception added.
- `python -B scripts/lint-spec-files.test.py`: 36 cases, 35 passed, one existing
  Windows `WinError 1314` symlink-creation privilege error. No privilege change.
- Read-only package preflight: all ten document files have valid existing local
  Markdown link targets, known work-order requirement/acceptance IDs, UTF-8
  without BOM, and sizes within their applicable ceilings.
- `.github/scripts/pr-docs.cjs: validateCoverage`: covered, four work orders,
  zero errors using all package contents plus one prospective `process.go`
  triggering path. This is local reference validation, not a live PR/CI status.
- Bounded independent read-only contract review found and corrected a mistaken
  StartRound persistence assumption, one broken continuation link, missing
  preview setup/heartbeat route definitions and the need for a controller-owned
  WSGI authority wrapper. The bounded recheck found no remaining blocking
  contradiction. This does not replace implementation tests or PR review.
- `git diff --check` on the package/spec/ADR paths passed; files remain
  unstaged and uncommitted for the design handoff. Existing 1.9.0 source changes
  remain untouched.

## Risks

Native event/output-file ordering, Windows pipe/Job Object cleanup races,
same-time content tampering, stale round evidence, large deferred resource
inputs, Flask import/data dependencies and preview code with absolute writes
need focused coverage. Preview copy isolation is not an OS sandbox. Cold-start
latency must be measured before attributing it or promising acceleration.
