---
id: "01-local-project"
title: "Admit and prepare local projects with real verification"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-RELIABLE-SNAPSHOT-001
  - REQ-RELIABLE-BOOTSTRAP-001
acceptance_criteria:
  - AC-RELIABLE-SNAPSHOT-001.1
  - AC-RELIABLE-SNAPSHOT-001.2
  - AC-RELIABLE-SNAPSHOT-001.3
  - AC-RELIABLE-SNAPSHOT-001.4
  - AC-RELIABLE-SNAPSHOT-001.5
  - AC-RELIABLE-SNAPSHOT-001.6
  - AC-RELIABLE-SNAPSHOT-001.7
  - AC-RELIABLE-SNAPSHOT-001.8
  - AC-RELIABLE-SNAPSHOT-001.9
  - AC-RELIABLE-BOOTSTRAP-001.1
  - AC-RELIABLE-BOOTSTRAP-001.2
  - AC-RELIABLE-BOOTSTRAP-001.3
  - AC-RELIABLE-BOOTSTRAP-001.4
  - AC-RELIABLE-BOOTSTRAP-001.5
  - AC-RELIABLE-BOOTSTRAP-001.6
  - AC-RELIABLE-BOOTSTRAP-001.7
system_design:
  - ../../specs/reliable-collaboration/system-design/local-snapshots.md
---

# Task 01: Admit and Prepare Local Projects with Real Verification

## Summary

Deliver ordinary-folder admission through local recovery, isolated implementation, actual verification and explicit acceptance. Preserve external Git ownership, legacy evidence and the shared three-round limit.

## In scope

- Reserve canonical projects before mutation; prepare eligible managed roots and protect staging, credentials and coverage.
- Recover/cancel preparation without repairing unrelated Git or rebasing the original after a worktree exists.
- Establish real checks, freeze/seal their plan and retain strict review/application gates.
- Update localized copy/states and focused draft/state browser checks.
- Record same-task initialization intent, safe interrupted-index limits, and explicit path exclusions without claiming comprehensive secret scanning.

## Out of scope

Remote operations, global configuration, commits of Agent results, broad tools roots, account integration changes and UI redesign.

## Acceptance

- A plain temporary folder can start with an inspectable local recovery point and separate worktree; acceptance requires actual successful checks and independent review.
- Unsafe/conflicting roots, manual staging, modified evidence and original drift stop without unintended source/index mutation. Failed new-folder preparation may retain controller-created metadata; an unowned staged index blocks without automatic reset or recommit.
- Cancellation/restart, explicit commands and legacy candidates retain task identity, evidence, source authority and budget.

## ASCII UI preview

UI-LOCAL-01 follows [the combined preview](plan.md#ascii-ui-preview); phones use the current stacked controls. This owns AC-RELIABLE-SNAPSHOT-001.9 and AC-RELIABLE-BOOTSTRAP-001.1/.4 alongside the backend criteria.

```text
[ Check and start collaboration ]
Preparing local recovery point... [ Stop task ]
Verification pending: establishing real project checks.
Failure: <localized next action>; submitted choices retained.
```

## Verification

Run from the repository root in PowerShell with the existing Go/compiler environment. Package commands use explicit package roots.

```powershell
go -C apps/backend test -race -trimpath -p 4 ./internal/reliable -count=1
go -C apps/backend test -race -trimpath -p 4 ./internal/backendapp -run TestReliableRoutes -count=1
go -C apps/backend vet ./internal/reliable
Push-Location apps/backend
try { golangci-lint run ./internal/reliable --new=false --timeout=3m } finally { Pop-Location }
node --test apps/backend/internal/reliable/page-state.test.cjs
Run the local project browser validation for your checkout.
node tools/domain-browser-check.cjs
git diff --check -- docs/decisions docs/specs/reliable-collaboration docs/plans/automatic-local-project
git status --short -- docs/plans/automatic-local-project
```

Run `python scripts/list-docs.py validate` and `python scripts/lint-spec-files.py --all` against the assembled full checkout containing their scripts/configuration. Record new owning-system findings separately from an unchanged upstream baseline. The owning implementations and tests are listed below; the plan maps their actual test names to criteria.

## Files likely touched

- `apps/backend/internal/reliable/http.go`, `setup_actions.go`, `engine.go`, `states.go`, `store.go`.
- `local_project.go`, `local_project_policy.go`, `local_project_snapshot.go`, `local_project_recovery.go`, `automatic_checks.go`, `workspace.go`, `process.go`, `stages.go`, prompts and seal helpers.
- `local_folder_http_test.go`, `bootstrap_http_test.go`, `local_project_safety_test.go` and existing admission/acceptance/source-limit suites in the same package.
- `page.html`, `page.js`, `page-state.test.cjs` and focused browser tooling.
- Owning requirement, design, decision and plan documents.

## Dependencies

None. Implement the mutation boundary before check execution and UI projection within this vertical slice.

## Risks

Git alone does not prove managed ownership. New flags/artifacts must preserve legacy defaults and sealed bytes. Missing verification consumes existing rework rather than creating an unlimited preparation loop. Explicit exclusions are path-based; source is not comprehensively scanned for secrets. A staged index without a pending checkpoint proof is preserved for maintainer recovery.

## Parallelism

`sequential`

## Inputs

- [Local project requirements](../../specs/reliable-collaboration/requirements/local-snapshots.md).
- [Local project design](../../specs/reliable-collaboration/system-design/local-snapshots.md).
- [Recovery decision](../../decisions/2026-10-19-managed-local-project-snapshots.md).
- Existing admission, source-limit, recovery, acceptance, domain-scope and draft tests.

## Results

Done. The reliable-package race run passed in 137.415 seconds; eight targeted race items after the zero-test reporting fix passed in 27.092 seconds. Static checking reported zero issues and the scoped backend route test passed in 9.567 seconds. Local-project/domain/legacy browser scenarios passed 5/5, 6/6 and 5/5; state checks passed 4/4. Native empty-folder proof task `q5zt2jbj4iket2u7rdiwbgkt4q` used four independent sessions in round one, froze `npm run test`, and passed five assertions after explicit application to its disposable sample without a Git remote.

Version 1.3.0 is deployed. The [plan](plan.md#verification-results) records its executable hash, rollback backup, preservation of all five original task states/rounds/seals, and the separate 0.65-second read-only validation of an old sealed candidate. Deployed UI smoke checks passed without page-script errors; original Kandev returned HTTP 200. Runtime catalog observations were six Codex/two Harness options, versus seven/two at candidate startup.

Five broader backendapp failures reproduced on clean base `d13a69d0` in 11.775 seconds; no full-Kandev-suite pass is claimed. Scoped docs checks passed; complete-checkout documentation/diff gates remain the final integration owner's responsibility before the normal local source commit. No push or remote publication is included.

Final assembly gates: the deployed full checkout validated 351 decisions and 1350 specifications. Owning reliable-collaboration spec findings remain zero; the unchanged upstream full-spec baseline has 107 size findings. Staged whitespace and changed-Go formatting checks passed. Browser harnesses and deployment tools are retained in the local assembly workspace; backend implementation and behavior tests are versioned in this source checkout. The deployed HTTP smoke test checks the actual executable independently of that assembly copy.
