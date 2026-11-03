---
status: current
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-MODEL-001
  - REQ-RELIABLE-MODEL-002
---

# Stage Model Handoff Design

The existing bbolt task is the only progress authority. A versioned model plan stores connection IDs for the three roles. Dispatch and plan updates serialize through Store.Update; dispatch captures the selected connection and plan version before starting. Running calls retain that identity while updates affect only later admissions.

Codex uses its existing ChatGPT-authenticated CODEX_HOME and an explicit OpenAI provider/model in a child process. Harness runs its own Node CLI with the existing DSH_HOME and deepseek-account provider. It does not run through a Codex DeepSeek profile. Browser requests cannot provide arbitrary executable paths, homes or credentials. No global configuration is modified. Availability means local configuration was found, not a promise of remote capacity.

Native invocations use full access with noninteractive approval, without modifying native global settings. Investigation and review retain read-only responsibilities through prompts and controller source comparisons; the operating system does not prevent their writes. These comparisons cover admitted project/candidate source, not arbitrary external side effects. A local Node preload sends console.info diagnostics to stderr while retaining strict native JSON events on stdout. A successful run requires an unambiguous session in the correct directory, a completed turn, one valid final report and a successful process exit. Credential values are read only for in-memory log redaction.

An immutable dispatch record and handoff input contain task goal, commands, source hash, required reads, audit and evidence locations. Results retain the actual session and output. Passing evidence seals these files alongside existing logs, snapshots and patch. Hashing the current transaction snapshot and entering awaiting acceptance happen in one write transaction, closing plan edits at the same boundary. Fresh read-only review and the persistent three-round budget remain mandatory.

The Chinese interface separates project setup, stage assignments, actual dispatch history and verification. Polling must not replace dirty selection controls. A plan version conflict asks users to reload the latest plan. A native folder picker and disposable cloned sample reduce typing without automatically committing user files.

New domain-mode tasks extend this design with frontend/backend writers and native model/effort catalogs; see [domain assignments](domain-assignment.md). Their review remains on Codex. Plans without a mode retain the legacy single-writer role contract and serialized dispatch history.

In 1.5.2, opening either new or future assignment controls triggers the existing authenticated native catalog refresh when a Codex provider has an unverified configured model or a directory error. The shared refresh preserves form input and dirty plans and remains serialized by the refresh control. Listed native efforts become selectable; absent native metadata still receives no invented effort capabilities. Running dispatches are unaffected.

Version 1.5.3 stores one projected observation per Codex connection in the controller's native-model-metadata.json cache. A matching connection ID, native home, provider, model and observation timestamp are required for fallback when the configured native entry is missing. Credentials and raw native catalog identity fields are excluded. Historical options use the codex_cache_history source and a visible historical label; native current metadata always wins. Atomic cache replacement preserves the latest observation across restarts. Corrupt or unavailable cache data cannot create capabilities. This metadata cache does not install skills/plugins or modify the native home.

## Shared-project assistant modes (local 1.17)

Optional assistant_mode metadata selects codex, deepseek or custom. Omission preserves legacy serialization and sealed evidence. Named modes resolve native catalog defaults before comparing active role selections; inactive roles do not block an assignment. Explicit modification modes require a Codex reviewer. Read-only tasks validate the selected investigator without imposing unused writer metadata. The existing versioned current/future plan APIs remain the only save path.

The composer and new-task cards expose named presets while the role drawer preserves custom assignments. Switching presets retains an assignment-only custom snapshot scoped to the current task or new-task form. Restoring it retains the latest authorized scope. Browser guards cover message dispatch, recovery, replan, backlog start, automatic continuation and acceptance whenever a selection is dirty, saving or conflicted. Read and cancellation actions remain available.

Before native dispatch or a read-only question, the controller follows the task's parent chain with cycle, ownership and bounded-record checks. It writes immutable SHA-256 JSON chunks and an index under Engine.Root/context, reusing exact content. Records contain original saved messages, replies, requirements, errors and attachment references. A small mutable task head supports status projection. Full records remain accessible through absolute paths; prompt excerpts are explicitly bounded reading aids, not substitutes for archived originals. Questions use a message cutoff so later queued requirements are not presented as active instructions.

Dispatch.ContextHash and TaskMessage.ContextHash anchor the expected archive before the runner starts. The controller verifies actual index, chunk and attachment bytes around execution; evidenceDigestSnapshot includes namespaced archive hashes before sealing. This validation performs no database reads inside the sealing write transaction. Blank legacy anchors skip the new checks, preserving old accepted checkpoints. Source/evidence verification remains mandatory after a provider switch.

GET tasks/{id}/context/status follows saved history and verifies an existing head without preparing archives or changing task records. The UI requests it only on details open or explicit refresh. Saved counts and archived snapshot counts are separate; a prepared snapshot does not imply the Agent read it or that later replies are included. Archive data is protected storage, included in usage but excluded from temporary cleanup. Cancellation, disk floor and explicit batch/message/byte/attachment limits stop preparation with an error while retaining original records. No native private sessions, credential homes, skills or plugins are copied.
