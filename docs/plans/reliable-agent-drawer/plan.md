---
status: completed
system: reliable-collaboration
---

# Agent settings drawer and continuation model choices

The user requested implementation: fix unavailable model/effort controls after inspection, and move agent controls to a toggleable side panel rather than the bottom of a long task page.

## Work order

1. Reproduce completed-task settings rejection and inaccessible drawer entry through HTTP/browser tests.
2. Store future choices separately from sealed execution plans. Apply them to subsequent questions and linked work batches. If a verified batch receives more requirements, promote its future choices into new immutable execution-plan evidence before rework.
3. Add a desktop side panel and mobile modal drawer, with role tabs, visible entry/close controls, Escape/focus handling, persistent drafts and no main-page scrolling.
4. Verify model/effort delivery, old seal validity, multi-window version guards, desktop/mobile interactions and production history; deploy the checked local binary.

## Interface

```text
Sidebar              Project conversation                  Agent settings [close]
[Agent settings]     ...                                   Front | Back | Plan | Review
                     [message composer]                     Tool [native connection]
                                                           Model [native catalog]
                                                           Reasoning [native effort]
                                                           [Save future choices]
```

On mobile, the drawer is modal; on desktop it remains complementary so the project conversation and toggle stay usable.

## Verification

The final owning-package race run passed 113 top-level tests with seven opt-in skips in 193.969 seconds. Six state-reconciliation tests and 33 browser scenarios passed, including a sticky cross-window conflict after stage completion. Go vet and owning golangci-lint passed. Native-catalog runtime checks confirmed GPT-6.1-Sol ultra selection, full-width stacked controls, desktop toggle, phone/landscape modality, dark-label contrast and no task-setting writes. The real HTTP/store/native-protocol conversation fixture retained two answers without changing source or spending modification rounds. No paid model call was made. Global specification lint retains unrelated baseline findings; this owner has zero findings.