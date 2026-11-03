---
id: "01-desktop-workspace"
title: "Deliver the desktop task workspace"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-RELIABLE-DESKTOP-001
  - REQ-RELIABLE-ARCHIVE-001
acceptance_criteria:
  - AC-RELIABLE-DESKTOP-001.1
  - AC-RELIABLE-DESKTOP-001.2
  - AC-RELIABLE-DESKTOP-001.3
  - AC-RELIABLE-DESKTOP-001.4
  - AC-RELIABLE-DESKTOP-001.5
  - AC-RELIABLE-DESKTOP-001.6
  - AC-RELIABLE-DESKTOP-001.7
  - AC-RELIABLE-DESKTOP-001.8
  - AC-RELIABLE-ARCHIVE-001.1
  - AC-RELIABLE-ARCHIVE-001.2
  - AC-RELIABLE-ARCHIVE-001.3
  - AC-RELIABLE-ARCHIVE-001.4
  - AC-RELIABLE-ARCHIVE-001.5
  - AC-RELIABLE-ARCHIVE-001.6
system_design:
  - ../../specs/reliable-collaboration/system-design/desktop-workspace.md
---

# Task 01: Deliver the Desktop Task Workspace

## Summary and scope

Deliver the complete archive, folder-access, and compact-workspace slice in `internal/reliable`. Implement server-authoritative archive metadata and folder targets before connecting sidebar views, shared menus, expandable model settings, and local theme preferences. Keep the current independent review, three-round, and explicit acceptance rules.

The authorized delivery also includes packaging, rollback-backed deployment to the existing `E:\小布丁` installation, preservation of existing task/account data, and verification of the desktop entry and deployed page. The delivery owner confirmed these results in the final production record.

The same delivery includes preserving an existing configured model when a stale native cache omits it. Retain its ID with an explicit unverified label and no invented reasoning-effort metadata; use native metadata again when the ID is later listed. This compatibility fix does not introduce native account or directory-fetch capabilities.

Exclude native account or default assignment changes, fake conversations, task deletion, unrelated public docs, and remote publication.

## Implementation acceptance

- Archive/restore persists separately, rejects every prohibited state, retains an awaiting-acceptance reservation, and preserves existing sealed candidates and execution fields.
- All three local folder targets are derived from the task, fail visibly when absent, and remain guarded explicit local actions.
- Desktop and phone provide the required task navigation and menu entry points, real result hierarchy, complete expandable settings, retained drafts, and persistent appearance selection.

## ASCII UI preview

UI-DESKTOP-01/02/03 follow [the combined desktop, creation, and phone previews](plan.md#ascii-ui-preview).

```text
Desktop: [ Search | Current / Archived | Task ... ] | Result
                                                    > Models
New task: Project -> Outcome -> Work summary -> [Start]
                                > All model / effort settings
Phone: [Tasks drawer] -> Result -> > Models
Menu: Open project / working copy / evidence; Archive / Restore
```

This excerpt owns the same DESKTOP-001.1-.7 and ARCHIVE-001.3/.6 criteria as the combined preview. Active-state archive is disabled with a reason; archived awaiting-acceptance work retains its warning. Menu activation, dismissal and focus, missing-folder errors, narrow layout, and local themes require rendered evidence.

## Verification

Run from the repository root with the existing Go/compiler environment and an isolated temporary `KANDEV_HOME_DIR` for tests that need it. These are intended commands, not completed results.

```powershell
go -C apps/backend test -race -trimpath -p 4 ./internal/reliable -count=1
go -C apps/backend test -race -trimpath -p 4 ./internal/reliable -run '^TestDesktop' -count=1
go -C apps/backend test -race -trimpath -p 4 ./internal/reliable -run '^TestConfiguredModelSurvivesAnOutdatedNativeCatalog$' -count=1
go -C apps/backend test -race -trimpath -p 4 ./internal/backendapp -run '^TestReliableRoutesAreOptInAndRejectUngatedAcceptance$' -count=1
go -C apps/backend vet ./internal/reliable
node --test apps/backend/internal/reliable/page-state.test.cjs
node scripts/reliable/desktop-browser-check.cjs
node tools/domain-browser-check.cjs
node tools/browser-check.cjs
node tools/local-project-browser-check.cjs
```

The focused `TestDesktop*` selection supports iteration; the complete package race run passed before the final model-cache patch. That patch has targeted model/catalog race verification rather than a claimed repeated complete-package pass. The [plan's acceptance table](plan.md#acceptance-coverage) maps all 14 criteria to actual backend tests and nine named browser scenarios, including the configured-model regression under DESKTOP-001.5/.8. The canonical browser runner is `scripts/reliable/desktop-browser-check.cjs`, uses existing `apps/web` Playwright with Windows Edge and local fixtures, and does not invoke paid models. Its output defaults to `.tmp/reliable-desktop-browser` and can be overridden with `XIAOBUDING_VALIDATION_DIR`. The old `tools/desktop-browser-check.cjs` is only the historical local verification copy. Validate frontmatter, requirement/criterion references, relative links, UTF-8, and specification size. In the assembled full checkout, run `python scripts/list-docs.py validate` and `python scripts/lint-spec-files.py --all`, separating new owning-system findings from existing upstream findings.

## Likely files and dependencies

`apps/backend/internal/reliable/store.go`, task HTTP actions, existing evidence/seal helpers where compatibility must be checked, `page.html`, embedded styles/scripts and their state tests, plus focused archive/folder/browser tests. This work depends on the current task store, local mutation guard, folder opening, model-plan controls, and automatic local-project contract; it has no independent preceding work order.

## Risks

Keep archive outside sealed execution data and unfinished-reservation release. Test eligibility in the same atomic update as persistence. Avoid shell interpolation or arbitrary folder targets. Preserve keyboard focus, selection, full model values, and current localized control semantics during UI rerenders.

## Parallelism

`sequential`

## Inputs

- [Desktop workspace requirements](../../specs/reliable-collaboration/requirements/desktop-workspace.md).
- [Desktop workspace design](../../specs/reliable-collaboration/system-design/desktop-workspace.md).
- [Local recovery and reservation design](../../specs/reliable-collaboration/system-design/local-snapshots.md).
- Existing task state, seal, acceptance, model selection, and rendered checks.

## Results

Done. All six `TestDesktop*` tests passed under the race detector, and the complete reliable package passed in 171.645 seconds. The local desktop browser suite passed 9/9 after the archived-resume, loading-placeholder, focus, and action-feedback fixes; state/domain/legacy/local-project regressions passed 4/4, 6/6, 5/5, and 5/5. Static checking and documentation checks passed. [The plan](plan.md#results) records exact evidence paths and the limits of fixture results.

The initial 1.4.0 deployment passed real candidate/production checks and preserved five tasks, including the actual evidence of one awaiting-acceptance candidate in a stopped-service database copy. Its backup is `E:\小布丁\backups\upgrade-1.4.0-20261019-170254`; the old sealed-candidate check also passed.

Production exposed a stale native cache silently replacing configured `gpt-6.1-sol` with the first listed model. The new `TestConfiguredModelSurvivesAnOutdatedNativeCatalog` reproduced that failure before the fix. It and five related model/catalog tests then passed in 10.627 seconds; the [plan](plan.md#results) identifies the RED/GREEN logs and the exact placeholder compatibility boundary. This is not a new account or live-directory capability.

The patched 1.4.0 production check passed with no page-script errors, original Kandev HTTP 200, desktop/mobile verification, and all five tasks' previous state/round/seal/archive values preserved. Its immediate rollback backup is `E:\小布丁\backups\model-preference-20261019-170806`. The final stopped-service database copy again passed validation of five tasks and the existing awaiting-acceptance candidate's actual evidence; `validation/desktop-final-legacy-seal.log` records a 2.832-second package pass. Windows PowerShell 5.1 repeated startup reused PID 28972; the existing shortcut retained its PowerShell target and uses the installed icon.

The displayed Codex count of seven includes six native directory entries and configured `gpt-6.1-sol` explicitly marked unverified; Harness has two directory entries. These counts are observations, not native availability guarantees. The canonical `scripts/reliable/desktop-browser-check.cjs` was separately copied into the complete source tree and actually passed 9/9, recorded in `validation/desktop-repository-browser.log`. Complete-source documentation validation passed for 351 decisions and 1352 specifications, with zero owning-system spec findings and 107 existing findings elsewhere. No source commit, full upstream test-suite pass, new native account capability, or remote publication is claimed.
