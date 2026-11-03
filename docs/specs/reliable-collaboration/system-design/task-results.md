---
status: draft
system: reliable-collaboration
requirements:
  - REQ-RELIABLE-RESULTS-001
  - REQ-RELIABLE-PREVIEW-001
---

# Task Results and Local Preview Design

## Ownership and existing contracts

Reliable collaboration owns result provenance, read-only projections and local
preview lifecycle. Ordinary Kandev task sessions and the upstream trusted
Browser-panel preview remain outside this extension. Reuse existing folder
actions, tokens, redaction, process ownership and UI primitives. Preserve the
[local loop](local-loop.md), [workspace](desktop-workspace.md),
[resource profile](large-resource-projects.md) and
[continuation](desktop-workspace.md) contracts.

| Requirement | Sections |
| --- | --- |
| REQ-RELIABLE-RESULTS-001 | Projection, provenance, viewer, presentation |
| REQ-RELIABLE-PREVIEW-001 | Preview recipes, preparation, authority, lifecycle |

`page-proof.js: TaskProof.render` currently links the patch only if `Task.Review`
exists and links only `check-1.log`. `stages.go` saves implementation summary,
documentation impact and residual risks; domain mode saves `domainOutcome`
objects with `Before`/`After` snapshots. These reports are not summarized in the
main task card. `Store.StartRound` clears `Checks` and `Review`, so ordinary
round admission already prevents reusing those task-level fields across rounds.
Historical selectors, raw reports, interrupted publication and individual logs
still need exact provenance rather than guessed current-round paths.

## Result projection and persistence

Add `GET /api/v1/reliable/tasks/{id}/result` with a versioned, bounded projection
implemented in proposed `result_projection.go` and `result_http.go`:

- `task_id`, `result_id`, `round`, requirement revision, dispatch/domain IDs,
  recorded source identity, task state and `target_kind`;
- `evidence_status`: sealed-at-generation, validated-stage, raw-partial,
  historical, missing, altered or ambiguous, with explicit explanations;
- attributed summaries, documentation impact and residual risks;
- paged changed-file descriptors, actual patch reference and counts/limitations;
- check descriptors with command, real exit/verification problem, round and
  each exact log reference; raw orphan logs remain incomplete evidence;
- review and decision references with accepted controller provenance, or an
  explicit unavailable status plus separately labelled raw output;
- allowlisted artifact descriptors and existing folder action availability;
- preview eligibility/setup status, not a running process created by GET.

A result ID binds task, round/revision, selected source/report references and
their digests. It is not a new acceptance seal. Start with task state, persisted
stage outputs and exact known evidence paths. Do not treat `Dispatch.Status ==
completed` as proof of stage validation: `completeDispatch` happens earlier.
Future stage completion writes a bounded projection index atomically outside
sealed evidence. Store it under `Root/results/{task}/{result}.json`; index
publication follows stage validation. It records artifact digests and expected
source file fingerprints. It is regenerable metadata, never authority for
acceptance. Do not add it to old seal inputs or mutate old artifacts.

Historical tasks use a lazy bounded adapter. Legacy single-writer reports are
objects; domain aggregate reports are arrays, with separate domain evidence.
Bind the first applicable `Before` to the last validated `After`; use
`changedFiles` semantics. Saved patch parsing can provide a file list when
snapshot provenance is absent. Do not count entries merely from model prose.
If old formats cannot establish revision/source association, mark them
historical/unbound and offer the raw files, not a current pass. The current
cancelled round has a patch, two domain reports and a raw failed-dispatch review;
it can expose partial artifacts but no acceptance action.

Projection GET reads only bounded persisted metadata/evidence, never a source
tree scan or `TakeSnapshot`, model invocation, check or preview launch. Load
selected round lazily, page file descriptors (default 100, maximum 500), cap
summary metadata at 1 MiB and reject bounded-read/parse errors explicitly.
Do not read the 22 MB resource library for every card request. Missing or
inconsistent metadata yields partial/unavailable detail instead of suppressing
the entire task. Return a generation token; the client rejects stale requests.

"Passed when generated" is distinct from current applicability. Result viewing
does not freshly certify the live candidate or original. Existing acceptance
performs full current integrity validation. Changed metadata/digests invalidate
projection reuse; do not trust timestamps alone. Preserve snapshots and saved
outcomes from earlier rounds, showing them in a labelled selector.

## File and evidence viewer

Add `GET .../result/files` and `GET .../result/artifacts/{artifact_id}` for a
specific result ID, with server-resolved descriptors rather than client paths.
Source defaults to text/diff; never execute HTML as the panel's document.
Existing `/evidence?path=` remains `text/plain` with its current traversal rules.

Read a selected file only on explicit view. Constrain it to the associated
result/evidence or candidate root; reject reparse points, aliases, traversal,
private/excluded paths, non-regular files and files above the existing per-file
ceiling. Compare its bytes against the saved expected fingerprint. If source
has changed or a historical version is absent, show the retained patch or an
unavailable reason, not current bytes labelled as historical. Binary entries
show type/size/download availability without fabricated text diff. Never render
diff text, summaries, paths or Markdown through unsanitized `innerHTML`.

Support bounded text/Markdown-as-text, PNG/JPEG/WebP images and static HTML via
the explicit preview service. SVG defaults to text/download because it can
contain active content. Downloads use attachment disposition and `nosniff`.
No arbitrary OS file-opening endpoint is needed. Existing POST folder actions
continue resolving project/candidate/evidence paths on the server.

## Presentation

Put a result card immediately below the existing status header, before the
conversation, with one summary, concise evidence status and primary preview
action when eligible. Keep the Agent drawer, portrait spacing, background and
theme untouched. A pending candidate retains the separate existing accept
action; viewing never accepts it. Use a collapsed file list and lazy viewer.

Desktop opens a bounded right-side result drawer with file navigation and
text/diff tabs. Phone uses a full-width dialog with back/close and its own file
selector, keeping long lines inside the code pane rather than scrolling the
page horizontally. Preserve open/selected result and scroll state through
polling. Use task/result generations, request cancellation, explicit empty/
busy/error/retry states, accessible labels, focus trapping/restoration, Escape,
and status text in addition to color. Touch targets are at least 44 px.

Interim domain limitations are expandable with domain/time attribution.
Latest accepted decision and current blockers get priority; old frontend
"backend not implemented yet" claims are not promoted to a final result.
The result card is a projection, not a fabricated agent chat message.

## Preview recipes and preparation

### Supported recipes

An additive typed preview configuration selects one of:

| Type | Valid inputs | Execution | Fallback |
| --- | --- | --- | --- |
| Static | Existing relative document root + HTML entry | Controller static server on dedicated loopback origin | Unsupported dynamic templates/builds show setup needed |
| Flask | Existing selected Python interpreter, relative module/file and existing app attribute, selected local health path | Fixed argv: Python isolated mode plus controller-owned WSGI bootstrap and private descriptor | Missing dependency/import/startup/data gives explicit failed status |

Discovery only reads bounded existing entry metadata, never imports the project
or runs a command on GET. One unambiguous supported entry can offer one-click
preview; multiple entries require a one-time typed setup. Store this operational
configuration outside frozen task evidence and bind it to task/project source
identity. Do not accept arbitrary shell strings, launchers, model-suggested
URLs or tunnel options. Executable selection uses a native file picker and
server validation, not an unchecked web request absolute-path field.

For a Flask project, source documentation may identify an application entry point, templates, static files, and a health endpoint. Treat those as configuration evidence, not proof that the application starts successfully; avoid project launchers that can start tunnels or manipulate shared processes.

### Disposable project copy

`POST .../preview` starts preparation for an explicit result ID. Reserve the
task's execution mutex while selecting/verifying a source generation, excluding
concurrent implementation, hydration and acceptance. Allow preview only for a
resting state with a concrete source identity; archive inspection is allowed,
but starting runtime requires restoring first. A cancelled task can explicitly
preview its partial candidate, labelled unverified, without resuming it.

Create a controller-owned disposable copy under `Root/previews`, with no `.git`,
native homes, secret files, candidate sidecars or shared writable links. Copy
bounded regular source/output files from the exact selected generation and
verify their saved fingerprints. Before acceptance the source is the candidate;
after application offer the applied original only after matching its expected
identity, otherwise retain the historical candidate option and explain drift.
Data/caches/logs/outputs stay under the disposable runtime root. Do not merge
preview output back into a project.

Deferred resource identities come from the frozen library. Hydrate only declared
inputs and requested asset paths into this separate copy; use existing path,
Git-object, per-file and batch-budget validation, with an independent preview
ledger. Do not alter the candidate's availability ledger or evidence. Missing
inputs fail or show a partial-preview warning, never a false working website.
Resource staging and copy progress must be cancellable and visible. Do not
pre-copy all 150081 resources. No runtime guesses may mutate a frozen resource.

## Flask bootstrap

Do not launch the stock Flask CLI as the public preview server: it cannot
enforce this extension's capability and origin boundary. Embed a small
controller-owned `preview-bootstrap.py` outside the disposable project root.
The fixed command is the selected Python interpreter with `-I`, the verified
bootstrap path and `--descriptor` naming a controller-created private JSON file.
The descriptor binds generation, disposable root, module/file, app attribute,
loopback port, health path and per-generation capability. It is not client JSON
passed unchecked to a shell. Support an existing Flask application object first;
factory calls with arbitrary arguments require a later typed recipe.

The bootstrap loads only the confirmed module from the disposable root, wraps
its WSGI app in controller middleware, and starts `werkzeug.serving.run_simple`
on the chosen `127.0.0.1` port with reloader, debugger and interactive evaluation
disabled. These are supported local-serving options in the
[Werkzeug serving contract](https://werkzeug.palletsprojects.com/en/stable/serving/).
Keep existing package versions; no installation is required by the bootstrap.
Its middleware enforces the exact Host, rejects inappropriate Origin and
cross-site requests, denies private/reserved path prefixes, and checks the
preview cookie/capability before calling the project application. Static preview
uses equivalent checks in the controller's dedicated listener.

The only unauthenticated content is a fixed bootstrap handoff page at
`/_xbd_preview/start`. A one-use fragment token is consumed by that page's own
script, removed from browser history, then POSTed to the same preview origin;
the middleware exchanges it for a short-lived, HttpOnly, SameSite=Strict,
generation-specific cookie and redirects to the selected entry. Relative
assets use that cookie. Tokens never enter query strings, referrers, application
environment or project request headers; middleware strips its internal headers
and cookie before forwarding to the app. Redact handoff tokens from logs and
revoke them when the preview stops. This protects controller authority from
routine project requests, not against malicious project code reading local
process memory/files as the same Windows user.

An internal `/_xbd_preview/health` route requires a distinct controller-only
probe nonce and returns the bound generation plus bootstrap instance identity.
Only after this matches a live owned process does the controller probe the
configured application health path through the authenticated middleware with
redirects disabled. Ordinary `/healthz` success alone proves neither ownership
nor the preview boundary. Check debug/reload disabled, Host/cross-site rejection,
capability exchange/expiry and asset requests in real local fixture tests.

## Preview authority and lifecycle

Implement proposed `PreviewManager` owned by `Engine`, closed by `Engine.Close`
through `internal/backendapp/reliable_routes.go`'s existing shutdown hook.
Routes are additive; all POSTs require existing local/capability authorization:

- GET `tasks/{id}/preview/config`: bounded typed setup/discovery status;
- POST `tasks/{id}/preview/pick-interpreter`: native picker with no client path;
  return an opaque interpreter ID and display path after server validation;
- POST `tasks/{id}/preview/config`: expected config revision, kind, interpreter
  ID, relative module/file and app attribute or static root/entry, and relative
  health path. Validate syntax/containment without importing/running the app.
  Persist outside evidence with source eligibility; return revision conflicts;
- POST `tasks/{id}/preview`: result ID and config revision, never a command;
- GET `tasks/{id}/preview`: bounded status/log cursor, never starts work;
- POST `tasks/{id}/preview/stop`: stop only the owned generation;
- POST `tasks/{id}/preview/heartbeat`: exact active preview generation; stale
  generations fail without touching the current preview or task evidence;
- separate loopback preview origin serves static content or the owned Flask
  port. It is not a generic reverse proxy on the reliable authority origin.

Bind exact loopback addresses; validate Host/Origin for the preview server,
disable directory listings, confine static paths, and do not expose `.git`,
private files or controller API routes. Use a random preview capability scoped
to its generation, separate from `X-XBD-Token`, passed via a non-referrer
handoff; redact it from logs. Do not share a controller token or window opener.
Open the returned URL using an explicit user action with `noopener,noreferrer`.
If opening is blocked, retain a copyable local URL and ready status.

Runtime code is trusted project code under the user's Windows account. An
independent copy and stripped environment prevent routine write pollution but
are not an OS sandbox against absolute paths/network access. Clearly state
this boundary in preview setup, without a repeated approval questionnaire.
Strip native login locations and sensitive environment values; do not copy
`.env` or load Codex/Harness credentials. Features requiring API keys can remain
unavailable in preview; do not perform paid network calls as validation.

Allow one owned preview per task and at most two globally. Preparation and
startup run asynchronously after POST. State machine: preparing -> starting ->
ready -> stopped/expired; any phase may fail with a diagnostic. Reuse a live
same-generation preview, stop an old generation before replacement. Use a
controller-selected available port with bounded collision retries, bind
verification and liveness of the owned process. Flask readiness requires the
expected app response on its configured health path with redirects disabled
and body/time limits; a different process listening on the port is a failure.
Startup deadline is 30 seconds after staging; log actual staging time separately.
An unexpected process exit after readiness updates status to failed.

Use the Windows Job Object owner and bounded drain/stop pattern, never kill a
process by an old PID alone. A visible selected task with a live preview sends
one explicit heartbeat per minute; hiding the tab or selecting another task
stops those heartbeats, and returning resumes only if that generation is still
alive. Ordinary status/log GET does not extend lifetime. Expire after thirty
minutes without a UI heartbeat and at two hours maximum. Stop on service shutdown;
on restart classify saved preview records as stopped and do not execute them.
After stopping, remove only verified owned temporary directories, retaining a
bounded redacted diagnostic log. A failed cleanup preserves files and reports
the problem. Preview records/logs remain outside evidence seals.

Application first reserves task execution, stops any owned preview and then
runs the existing full acceptance validation. Preview never grants acceptance
or consumes a round. No automatically started process is allowed to mutate the
candidate during source selection or publication.

## Related decisions and plans

- [Local preview provenance boundary](../../../decisions/2026-10-20-reliable-result-preview-boundary.md)
- [Upstream trusted browser preview](../../../decisions/2026-09-05-trusted-browser-html-preview.md), a separate feature without this extension's origin guarantee.
- [Execution efficiency and task results](../../../plans/reliable-speed-results/plan.md)
