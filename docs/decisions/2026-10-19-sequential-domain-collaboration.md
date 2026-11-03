# ADR-2026-10-19-sequential-domain-collaboration: Sequential domain writers with native model selection

**Status:** accepted
**Date:** 2026-10-19
**Area:** workflow

## Context

The user wants Harness frontend work and Codex backend/review work, with custom tool, model and effort choices. One generic implementation stage cannot express this division. Concurrent writers in one worktree would invalidate source provenance and scope attribution.

## Decision

Use sequential domain writers within one persistent round. Preserve one source authority and enforce domain scope at each handoff. Catalog-validated selections are frozen per dispatch, while mode and authorization remain fixed for the task. Legacy single-writer tasks keep their evidence contract.

## Consequences

Both domains share current source without concurrent writes. Total rework stays bounded. Execution order is visible, and model lists reflect existing account sources. Parallel worktrees and merge coordination remain outside this extension.

## Alternatives Considered

- Renaming the generic implementation dropdown would not implement domain responsibility.
- Concurrent writes in one worktree would make evidence and scope ambiguous.
- One round budget per domain would double the authorized limit.
- Rewriting legacy task plans would invalidate already sealed candidates.
