---
status: current
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-SNAPSHOT-001
  - REQ-RELIABLE-BOOTSTRAP-001
---

# Automatic Local Project Preparation Design

## Boundary and mapping

`internal/reliable` owns admission, original-folder preparation, source evidence and automatic check establishment. Native runners, independent review, the evidence seal and human acceptance remain authoritative. Ordinary upstream Kandev tasks are unaffected.

| Requirement | Design sections |
| --- | --- |
| REQ-RELIABLE-SNAPSHOT-001 | Admission and ownership; Recovery points; Preparation lifecycle; UI projection |
| REQ-RELIABLE-BOOTSTRAP-001 | Automatic verification; Evidence and compatibility; UI projection |

## Admission and ownership

HTTP creation resolves the original path to its canonical identity and validates the goal, scopes and tool/model selections before filesystem mutation. `Store.Create` reserves the project through its atomic unfinished-task constraint, including equal and overlapping parent/child roots. Only the admitted task enters `Engine.prepare` with `auto_snapshot` enabled. A failed preparation remains inspectable and cancellable under that reservation; retries reuse the task rather than racing a second initializer.

A repository-local `xiaobuding.managed-root` marker binds ownership to the canonical original folder. Existing Git is managed only when that marker matches the folder. Missing or mismatched ownership never authorizes adoption. Reject parent-repository subdirectories, runtime/worktree paths, broad tool installation roots, nested repositories, links/reparse-point source and source beyond the fixed snapshot limits. Refusal preserves source contents and existing user history/index; a new root can retain controller-created Git metadata or `info/exclude` rules from incomplete preparation.

## Recovery points

For an ordinary admitted root, initialize local Git, controller exclusions and ownership, then create an initial commit. An empty source may use an empty initial commit. Identity is command-local; global identity and remotes are untouched. All controller Git helpers clear inherited `GIT_*` environment variables so the original repository, alternate index and candidate operations cannot be redirected by the parent shell. Disable hooks and inherited Git execution helpers at this boundary.

For a managed root, reject an in-progress Git operation or nonempty user index before snapshotting admitted unstaged changes. Commit only when included source changed. Externally managed roots retain `requireClean` with no automatic preparation commit. The separate [accepted-result checkpoint](local-loop.md#accepted-local-versions) can record an exact human-accepted patch in either root without adopting external ownership. No preparation path configures a remote, fetches or pushes.

Controller source exclusions are independent of project ignore text. `local_project_policy.go` owns an explicit credential/dependency/cache path list; it does not scan arbitrary file contents for secrets or rewrite existing history. Inspect paths before staging and pass the explicit admitted list using literal semantics and NUL delimiters. Do not blanket-stage or force-add files. Inspect tracked/changed entries and ignore conflicts so excluded paths cannot be newly staged through a project override. Unsafe paths and oversized source fail before a recovery commit. Exclusions do not imply a backup of every file in the folder.

After preparation, establish the base commit, baseline snapshot, worktree and frozen task contract through the existing path. A recovery point is an original-source baseline, not an Agent result or an implementation round. Candidate application still requires explicit human acceptance.

## Preparation lifecycle

Persist `preparing` for work that outlives the HTTP request. Cancellation uses the existing task context and process ownership. Restart records interruption without resetting rounds or silently replaying preparation. Before creating `.git`, persist `local-init-intent` evidence bound to this task and canonical original. A retry can recover this task's interrupted initialization only if there is no index, ref or remote. It never adopts an externally initialized repository.

`local_project_snapshot.go` records a pending checkpoint with the original root, base HEAD and staged-index hash. A matching pending record distinguishes owned interrupted staging from a user index. The crash window after `git add` but before writing that record cannot be proven owned: continuation blocks and preserves the index for maintainer recovery. It does not reset or blindly recommit it. Incomplete initialization or exclusion setup may retain local metadata even though source contents and prior user HEAD/index stay protected.

Handle an existing `Worktree` before any automatic preparation on resume. Restore its frozen contract instead of committing original changes again. This preserves source-drift rejection and prevents continuation from legitimizing unrelated edits after verification.

## Automatic verification

Explicit user commands remain unchanged. Otherwise discover real existing checks. With none, persist `auto_checks` and allow planning/implementation to establish source and verification.

An empty planner read acknowledgement is permitted only while the current controller-captured manifest is empty. Once files exist, normal source-read requirements apply, including rework. Invented-read rejection remains unchanged. Planning describes the missing foundation; enabled writers establish implementation and real checks within their audited scope and existing shared round.

After implementation, discover candidate commands and reject known placeholder/no-op scripts. Persist controller-owned `check-plan` evidence binding commands to source before executing them. Once established, retain that plan through rework. Missing real checks requests rework without review, consuming the same three-round implementation budget. `echo`, unconditional zero exits and model assertions never substitute for actual test/build processes.

Run accepted commands through `runCheck` with output, log hashes and actual exits. Automatic-check output recognizes zero-test Node, Python unittest and Go test results as insufficient evidence; retain the real exit code and record a separate optional `verification_problem` instead of changing a successful process exit into a fabricated failure exit. The UI and policy treat that outcome as unverified. A Go run with at least one successful real test package remains valid even if other packages have no tests. Only successful verified checks proceed to a fresh Codex review for domain-mode tasks. Review evaluates feature behavior and check relevance; command execution alone does not prove coverage or comprehensively classify arbitrary scripts.

## Evidence and compatibility

New flags and check-plan fields are optional and omitted when absent. Existing tasks preserve clean-Git admission and historical serialized evidence. Seal the bootstrap plan for applicable new tasks and revalidate its integrity, check outcomes, frozen source, patch and unchanged clean original at acceptance. Do not rewrite a sealed legacy contract to add new fields or artifacts.

Retain recovery provenance and preparation failures without credentials. Task identity, source authority and round count remain in the existing store; private native conversations are not synchronized.

## UI projection

Keep the project/goal form and assignment controls. Explain ordinary folders, automatic local recovery and absence of GitHub uploads beside the path. Submission/task views distinguish preparation, execution and verification. Localized failure messages preserve drafts instead of requiring Git commands. Empty projects show pending verification rather than a pass. Preserve labels, keyboard behavior and current mobile stacking.

## Related decision

- [Managed local recovery points before source baseline](../../../decisions/2026-10-19-managed-local-project-snapshots.md)
