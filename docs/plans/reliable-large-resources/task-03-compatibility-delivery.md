---
id: "03-compatibility-delivery"
title: "Validate the user project and deliver the local fix"
status: done
wave: 3
depends_on:
  - 02-resource-evidence
plan: "plan.md"
requirements:
  - REQ-RELIABLE-RESOURCES-001
acceptance_criteria:
  - AC-RELIABLE-RESOURCES-001.1
  - AC-RELIABLE-RESOURCES-001.6
  - AC-RELIABLE-RESOURCES-001.7
system_design:
  - ../../specs/reliable-collaboration/system-design/large-resource-projects.md
---

# Task 03: Compatibility and Local Delivery

## Summary

Confirm that the actual question-bank project passes bounded preparation, that old history remains valid, and that the existing interface explains real source/resource progress. Deliver only after the preparation/evidence contracts and legacy acceptance checks pass.

## In scope

- Chinese activity/error copy in existing surfaces, actual inventory counts, clear missing-resource failures and restart/continue behavior.
- Desktop/mobile browser checks, existing model drawer regression, real-project opt-in preparation with no native model call, legacy pending seal verification without application.
- Reconcile active source-ceiling and preparation specs, update public how-to documentation, complete plan results, build candidate, backup, safe local rollout and post-deployment verification.

## Out of scope

Automatic paid model execution, automatically applying any candidate, changing Codex/Harness native settings, layout redesign, resource deletion or source history rewriting.

## Acceptance

1. A representative large project inventory and bounded preparation succeed with measured source/resource counts, original contents/index/HEAD preserved, and no model dispatch. Validate source and selected input resources, not the application's correctness.
2. Desktop/mobile show real source/resource progress and actionable errors. Existing model/effort controls and all saved task states/rounds/seals remain intact; old pending acceptance evidence still verifies without application.
3. Deploy the tested local binary with backup/rollback and verify the existing shortcut/URL. Retain the blocked task's goal, assignments and round zero; leave continuation to the user's explicit action and record this clearly.

## ASCII UI preview

UI-01, existing activity region from [the complete plan](plan.md#ascii-ui-preview):

```text
Source: 3,263 files | Deferred input resources: 150,081
Current activity: verifying requested input resources
[Cancel]

Paused: required input is unavailable.
[Continue after resolving] [Stop task]
```

Chinese product text is required. Desktop and phone share the same order; counts wrap on phones. No new dialog, long form or model drawer layout is introduced. Covers AC-RELIABLE-RESOURCES-001.7.

## Verification

From `apps/backend`:

```powershell
go test -trimpath -race ./internal/reliable -run TestExistingResourceProjectPreparation -count=1 -v
golangci-lint run ./internal/reliable --new-from-rev=61ff2c38447eae434eb9c97c22f1745f3b8b9d1a --timeout=5m
```

The real-project test must be implemented as opt-in: set `XIAOBUDING_RESOURCE_PROJECT` to the exact original root only for the controlled preparation check, then clear it. It must not invoke any native model or apply a patch. Build with the established `-trimpath` Windows toolchain and run the existing legacy sealed-candidate validator against a closed database copy.

From repository root:

```powershell
node scripts/reliable/desktop-browser-check.cjs
python -B scripts/list-docs.py validate
python -B scripts/lint-spec-files.py --all
node --test scripts/validate-public-docs.test.mjs
node scripts/validate-public-docs.mjs
git diff --check
```

Use the deployment/runtime procedure already recorded under `E:\小布丁\data\validation\upgrade-1.8.0`, adapt version-specific paths and assertions, and record every returned long-running process handle until completion. Preserve unrelated upstream specification-lint findings separately from this owner's result. Stop rollout if any task is active or legacy evidence verification fails.

## Files likely touched

`apps/backend/internal/reliable/page.js`, activity producers, opt-in integration tests; `scripts/reliable/desktop-browser-check.cjs`; owning source-limit requirements/designs; `docs/public/tasks-and-workflows.md`; local deployment manifests/reports outside the repository.

## Dependencies

Tasks 01 and 02 and all their final targeted checks. Read `/e2e` and the existing deployment scripts before executing their corresponding work.

## Risks

Incorrectly treating a preparation test as app validation, a real-project check silently making paid calls, unrelated active user tasks during rollout, and source/seal migration changing old evidence.

## Parallelism

Sequential. No subagent delegation is authorized.

## Results

All 36 desktop/mobile browser scenarios passed after the last UI edit. Go vet and owning golangci-lint passed with zero issues. Public-doc checks passed 62 cases and validated 47 pages. The final binary was installed as 1.9.0 with backup, preserving eight tasks, assignments and native settings. A closed DB copy validated one old sealed candidate without application. Exact-binary runtime checks passed native model/effort selection, appearance, desktop/mobile controls and the version-aware retry on the current blocked task. No task-setting write or paid model call occurred during runtime verification.
