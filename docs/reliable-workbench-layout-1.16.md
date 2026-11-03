# Workbench layout revision

Date: 2026-10-26. Scope: the Chinese local `/reliable` workspace.

## Observed references

- [Vibe Kanban](https://github.com/BloopAI/vibe-kanban), 28,462 stars at lookup.
  Its official workspace screenshot separates workspace navigation, agent
  conversation, code review and contextual actions. The repository README
  announces sunsetting; this revision uses visual research only.
- [Ekko Studio](https://github.com/EKKOLearnAI/ekko-studio), 11,566 stars at lookup.
  Official new-chat and conversation screenshots show quiet navigation, a
  persistent composer and readable conversation surfaces. The README identifies
  its captures as demo data; no screenshot is treated as execution evidence.
- AutoGen Studio documentation was read inside the 61,572-star
  [AutoGen repository](https://github.com/microsoft/autogen/tree/main/python/packages/autogen-studio).
  That star count belongs to the framework repository. Studio's prototype scope
  is documented upstream; it is not a replacement dependency for this app.

Reference URLs and rendered observations are retained in
`validation/reference-layouts.json`. No reference application was installed.
The installed UI skill supplied general dashboard guidance; its marketing-page
patterns did not match this workspace and were not used. Layout decisions use
the observed workspace references and the existing local product contract.

## Layout decisions

Use a 232-pixel project sidebar, a central reading/working area, and a 284-pixel
batch overview. Move the existing summary host into the right column rather
than recreating its actions. Four tabs and the persistent composer retain
their original DOM hosts, handlers and scroll owner.

The overview can collapse. At widths up to 1100 pixels it starts collapsed and
opens inline above the content. Attention states remain visible and named on
the header button. The native Agent settings panel temporarily takes priority;
closing it restores the stored overview preference. Preferences for wide and
compact viewports are separate.

Keep the existing component stylesheet in an explicit `components` cascade
layer, with shell decisions in `workbench`. Existing hidden semantics, reduced
motion, component-specific forms and protected preview controls remain.
Preserve the avatar, custom local backgrounds, transparency and portrait mode.

## Preserved behavior

- Native Codex and DeepSeek Harness authentication, models, reasoning options,
  original skills and plugins; changes apply to future dispatches.
- Shared project state, continuous questions/requirements, attachments, draft
  persistence, original and candidate folder actions, and reversible archive.
- Source evidence, actual commands, independent Codex review, acceptance gates,
  source and process ownership checks, and three quality rounds per batch.
- Explicit subsequent batches, project backlog/dependencies, source-labelled
  handoffs, changed files, supported previews, and storage budget controls.

The only Go source change is the extension version string in `http.go`.
Other reliable-package Go files are compared with retained SHA-256 values.
No new runtime, model configuration, package installation or business task is
required for this layout revision.

## Validation and limits

Forty isolated browser checks cover the retained 27 interaction cases and 13
layout scenarios, including 1101-pixel Agent-panel composition. Fifty-four
existing page logic tests pass. Thirteen selected older feature scenarios are
covered; one attachment screenshot step detached during polling in the first
run and passed when rerun separately. The original failure log is retained.
Sixteen bounded HTTP/workbench regression tests pass in 23.488 seconds.

Independent review found compact-state discoverability and narrow-desktop
attachment/help overlap issues. Both were corrected and checked again.
Final candidate/runtime, unchanged task history and desktop launch are recorded
in separate release receipts. Layout changes do not widen preview support or
guarantee correct model answers or faster model execution.
