---
created: 2026-10-19
status: complete
requirements:
  - REQ-RELIABLE-SNAPSHOT-001
  - REQ-RELIABLE-BOOTSTRAP-001
system_design:
  - ../../specs/reliable-collaboration/system-design/local-snapshots.md
legacy_specs: []
---

# Implementation Plan: Automatic Local Project Preparation

## Overview

Accept ordinary folders without user Git commands, preserve admitted source in managed local recovery points, and establish real checks for empty projects. The mutation boundary, check bootstrap/evidence and Chinese UI projection are complete. Version 1.3.0 was deployed locally on 2026-10-19; the verification record below states its scope and upstream limitations.

## Scope

### In scope

- Canonical admission and atomic reservation before mutation.
- Managed initialization/ownership and local recovery commits with protected staging and exclusions.
- Persisted preparation, cancellation/recovery, and real verification bootstrap.
- External Git, legacy seals and shared domain/round compatibility.
- Localized preparation/failure states and retained drafts.

### Out of scope

Remote operations, global Git identity changes, external repository adoption, automatic commits of Agent results, unattended acceptance, broad tools-root imports, and unrelated UI redesign.

## Technical approach

`http.go` normalizes and validates creation; `setup_actions.go` reserves through `Store.Create`, including parent/child overlap. Optional `auto_snapshot`/`auto_checks` fields preserve legacy defaults. `Engine.prepare` owns `preparing`, managed recovery points and the existing worktree contract. `local_project.go`, `local_project_policy.go`, `local_project_snapshot.go` and `local_project_recovery.go` implement canonical ownership, explicit exclusions, literal NUL-delimited staging and recoverable initialization. All controller Git helpers discard inherited `GIT_*` variables.

`automatic_checks.go` discovers and freezes actual candidate checks after bootstrap implementation and binds the plan into applicable evidence seals. Empty planner reads are allowed only for the current empty source. Missing checks and detected zero-test output request bounded rework; explicit commands and independent review remain authoritative.

| Project shape | Preparation | Verification or refusal |
| --- | --- | --- |
| Ordinary folder | Create managed local history | Discover checks or bootstrap; refuse unsafe source before commit, with possible controller metadata retained |
| Empty folder | Empty source baseline | Establish source and checks in the shared budget |
| Managed root | Snapshot unstaged admitted changes | Block manual staging, marker mismatch or unfinished Git operations |
| External Git root | Existing clean-root policy | No adoption, automatic commit or remote changes |
| Parent/nested repo, link, internal worktree, tools root | Refuse automatic preparation | Localized reason; no Agent dispatch or destructive repair |

## ASCII UI preview

UI-LOCAL-01: Existing `/reliable` setup and submitted-task states.

```text
Project folder  [ Choose folder / paste path                 ]
Ordinary folders work. A local recovery point is created first.
No GitHub upload.
Goal            [ Describe the outcome                      ]
Assignments     [ Existing tool / model / effort controls    ]
[ Check and start collaboration ]

Preparing: Creating local recovery point... [ Stop task ]
New project: Establishing source and real verification...
Failure: <reason and next action>; submitted choices retained.
```

Phones stack the same existing controls. Localized wording, retained drafts, preparation visibility and no false pass are required by AC-RELIABLE-SNAPSHOT-001.9 and AC-RELIABLE-BOOTSTRAP-001.1/.4. Spacing is illustrative.

## Tests

| Criteria | Implemented evidence |
| --- | --- |
| SNAPSHOT-001.1/.3 | `local_folder_http_test.go`: `TestPlainFolderIsStoredLocallyAndReusedAfterAcceptance` exercises preparation, acceptance and the next unstaged checkpoint |
| SNAPSHOT-001.2 | `TestUnfinishedTaskIsRejectedBeforeAnotherLocalCheckpoint` and `TestOverlappingLocalProjectsReserveOnePreparationSlot` reject duplicate and overlapping reservations |
| SNAPSHOT-001.4/.5 | `local_project_safety_test.go`: `TestLocalSnapshotsPreserveExternalAndManualStaging`, `TestLocalSnapshotRejectsPrivateIgnoreOverrideAndNestedGit`; `local_folder_http_test.go`: `TestLocalStoragePreservesButDoesNotTrackSecretsOrDependencies` |
| SNAPSHOT-001.6/.8 | `TestLocalSnapshotDoesNotRebaseWorkingCopiesOrNestedSelections`, the nested-root case above and existing source-limit tests; `TestGitEnvironmentCannotRedirectLocalStorageOrCandidateIndex` protects Git routing |
| SNAPSHOT-001.7 | `TestPreparingRecoveryAndCancellationKeepZeroImplementationRounds`, `TestInterruptedOwnedInitializationResumesWithoutAdoptingExternalGit` |
| BOOTSTRAP-001.1/.2/.3 | `bootstrap_http_test.go`: `TestEmptyFolderCreatesRealChecksWithoutManualConfiguration`, `TestEmptyAuditExceptionEndsAsSoonAsSourceExists` |
| BOOTSTRAP-001.4 | `TestMissingAutomaticChecksUsesThreeRoundsAndCannotBeAccepted`, `TestAutomaticChecksDoNotTreatZeroTestRunnersAsVerification`, `TestAutomaticGoChecksAllowPackagesWithoutTestsAlongsideRealTests`, `TestZeroTestExecutionRemainsUnverifiedWithoutChangingActualExit`; existing `TestCheckDiscoveryRejectsPlaceholderTests` |
| BOOTSTRAP-001.5/.6/.7 | Explicit-command local-folder flow, `TestAutomaticCheckPlanTamperingBlocksAcceptance` and existing acceptance/legacy/domain tests |
| SNAPSHOT-001.9 | `tools/local-project-browser-check.cjs`: localized failure, retained input, preparation and truthful verification setup |

## E2E tests

`tools/local-project-browser-check.cjs` passed five Windows Edge fixture scenarios: ordinary-folder submission without manual commands, localized failure with native-model choices retained, automatic-verification/recovery-point visibility, mobile optional-command layout, and an exit-zero/no-tests result that stays unverified. These cover AC-RELIABLE-SNAPSHOT-001.9 and AC-RELIABLE-BOOTSTRAP-001.1/.4. Existing domain, legacy and JavaScript state runners passed six, five and four scenarios respectively. Browser fixtures do not prove Git behavior; backend temporary-project tests prove source/index/remote invariants. These fixture gates require no paid model calls; the separate native proof is recorded below.

## Work orders

- [x] [Task 01: Admit and prepare local projects with real verification](task-01-local-project.md)

## Verification results

Completed for the reliable extension and local deployment. The complete reliable-package race run passed in 137.415 seconds. After the zero-test reporting fix, eight targeted race items passed in 27.092 seconds and package static checking reported zero issues. The scoped backend route check passed in 9.567 seconds. Local-folder browser checks passed 5/5, domain browser checks 6/6, legacy browser checks 5/5, and state checks 4/4. The temporary-folder integration also executes actual npm/Node assertions with fixture Agent responses and exercises review/application.

A separate actual native proof task, `q5zt2jbj4iket2u7rdiwbgkt4q`, started from an empty fullstack folder, completed in round one with four independent native sessions, automatically discovered and froze `npm run test`, and reached explicit acceptance in the disposable sample. Five assertions passed after application; no Git remote was configured.

Version 1.3.0 was rebuilt and deployed to `E:\小布丁`. Rollback backup: `E:\小布丁\backups\upgrade-1.3.0-20261019-155143`. Final executable SHA-256: `3c388b4bf5228aef123d4cba217a7782a58a65b4a735f0a5c96d41f47b875bf1`. Deployment preserved the state, round and evidence hash of all five existing tasks. After stopping the service and copying its closed database, `TestExistingLegacySealedCandidateRemainsValid` revalidated one existing awaiting-acceptance candidate in 0.65 seconds without applying its patch.

The deployed-page runtime smoke check reported no page-script errors and the original Kandev route returned HTTP 200. Its live catalogs contained six Codex options and two Harness options; the earlier candidate startup observed seven and two. These are separate observations of a changing native catalog, not fixed model-count guarantees.

Five failures in the broader `backendapp` suite reproduced on the unmodified `d13a69d0` base in 11.775 seconds. This record does not claim that the full Kandev test suite passed. Scoped documentation checks passed with zero new specification/encoding/link/criteria findings; the complete-checkout catalog, lint and diff result is owned by the final source-integration gate before the local commit. No remote publication is part of this deployment.

## Risks

- Interrupted initialization can resemble external Git without an ownership check.
- A crash after staging but before the pending-index record blocks safely and requires maintainer recovery; preparation does not automatically reset that index.
- Failed new-folder preparation can leave local `.git` or exclude metadata, while source contents and pre-existing user history/index remain protected.
- The explicit path list does not comprehensively detect secrets inside arbitrary source files.
- Windows canonical paths, literal staging and ignore conflicts can include unrelated source if bypassed.
- Successful generated checks may lack feature assertions; independent review remains required.
- Optional fields and artifacts must preserve old evidence serialization.

Final assembly gates: the deployed full checkout validated 351 decisions and 1350 specifications. Owning reliable-collaboration spec findings remain zero; the unchanged upstream full-spec baseline has 107 size findings. Staged whitespace and changed-Go formatting checks passed. Browser harnesses and deployment tools are retained in the local assembly workspace; backend implementation and behavior tests are versioned in this source checkout. The deployed HTTP smoke test checks the actual executable independently of that assembly copy.
