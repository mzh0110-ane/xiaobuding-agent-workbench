---
status: active
system: reliable-collaboration
created: 2026-10-19
owners:
  - local-distribution
---

# Automatic Local Project Preparation Requirements

## Overview

A novice selects an ordinary project folder and describes an outcome. Xiaobuding prepares a local recovery point and isolated candidate without requiring Git commands or a GitHub account. A managed project has local history created by Xiaobuding and ownership bound to that canonical original folder. Existing repositories without that ownership remain externally managed.

### REQ-RELIABLE-SNAPSHOT-001: Automatic local recovery points

- **AC-RELIABLE-SNAPSHOT-001.1:** When a valid task targets an ordinary folder, the system shall create local project history and a recovery point before starting any Agent, without configuring a remote, uploading files, or changing global Git identity.
- **AC-RELIABLE-SNAPSHOT-001.2:** At most one unfinished task shall own preparation and execution for the same canonical project folder or overlapping parent/child folders; a competing request shall not create a recovery point or start an Agent.
- **AC-RELIABLE-SNAPSHOT-001.3:** Before a later task starts in a managed project, unstaged source changes shall be preserved in a new recovery point. User-staged changes shall block automatic preparation without altering the index.
- **AC-RELIABLE-SNAPSHOT-001.4:** Externally managed Git repositories shall retain clean-root admission; preparation shall not automatically commit their changes or adopt their ownership. The narrowly authorized post-acceptance local version is governed by REQ-RELIABLE-LOOP-002 and shall not permit general preparation commits in an external repository.
- **AC-RELIABLE-SNAPSHOT-001.5:** New automatic staging shall exclude paths in the explicit credential, dependency and cache exclusion list. A conflict between admitted source and those exclusions shall block preparation with a reason rather than force-add excluded paths. This is path filtering, not a comprehensive scan of file contents for secrets.
- **AC-RELIABLE-SNAPSHOT-001.6:** Automatic preparation shall refuse parent-repository subdirectories, internal worktrees, tool installation roots, nested repositories, source links, and projects exceeding AC-RELIABLE-LOOP-001.7 limits without altering their source contents or existing user history/index. Failed new-folder preparation may retain controller-created Git metadata and local exclusion rules.
- **AC-RELIABLE-SNAPSHOT-001.7:** Preparation shall be persisted and cancellable. Restart shall retain the task, completed recovery points and round count, and require explicit continuation rather than silently replaying preparation.
- **AC-RELIABLE-SNAPSHOT-001.8:** Once a task has a worktree, continuation shall not create another original-project recovery point. Original drift shall remain an application blocker under AC-RELIABLE-LOOP-001.5.
- **AC-RELIABLE-SNAPSHOT-001.9:** The UI shall explain ordinary-folder support, local-only recovery points and separate candidate work. Preparation failures shall retain the project, goal and assignments and provide a localized next action.

### REQ-RELIABLE-BOOTSTRAP-001: Real verification for new projects

- **AC-RELIABLE-BOOTSTRAP-001.1:** With no explicit command or discoverable check, a new task shall enter automatic-check preparation instead of requiring the novice to invent a command. The UI shall distinguish preparation from a verified result.
- **AC-RELIABLE-BOOTSTRAP-001.2:** An automatic-check planner may acknowledge an empty source only when its current captured source manifest is empty; an empty acknowledgement for nonempty source shall remain invalid, including during rework.
- **AC-RELIABLE-BOOTSTRAP-001.3:** An automatic-check implementer shall establish executable tests or a real build/type check. The controller shall discover and retain a verification plan, execute its commands, and retain real output and exits before review or acceptance.
- **AC-RELIABLE-BOOTSTRAP-001.4:** Missing or placeholder-only checks and detected zero-test executions shall request rework within the same three-round task budget. They shall not synthesize passing checks, dispatch a passing review, or enable acceptance. A zero-test execution shall retain its actual exit code and be shown as unverified even when that exit code is zero.
- **AC-RELIABLE-BOOTSTRAP-001.5:** User-provided commands shall remain unchanged; automatic discovery shall not replace them or relax an established verification plan during rework.
- **AC-RELIABLE-BOOTSTRAP-001.6:** Automatic verification evidence shall participate in source-bound evidence and acceptance checks. Missing, altered or stale verification evidence shall prevent application.
- **AC-RELIABLE-BOOTSTRAP-001.7:** Local preparation and check bootstrap shall preserve fresh-session independent review, explicit human application, and the shared three-round frontend/backend limit.

## Exclusions

Automatic commits of unaccepted Agent-produced candidates, adoption of external repositories, remote publication, global identity changes, comprehensive secret-content scanning, unattended acceptance, and guarantees of zero model errors remain excluded. Accepted-result versions are owned by [the local collaboration loop](local-loop.md), not preparation. Recovery points cover admitted source under the explicit path policy, not every file in a folder or an entire drive. Existing history is not rewritten to remove previously committed material.
