---
status: draft
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-EFFICIENCY-001
---

# Execution Efficiency Design

## Ownership and grounding

`internal/reliable` owns native role execution and the snapshots consumed by its
controller. Keep ordinary Kandev sessions and shared agentctl behavior outside
this change. Existing [local loop](local-loop.md),
[domain assignment](domain-assignment.md) and
[resource preparation](large-resource-projects.md) gates still apply.

| Requirement | Design sections |
| --- | --- |
| REQ-RELIABLE-EFFICIENCY-001 | Native completion, timing, preparation reuse, compatibility |

Observed on task `next-9d81684e55eb5320fba4e40ec0`: the reviewer emitted
`turn.completed` and a complete `response.json` near 06:33:35Z on 2026-10-20,
but dispatch failed near 06:53:31Z after the thirty-minute timeout with a
Windows termination error. The raw verdict was `rework`, not pass. This is
evidence of approximately twenty minutes of post-completion waiting; it is not
proof of the exact Windows/pipe root cause. The task is now cancelled, round 1.
Do not recover, resume or adopt its raw result during implementation tests.

## Native completion

### Existing boundaries

`process.go: runProcess` buffers bounded stdout/stderr and calls
`runOwnedCommand`, which waits for `cmd.Wait`. `process_windows.go:
isolateProcess` supplies a kill-on-close Job Object. `WaitDelay` handles exit
and pipe cleanup, not a still-live parent after a native terminal event.
`CodexRunner.Run` rejects process error before reading its report.
`HarnessRunner.Run` uses `harnessStream`, with a different terminal protocol.

`activity_process.go` is a redacted presentation adapter. Its line truncation
and public completion messages are unsuitable as authority for execution.

### Proposed native-only lifecycle

Introduce a native execution supervisor (`native_process.go`, proposed), using
the existing process ownership primitives. Leave Git and `runCheck` on their
ordinary exit-based runner. The supervisor has structured outcome fields:
session, terminal event time, report digest, observed exit/code, cleanup reason,
output/timeout/cancellation/cleanup errors, and event validation result.

1. Capture/framing is independent of UI activity. Parse complete JSON records
   incrementally, including records split across writes. Never silently discard
   oversized/invalid lines; retain the current 16 MiB stdout/stderr budgets and
   1 MiB report ceiling. A final unterminated JSON record is eligible only at
   actual EOF, not after an arbitrary quiet interval.
2. Codex requires the current invocation's single identified session, valid
   terminal turn and a fully written, valid report at its exclusively owned
   response path. Harness requires its matching project/session, completed
   `turn_end` and structured `final`, using the existing stream semantics.
   Report availability may lag the terminal event. Observe/report-bound reads
   until valid or timeout; never treat a parse failure as success.
3. When complete evidence is available, start a five-second natural-exit grace.
   Continue consuming stdout/stderr. Normal explicit nonzero exits, errors,
   extra invalid protocol data and parent cancellation/deadline still fail.
4. If still live at grace expiry, stop only that owned native process job/tree.
   Record a distinct controlled-completion cleanup cause, not user cancellation.
   Bound reaping/draining to the existing two-second cleanup budget. A job that
   cannot be confirmed stopped blocks; never detach it and advance a writer.
5. After exit/drain, revalidate the entire captured protocol and complete report,
   then check parent context again. Only the supervisor's own recorded cleanup
   can explain an expected forced-exit status. Genuine explicit nonzero exit,
   output overflow, timeout, cancellation or cleanup failure remains an error.
   If evidence is incomplete/ambiguous, use existing bounded failure behavior.
6. Persist redacted native logs and cleanup outcome even on failure. Pass a
   valid identified report through existing `completeDispatch` and stage schema,
   read-set, source and policy validation. Native completion does not bypass
   stage validation or determine the review verdict.

At every later controller state commit, cancellation wins over a stale success.
Keep existing fresh-session checks and per-role resource exchanges. A resource
request is a completed preparation exchange, not completed implementation.
Do not reinterpret old failed dispatches or modify their saved end records.

### Provider compatibility

| Runner/transport | Completion identity | New behavior | Unsupported fallback |
| --- | --- | --- | --- |
| Codex CLI JSON | One runtime thread/session + terminal turn + output-schema report | Grace, owned cleanup, full revalidation | Exit-based bounded failure; visible protocol problem |
| DeepSeek Harness CLI JSON | Project/session + completed turn_end + final object | Same lifecycle; existing strict Harness parser | No Codex-event inference or arbitrary final-message success |
| Git, checks, folder picker | Process exit only | Existing behavior | Existing timeout/cancellation |
| Historical dispatch evidence | Persisted exact attempt | Read-only display | Never run or promote it |

Tests use paid-call-free helper processes. Include lingering parents/children,
inherited pipes, normal exits, missing/truncated files, session conflicts,
duplicate/failed turns, errors after final, final without newline, output
overflow, cancellation before/during grace, timeout and teardown failure.

## Timing and observable progress

Persist a versioned diagnostic sidecar under `Root/activity`, separate from all
seal inputs. Record task/attempt/dispatch identity, monotonic duration,
operation, counts, outcome and UTC timestamps. Redact commands/output through
existing redactors. Do not add fields to legacy sealed dispatch serialization.
Clock changes must not yield negative durations or invented elapsed time.

Instrument these boundaries separately: admission inventory; Git enumeration
start/end; physical path enumeration; source byte hashing; resource verification;
manifest read/digest/parse; native start/terminal/exit; actual verification; and
review. Distinguish nested spans to avoid summing overlapping work. Diagnostics
are not proof of success and are optional for older tasks.

Expose bounded diagnostics through an additive read-only task endpoint and the
existing activity/status row. Show e.g. "独立审查已返回，正在收尾" during cleanup,
then the actual next state. Current operation and elapsed time are sufficient;
do not turn one Git checkout percentage into whole-workflow progress.

## Preparation reuse

Observed resource manifest size is about 22 MB. Current `engine.go` preflight,
`workspace.go` preflight and `resourceInventory` can repeat admission for one
attempt. `stages.go`, `resource_dispatch.go: Engine.call`, and
`dispatch.go: beginDispatch` can snapshot the same actor-free boundary.

Create an attempt-scoped immutable prepared-input descriptor. Transfer the
bounded source/resource inventory from admission into owned candidate creation;
revalidate original HEAD, index identity and clean-state conditions at the
publication boundary. Any drift or interruption discards reuse. Do not cache a
clean working tree solely from HEAD/index: unstaged original drift remains a
failure at all existing original-source gates.

Pass a boundary-owned snapshot explicitly to consumers that need that exact
snapshot; compare task/base, requirement revision, resource-manifest digest and
materialization generation. Do not use a time-window or global "latest"
snapshot. Hydration, actor admission, a check process, restart or cancellation
invalidates it. Snapshot after each native call and each existing check/review
boundary. Preserve every source, resource, patch and acceptance verification.

`loadResourceProfile` currently reads, hashes and parses the manifest. Reuse the
immutable parsed object keyed by verified digest, bounded to one task/attempt;
verify current manifest bytes against the expected digest before reuse at an
integrity boundary. This saves repeat parse/allocation without trusting mtime,
size, a marker or resource skip-worktree flags. Never keep an unbounded cache.

Regression evidence must count admission inventories and snapshot invocations,
compare old/new fingerprints, and tamper with same-size/same-time files.
Benchmark cold and warm paths separately. The observed initial eleven-minute
`ls-files`-to-`diff` gap includes multiple operations; attribute it only after
the new spans isolate those operations. Do not claim hashing alone caused it.

## Compatibility and rollout

No model/effort changes, parallel writers, new logins, skill installs, changed
round accounting, automatic acceptance or broad ceiling increases. Existing
seals remain byte-compatible. Native protocol support is explicit; unknown
shapes fail closed or retain the prior bounded exit path.

Use isolated fixtures and a closed database copy for historical seal checks.
Do not run the user's cancelled task, accept a demo, stop active work or mutate
the original project during validation. Local upgrade must preserve current
task state and be deferred if production work is active.

## Implementation Plans

- [Execution efficiency and task results](../../../plans/reliable-speed-results/plan.md)
