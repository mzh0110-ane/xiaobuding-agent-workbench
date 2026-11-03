---
created: 2026-10-19
status: complete
requirements:
  - REQ-RELIABLE-ACCESS-001
  - REQ-RELIABLE-PERSONALIZATION-001
system_design:
  - ../../specs/reliable-collaboration/system-design/desktop-personalization.md
legacy_specs: []
---

# Implementation Plan: Desktop Personalization

## Scope and sequence

Deliver version 1.5.0 with application-scoped native full-access defaults and a small appearance update. First verify both native permission contracts and preserve read-only role source checks. Then reuse the original portrait locally, enlarge the existing avatar, and connect background selection, transparency, and local persistence. The first visit defaults to portrait at 80% transparency; saved `none` remains off. Finish with focused backend/state/browser evidence and actual local delivery verification.

Keep accounts, global tool settings, default assignments/models, task state, seals, independent review, and the three-round budget unchanged. Reuse the already installed skills/plugins in the selected native environments; do not create or install a separate Xiaobuding collection. Verify effective inheritance for both tools instead of assuming it from their home paths. Do not generate or edit images. Local delivery may use the existing rollback-backed deployment process; remote publication and unrelated UI work are excluded.

## ASCII UI preview

UI-PERSONAL-01: Existing `/reliable` sidebar appearance region and main workspace.

```text
[ Larger existing avatar ]     Task / new requirement
Appearance                     Readable foreground surfaces
Theme        [ System v ]      Existing result and model controls
Background   [ Portrait v ]
              None / Portrait
Transparency [----o-----] 80%
              0 visible -> 100 invisible
```

UI-PERSONAL-02: Phone uses the existing navigation drawer.

```text
[Tasks]  Larger avatar / title
Task or requirement content

Tasks drawer:
Theme [System v]  Background [Portrait v]
Transparency [----o-----] 80%
```

Control availability, percentage semantics, readable foreground, and phone access are required by PERSONALIZATION-001.1-.5. Spacing and illustrative English labels are not pixel or localization specifications. Include none/portrait, both slider endpoints, stored/corrupt values, storage failure, and light/dark/system rendered states in verification.

## Acceptance coverage

The canonical `scripts/reliable/desktop-browser-check.cjs` contained these 13 baseline scenarios in 1.5.0. B1-B9 retain the desktop-workspace regression names; B10-B13 cover this appearance increment.

| Label | Exact scenario name |
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
| B10 | `portrait background has adjustable transparency and slightly larger avatars` |
| B11 | `background can be disabled and restored without changing theme or task input` |
| B12 | `corrupted or blocked appearance storage keeps controls usable` |
| B13 | `mobile background controls remain reachable and trap focus only on visible controls` |
| B14 (1.5.1) | `portrait reserves desktop subject space beside forms and task results` |

| Acceptance criterion | Actual test mapping and verification boundary |
| --- | --- |
| AC-RELIABLE-ACCESS-001.1 | `TestCodexDefaultsToFullAccessForEveryRole`: Codex arguments for auditor/writer/reviewer; `TestHarnessUsesNativeHomeFullAccessAndNoCodexKeyFallback`: real Node fixture processes verify Harness role overlays, approval policy, and selected native parameters for all roles |
| AC-RELIABLE-ACCESS-001.2 | The two permission tests above and `TestExistingNativeHarnessExtensionsResolveWithoutModelCalls`: per-call settings and unchanged existing desktop manifest/patch hashes; no global permission or login rewrite |
| AC-RELIABLE-ACCESS-001.3 | `TestReviewerCannotChangeCandidateAndPass`, `TestSourceDriftInvalidatesHumanAcceptance`, `TestIndependentReviewRequestsReworkThenHumanAcceptance`, and `TestFailedVerificationStopsAfterThreeRounds` are included in the passing core run; investigation source comparison remains in `stages.go` |
| AC-RELIABLE-ACCESS-001.4 | `TestHarnessReusesInstalledDesktopExtensionsAndKeepsPreferencesPrivate`, `TestHarnessInheritanceFailureStopsBeforeCallingTheModel`, and `TestExistingNativeHarnessExtensionsResolveWithoutModelCalls`: original module references/preferences, named-override identity, private-overlay cleanup, safe failure, and installed-environment metadata resolution. Native composition also retained the installed `edu-mode` disabled preference; Codex preserves its original home and does not disable plugins. This does not assert every installed plugin is active |
| AC-RELIABLE-PERSONALIZATION-001.1 | B10/B13 and existing `TestDesktopAvatarIsServedAsAnImage`: larger avatar geometry, actual avatar route, and phone controls; actual candidate/production desktop and phone checks passed |
| AC-RELIABLE-PERSONALIZATION-001.2 | B10/B11/B12: portrait default, none/portrait switching, local background image route, and fallback. Candidate and production served the original image bytes with matching SHA-256 |
| AC-RELIABLE-PERSONALIZATION-001.3 | B10/B11: 80% default, keyboard 0/100 endpoints, derived background opacity, and an independently retained foreground/theme |
| AC-RELIABLE-PERSONALIZATION-001.4 | B10/B11/B12: numeric persistence, none persistence, malformed JSON, rejected mode, clamped 180-to-100 input, and blocked storage; this does not claim every malformed numeric representation was separately browser-tested |
| AC-RELIABLE-PERSONALIZATION-001.5 | B8/B10/B11/B13 plus B9: system appearance, light/dark backgrounds, keyboard 0/100 endpoints, visible-control focus trap, and phone bounds. Actual candidate/production light/dark and phone screenshots passed independent visual review; endpoints remain specifically covered by the 13-scenario suite |

Fixture tests do not prove that a full-access process enforces read-only behavior through operating-system permissions. Installed module resolution is also distinct from runtime activation and native model execution.

## Work orders

- [x] [Task 01: Deliver native access defaults and local appearance](task-01-desktop-personalization.md)

## Risks and verification

Role-level source guards must survive removing native sandbox restrictions. Permission changes must remain per invocation. Persisted appearance data must not become arbitrary CSS or asset URLs. Transparency must affect the background alone, and both endpoints must remain usable. Verify original asset identity without invoking image generation or modifying the source file.

Use the work order's commands with the existing toolchain. Record successful checks separately from remaining native/runtime/deployment limitations. Validate docmeta and owning-system specifications without relabeling unrelated upstream findings.

## Results

Complete. Version 1.5.0 is deployed at the existing local port 38429 and passed actual candidate/production checks. All 13 canonical browser scenarios passed, as did Codex full-access argument checks and Harness all-role process fixtures. The installed-environment `TestExistingNativeHarnessExtensionsResolveWithoutModelCalls` passed under the race detector: 12 existing extension bundles were referenced, the original desktop manifest/patch hashes were unchanged, and no model call occurred. `E:\小布丁\data\validation\upgrade-1.5.0\native-installed-check.log` records 6.489 seconds including inheritance failure-path verification.

The later core run passed 121 tests with five opt-in tests skipped in that run, in 150.447 seconds. The opt-in installed-environment metadata check was exercised separately rather than counted as a core pass. Follow-up named-preference identity normalization and native composition checks passed in 5.780 seconds, including retention of the installed `edu-mode` disabled preference and unchanged original configuration hashes. The backend route race check passed in 6.997 seconds, and the implementation owner reported independent review passing. No model call was used for these checks.

Harness inheritance uses native profile parsing/serialization and `file:` module references, a deleted private temporary overlay, and a configuration-free evidence inventory. Original native warnings remain visible; UI-only extensions may require `webServer` or other native host services. No claim is made that all plugins activated. Codex continues using its original home without plugin-disabling overrides.

The public task/workflow guide and local usage/check report describe the delivered 1.5 behavior and its native-extension boundary. Final evidence is retained under `E:\小布丁\data\validation\upgrade-1.5.0`:

- `candidate-runtime.json` and `production-runtime.json` record version 1.5.0, zero browser-script errors, original Kandev SPA HTTP 200, and seven Codex/two Harness displayed choices. Light/dark, desktop/phone, production context-menu, and archived pending-result screenshots passed. Counts do not establish successful execution of every model.
- The original image SHA-256 is `9a369770514ab928193def95b4539ce3ed69f025c3dd5cf043dce7d74d3a755b`; served candidate and production image bytes matched it. No image was generated or edited for this increment.
- All five existing tasks retained state, round, evidence hash, and archive values. Connection-settings SHA was unchanged. `legacy-seal.log` records `TestExistingLegacySealedCandidateRemainsValid` passing in a 3.123-second package run against the stopped-service copy, including the one awaiting-acceptance candidate; its changes were not applied.
- Rollback backup: `E:\小布丁\backups\upgrade-1.5.0-20261019-182353`. Final vet/lint and independent code/candidate visual review passed, with zero lint issues.
- `state-tests.log` records all four page-state tests passing in approximately 83 milliseconds. A repeated PowerShell 7 launch reused PID 31832 (`repeat-launch.json`); `delivery-check.json` records successful deployed-file and original-shortcut fingerprint checks. The production window was opened and the separate candidate process was stopped.

The first deployment launch used Windows PowerShell 5.1 without an execution-policy argument and was blocked by the local script policy. The delivery owner restored the previous executable/service, then completed deployment with the installed PowerShell 7 using normal `-File`. No execution-policy, original-shortcut, or database change was made to bypass that restriction. This record does not claim a successful Windows PowerShell 5.1 cold start for this release.

The first production check expected an acceptance button on an archived pending-result task. Its assertion was corrected to require Restore and no acceptance button; the rerun passed. This was a verification expectation correction, not an application fix.

Static inspection of the installed native source indicates that pending nonrequired UI extensions do not block headless submission; this is a source-level finding, not an additional runtime/model verification or a claim that every plugin activates.

No paid native model call, full upstream suite pass, source commit, or remote publication is claimed by this delivery record. Reference resolution still does not prove activation of every plugin or operating-system isolation of full-access read-only roles.

### 1.5.1 layout correction

The centered foreground covered the portrait at low transparency. Reserve a desktop subject region beside left-aligned forms and task results; contain the original image without repetition. The added browser scenario checks 1181px, 1440px and 2048px desktop bounds, task results, phone overflow and restoration of centered no-background mode. All existing collaboration contracts remain unchanged. Record final delivery evidence in the local 1.5.1 validation directory.

Delivery verified: 1.5.1 candidate and production pages passed with zero script errors and original Kandev HTTP 200. All 14 browser scenarios and four state tests passed. The stopped-database check verified five preserved tasks and one pending seal without applying it (2.930 seconds). Backup: E:\小布丁\backups\upgrade-1.5.1-20261019-185057. Documentation metadata validated 351 decisions and 1354 specifications; owning specification findings remain zero, with 107 unchanged upstream findings. No new native model call or full core regression rerun was needed for this CSS/version-only runtime change.
