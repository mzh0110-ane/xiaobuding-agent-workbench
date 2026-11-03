---
status: current
system: reliable-collaboration
created: 2026-10-21
owners:
  - local-distribution
requirements:
  - REQ-RELIABLE-PROJECT-001
  - REQ-RELIABLE-PROJECT-002
  - REQ-RELIABLE-PROJECT-003
---

# Long Project Continuity Design

The existing local workflow remains the execution and acceptance authority. A project ledger in the same Bolt database adds requirements, provenance-bearing memories and an opt-in confirmed backlog. Reads derive batch progress from task records; they never turn model prose into an accepted fact. Stable project identity uses the existing normalized project-root key. Revision checks protect concurrent edits. No per-message project-memory files or additional database are created.

Read-only project planning reuses the existing question queue and auditor connection. Suggested items are retained as unconfirmed. Starting an item uses the normal linked-batch API with deterministic message identity and unchanged native accounts. Only a human-accepted batch can advance an opted-in confirmed backlog. Current task goals remain the execution scope; other project items are context only.

Writer recovery validates independent dispatch identities, saved reports, requirement revisions, source snapshots and domain scopes. Completed frontend output is reused only when its retained evidence matches the current source. Failed backend partial output must remain within its authorized domain. Recovery keeps the implementation round, uses bounded operational retries and still runs actual checks and independent review. A completed writer is never displayed as a verified project requirement.

The optional task recovery journal stores the current round/domain, cumulative execution attempts, explicit resumptions, frozen audit/revision identities and last failure in the existing database. Each invocation permits three automatic attempts only for operational invocation errors after validating current source. Failed attempts can leave allowed partial edits; their first dispatch input anchors the completed-domain handoff, and the successful retry's source identity anchors those partial edits. A changed completed frontend, altered report, missing dispatch or uncertain process cleanup blocks reuse. New requirements use the existing fresh-audit path; operational retry does not reset or consume a quality-rework round. Checking and reviewer interruptions are not given a new recovery protocol by this delivery.

The fourth workspace tab loads the project ledger on demand, exposes requirement acceptance conditions and existing evidence, and preserves keyboard and phone behavior. A bounded patch reader uses existing immutable patch artifacts. Existing open-folder actions expose working files; unavailable application previews are not fabricated.

Patch fetches are limited to 512 KiB even when the server omits a content length, and display stops at 3,000 lines. The immutable original remains separately accessible. Requirement statuses derive from their linked task records; a general passing batch must not silently establish every backlog item or acceptance condition. User memories and read-only model observations retain provenance and remain context requiring current-source validation.

Runtime context selects deployment-owned temporary roots for child processes without changing native credential or skill homes. Build and browser validation use explicit deployment-drive temporary roots. Explicit storage analysis reports application categories. Cleanup is limited to old inactive controller-owned transient roots, rejects traversal and symlinks, and never removes evidence or working copies.

Owned processes receive deployment-drive temporary and supported npm/pip/uv/Go/Playwright cache settings. Existing absolute non-C-drive Go cache locations are retained. Child Codex invocations receive application-owned log and SQLite paths and disable additional local history persistence; original Codex/Harness homes and installed extensions remain unchanged. Each process scratch directory carries an ownership marker and is removed on normal completion. Cleanup first previews candidates, then requires an explicit idle-task deletion request; only marked inactive process directories with all entries older than seven days are eligible. Neither cleanup nor bounded storage analysis claims to reclaim the existing native home, worktrees, attachments, evidence or all cache contents.

## Requirement mapping

| Requirement | Implementation boundary |
| --- | --- |
| REQ-RELIABLE-PROJECT-001 | Project ledger store, read-only plan queue, contextual handoff, confirmed backlog dispatcher |
| REQ-RELIABLE-PROJECT-002 | Writer recovery journal and domain evidence validation |
| REQ-RELIABLE-PROJECT-003 | Project workspace view, bounded patch reader, storage context and retention endpoints |
