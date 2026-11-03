# ADR-2026-10-18-local-reliable-collaboration: Controller-owned local collaboration

**Status:** accepted
**Date:** 2026-10-18
**Area:** workflow

## Context

This local distribution must reuse an existing Windows Codex/DeepSeek installation and preserve evidence, independent review, bounded rework, and explicit human acceptance. Native review actions do not themselves block every workflow transition. A completion signal is an agent claim rather than verification evidence.

## Decision

Add an opt-in, backend-hosted collaboration entry at `/reliable`. A dedicated controller owns its tasks and durable state. These tasks are separate from ordinary Kandev tasks, so ordinary task movement, completion signals, and review actions cannot grant acceptance to them. Three fresh CLI role sessions run sequentially: auditor, implementer, and reviewer. Real verification processes run between implementation and review.

Three rounds means one initial implementation and at most two reworks. Persist admission before launching an implementer. Stop on missing evidence, invalid reports, source drift, or unsupported scope. Apply changes to the original clean project only after an explicit browser action and fresh source/patch verification.

## Consequences

The installed CLI reads its existing `CODEX_HOME`; no credentials are copied into Kandev settings. The integration avoids an additional agent bridge and preserves upstream Kandev behavior. The custom entry must be used to receive these guarantees; native tasks are not transparently converted. The local panel is intentionally Simplified Chinese.

Read acknowledgements and model judgments remain claims. Source fingerprints, process output and session identities provide independent structural evidence, not a guarantee of semantic correctness or tamper resistance against a privileged local user. Windows verification commands execute project code under the local user account with credential environment variables removed.

## Alternatives Considered

- Prompt and workflow configuration alone: cannot enforce durable round limits and passing verification.
- Reusing ordinary task state as acceptance authority: would require invasive changes across existing execution and transition paths.
- External standalone application: adds another server and an independent launch surface. Hosting the controller in Kandev gives one local origin and one lifecycle.

## Specifications

- [Reliable collaboration](../specs/reliable-collaboration/README.md)
