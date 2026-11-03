---
created: 2026-10-19
status: complete
requirements:
  - REQ-RELIABLE-DESKTOP-001
  - REQ-RELIABLE-ARCHIVE-001
system_design:
  - ../../specs/reliable-collaboration/system-design/desktop-workspace.md
legacy_specs: []
---

# Implementation Plan: Desktop Task Workspace

## Overview

Deliver a compact local task workspace with result-first task detail, expandable model settings, task-bound folder actions, and reversible archiving. Preserve execution and evidence behavior while improving navigation.

## Scope and sequence

Implement one vertical work order. Establish archive persistence and server-side eligibility, prove unchanged reservations and sealed candidates, add task-bound folder targets, then connect the sidebar, shared menus, result layout, and appearance preferences. Finish with focused backend, state, and rendered checks.

Local delivery includes preparing the updated package, deploying it to the existing `E:\小布丁` installation with a rollback backup, preserving account configuration and existing task data, and checking the existing desktop entry and deployed page. These steps are complete to the verification boundaries recorded below.

Production verification also owns a discovered compatibility fix: a stale native model cache must not silently replace the already configured model with the first listed entry. Preserve the configured ID as an explicitly unverified option when absent from the cache, retain real listed alternatives, and use native metadata again when that ID appears in a later directory. This does not add an account connection, perform a model request, or turn a local cache into a live availability guarantee.

Excluded: native connection changes, default model or assignment changes, chat synthesis, deletion, automatic cancellation, changed evidence rules, unrelated public documentation, and remote publication.

## ASCII UI preview

UI-DESKTOP-01: `/reliable`, current task selected; sidebar remains available while the main result area scrolls.

```text
+-----------------------+------------------------------------------+
| Xiaobuding      [+ New]| Task outcome                      [ ... ]|
| [ Search tasks      ] | State / round / next permitted action    |
| [Current] [Archived]  |                                          |
| Selected task   [ ...]| Verification / review / result           |
| Other task      [ ...]| [Inspect evidence] [Accept when eligible]|
|                       |                                          |
| Theme [System v]      | > Tool and model arrangements            |
+-----------------------+------------------------------------------+

Task menu (overflow, right-click, or keyboard):
  Open original project / working copy / evidence folder
  Archive or restore
  Active execution: archive unavailable, with reason
  Awaiting acceptance: archive allowed; still needs acceptance
```

UI-DESKTOP-02: New task; request-first composition, with complete settings on expansion.

```text
Project       [ Choose folder / path                    ]
Outcome       [ Describe what should be completed       ]
Work summary  [ Frontend / backend / reviewer summary    ]
              > All tool, model, and effort settings
              > Optional verification and scope
[ Start collaboration ]
```

UI-DESKTOP-03: Phone; navigation is a dismissible drawer, not a compressed fixed sidebar.

```text
[Tasks]  Xiaobuding                         [+ New]
Task outcome                               [ ... ]
State / round / next action
Verification / review / result
[Inspect evidence] [Eligible primary action]
> Tool and model arrangements

Tasks drawer: search, Current / Archived, rows, theme
```

Navigation, ordering, shared menu behavior, preserved full settings, and pending-acceptance visibility are required by DESKTOP-001.1-.7 and ARCHIVE-001.3/.6. Spacing and illustrative English labels are not pixel or localization specifications. Product copy remains localized. Empty search results, missing folders, failed archive requests, disabled active-task archive, and menu/drawer focus behavior require rendered checks.

## Acceptance coverage

Archive/folder backend tests below are in `apps/backend/internal/reliable/desktop_actions_test.go`; the follow-up configured-model regression is in `catalog_preference_test.go` in the same package. Browser scenario labels refer to the canonical runner `scripts/reliable/desktop-browser-check.cjs`. The earlier `tools/desktop-browser-check.cjs` is a local verification copy; historical logs retain its tested nine-scenario results.

| Browser label | Exact scenario name |
| --- | --- |
| B1 | `desktop starts with focused requirement and collapsible complete native assignments` |
| B2 | `sidebar searches source paths and separates archived records` |
| B3 | `right click on inactive row opens its folder without navigating active task` |
| B4 | `archive is reversible metadata and waiting acceptance remains explicit` |
| B5 | `a rejected archive keeps the record in place and shows actionable feedback` |
| B6 | `keyboard menu returns focus and cannot archive a running task` |
| B7 | `task result, disclosure state and future plan survive polling` |
| B8 | `appearance setting persists and dark controls remain readable` |
| B9 | `mobile drawer offers the same task menu and stays within viewport` |

| Acceptance criterion | Actual test mapping and verification boundary |
| --- | --- |
| AC-RELIABLE-DESKTOP-001.1 | B2: source-path search and current/archive separation |
| AC-RELIABLE-DESKTOP-001.2 | B3/B4/B6/B9: context menu, overflow, keyboard, focus return, disabled archive, and phone placement; final browser rerun passed |
| AC-RELIABLE-DESKTOP-001.3 | `TestDesktopFolderActionsReportUnavailableDirectories`, `TestDesktopFolderActionsUseOnlyTaskOwnedTargets`, and B3: all server targets, absent paths, rejected arbitrary/untrusted requests, and row-specific opening |
| AC-RELIABLE-DESKTOP-001.4 | B7: actual result page, collapsed settings and polling-stable disclosures; real candidate and final patched-production desktop/mobile checks passed |
| AC-RELIABLE-DESKTOP-001.5 | B1/B7 plus the passing domain-browser and page-state regressions: requirement-first entry, full settings expansion, defaults, preserved model/effort choices, and retained disclosure state. `TestConfiguredModelSurvivesAnOutdatedNativeCatalog` additionally proves the configured model is not replaced by a stale catalog's first entry |
| AC-RELIABLE-DESKTOP-001.6 | B8: dark-theme reload persistence, viewport layout, and live `system` mode changes under emulated operating-system light/dark appearance |
| AC-RELIABLE-DESKTOP-001.7 | B6/B9: keyboard menu, mobile drawer, bounded menu/page geometry, and focus returned to the navigation toggle after mobile task selection |
| AC-RELIABLE-DESKTOP-001.8 | B7 and final real candidate/production inspection; `TestConfiguredModelSurvivesAnOutdatedNativeCatalog` proves an unlisted configured option is labeled unverified, has no invented reasoning-effort metadata, and regains native metadata when listed. Production exposed six listed Codex options plus the configured unverified `gpt-6.1-sol`, without claiming that placeholder proves native availability |
| AC-RELIABLE-ARCHIVE-001.1 | `TestDesktopArchivePersistsAndKeepsTheSealedCandidate`, `TestDesktopArchiveEligibilityIdempotenceAndResumeGuard`, and B4: persisted timestamp, database reopen, repeat archive and restore |
| AC-RELIABLE-ARCHIVE-001.2 | `TestDesktopArchiveEligibilityIdempotenceAndResumeGuard`, `TestDesktopArchiveCannotRaceExecutionOrReleaseAnUnfinishedProject`, and B6: stage eligibility, reserved-run guard, and disabled menu |
| AC-RELIABLE-ARCHIVE-001.3 | `TestDesktopArchivePersistsAndKeepsTheSealedCandidate`, `TestDesktopArchiveCannotRaceExecutionOrReleaseAnUnfinishedProject`, and B4: awaiting-acceptance preservation, unfinished-project reservation, and a direct archived pending-acceptance count assertion |
| AC-RELIABLE-ARCHIVE-001.4 | `TestDesktopArchivePersistsAndKeepsTheSealedCandidate`: unchanged state/round/evidence hash, database reopen, and successful acceptance after restore. Final production preserved all five real tasks' state/round/seal/archive values; `TestExistingLegacySealedCandidateRemainsValid` passed again on one awaiting-acceptance candidate in the final stopped-service database copy |
| AC-RELIABLE-ARCHIVE-001.5 | `TestDesktopArchiveCannotRaceExecutionOrReleaseAnUnfinishedProject`, `TestDesktopArchiveEligibilityIdempotenceAndResumeGuard`, `TestDesktopArchivePersistsAndKeepsTheSealedCandidate`, and B4: execution/edit/application guards, synchronous HTTP resume rejection, and independent restore; all named backend tests passed under the race detector |
| AC-RELIABLE-ARCHIVE-001.6 | B2/B4/B5: view separation, restore/reload, and a rejected archive retaining the current row while clearing an earlier success message and displaying the actual error through the shared action-feedback path |

The nine-scenario browser rerun, real candidate/final production checks, and final existing-seal check passed. The stale-cache model-selection fix has targeted passing tests and was verified in production. Fixture rendering and a configured-model placeholder still do not establish native-model execution or availability.

## Work orders

- [x] [Task 01: Deliver the desktop task workspace](task-01-desktop-workspace.md)

## Risks and verification

Archive metadata can invalidate historical seals if accidentally added to the sealed contract. A view filter can accidentally release an unfinished reservation. Eligibility checks outside the store transaction can race with execution. Client-supplied folder paths can cross the intended local-action boundary. Menu and theme changes can lose focus or form values on rerender. Focused tests must prove these boundaries before broadening verification.

A native cache can temporarily omit a legitimately configured model. Preserve the configured ID rather than silently selecting another model, but show that the directory did not list it. Do not invent supported reasoning efforts or describe the retained placeholder as a verified native directory entry.

Use the exact commands in the work order. Run the existing reliable-package checks as justified by changed backend behavior and bounded rendered scenarios for desktop and phone. Complete-checkout docmeta/spec validation belongs to integration; this partial assembly can run the same validators against its owning specs using the upstream read-only tooling.

## Results

Complete. Version 1.4.0, including the configured-model compatibility patch, is deployed locally and passed the recorded production and existing-evidence checks.

- The complete `internal/reliable` race run passed with a final package event elapsed time of 171.645 seconds in `validation/desktop-final-race.jsonl`. All six `TestDesktop*` tests passed in that run; the focused race evidence is also in `validation/desktop-boundary-green.log`.
- The local verification copy of the desktop browser runner passed 9/9 scenarios in `validation/desktop-browser-final.log`, including synchronous-action feedback after an earlier success, pending-acceptance archive count, operating-system appearance following, loading-placeholder removal, and mobile focus restoration. The action-feedback regression was first reproduced failing, then fixed and rerun; the earlier failure is retained in `validation/desktop-feedback-red.log`.
- Page-state tests passed 4/4 in `validation/desktop-state.log`; domain, legacy, and local-project browser regressions passed 6/6, 5/5, and 5/5 in `validation/desktop-domain-browser.log`, `validation/desktop-legacy-browser.log`, and `validation/desktop-local-browser.log`.
- Static verification passed: the implementation owner reported `go vet`, and `validation/desktop-lint.log` records zero lint issues. Documentation metadata, owning-system specification lint, criterion/test-name references, links, UTF-8, and lifecycle statuses passed the documentation check.
- The implementation owner reported a successful initial 1.4.0 deployment with rollback backup `E:\小布丁\backups\upgrade-1.4.0-20261019-170254`. The real candidate and deployed checks passed. All five existing tasks were retained; a stopped-service database copy retained those five records and one awaiting-acceptance candidate's actual evidence. `validation/desktop-legacy-seal.log` records `TestExistingLegacySealedCandidateRemainsValid` passing, with 2.863 seconds for that package run.
- Production then exposed an older `setModelCatalog` fallback: a native cache exposing six model options omitted configured `gpt-6.1-sol`, so the UI silently selected its first model. `TestConfiguredModelSurvivesAnOutdatedNativeCatalog` first failed with that replacement in `validation/desktop-model-preference-red.log`. The fix preserves the configured ID as an explicitly unverified placeholder without effort metadata and replaces the placeholder with native metadata when later listed. The new regression and five catalog/model-routing regressions passed in `validation/desktop-model-preference-green.log` in 10.627 seconds. These targeted results follow the earlier 171.645-second package run and do not claim that the complete package was rerun after this final patch.
- The final compatibility patch is deployed; its immediate rollback backup is `E:\小布丁\backups\model-preference-20261019-170806`. `validation/desktop-production-runtime.log` records actual version 1.4.0, five preserved tasks, no page-script errors, and HTTP 200 from the original Kandev SPA. The delivery owner verified unchanged state, round, seal, and archive values for all five tasks and passed actual desktop and phone checks. Screenshots include `validation/desktop-production-desktop.png`, `validation/desktop-production-mobile.png`, `validation/desktop-production-menu.png`, and `validation/desktop-production-pending.png`.
- The final production directory presentation contained seven Codex choices: six native-directory entries plus configured `gpt-6.1-sol` explicitly marked unverified, and two Harness directory entries. The placeholder is not a new or verified native directory entry and does not create reasoning-effort support.
- A repeated launch through Windows PowerShell 5.1 reused PID 28972. The existing desktop shortcut retained its PowerShell target and uses `E:\小布丁\assets\小布丁.ico`. The delivery owner rechecked five records and one awaiting-acceptance candidate's actual evidence from the final database copy; `validation/desktop-final-legacy-seal.log` records `TestExistingLegacySealedCandidateRemainsValid` passing in a 2.832-second package run without applying its changes.

The canonical reproducible runner is `scripts/reliable/desktop-browser-check.cjs`. It resolves the repository root and existing `apps/web` Playwright dependency relatively, writes to `.tmp/reliable-desktop-browser` by default, and accepts `XIAOBUDING_VALIDATION_DIR` as an output override. The delivery owner copied it into the complete source tree and actually ran all nine scenarios successfully; `validation/desktop-repository-browser.log` records that separate 9/9 result. The local historical evidence remains under `validation/desktop-browser/`. These fixture checks do not start native model requests.

Final complete-source documentation validation passed for 351 decisions and 1352 specifications. Owning reliable-collaboration specification findings are zero; 107 existing findings outside this owning system remain recorded separately.

No new native account or live-directory capability is claimed. No full upstream Kandev test-suite pass, source commit, or remote publication is asserted by this delivery record.
