---
id: "01-desktop-personalization"
title: "Deliver native access defaults and local appearance"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-RELIABLE-ACCESS-001
  - REQ-RELIABLE-PERSONALIZATION-001
acceptance_criteria:
  - AC-RELIABLE-ACCESS-001.1
  - AC-RELIABLE-ACCESS-001.2
  - AC-RELIABLE-ACCESS-001.3
  - AC-RELIABLE-ACCESS-001.4
  - AC-RELIABLE-PERSONALIZATION-001.1
  - AC-RELIABLE-PERSONALIZATION-001.2
  - AC-RELIABLE-PERSONALIZATION-001.3
  - AC-RELIABLE-PERSONALIZATION-001.4
  - AC-RELIABLE-PERSONALIZATION-001.5
system_design:
  - ../../specs/reliable-collaboration/system-design/desktop-personalization.md
---

# Task 01: Deliver Native Access Defaults and Local Appearance

## Summary and scope

Deliver version 1.5.0 with native full-access defaults only for this application's Codex/Harness invocations while preserving source-verified read-only investigation/review. Reuse the original environments' installed skills/plugins without creating or installing another collection. Enlarge the existing avatar and reuse the supplied original portrait as the default optional background at 80% transparency; saved `none` remains off.

Exclude global tool/account changes, new model defaults, image generation/editing, task/evidence migration, unrelated UI work, and remote publication.

## Implementation acceptance

- Both native runners receive supported full-access/noninteractive settings without rewriting native configuration; investigation/review source changes remain rejected, and existing native skills/plugin inheritance is verified without another installation.
- Original image content is reused locally; none/portrait and 0-100% background transparency work with tolerant local persistence and an enlarged undistorted avatar.
- Desktop/phone themes, keyboard controls, task/model defaults, independent review, seals, and round/application rules retain their existing behavior.

## ASCII UI preview

UI-PERSONAL-01/02 follow [the combined preview](plan.md#ascii-ui-preview).

```text
Larger avatar
Theme [System v]  Background [Portrait v] (None available)
Transparency [----o-----] 80%   (background only)
Phone: same controls inside the existing tasks drawer
```

The preview maps to PERSONALIZATION-001.1-.5. Inspect endpoints, malformed stored settings, theme changes, and narrow layouts instead of treating a sketch as rendered proof.

## Verification

Run with the existing toolchain from the repository root and isolate test state where needed. The [plan's coverage table](plan.md#acceptance-coverage) maps all nine criteria to actual test names and the 13 canonical browser scenarios. Commands define verification gates; only the results explicitly recorded below are claimed complete.

```powershell
go -C apps/backend test -race -trimpath -p 4 ./internal/reliable -count=1
go -C apps/backend test -race -trimpath -p 4 ./internal/reliable -run '^(TestCodexDefaultsToFullAccessForEveryRole|TestHarnessUsesNativeHomeFullAccessAndNoCodexKeyFallback|TestHarnessReusesInstalledDesktopExtensionsAndKeepsPreferencesPrivate|TestHarnessInheritanceFailureStopsBeforeCallingTheModel)$' -count=1
go -C apps/backend test -race -trimpath -p 4 ./internal/reliable -run '^TestExistingNativeHarnessExtensionsResolveWithoutModelCalls$' -count=1
go -C apps/backend vet ./internal/reliable
node --test apps/backend/internal/reliable/page-state.test.cjs
node scripts/reliable/desktop-browser-check.cjs
python scripts/list-docs.py validate
python scripts/lint-spec-files.py --all
```

The installed-environment test requires `XIAOBUDING_NATIVE_PROFILE_HOME`, `XIAOBUDING_NATIVE_ENTRY`, and `XIAOBUDING_NATIVE_NODE` to point to the existing Harness installation. Without these, it skips; a skipped run is not inheritance proof. It resolves metadata and checks original profile hashes without calling a model. The optional `XIAOBUDING_NATIVE_INVENTORY` output is module inventory only. Fixture checks cover private-overlay cleanup and failure before model startup. Codex inheritance remains its original `CODEX_HOME` without disabling plugins; do not infer every plugin's activation from a matching path or module resolution.

Verify the original asset and actual candidate rendering separately. The canonical browser runner now has five personalization scenarios in addition to its original nine. It needs no paid model or image-generation request. Local runtime delivery verification belongs to the implementation owner.

## Likely files and dependencies

`apps/backend/internal/reliable/process.go`, `harness.go`, `harness_inheritance.go`, `harness-native.mjs`, existing source-stage verification, `page-appearance.js`, embedded page/styles, local image serving/assets, focused tests, and `scripts/reliable/desktop-browser-check.cjs`. Reuse the native connections/profile facilities, existing avatar and supplied portrait, drawer/theme controls, source manifests, and acceptance evidence.

## Risks

Full native access and read-only responsibility are distinct; source comparison must fail on observed writes rather than claim filesystem isolation. Keep per-call changes out of native homes. Reject arbitrary persisted asset/style values and preserve foreground readability independently of background transparency.

## Parallelism

`sequential`

## Inputs

- [Requirements](../../specs/reliable-collaboration/requirements/desktop-personalization.md).
- [System design](../../specs/reliable-collaboration/system-design/desktop-personalization.md).
- [Existing desktop workspace](../../specs/reliable-collaboration/system-design/desktop-workspace.md).

## Results

Done. Version 1.5.0 is deployed locally. All 13 canonical browser scenarios and both native permission fixture contracts passed. The installed Harness metadata test passed under the race detector with 12 existing extension bundle references and unchanged original desktop manifest/patch hashes. The recorded `native-installed-check.log` run took 6.489 seconds including failure-path coverage; no model call or extension installation was used.

The later core run passed 121 tests with five opt-in skips in 150.447 seconds; installed native metadata was verified separately. Follow-up named-preference/native composition checks passed in 5.780 seconds, preserving the original `edu-mode` disabled preference and configuration hashes. The backend route race check passed in 6.997 seconds. Final vet/lint and independent code/candidate visual review passed. Public/local documentation reflects the completed deployment and the same support boundaries.

Actual candidate/production records under `E:\小布丁\data\validation\upgrade-1.5.0` show version 1.5.0, zero page-script errors, original Kandev HTTP 200, seven Codex/two Harness displayed choices, and successful light/dark/phone and task-menu checks. The supplied original image and served bytes share SHA-256 `9a369770514ab928193def95b4539ce3ed69f025c3dd5cf043dce7d74d3a755b`. The 0/100 endpoints were tested in the 13-scenario suite; actual candidate/production screenshots provide separate rendering evidence.

All five existing tasks and the connection-settings SHA were preserved. The final stopped-service-copy seal check passed in 3.123 seconds without applying the pending candidate. Backup: `E:\小布丁\backups\upgrade-1.5.0-20261019-182353`. The initial Windows PowerShell 5.1 script-policy failure was rolled back, then PowerShell 7 normal `-File` deployment succeeded without modifying execution policy, original shortcut, or database. Four page-state tests passed in approximately 83 milliseconds; repeated PowerShell 7 startup reused PID 31832 and delivery/shortcut fingerprints passed their final checks. The production window was opened and the separate candidate stopped. A production-check expectation for an archived pending task was corrected to Restore/no acceptance button and rerun successfully. [The plan](plan.md#results) records these boundaries; this release does not claim a Windows PowerShell 5.1 cold-start pass.

Native reference inheritance does not establish activation of UI-only plugins, which may still require original host services. Static inspection of the installed native source indicates that nonrequired pending UI entries do not block headless submission; no extra runtime/model check is claimed for that finding. No image generation, paid model call, full upstream suite pass, source commit, or remote publication is claimed.

### 1.5.1 follow-up

Correct portrait occlusion through desktop column layout and non-repeating contained imagery. Extend the canonical browser suite with a fourteenth subject-visibility scenario; keep narrow-screen and no-background behavior intact. This change does not modify native agents or task execution.

Delivery verified: 1.5.1 candidate and production pages passed with zero script errors and original Kandev HTTP 200. All 14 browser scenarios and four state tests passed. The stopped-database check verified five preserved tasks and one pending seal without applying it (2.930 seconds). Backup: E:\小布丁\backups\upgrade-1.5.1-20261019-185057. Documentation metadata validated 351 decisions and 1354 specifications; owning specification findings remain zero, with 107 unchanged upstream findings. No new native model call or full core regression rerun was needed for this CSS/version-only runtime change.
