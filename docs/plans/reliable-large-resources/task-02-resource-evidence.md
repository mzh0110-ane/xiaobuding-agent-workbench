---
id: "02-resource-evidence"
title: "Verify materialized inputs and resource-bound evidence"
status: done
wave: 2
depends_on:
  - 01-resource-preparation
plan: "plan.md"
requirements:
  - REQ-RELIABLE-RESOURCES-001
acceptance_criteria:
  - AC-RELIABLE-RESOURCES-001.3
  - AC-RELIABLE-RESOURCES-001.4
  - AC-RELIABLE-RESOURCES-001.5
  - AC-RELIABLE-RESOURCES-001.6
system_design:
  - ../../specs/reliable-collaboration/system-design/large-resource-projects.md
---

# Task 02: Materialization and Evidence

## Summary

Supply bounded independent input resources without inventing successful reads or exposing original bytes to candidate writes. Bind all new evidence to the frozen resource baseline while preserving old snapshot hashes and acceptance seals.

## In scope

- Controller-owned exact-path hydration from Git objects with stream limits, atomic publication, destination guards and cancellation.
- Bounded structured resource requests for both native providers, fresh identified redispatch, preparation evidence and unchanged round budgets.
- Physical resource validation independent of skip-worktree flags; invocation-specific read availability; immutable resource guards in snapshots, patch export, review and acceptance.
- Versioned snapshot hashing and compact resource descriptors, without changing legacy serialization/hashes.

## Out of scope

Original writable links, editing baseline resources, new authentication, plugin/skill installation, automatic test passes or acceptance, production rollout.

## Acceptance

1. Exact resources become independent byte-verified candidate files; case/traversal/link/race/oversize/cancel cases fail closed and never modify original resource contents.
2. Unavailable-resource citations, forged availability, resource tampering/deletion, changed manifests and resource patches cannot pass; identical hydration preserves logical source identity.
3. Valid resource requests are bounded, recorded, rerun with fresh sessions, and do not increase/reset implementation rounds. Existing candidate seals remain valid through the old algorithm.

## Verification

From `apps/backend`:

```powershell
go test -trimpath -race ./internal/reliable -run 'TestResourceMaterialization|TestResourceDestinationGuard|TestMaterializationCancellation|TestDeferredResourceReadRejected|TestResourceRequestDispatch|TestResourceDriftBlocksAcceptance' -count=1 -v
go test -trimpath -race ./internal/reliable -count=1 -v
go vet ./internal/reliable
```

Record exact results after the last production edit. Use real filesystem/store/Git boundaries with the external model boundary faked; no paid native invocation is required.

## Files likely touched

`workspace.go`, `stages.go`, `dispatch.go`, `domain_execution.go`, `evidence.go`, resource snapshot/materializer modules and their new integration tests under `apps/backend/internal/reliable`.

## Dependencies

Task 01's frozen descriptor and owned sparse preparation. Keep the resource-manifest digest and legacy snapshot dispatch contract synchronized.

## Risks

Git filters and CRLF, sparse flags hiding physical files, mutating role resource requests, stable source hashes versus changing input availability, evidence serialization, and cancellation between verify and publish.

## Parallelism

Sequential. No subagent delegation is authorized.

## Results

The final resource/legacy-patch race run passed 12 tests in 214.657 seconds after the final Go production edit. The full owning suite passed 124 top-level cases with eight opt-in skips in 479.440 seconds before the final resource-only patch correction; the focused run covers that correction and all new resource regressions. The additional resource protocol/schema contract test passed with race enabled. Actual WMF and question-bank JSON hydration passed in the opt-in preparation test in 120.12 seconds without a model call. No frozen input entered an accepted patch.
