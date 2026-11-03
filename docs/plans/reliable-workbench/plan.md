---
status: completed
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-WORKBENCH-001
  - REQ-RELIABLE-WORKBENCH-002
  - REQ-RELIABLE-WORKBENCH-003
  - REQ-RELIABLE-WORKBENCH-004
  - REQ-RELIABLE-WORKBENCH-005
system_design:
  - ../../specs/reliable-collaboration/system-design/workbench.md
---

# Long Project Workbench Implementation

The user approved the prior workbench plan for the existing local distribution.
The target update is 1.15.0. Implementation, independent review, partitioned
backend regression, and browser validation are complete. Candidate startup,
production preservation checks, deployment, and actual desktop shortcut launch
passed. The local release is complete; no user business task was resumed or
accepted during release verification.

## Work package

1. [Task 01: navigation and lightweight reads](task-01-navigation.md) supplies the
   project-centered shell, conditional summaries/details, paged conversation,
   actionable recovery presentation, and future-dispatch settings clarity.
2. [Task 02: project continuation and recovery](task-02-project-continuity.md)
   extends the existing ledger, handoff provenance, explicit dependency gates,
   planning recovery, notification preferences, and advisory space policy.
3. [Task 03: candidate results and preview](task-03-candidate-results.md) adds
   safe changed-file inspection and explicit owned-candidate preview, then owns
   final integration, release checks, and rollback-ready local installation.

These are vertical delivery/validation boundaries, not three additional quality
attempts. Same-candidate writers remain sequential. Native Harness supplied
frontend layout work; Codex owns controller integration and independent review.

## Dependencies and risks

The existing task store, evidence layout, accepted checkpoints, native account
homes, and desktop shortcut remain authoritative. New reads preserve legacy API
shapes. Project fields are optional additions in the existing ledger; budget
policy uses one additional bucket in the same database.

Critical risks are stale asynchronous responses, concurrent project edits,
source mutation during failed read-only planning, preview process ownership,
build/source confusion, and low-memory Windows validation. Keep builds/tests
serialized and use deployment-drive caches. Never test by automatically
accepting a user's business task or deleting retained project data.

## ASCII UI preview

UI-WORKBENCH-01: Desktop, existing task, navigation/current state.

```text
+----------------------+------------------------------------+----------------------+
| Projects             | Project / Batch / State           | Agent panel          |
| Project A            | Actual agent / latest activity    | Frontend             |
|   Current batch      | [Specific blocker + next action]  | Backend              |
|   History (collapsed)|------------------------------------| Planning             |
| Needs attention      | Conversation | Execution          | Review               |
| Awaiting acceptance  | Results      | Project plan       | Tool / model / effort|
|                      |                                    | Current vs next run  |
|                      | Scrollable content                 | [Collapse]           |
|----------------------|------------------------------------|                      |
| Theme / Background   | Ask / Request changes / Attach    |                      |
| Notifications        | [Fixed draft composer] [Send]     |                      |
+----------------------+------------------------------------+----------------------+
```

The header and composer stay fixed relative to the central scroll region. The
Agent panel is optional and collapsible. Project identity, separate actual/future
assignments, explicit blocker action, and retained draft are structural
requirements (AC-RELIABLE-WORKBENCH-001.1 through .4). Spacing and English sketch
copy are illustrative; product labels remain localized.

UI-WORKBENCH-02: Phone, navigation/settings closed and expanded alternatives.

```text
+-------------------------------------+
| [Projects] Project / State [Agents]  |
| Compact blocker + available action  |
| Conversation | Execution            |
| Results      | Project plan         |
|-------------------------------------|
| Scrollable content                  |
|-------------------------------------|
| Ask / Changes  [Attach]             |
| [Retained draft]             [Send] |
+-------------------------------------+
| Projects drawer OR Agent overlay   |
| Labeled controls / focus / [Close] |
+-------------------------------------+
```

Phone navigation and settings are dismissible overlays, not permanently narrow
desktop columns. Content must remain reachable with touch and keyboard; opening
an overlay must not discard the composer (AC-RELIABLE-WORKBENCH-001.3/.4).

UI-WORKBENCH-03: Project and results, pending/blocked/ready states.

```text
Project plan: short title | state | milestone
  [Full requirement / acceptance]
  [Edit unassigned item] [Up] [Down]
  Prerequisites: accepted before start
Handoff: saved version | reported facts | inference | unresolved

Results: actual checks | independent review | remaining issues
  [Changed files] [Read text] [Patch / raw evidence]
  Preview: supported entry | source identity | build warning
  [Start preview] [Open] [Stop]
  Unsupported / busy / stale / cleanup unknown: explicit reason
  [Human accept and apply] OR [Continue requirements]
```

Preview is optional and never substitutes for verification. Full requirements,
provenance, unresolved conditions, and runtime failure states must remain
inspectable (REQ-RELIABLE-WORKBENCH-002/.003/.004). Ordinary phone composition uses
the same content in one scrollable column.

## Verification strategy

Each work order records its exact targeted commands. Run backend package checks
from `apps/backend`, Node commands from the repository root, and documentation
validators after staging has been integrated into the complete checkout.
Existing browser fixtures must include the new embedded modules and new API
views before their pass count can establish coverage for this update.

Final release verification must cover the real Windows PowerShell 5.1 desktop
entry, the existing database with FTS5, desktop/phone dark and light layouts,
drafts and older messages, dependent backlog dispatch, candidate file/preview
failure states, original/candidate distinction, low-space admission, native
connections, and unchanged existing task records. Keep per-check receipts on E.

## Current results

- Project implementation: all `TestProject` cases passed with `-race` after final
  project changes (92.188 seconds). The 11 project page logic tests and syntax
  check passed. Receipts are in the staged `validation/project-*.log` and
  `project-slice-receipt.json`.
- Candidate files/preview: the final 11-test `TestResult(Preview|Files)` race group
  passed (68.316 seconds), including generated HTML, old-build/source selection,
  and stopping-ownership regressions. See `validation/preview-validation.json`.
- Native Harness completed layout work using the existing account and
  `deepseek-flash`; 54 tool calls completed successfully. Controller integration
  and independent review remained with Codex.
- All 54 page logic checks and 27 isolated browser checks passed. Desktop
  1440-by-1000 measurements were 135.8 pixels for the summary, 137 for the
  composer, and 613.2 for central content. Phone light/dark layouts had no
  horizontal overflow. Browser mutations targeted fixtures only.
- Backend race regression covered 277 distinct passing top-level cases and 15
  explicit skips across partitioned runs. The first long run timed out after
  recording 235 passes; a separate 46-case selection completed in 163.309
  seconds with no missing selected cases. This is not one successful full-suite
  command. See `validation/backend-regression-receipt.json`.
- Source-change recovery received an independent final review after immutable
  receipt chaining, later-round tamper checks, and fixed writer-scope regressions
  passed. Human confirmation does not reset quality rounds or widen writers.
- Documentation validation passed 62 public-docs tests, 47 published pages, and
  a catalog of 352 decisions/1370 specifications. Global spec lint retains 107
  existing size findings; this changed system has none.
- After the final style refactor, 19 targeted race checks passed in 135.976
  seconds. Reliable-package Go vet passed; the real backend route check passed
  in 3.213 seconds; incremental lint reported zero issues. Four maintained
  browser regressions also passed without being double-counted as new scenarios.
- Final candidate SHA-256 is
  `f57f007f65760b7f62a7330e4bf8a6994eb041469d135e9dc5aa955289deec83`.
- Candidate startup passed using a read-only existing-database backup with 253
  tables, one FTS5 table, and a successful integrity check. Final embedded sources
  matched the candidate and production pages.
- Deployment completed to `E:\小布丁`. All 13 raw task records remained identical;
  native connections and the desktop shortcut were preserved. Two closed-copy
  legacy evidence/quiescence checks passed in 2.542 seconds. No user task was
  resumed or accepted.
- Production desktop/phone read-only checks passed with zero browser errors or
  mutation requests. Summaries measured 7,564 bytes versus 2,770,337 legacy bytes
  (approximately 99.7% less); selected workspace measured 13,705 bytes and
  unchanged reads returned 304. These are payload measurements, not Agent runtime
  speedup claims. See `validation/deployment.json` and `production-runtime.json`.
- Backup: `E:\小布丁\backups\upgrade-1.15.0-20261026-204601`. The original Windows
  shortcut was actually launched and reused the same backend PID without a
  duplicate service. Windows PowerShell 5.1 target, launcher BOM, and parse checks
  passed. See `validation/desktop-shortcut-verification.json`.

Public documentation impact: update the existing local-extension how-to section
in [Tasks and Workflows](../../public/tasks-and-workflows.md). The root README and
screenshot catalog have no changed local-extension entry requiring a new claim.
