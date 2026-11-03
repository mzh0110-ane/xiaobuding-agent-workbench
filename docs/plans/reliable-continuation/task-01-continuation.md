---
status: completed
system: reliable-collaboration
---

# Task 01: conversation and controlled requirement continuation

## Interface

```text
Task summary and current state
Conversation
  You: question or requirement [queued / reading / answered / applied to round]
  Agent: source-based response and expandable facts / unknowns / citations
  [Continue asking (read-only) | Supplement modification requirements]
  [Describe the next question or requirement.......................]
  Delivery explanation                             [Send]
Live activity
Results, evidence, native model assignments
```

## Acceptance and validation

- HTTP tests prove multiple messages, idempotency, queued delivery, failed delivery, restart preservation and archived-task guards.
- A gated runner proves a mid-review requirement cannot pass under the old goal and reaches both writers and fresh review within the original round budget.
- Read-only follow-up does not change a candidate's seal, implementation round, or source. Verify acceptance evidence remains valid afterward.
- Browser checks cover posting, per-task drafts, stale completion isolation, reload, errors, Chinese input and mobile overflow.
- Run the owning Go package with race detection, relevant browser scenarios, Go vet and owning lint; preserve existing task history and sealed evidence during deployment.

## Verification

`go test -trimpath -race ./internal/reliable -count=1 -v` passed 109 top-level tests (seven opt-in skips). `scripts/reliable/desktop-browser-check.cjs` passed 29 scenarios. The opt-in `TestLiveActivityBrowserWithProtocolFixture` using `scripts/reliable/conversation-integration.cjs` passed against the actual HTTP/store/runner pipeline; the fixture is not a live model call. Go vet, owning golangci-lint and whitespace checks passed. Missing message delivery was first reproduced through HTTP; stale-goal review and cancellation failures were reproduced before their fixes.

