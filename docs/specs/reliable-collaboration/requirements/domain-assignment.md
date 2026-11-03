---
status: active
system: reliable-collaboration
created: 2026-10-19
owners:
  - local-distribution
---

# Domain Assignment Requirements

### REQ-RELIABLE-DOMAIN-001: Native tools assigned by work domain

- **AC-RELIABLE-DOMAIN-001.1:** A new task shall select frontend, backend or both. Frontend shall default to native DeepSeek Harness; backend and planning shall default to Codex with ChatGPT login. Independent review in domain-mode tasks shall stay on Codex with a fresh session.
- **AC-RELIABLE-DOMAIN-001.2:** Each enabled assignment shall independently select tool, model and reasoning effort from its native catalog, with an explicit automatic effort option. Unsupported selections shall fail rather than fall back.
- **AC-RELIABLE-DOMAIN-001.3:** Both domains shall write sequentially to one worktree within one admitted round. Each writer shall respect its audited domain scope; a scope violation or failed writer shall prevent verification and review.
- **AC-RELIABLE-DOMAIN-001.4:** Running updates shall affect subsequent dispatches only. Domain participation and user scope shall remain fixed for the task. Actual domain, model, effort, source and fresh session shall be retained in evidence.
- **AC-RELIABLE-DOMAIN-001.5:** The persistent limit shall remain three total rounds per task, including both writers, with independent review and human acceptance. Legacy tasks and already sealed evidence shall remain valid.
- **AC-RELIABLE-DOMAIN-001.6:** Catalog refresh shall reuse existing account data without exposing credentials or changing global tool configuration. Failures shall distinguish configured defaults from catalog-verified options.

The controller detects out-of-scope worktree writes before delivery. This is not a claim that all filesystem writes are prevented by operating-system permissions.
