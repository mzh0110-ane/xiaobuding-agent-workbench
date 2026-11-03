# Recovery and storage contracts

## Recovery guidance

`RecoveryGuidanceFor(task)` is a read-only projection with `code`, localized
`summary`, `next_action`, and the unchanged raw `detail`. It never expands
scope, starts a task, accepts a result, or resets a quality round.

Actions are `resume`, `replan`, `new_batch`, `login`, `checks`, or `none`.
Superseded and archived batches remain historical views. The UI must use
existing task eligibility checks before offering a corresponding mutation;
guidance is not an authorization grant.

## Planning runtime retries

The read-only audit invocation has at most three operational attempts within
one shared 30-minute window. A complete 30-minute timeout pauses instead of
opening another 30-minute window. The implementation-round counter is unchanged.
Existing dispatch input, start, and result evidence retain every attempt.

Only a failed auditor dispatch with an operational error may retry. Before the
next attempt, the controller checks owned-process cleanup, task state, frozen
requirement and model-plan revisions, exact candidate source identity, Git
history, and resource fingerprint. Changed input, source mutation, invalid
schema, scope violations, and missing evidence never trigger automatic retries.
Reports still pass the existing strict decode, read acknowledgement, scope,
and domain checks outside the invocation retry loop.

## Storage budgets

`GET /api/v1/reliable/storage/budget` explicitly scans only cache and temporary
categories with the existing bounded, link-safe scanner. It also probes actual
available disk bytes. It does not delete data. `POST` requires the browser
capability, an expected policy revision, and all three limits in bytes.

Policy lives in `reliable-storage-budget-v1` inside the existing Bbolt database.
Defaults are 8 GiB cache, 2 GiB temporary storage, and 2 GiB minimum free space.
Each configurable amount is 64 MiB through 1 TiB. Cache and temporary values
are advisory budgets, explicitly reported as `limit_mode: advisory`; this is
not a producer-enforced cache quota.

`CheckStorageBudget(ctx)` only reads policy and probes disk free space. Call it
before new worktree or preview allocations. Do not run a historical storage
scan per dispatch, status poll, or source read. Space pressure must not prevent
editing settings, reading evidence, or inspecting retained work. Unknown disk
space fails new allocation with a separate diagnostic instead of claiming
space is sufficient.

Existing cleanup remains the sole deletion path: an explicit preview and human
confirmation, old marker-owned inactive temporary directories, and all existing
identity checks. Cache, worktrees, evidence, attachments, credentials, native
account homes, and user project files are never deleted by budget handling.
