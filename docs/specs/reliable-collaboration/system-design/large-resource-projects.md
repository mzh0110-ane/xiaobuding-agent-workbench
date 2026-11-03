---
status: current
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-RESOURCES-001
---

# Resource-Aware Candidate Preparation Design

## Ownership and evidence

`internal/reliable` owns this extension to candidate preparation, snapshots, role handoff, resource availability and acceptance. Ordinary upstream Kandev workspaces are unaffected. The existing [local loop](local-loop.md), [local recovery design](local-snapshots.md), and [desktop workspace](desktop-workspace.md) remain authoritative outside the explicitly proposed resource profile.

| Contract | Mechanism |
| --- | --- |
| AC-RELIABLE-RESOURCES-001.1-.2 | Admission inventory and sparse candidate |
| AC-RELIABLE-RESOURCES-001.3-.4 | Bounded materialization and immutable resource checks |
| AC-RELIABLE-RESOURCES-001.5 | Versioned snapshot and dispatch evidence |
| AC-RELIABLE-RESOURCES-001.6 | Owned preparation recovery and legacy compatibility |
| AC-RELIABLE-RESOURCES-001.7 | Existing activity/error projection |

## Admission inventory

Keep the existing preparation path for projects within the current full-snapshot limits. For an oversized existing clean Git root, build a bounded NUL-delimited index/HEAD manifest without reading the contents or calculating the decompressed size of every Git object. Cross-check stage-zero index entries against the frozen HEAD tree. A staged change, unmerged entry, symlink, submodule, unsafe path or unsupported platform alias remains a blocker. Do not change the original index to qualify a project.

Classify only existing regular baseline files with a case-insensitive extension in this initial binary set: `.png`, `.jpg`, `.jpeg`, `.webp`, `.gif`, `.wmf`, `.emf`, `.tif`, `.tiff`, `.bmp`, `.ico`, `.pdf`, `.docx`, `.xlsx`, `.pptx`, `.mp3`, `.mp4`, `.wav`, `.zip`, `.epub`, `.gz`. This list is a product policy, not a content-security verdict. Existing `.json` and `.jsonl` files under the top-level `data/` and `artifacts/` directories are also frozen input data. This includes the observed 443 MB candidate bank; application configuration JSON elsewhere remains mutable. Everything else stays mutable source, including `.svg`, executable source under `artifacts`, and extensionless entries. Do not exclude a whole directory because its name sounds generated. Refuse unsupported attributes or transformations that prevent verifying a resource's actual bytes against its frozen object identity.

Admit at most 250000 total entries, 30000 source entries, and 64 MiB of inventory metadata. The ordinary 16 MiB/file and 256 MiB/source-byte ceilings continue for materialized mutable source. Resource materialization has separate limits of 64 MiB/file and 2 GiB/batch. Check cancellation during inventory, enumeration and streaming copies. Oversized, all-source repositories still fail before checkout.

Freeze a canonical manifest of resource path, mode, Git object format and object ID. Write it exclusively under the task's evidence directory, retain its SHA-256 digest and counts in the contract, and avoid putting 150000 resource entries in every task-list response. The source and resource inventories together cover every tracked baseline entry. Record admission timing and counts as real activity, without claiming that a file-list percentage measures uploaded bytes.

## Sparse candidate and initial recovery

Use `git worktree add --no-checkout` for a new task-owned branch and configure non-cone sparse checkout only for that candidate. Include source paths and exclude the exact frozen binary paths using escaped Git patterns. Verify case aliases and metacharacter names before mutation. Initialize the sparse index/worktree from the frozen tree; `worktree add --no-checkout` plus `sparse-checkout set` alone is insufficient to establish the candidate index.

The Windows integration regression verifies sparse source/SVG inclusion and resource deferral at the production preparation boundary. Controller materialization reads frozen blob bytes directly and publishes an independent task-local copy through an `os.Root`-confined temporary file and exclusive link. It does not depend on a native Agent running sparse-checkout commands.

Persist a task-specific preparation-intent record before creating the sparse candidate. Recovery verifies canonical Git common directory, expected branch, base and manifest digest. A completed clean sparse candidate can recover a missing completion marker after another snapshot; partially initialized or dirty candidates require inspection and are preserved. An ambiguous or modified directory blocks rather than being reset, removed or overwritten. Existing full worktrees and old cancelled partial worktrees are not silently migrated or reused.

## Bounded materialization

Hydrate from the frozen Git object, never from a writable junction or hard link to the original. Prefer a controller-owned stream from the local Git object database into a task-local temporary regular file, verify its object identity, then atomically publish the independent file inside the candidate. Reject traversal, case aliases, non-regular parent components and destination races. Materialized inputs may retain their skip-worktree flag; physical verification is authoritative. Patch export uses `git add --sparse` so newly generated assets are included, followed by an immutable-resource index guard. Original history and ignore rules are not changed.

Role handoffs carry the resource-manifest digest, existing materialization records, remaining limits and the resource-read-only policy. Add a bounded `resource_requests` alternative to role output: at most 64 exact baseline paths per request, two request/redispatch cycles per role invocation. This is a controller preparation exchange, not a new implementation round. A request cannot simultaneously count as a completed audit, implementation or passing review. The controller first verifies that the resource request did not modify source, hydrates its input, records request/result evidence, and starts a fresh identified role session with the same frozen source and implementation round. Read-only audit/review never need permission to write their own resources.

Native roles are instructed to use controller resource requests rather than modifying Git sparse state. The next controller snapshot must detect and validate all locally materialized resources regardless of skip-worktree flags. Enumeration must inspect the physical candidate rather than treating index flags as proof of absence. Reject modifications and direct writes replacing deferred resources. A test that needs additional files without a valid resource request remains a real failed check; missing images never justify a synthetic pass. Resource groups needed for a declared integration check must be hydrated within budget before the controller executes it.

## Snapshots, citations and patches

Add an explicit snapshot version only for new resource-profile tasks. Preserve the existing snapshot algorithm and serialization for all legacy contracts and seals. The new hash binds mutable-source byte fingerprints, base commit and immutable resource-manifest digest. Materializing identical frozen inputs does not change that logical source hash. Separate verified available-resource fingerprints from mutable `Snapshot.Files`; availability is evidence, not an editable source entry or a way to waive original-source drift.

At each stage boundary, verify the physical materialized-resource set and byte identities before creating evidence or accepting citations. Resource deletion after acknowledged hydration is a failure until explicitly restored by the controller. `checkReads` must check the invocation's verified availability as well as source membership; a resource's presence in an immutable manifest is not proof that an Agent read it. A requested but unavailable resource cannot satisfy required reads.

Before `ExportPatch`, validate candidate index and resources. The candidate patch may include scoped source and newly generated files, but may not modify any frozen resource path. `acceptanceEvidence` rechecks the manifest file, candidate source, currently materialized resources, patch digest and all existing sealed evidence. `ApplyPatch` retains clean original/base checks and explicit human acceptance. Source or resource drift blocks application. This workflow reduces accidental writes; full-access native Agents are not an OS security boundary.

## Compatibility and visible behavior

| Context | Intended behavior | Evidence / fallback |
| --- | --- | --- |
| Small existing Git project | Existing full copy and old snapshot version | Existing regression suite and old acceptance seal |
| Oversized clean Git project with bounded binary resources | New sparse resource profile | Frozen inventory, bounded source, independent materialization |
| Large all-source or unsupported tree | Refuse before checkout | Specific reason; no silent exclusions |
| Ordinary folder or managed local recovery project | Existing preparation policy for this package | No unrequested adoption or history rewrite |
| Codex / DeepSeek Harness | Same profile and structured resource-request handoff | Native provider settings unchanged; malformed requests fail |
| Archived or previously sealed task | Existing controller behavior | No silent migration or seal rewrite |

Use existing Chinese activity rows for source/resource counts and materialization progress. Keep actions, desktop/mobile layout, model drawer and background unchanged. Update only misleading whole-repository error text. The blocked task `next-9d81684e55eb5320fba4e40ec0` keeps its identity, original goal, current assignments and round zero; a later explicit continue action performs new admission. Do not automatically dispatch a paid Agent during validation or apply a prior pending candidate.

## Alternatives and risks

Raising every snapshot ceiling retains repeated full-library reads and copying. Original-file hard links or writable directory junctions expose the original to accidental changes. Editing `.gitignore` cannot remove already tracked files and changes project policy. A new lightweight repository would split the user's history and needs data-path adaptations. Sparse isolated candidates retain project-relative paths while materializing only proven inputs, at the cost of explicit resource availability and recovery logic.

Windows sparse-checkout initialization, arbitrary path escaping, filters, sparse index flags, resource availability in actual checks, and old seal compatibility require integration tests before deployment. Actual project preparation admitted 3263 mutable source/text entries and 150081 deferred inputs in 101.50 seconds in the first successful controlled run, preserving original HEAD/index/contents. Final deployment verification is recorded in the implementation plan; application correctness is a separate check.
