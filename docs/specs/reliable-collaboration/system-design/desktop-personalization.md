---
status: current
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-ACCESS-001
  - REQ-RELIABLE-PERSONALIZATION-001
---

# Desktop Personalization Design

## Boundary and mapping

`apps/backend/internal/reliable` owns native dispatch construction and the embedded workspace. Reuse `CodexRunner`, `HarnessRunner`, the current task/source evidence pipeline, and existing appearance controls. Do not change native home configuration or ordinary upstream Kandev tasks.

| Requirement | Design sections |
| --- | --- |
| REQ-RELIABLE-ACCESS-001 | Invocation defaults and role verification |
| REQ-RELIABLE-PERSONALIZATION-001 | Local assets and rendering; Appearance preference contract |

## Invocation defaults and role verification

`process.go` passes Codex `danger-full-access` with `approval_policy="never"` for application-owned child processes across all three roles. `harness.go` sets the matching sandbox/permission preset and noninteractive approval policy in its per-call role overlay, appended after inherited native settings. Keep model, account-home, credential isolation, output protocol, and fresh-session behavior unchanged.

Codex continues with its selected original `CODEX_HOME` and does not disable its native plugins. Harness uses `harness_inheritance.go` and the embedded `harness-native.mjs` to reuse the existing desktop extensions through native `loadProfileDirectory`, parser, and serializer facilities. Resolve installed extension modules to `file:` URLs, reuse original preferences, and avoid duplicating bundles already provided by the native headless host. Do not copy, install, or maintain another skills/plugins collection.

Keep inherited private configuration in a temporary overlay outside task evidence, delete it after success or failure, and append the application role-policy overlay last. Evidence retains a module inventory in `native-extensions.json`, without private configuration. An inheritance preparation error stops before starting the model process. The installed-environment metadata check observed references to 12 existing extension bundles with unchanged desktop manifest/patch hashes and no model call. This proves reference resolution and preservation, not activation of every plugin: UI-only extensions can wait for native host services such as `webServer`. Preserve native startup warnings rather than labeling all extensions active.

Separate native permission selection from `RoleRequest` responsibility. Investigation/review prompts still prohibit source changes, and `stages.go` compares admitted candidate source before and after those stages. A detected mutation fails the role and prevents its evidence from authorizing delivery. Original-source drift remains an acceptance blocker. Preserve implementation scope validation, checks, seals, rework budget, and acceptance gates. Full access does not impose an operating-system read-only boundary; comparison covers the controller's admitted source, not every possible external side effect.

## Local assets and rendering

Copy the supplied original portrait into the application's local asset delivery path. Serve it through the embedded/local page boundary using the same controlled route pattern as the existing avatar; do not fetch a remote image at runtime. Preserve the original asset bytes and image contents. Increase the existing avatar's rendered size through layout styles without replacing its asset.

Use a decorative background layer independent of foreground opacity. Keep it behind the workspace with no pointer interaction and no contribution to document geometry. Maintain readable theme surfaces, controls, menus, drawers, and focus indicators in light/dark/system modes and at phone widths. Positioning and cropping may adapt to the viewport without editing the source image.

## Appearance preference contract

Reuse the existing browser-local appearance initialization through `page-appearance.js`. Store only the allowlisted mode (`none`, `portrait` or `custom`) and a finite numeric transparency percentage under `xiaobuding.background`. Default an absent/invalid mode to `portrait`; when no valid percentage is saved, use 80% transparency as the soft default. Preserve the saved percentage while background mode is `none` so re-enabling portrait restores it. Disabling the background disables its slider without changing theme or task input.

Normalize the percentage to the inclusive 0-100 range, reject empty/non-numeric/non-finite input, and derive background opacity as `1 - transparency / 100`. Do not interpolate unvalidated persisted text into CSS. Keep slider, displayed percentage, and background style synchronized on initialization and change. Catch browser-storage read/write failures and continue with usable in-memory settings. These preferences never enter server task records, model settings, or evidence seals.

## Related contracts

- [Desktop workspace design](desktop-workspace.md) owns theme choice, navigation, and task organization.
- [Local collaboration loop](local-loop.md) owns source evidence, independent review, bounded rework, and explicit acceptance.

## Local image selection

Version 1.5.2 adds a native file picker inside the existing background settings. The mode allowlist also accepts `custom`. Validate PNG/JPEG/WebP MIME type, a 20 MiB bound and browser image decoding before activation. Keep the image Blob in the origin's IndexedDB rather than putting its base64 content into quota-limited localStorage. The existing preference key retains only mode/transparency. Temporary browser object URLs supply CSS and are revoked after replacement or unload; arbitrary stored URL strings never become CSS.

Restore the Blob asynchronously without overriding a newer user selection. A selection version guards older decodes, while a mode version lets a later default/none choice win. A failed Blob write allows temporary rendering but persists the default mode instead of a broken custom reference. Missing or corrupt restored images fall back with a visible status. Successful replacement retains only one stored custom image. There is no upload endpoint or native-tool configuration change. Custom mode shares the existing subject-preserving desktop layout and transparency controls.

## Desktop subject visibility

Version 1.5.1 reserves the right 42% of the workspace at viewport widths of at least 1181px. Forms, notices and task results align left with a 720px maximum width. The portrait uses contain/no-repeat so the face stays visible without tiling or source-image edits. Narrow windows and no-background mode retain the existing full-width or centered layout. Theme, stored transparency and task state are unaffected.

The page image policy permits self-origin and browser Blob image URLs. Other source policies remain unchanged. The browser fixture projects the actual served CSP header so local image decoding is verified under the production policy.
