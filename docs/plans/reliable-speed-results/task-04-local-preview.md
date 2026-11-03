---
id: "04-local-preview"
title: "Add explicit local website previews and deliver locally"
status: pending
wave: 4
depends_on:
  - 03-result-cards
plan: "plan.md"
requirements:
  - REQ-RELIABLE-EFFICIENCY-001
  - REQ-RELIABLE-RESULTS-001
  - REQ-RELIABLE-PREVIEW-001
acceptance_criteria:
  - AC-RELIABLE-EFFICIENCY-001.7
  - AC-RELIABLE-RESULTS-001.4
  - AC-RELIABLE-PREVIEW-001.1
  - AC-RELIABLE-PREVIEW-001.2
  - AC-RELIABLE-PREVIEW-001.3
  - AC-RELIABLE-PREVIEW-001.4
  - AC-RELIABLE-PREVIEW-001.5
  - AC-RELIABLE-PREVIEW-001.6
system_design:
  - ../../specs/reliable-collaboration/system-design/execution-efficiency.md
  - ../../specs/reliable-collaboration/system-design/task-results.md
---

# Task 04: Add Explicit Local Website Previews and Deliver Locally

## Summary

Make supported result websites open from the result card through an explicitly
owned preview lifecycle. Ship the integrated local extension only after the
implementation checks pass, retaining the installed state and historical seals.

## In scope

- Static and typed Flask recipes with a controller-owned WSGI bootstrap and
  enforced preview capability middleware; existing environment selection, source and
  resource staging to a disposable copy; separate loopback content origin.
- Explicit start/status/heartbeat/stop endpoints with task/result identity,
  capability checks, process/readiness ownership, bounded logs and expiration.
- Preview card states/UI-04, startup failures, blocked opening fallback and
  serialized acceptance/preview selection; shutdown cleanup integration.
- Update local/public usage docs, isolated candidate build/validation, idle-only
  local replacement, source/runtime/data backup and a delivery receipt.

## Out of scope

Arbitrary commands/proxies/frameworks, automatic dependency installation or
tunnels, OS sandboxing, real app paid API calls, resuming cancelled tasks,
accepting demos or modifying the user's original project during validation.

## Acceptance

1. Explicit static/Flask fixture previews run from verified disposable input
   generations, open on their separate loopback origins only after readiness,
   and show accurate missing-input/dependency/startup/exit/expiry states.
2. GET/polling cannot start code; client paths/commands/URLs cannot select
   arbitrary execution or reads; stop/shutdown/cancellation remove owned
   descendants without changing original/candidate/sealed evidence or rounds.
3. Completed targeted/package checks and isolated installation smoke checks are
   recorded with real outputs. Local delivery preserves tasks/preferences/seals
   and records its binary hash/backup/rollback path; active production work
   defers replacement instead of being cancelled for deployment.

## ASCII UI preview

[UI-04](plan.md#ui-04-explicit-preview-lifecycle), mapped to
AC-RELIABLE-PREVIEW-001.1-.6:

```text
+--------------------------------------------------------+
| Website preview | Candidate / round 1 | Not verification |
| Preparing copy -> Starting -> Ready                     |
| Ready URL: http://127.0.0.1:<owned-port>/...              |
| [Open website] [Stop preview] > Logs                     |
| Failed / expired: reason. [Retry] [Open copy]            |
+--------------------------------------------------------+
```

Only the current state is shown. Desktop uses the compact card; phone wraps URL
and stacks actions below. Stop is available during preparation/startup. Never
show a Flask template as a ready static website. Keep UI-01 result acceptance
separate; no preview request performs acceptance.

## Verification

From repository root after plan environment setup. New test names are planned:

```powershell
go -C apps/backend test -trimpath -race ./internal/reliable -run 'TestPreviewLifecycle|TestPreviewRecipes|TestPreviewReadinessOwnership|TestPreviewIsolationAndAuthorization|TestPreviewResourceStaging|TestPreviewAcceptanceSerialization' -count=1 -v
go -C apps/backend test -trimpath ./internal/backendapp -run Reliable -count=1 -v
node scripts/reliable/desktop-browser-check.cjs
node --test scripts/validate-public-docs.test.mjs
node scripts/validate-public-docs.mjs
```

After these pass, run the plan's package/document integration commands exactly
once; rerun affected checks only for new changes/failures. Build using the
existing local release method and validate with a separate port/data root
before replacement. No native model calls are needed to prove the fixture
flows. Run recorded-state/projection smoke checks against production read-only.
Do not claim an end-to-end paid model run or actual user feature correctness.

## Files likely touched

Proposed preview manager/recipe/staging/server files, embedded
`preview-bootstrap.py` and `preview_test.go` under
`apps/backend/internal/reliable`; existing `http.go`, `engine.go`, result UI,
`apps/backend/internal/backendapp/reliable_routes.go`/tests,
`scripts/reliable/desktop-browser-check.cjs`, public task documentation and the
local deployment receipt/usage documents. Retain current shortcut targets.

## Dependencies

Task 03 result identities and file descriptors; Task 01 process ownership and
Task 02 frozen prepared input/integrity semantics. Use the paired preview ADR.

## Risks

Flask may write absolute paths despite a disposable cwd. Preview is trusted
project execution, not OS isolation. Large data requirements may exceed preview
input budgets. A port collision cannot be mistaken for owned readiness.

## Parallelism

Sequential. Shared result contracts and local process lifetime must integrate
before installation replacement.

## Inputs

Task-results design, preview ADR, plan UI-01/03/04, existing backend shutdown
and Windows ownership tests, local 1.9.0 receipt and backup layout.

## Results

Pending.
