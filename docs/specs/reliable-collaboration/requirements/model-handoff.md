---
status: active
system: reliable-collaboration
created: 2026-10-19
owners:
  - local-distribution
---

# Stage Model Handoff Requirements

### REQ-RELIABLE-MODEL-001: Evidence-preserving model selection

- **AC-RELIABLE-MODEL-001.1:** Users shall select existing Codex or DeepSeek connections independently for audit, implementation and review before starting. Codex shall use its ChatGPT login; DeepSeek shall run in its native Harness using the Harness account configuration.
- **AC-RELIABLE-MODEL-001.2:** A running task shall accept versioned changes to subsequent dispatches without changing its active dispatch, identity, worktree or consumed rounds.
- **AC-RELIABLE-MODEL-001.3:** Every dispatch shall retain actual provider, model, role, round, plan version, source hash, fresh session identity and controller-built handoff evidence.
- **AC-RELIABLE-MODEL-001.4:** Unavailable connections, stale updates, missing evidence or source drift shall never silently fall back or bypass acceptance. Final sealing and closing plan edits shall be atomic.
- **AC-RELIABLE-MODEL-001.5:** The UI shall distinguish actual execution from future assignments, preserve unsaved selections during polling and explain the next action in Chinese.
- **AC-RELIABLE-MODEL-001.6:** A fresh sample shall be available without editing file paths or reusing a dirty prior sample. Legacy single-writer samples may retain Harness audit/review and Codex implementation. New domain and explicit assistant-mode tasks shall retain independent Codex review.
- **AC-RELIABLE-MODEL-001.7:** If a configured Codex model temporarily disappears from its shared native catalog, previously observed native capabilities may remain selectable only for the same connection, native home, provider and model. The UI shall identify this as historical native information, not current account entitlement. Newly listed native metadata shall replace prior capabilities, including removals. Without a matching observation, retain the unverified configured choice without inventing effort options.

Private model conversations are not shared or synchronized. Shared task facts, source and controller-owned evidence define continuity. The system does not guarantee absence of model errors.

### REQ-RELIABLE-MODEL-002: Shared-project assistant modes

- **AC-RELIABLE-MODEL-002.1:** Users shall choose native Codex, native DeepSeek Harness or custom collaboration. Named modes shall apply one native model/effort selection to planning and active implementation roles while retaining real checks, fresh independent Codex review and explicit human acceptance.
- **AC-RELIABLE-MODEL-002.2:** Existing tasks shall save switches through revisioned future plans without changing an active dispatch, task identity, worktree, conversation or quality-round count. Unsaved, saving or conflicting selections shall block work-starting actions while leaving stop and read actions usable.
- **AC-RELIABLE-MODEL-002.3:** The next native Agent shall receive paths to original saved conversation, requirements, replies and verified attachments across the same project's linked batches, together with a bounded reading aid. Historical claims shall remain unverified. Missing or corrupt records, changed attachments, parent cycles and cross-project ancestry shall fail explicitly before successful delivery.
- **AC-RELIABLE-MODEL-002.4:** Shared archives shall be content-addressed and checked against controller-owned dispatch/message anchors. Candidate sealing and acceptance shall include archive integrity. Legacy tasks with no archive anchor shall retain their existing evidence contract.
- **AC-RELIABLE-MODEL-002.5:** Opening the shared-record status shall be read-only and shall distinguish saved records, prepared snapshots and actual model understanding. Archiving shall reuse unchanged content on the configured data drive, count it in storage usage and preserve originals on any bounded-history failure.

Provider-private sessions and hidden reasoning remain outside this contract. A shared archive makes recorded context available; it does not guarantee model recall, comprehension or absence of hallucinations. Limits shall be visible in public guidance rather than silently dropping history.
