# Domain and model work order

Implement REQ-RELIABLE-DOMAIN-001 and its six acceptance criteria. Owning design: ../../specs/reliable-collaboration/system-design/domain-assignment.md.

Outcome: real frontend/backend responsibility with independent native model/effort selection, one worktree, one round budget and explicit evidence. Scope is internal/reliable, localized distribution docs/settings and validation tooling. Do not modify upstream ordinary tasks, account configuration or user source projects.

## ASCII UI preview

UI-01 follows the [combined preview](plan.md#ascii-ui-preview): mode selection, enabled domain cards, each card's Tool / Model / Effort, then planning and review; controls stack on phones. Running cards label edits as future dispatch changes and lock participation/scope.

## Verification

- go test -race -trimpath -p 4 ./internal/reliable -count=1
- go test -race -trimpath -p 4 ./internal/backendapp -run TestReliableRoutes -count=1
- go vet ./internal/reliable
- golangci-lint run ./internal/reliable --new=false --timeout=3m
- node --test internal/reliable/page-state.test.cjs
- Focused headless browser checks for linked tool/model/effort inputs, persisted edits, work modes, catalog failure and narrow layout.
- Opt-in real native frontend/backend sample with Codex planning/review and Harness frontend implementation, plus applied checks in a disposable project.

Test the historical dispatch serialization contract, one-round dual writes, scope violations, second-writer failure, plan switching during dispatch, and rejection of missing evidence.

Completed on 2026-10-19. All listed checks passed for the extension; actual native proof task vdk2kfb23iyjwghxhy4xjlqqx3 used four unique sessions and eight applied checks. See the plan for production preservation and the separate read-only legacy candidate verification.
