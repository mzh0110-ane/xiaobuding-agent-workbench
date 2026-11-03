# ADR-2026-10-20-reliable-result-preview-boundary: Evidence-Bound Local Result Previews

**Status:** accepted
**Date:** 2026-10-20
**Area:** workflow

## Context

The local reliable-collaboration extension retains source snapshots, patches,
reports and checks but does not currently provide a result artifact card or
project preview lifecycle. A raw model report can exist after dispatch failure,
and a cancelled task deliberately retains its candidate. Those artifacts must
be inspectable without being promoted to accepted output.

Upstream Kandev's Browser panel accepts trusted same-origin workspace execution
under its own task/session model. The standalone `/reliable` panel has a
different browser capability and sealed acceptance contract. Its HTML evidence
endpoint deliberately serves plain text. Reusing that origin for project HTML
would expose controller authority and conflate operational preview with proof.

## Decision

For the proposed local extension, result metadata is a regenerable read-only
projection of exact task/round/source/evidence identities. It is never authority
for acceptance. Raw, historical and partial artifacts remain visibly distinct
from controller-validated stage results and applied changes.

Project execution is an explicit task-bound preview action, using a disposable
copy and separate loopback origin with an owned lifetime. Static HTML and a
typed Flask recipe are the initial supported cases. Runtime configuration,
process records, preview outputs and diagnostics stay outside sealed evidence.
No arbitrary launcher/proxy or implicit project execution is provided by result
GET requests. Existing source, evidence and human-acceptance gates remain the
sole path for application.

This selects the architecture for a draft design package, not authorization or
evidence of completed implementation. It does not amend upstream trusted
Browser-panel behavior. Disposable copies reduce routine mutation but do not
isolate malicious local project code from the user's Windows account.

## Consequences

Result cards can display existing artifacts without additional model calls.
Preview needs bounded source/resource staging, typed launch configuration,
process cleanup and explicit missing-dependency states. Preview startup or a
successful HTTP response does not prove feature correctness. Unsupported
frameworks keep report/file/folder access and need a later typed recipe.

## Alternatives Considered

- Launch directly in the reviewed candidate: simpler, but requests, imports,
  caches and database writes can pollute or invalidate the reviewed result.
- Serve project HTML under `/reliable`: smaller routing surface, but exposes
  controller-origin authority and changes the plain-text evidence boundary.
- Accept arbitrary commands from an agent report: flexible, but confuses model
  output with operational authority and can silently start tunnels/installers.
- Support report/file links only: useful first slice, but does not let a novice
  inspect the actual supported website after a task.
- Add universal OS isolation/framework support now: much larger scope than a
  local result viewer; unnecessary to deliver typed trusted-project previews.
