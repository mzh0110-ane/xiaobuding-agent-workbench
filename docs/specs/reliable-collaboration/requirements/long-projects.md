---
status: active
system: reliable-collaboration
created: 2026-10-21
owners:
  - local-distribution
---

# Long Project Continuity Requirements

### REQ-RELIABLE-PROJECT-001: Durable project knowledge

- **AC-RELIABLE-PROJECT-001.1:** One project ledger shall span linked batches, retain user requirements and acceptance conditions, reject stale edits, and distinguish planned work, runtime completion, verified candidates, and human acceptance.
- **AC-RELIABLE-PROJECT-001.2:** Project memories shall cite their user or task origin and version. Historical model observations shall remain unverified context; future agents shall freshly inspect current source.
- **AC-RELIABLE-PROJECT-001.3:** Read-only planning may propose bounded backlog items. Model suggestions shall require explicit confirmation before execution. Opt-in continuation shall start only confirmed pending items after human acceptance, without automatic acceptance or round resets.

### REQ-RELIABLE-PROJECT-002: Resumable writer stages

- **AC-RELIABLE-PROJECT-002.1:** Recoverable writer failures shall preserve the current implementation round and validated completed domains. Source, scope, requirements and dispatch evidence shall be checked before continuing.
- **AC-RELIABLE-PROJECT-002.2:** Operational retries shall have a visible bounded budget separate from quality rework. A resumed candidate shall still receive actual verification, fresh independent review and human acceptance.

The delivered recovery boundary is the frontend/backend writer stage. Each invocation permits at most three automatic operational attempts; an explicit later resume remains available with cumulative counters, including eligible writer failures at implementation round three. New requirements require a fresh audit, and quality rework retains its original three-round limit. Checking/reviewing interruption recovery is outside this delivery.

### REQ-RELIABLE-PROJECT-003: Inspectable results and bounded storage

- **AC-RELIABLE-PROJECT-003.1:** The project view shall expose evidence links, batch history and a bounded on-demand patch reader without claiming missing previews or verified requirements.
- **AC-RELIABLE-PROJECT-003.2:** Application-owned runtime temporary files, build caches and validation artifacts shall use the deployment drive. Existing native account and extension roots shall remain unchanged.
- **AC-RELIABLE-PROJECT-003.3:** Storage analysis shall be explicit and cancellable. Cleanup shall preview and remove only inactive, sufficiently old application-owned transient files, preserving source, working copies, attachments, audit evidence, native accounts and symlink targets.

The patch reader is bounded to 512 KiB and 3,000 displayed lines. Temporary cleanup considers only controller-marked process directories with contents at least seven days old, requires idle tasks for deletion, and does not establish a cache quota or evidence-retention purge policy.
