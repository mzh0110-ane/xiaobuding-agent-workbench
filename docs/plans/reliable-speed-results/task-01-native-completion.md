---
id: "01-native-completion"
title: "Bound native completion cleanup"
status: pending
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-RELIABLE-EFFICIENCY-001
acceptance_criteria:
  - AC-RELIABLE-EFFICIENCY-001.1
  - AC-RELIABLE-EFFICIENCY-001.2
  - AC-RELIABLE-EFFICIENCY-001.7
system_design:
  - ../../specs/reliable-collaboration/system-design/execution-efficiency.md
---

# Task 01: Bound Native Completion Cleanup

## Summary

Reproduce valid native completion followed by lingering Windows parent/child
processes without paid calls. Deliver a native-only completion supervisor with
strict provider protocol validation and bounded owned cleanup.

## In scope

- Incremental Codex/Harness event/report completion, trailing-output validation,
  five-second grace, two-second drain and separate controlled cleanup outcome.
- Cancellation/deadline/error precedence, session/report integrity, native log
  persistence and cancellation checks before controller state publication.
- Helper executables and Windows descendant/pipe failure fixtures.

## Out of scope

Changing Git/check exit semantics, adopting historical reports, result UI,
resource caching, model changes and production task execution.

## Acceptance

1. Both supported native protocols settle a valid lingering-completion fixture
   within ten seconds, retain its exact verdict and leave no owned descendant.
2. Missing/invalid/stale evidence, later errors, unexpected exits, overflow,
   cancellation/deadline and cleanup failure remain failures with saved logs.
3. Existing fresh-session, resource-request, Git/check cancellation and sealed
   evidence behavior passes without rewriting old data.

## Verification

From repository root after the plan's toolchain setup. New test names are to be
implemented, not existing verification evidence.

```powershell
go -C apps/backend test -trimpath -race ./internal/reliable -run 'TestNativeCompletionLifecycle|TestNativeCompletionRejectsInvalidEvidence|TestWindowsCancellationTerminatesOwnedDescendants|TestGitCancellationStopsDescendantsAndReturns|TestResourceRequestProtocol|TestResourceRequestDispatch|TestAcceptanceRejectsMissingOrAlteredEvidence' -count=1 -v
```

## Files likely touched

`apps/backend/internal/reliable/process.go`, `process_windows.go`, `harness.go`,
`harness_stream.go`, `dispatch.go`, proposed `native_process.go`,
`native_process_test.go`, and existing Windows process tests in that package.

## Dependencies

None. Read both native parsers and the existing process ownership tests first.

## Risks

Output report publication can lag the terminal event; treating UI activity as a
completion validator or masking cancellation errors would admit invalid work.

## Parallelism

Sequential. Later work consumes the completion/timing result shape.

## Inputs

Paired execution-efficiency design, current native runner/parser source and the
retained failed reviewer evidence described by the plan, read-only.

## Results

Pending.
