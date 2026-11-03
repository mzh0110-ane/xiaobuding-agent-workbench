---
id: task-01-navigation
title: Project navigation and lightweight workbench reads
status: completed
wave: 1
depends_on: []
plan: plan.md
requirements:
  - REQ-RELIABLE-WORKBENCH-001
  - REQ-RELIABLE-WORKBENCH-003
acceptance_criteria:
  - AC-RELIABLE-WORKBENCH-001.1
  - AC-RELIABLE-WORKBENCH-001.2
  - AC-RELIABLE-WORKBENCH-001.3
  - AC-RELIABLE-WORKBENCH-001.4
  - AC-RELIABLE-WORKBENCH-003.1
system_design:
  - ../../specs/reliable-collaboration/system-design/workbench.md
---

# Project Navigation and Lightweight Reads

## Outcome and scope

Load project navigation independently from selected-task detail and old messages.
Retain the existing compact workspace and make recovery causes, real execution,
and future Agent settings inspectable. Use the existing embedded frontend; do
not migrate frameworks, native accounts, or ordinary Kandev task APIs.

## Acceptance

1. Summaries omit full histories; conditional selected-task reads and older
   25-message pages preserve data, drafts, scroll position, and selection identity.
2. Canonical projects organize current/history batches without merging unrelated
   folders. Desktop and phone retain usable conversation, results, and settings.
3. Blockers show actionable localized guidance with raw details available,
   without making guidance an authorization to resume or broaden scope.

## ASCII UI preview

UI-WORKBENCH-01, excerpt from the [combined preview](plan.md#ascii-ui-preview):

```text
Projects / current + history | Project / state / actual Agent | [Agent panel]
Needs attention / acceptance| Conversation / Execution       | Next settings
                            | Results / Project plan         | [Collapse]
                            | Scrollable content             |
                            | Fixed draft + attachment + Send|
```

UI-WORKBENCH-02: On phones, Projects and Agents open dismissible overlays above
the same central tabs and fixed composer. Test keyboard focus and long localized
labels, not ASCII geometry. This maps to AC-RELIABLE-WORKBENCH-001.1 through .4.

## Files and dependencies

`workbench_http.go`, `http.go`, embedded page modules, the existing project-key
canonicalizer, and the native model-plan editor. Result and budget modules from
later work orders use this shell; they do not change legacy full-read responses.

## Verification

```sh
# From apps/backend; use the established E-drive runtime/cache environment.
go test -race -trimpath -p 1 ./internal/reliable -run '^TestWorkbench' -count=1
# From the repository root.
node --test apps/backend/internal/reliable/page-workbench-state.test.cjs
node --test apps/backend/internal/reliable/page-workspace-state.test.cjs
node --test apps/backend/internal/reliable/page-state.test.cjs
node scripts/reliable/desktop-browser-check.cjs
```

Extend the owning browser fixture for lightweight views, required new modules,
history paging, project groups, stale responses, drafts, and the phone overlays
before counting it as new-feature browser coverage.

## Results

Implementation and Harness layout work are complete. All 54 combined page-logic
checks and 27 isolated browser checks passed, including message paging/backfill,
full-goal search, stale responses, project groups, retained drafts, and phone
light/dark layouts. Browser requests made no production task mutations.

Four maintained browser regression cases additionally passed, covering the
workspace panels/composer and attachments (see `validation/legacy-browser/results.json`).
These are separate from the 27 new scenarios, not a replacement for broader
legacy-suite coverage.

The desktop summary measured 135.8 pixels, composer 137, and central content
613.2 at a 1440-by-1000 viewport. Production summary payload measured 7,564 bytes
versus 2,770,337 bytes for the legacy full response (approximately 99.7% less);
selected workspace measured 13,705 bytes. Unchanged conditional reads returned
304. These are payload measurements, not end-to-end Agent speedup claims.

Candidate/runtime startup and 1.15.0 deployment passed, with all 13 raw task
records unchanged and no production task mutations. Task 03 also verified the
actual original desktop shortcut launch against the deployed runtime.
