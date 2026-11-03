---
status: current
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-DOMAIN-001
---

# Domain Assignment Design

New plans explicitly name a mode and separate frontend/backend assignments. Legacy plans without a mode keep their single-writer behavior and artifact list. New optional serialized fields are omitted when empty so historical dispatch hashes remain identical.

Planning establishes separate non-overlapping path scopes when both domains are enabled. Optional user domain paths narrow those scopes. The task mode and user paths are immutable after the initial plan, while tool, model and effort selections remain versioned and editable for future dispatches.

The controller admits one round, freezes its audit, then invokes enabled frontend/backend implementers sequentially. Role remains implementer to preserve writing responsibility and report schema; domain is separate dispatch metadata. Native full access does not change these responsibilities. Each writer receives the current source hash, prior domain evidence and the shared task facts. Per-writer source changes are checked against domain, audit and user scopes. A controller-owned aggregate report binds both results and their source snapshots. Only then run verification and a fresh read-only review. The existing atomic evidence seal and acceptance gates remain authoritative.

Codex catalog data is a whitelist projection of its existing models_cache.json. Harness model data comes from its native account's fixed official models endpoint with credentials retained only in memory, redirects disabled and bounded response/timeout. Local version compatibility and text-output capability constrain options. Catalog errors expose a generic status and an explicitly unverified configured default. Catalog replacement is synchronized; active dispatches retain frozen selections. Per-call model and effort overrides never mutate a shared runner or global config.

The UI shows work mode, separate domain cards with dependent tool/model/effort controls, optional path settings, and planning/review controls. Running views distinguish frozen actual execution from editable future assignments. See [the decision](../../../decisions/2026-10-19-sequential-domain-collaboration.md).
