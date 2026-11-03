# ADR-2026-10-19-managed-local-project-snapshots: Managed local recovery points before source baseline

**Status:** accepted
**Date:** 2026-10-19
**Area:** workflow

## Context

Requiring Git initialization and commits blocks novice ordinary-folder use. Automatically committing every Git repository would instead consume user staging and affect repositories Xiaobuding does not own. Empty folders lack checks; a placeholder command would undermine the evidence contract.

## Decision

Reserve a canonical project before original-folder preparation. Create local history only for ordinary roots and subsequently manage roots carrying a matching canonical ownership marker. Protect manual staging and retain clean-root admission for external repositories. Establish recovery before freezing the worktree contract; once a worktree exists, resume never commits the original again. Do not configure remotes, push or change global identity.

Allow new automatic-check tasks to establish real verification in the bounded implementation loop. Freeze and seal discovered checks before review/application. Missing checks consumes existing rework rather than creating a synthetic pass. The reliable-collaboration system owns both contracts.

## Consequences

Ordinary folders become usable without Git commands while external ownership and original-source drift remain protected. Preparation needs recovery state, cancellation, explicit path exclusions and failures. This policy is not comprehensive secret-content scanning. Empty projects can begin planning but cannot reach acceptance without executable verification. Local history covers admitted source rather than every file or entire tool roots. Failed preparation may retain controller-created metadata; source contents and pre-existing user history/index remain protected. An index left after staging but before a pending checkpoint record requires maintainer recovery instead of an unsafe automatic reset.

## Alternatives Considered

- Manual Git preparation retains a novice-only blocker.
- Adopting every Git root can alter another workflow's staging and history.
- Initializing a parent repository from a subdirectory can include unrelated files.
- Committing again on resume can hide drift after a candidate was verified.
- A placeholder successful check weakens review and acceptance.
