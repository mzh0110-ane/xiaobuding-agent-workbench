---
status: draft
system: reliable-collaboration
created: 2026-10-20
owners:
  - local-distribution
---

# Task Result Requirements

## Overview

A novice shall see what a collaboration attempt produced and open the actual
files or a local website preview from its result. Reliable collaboration owns
the contract because a result belongs to its task, round, evidence and human
acceptance lifecycle. This extends rather than replaces
[desktop workspace actions](desktop-workspace.md).

## Terminology

- **Candidate result:** Changes in the isolated working copy, not yet applied.
- **Partial result:** Artifacts retained after interrupted, blocked or cancelled
  work. Their existence does not establish correctness or acceptance.
- **Historical result:** Evidence for a specific earlier round or requirement
  revision, explicitly distinguished from the current attempt.
- **Preview:** Viewing retained output or explicitly running a supported local
  project. Preview readiness is not a successful verification or review.

## Requirements

### REQ-RELIABLE-RESULTS-001: Inspectable evidence-bound results

**Intent:** Make actual outputs accessible without reading an entire event log.

#### Acceptance criteria

- **AC-RELIABLE-RESULTS-001.1:** A task detail shall place a compact result card
  next to its execution status, above long conversation/evidence sections.
  It shall distinguish running work, read-only investigation, partial results,
  candidates awaiting acceptance and applied results. Cancelled or unreviewed
  output shall never receive an accepted/verified badge or apply control.
- **AC-RELIABLE-RESULTS-001.2:** The card shall show saved implementation
  summaries, actual changed-file count where derivable, required check outcomes,
  accepted review decision and remaining blockers. Model claims, validated
  reports and actual checks shall be labelled distinctly. Missing, altered or
  ambiguous evidence shall show an unavailable/unverified reason, not a guessed
  value or a synthetic successful result.
- **AC-RELIABLE-RESULTS-001.3:** Users shall open the actual patch, individual
  changed-file text/diff, each available check log, review report and supported
  output artifacts. Each link shall identify its own round, report/check
  provenance and candidate identity; an earlier pass shall not be presented as
  the current round's pass. A patch shall remain accessible when review failed
  or was never saved, provided that actual patch evidence exists.
- **AC-RELIABLE-RESULTS-001.4:** Result inspection shall survive reload, archive
  and service restart without changing task evidence or acceptance validity.
  Rapid task/result switches and failed requests shall not publish another
  task's output. Inspection shall never start work, scan the project, execute a
  model, run a check, accept a candidate or fetch every large artifact eagerly.
- **AC-RELIABLE-RESULTS-001.5:** Desktop shall show summary and primary actions
  before expandable file/evidence details. Phone shall stack the summary and
  actions, using a dismissible full-width result viewer instead of squeezing a
  desktop split pane. Controls shall have accessible names, keyboard operation,
  visible focus, loading/error/retry states and no horizontal page overflow.
  User content and paths shall wrap or disclose their full value.
- **AC-RELIABLE-RESULTS-001.6:** Legacy, domain-mode, archived, read-only and
  cancelled tasks shall have truthful result projections. Earlier per-domain
  limitations shall retain their report/round attribution rather than appearing
  as an unqualified final limitation. Unaccepted raw review output may be read
  as raw evidence, but shall not be adopted as the controller's review decision.

### REQ-RELIABLE-PREVIEW-001: Explicit local result preview

**Intent:** Open a usable result with one clear action when a supported entry is
known, without confusing preview with validation.

#### Acceptance criteria

- **AC-RELIABLE-PREVIEW-001.1:** A supported result shall expose an explicit
  preview action and identify the task, result generation and source target.
  The action shall open the controller-selected local preview only when ready.
  Task selection, ordinary GET requests, reload and polling shall not launch
  project code. Ambiguous or unsupported entry points shall show a clear
  setup/unavailable reason and retain report/file/folder access.
- **AC-RELIABLE-PREVIEW-001.2:** Initial runtime support shall cover existing
  static HTML with local assets and a confirmed Flask application entry with an
  existing Python environment. Jinja templates shall require their application
  runtime, not be offered as standalone websites. Missing dependencies, resource
  budgets, unavailable inputs and startup failures shall be visible; the system
  shall not silently install dependencies, start tunnels or upload files.
- **AC-RELIABLE-PREVIEW-001.3:** Preview shall use a separate disposable copy and
  separate runtime storage, never the original project or reviewed candidate as
  its writable working directory. No Git metadata, native login credentials or
  controller browser capability shall be copied or sent to the preview.
  Previewed project code remains trusted local code; a copy alone shall not be
  described as OS isolation from malicious absolute-path writes.
- **AC-RELIABLE-PREVIEW-001.4:** Executable preview content shall use a separate
  loopback origin, without access to the reliable panel's authority. Paths,
  hostnames and file serving shall be confined to the selected preview; client
  input or model reports shall not supply arbitrary commands, absolute paths,
  external URLs or arbitrary proxy targets.
- **AC-RELIABLE-PREVIEW-001.5:** The UI shall expose preparing, starting, ready,
  failed, stopped and expired states, the local URL when ready, a bounded log
  viewer and stop/retry actions. Owned processes shall stop on explicit stop,
  service shutdown and the stated lifetime limit, and shall not automatically
  restart after recovery. Readiness shall not be claimed for a port owned by
  another process or an unrelated HTTP response.
- **AC-RELIABLE-PREVIEW-001.6:** Preview generation shall serialize with source
  selection and acceptance, use frozen input identities and preserve original,
  candidate, tests and sealed evidence. Opening a preview shall not create a
  passing review, consume an implementation round, accept work or extend the
  three-round budget. Applying a candidate shall still independently revalidate
  source and evidence; an old preview shall be labelled with its old generation.

## Out of scope

Remote hosting, arbitrary shell launchers, automatic paid API use, automatic
dependency installation, live editing from the viewer, arbitrary framework
support, native desktop artifact widgets, private chain of thought and
unattended acceptance are excluded. Preview does not verify feature behavior.

## Implementation Plans

- [Execution efficiency and task results](../../../plans/reliable-speed-results/plan.md)
