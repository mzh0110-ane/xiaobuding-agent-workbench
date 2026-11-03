---
status: active
system: reliable-collaboration
created: 2026-10-18
owners:
  - local-distribution
---

# Local Collaboration Requirements

## Overview

A local user submits a concrete goal for a project and receives a verified candidate with retained evidence before deciding whether to apply it. Ordinary folders use [automatic local recovery points](local-snapshots.md); externally managed Git projects retain clean-root admission. Explicit acceptance also preserves the exact applied result as a local Git version so the next batch can continue in the same folder.

## Requirements

### REQ-RELIABLE-LOOP-001: Bounded independent collaboration

**Intent:** Reduce unsupported completion claims and uncontrolled rework.

#### Acceptance criteria

- **AC-RELIABLE-LOOP-001.1:** Each implementation and review shall use a fresh, runtime-identified session; audit and review shall request read-only execution.
- **AC-RELIABLE-LOOP-001.2:** The controller shall execute nonempty verification commands and retain exit results and output. Failed required checks shall prevent acceptance.
- **AC-RELIABLE-LOOP-001.3:** Malformed, incomplete, stale or serious-defect review evidence shall prevent acceptance. Missing or altered saved contracts, reports or check logs shall also prevent application.
- **AC-RELIABLE-LOOP-001.4:** At most three implementation rounds shall be admitted per task. Concurrency and restarts shall not reset this count.
- **AC-RELIABLE-LOOP-001.5:** Original project source contents shall remain unchanged until explicit acceptance, and source/patch drift or a dirty original shall prevent application. Controller-owned preparation metadata and original-source recovery commits are governed by REQ-RELIABLE-SNAPSHOT-001.
- **AC-RELIABLE-LOOP-001.6:** Cross-origin requests, missing browser capabilities, forged task state, and evidence path traversal shall be rejected.
- **AC-RELIABLE-LOOP-001.7:** Full-snapshot admission shall include every source entry within 30000 files, 256 MiB total and 16 MiB per file. The resource-aware exception in REQ-RELIABLE-RESOURCES-001 shall account for frozen input resources separately without silently omitting source or weakening the mutable-source ceilings. Exceeding the applicable profile limits shall block preparation.
- **AC-RELIABLE-LOOP-001.8:** Audited directory scopes shall accept a single terminal slash and use the same canonical path for authorization, domain overlap checks, persistence and writer handoff. Invalid paths shall identify the offending scope field and entry. Canonicalization shall not resolve traversal, broaden user authorization or weaken evidence-file path checks.

### REQ-RELIABLE-LOOP-002: Accepted local versions and continuous batches

**Intent:** Complete the acceptance-to-next-batch cycle without manual Git commands or duplicated project folders.

#### Acceptance criteria

- **AC-RELIABLE-LOOP-002.1:** Following explicit acceptance of a verified candidate, the system shall create a local Git version containing exactly the accepted changes against their verified original base. This bounded authorization shall apply to both controller-managed and externally managed original roots without adopting ownership of an external repository.
- **AC-RELIABLE-LOOP-002.2:** Automatic acceptance history shall not include unrelated or manually staged changes, rewrite prior history, configure or contact a remote, or change global Git identity. Ambiguous source, index, repository identity, or in-progress Git operations shall stop the operation with an actionable reason while preserving existing work.
- **AC-RELIABLE-LOOP-002.3:** With no later user changes, an explicitly requested next batch shall use the recorded accepted version as its base in the same original project, retaining prior conversation, native assignments and evidence. Later user changes shall follow the existing project-admission policy rather than be silently included in the prior accepted result. The new batch shall start at zero implementation rounds, require fresh checks and independent review, and shall not reset its parent's round count.
- **AC-RELIABLE-LOOP-002.4:** An accepted result from an older release shall be eligible for a missing local version only when retained acceptance evidence proves its original base, exact accepted patch and current project contents. Recovery shall not adopt additional edits or mark unverified source as accepted.
- **AC-RELIABLE-LOOP-002.5:** Interruption and explicit retry shall distinguish an unapplied patch, applied-but-unrecorded source and an already-created matching local version. A retry shall not reapply accepted changes, create duplicate commits, reset rounds or replay an Agent call. Uncertain outcomes shall remain blocked for inspection.
- **AC-RELIABLE-LOOP-002.6:** Each acceptance shall retain bounded intent/result metadata outside the project source. Opening, polling, refreshing or retrying a completed operation shall not append new receipts, create another project copy or create per-file backups. Existing evidence and the working candidate shall remain available.
- **AC-RELIABLE-LOOP-002.7:** The UI shall distinguish saved files, an accepted local Git version, a missing or failed local version and the next batch. It shall expose a localized retry reason without requiring users to repeat their goal, re-upload attachments or manually create Git history for a proven accepted result.

## Out of scope

- Native Kandev task conversion, commits of unaccepted Agent-produced candidates, automatic commits of unrelated external-repository edits, remote publishing, deployments, and guarantees of zero model errors. Controller-owned original-source recovery commits are defined separately by REQ-RELIABLE-SNAPSHOT-001.
- Projects with source symlinks, submodules or mutable source beyond 30000 files, 256 MiB total or 16 MiB per file. Resource-heavy existing repositories have the additional bounded profile defined in [large resource projects](large-resource-projects.md).
