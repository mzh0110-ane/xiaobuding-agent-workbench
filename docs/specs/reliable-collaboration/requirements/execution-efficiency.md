---
status: draft
system: reliable-collaboration
created: 2026-10-20
owners:
  - local-distribution
---

# Execution Efficiency Requirements

## Overview

Local collaboration shall spend time on actual project work rather than waiting
after a native turn finishes or repeating preparation within one protected
handoff. This system owns the result because its dispatches, snapshots and
acceptance policy determine when a stage can safely advance.

This capability extends the [local loop](local-loop.md) and
[observable execution](desktop-workspace.md). Those contracts remain
authoritative for independent review, real checks and human acceptance.

## Terminology

- **Native completion:** A provider-specific terminal event, identified session
  and complete report belonging to the current invocation. A public message or
  a file appearing alone does not establish completion.
- **Handoff boundary:** An interval with no agent, verification process or other
  admitted controller actor allowed to change the candidate.
- **Cleanup:** Stopping and reaping all subprocesses owned by an invocation.

## Requirements

### REQ-RELIABLE-EFFICIENCY-001: Bounded execution overhead

**Intent:** Eliminate avoidable waiting while preserving every correctness gate.

#### Acceptance criteria

- **AC-RELIABLE-EFFICIENCY-001.1:** When a supported native invocation has fully
  valid completion evidence but its owned process remains alive, the controller
  shall finish bounded cleanup and classify the invocation within ten seconds
  of complete evidence becoming available in the controlled regression fixture,
  rather than waiting for the thirty-minute invocation timeout. Cleanup failure
  shall block advancement and expose its cause.
- **AC-RELIABLE-EFFICIENCY-001.2:** User cancellation, expired execution deadline,
  output overflow, failed turns, malformed/incomplete/wrong-session reports,
  unexpected nonzero exits and errors received after an apparent terminal event
  shall not become successful completion. A report requiring rework shall
  retain that verdict; native completion shall not mean review approval.
- **AC-RELIABLE-EFFICIENCY-001.3:** The controller shall retain real durations for
  preparation, source enumeration/verification, each native invocation, native
  completion-to-exit cleanup, verification and review. The task view shall show
  the current operation and elapsed duration without invented overall progress
  or unsupported estimated finish times.
- **AC-RELIABLE-EFFICIENCY-001.4:** Unchanged preparation within one owned attempt
  and unchanged handoff inputs within one actor-free boundary shall not repeat
  a complete admission inventory or equivalent snapshot solely because several
  controller helpers consume it. Changed original/index identity, evidence,
  resource availability, restart or actor boundaries shall invalidate reuse.
- **AC-RELIABLE-EFFICIENCY-001.5:** Reuse shall preserve full content verification
  after each actor and around checks, review and application. Resource tampering,
  same-size/same-timestamp edits, missing resources and original drift shall
  remain detectable. Cold/warm measurements and counted operations shall
  substantiate any claimed reduction; no speed percentage is promised.
- **AC-RELIABLE-EFFICIENCY-001.6:** Existing native logins, configured skills and
  plugins, selected tools/models/efforts, domain execution order, requirement
  revisions and three-round budget shall remain unchanged. Reading timing or
  result information shall not trigger a model call, test or project scan.
- **AC-RELIABLE-EFFICIENCY-001.7:** Existing sealed candidates, cancelled tasks
  and incomplete historical invocations shall remain inspectable without
  rewriting evidence, replaying work, automatically adopting raw reports,
  resetting rounds or applying changes. An unsupported native event shape
  shall keep the existing bounded exit/failure behavior with an explicit reason.

## Out of scope

Parallel frontend/backend writes into the same directory, automatic model or
effort reduction, skipping verification, restarting cancelled work, GitHub
uploads, private reasoning disclosure and guarantees that a model makes no
mistakes are excluded.

## Implementation Plans

- [Execution efficiency and task results](../../../plans/reliable-speed-results/plan.md)
