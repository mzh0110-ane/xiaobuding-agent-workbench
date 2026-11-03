# Model handoff work order

Implement REQ-RELIABLE-MODEL-001 in internal/reliable only, plus local distribution documentation and settings. Native Kandev tasks remain outside the extension.

UI structure:
```text
Project history | New project: folder + goal
                | Stage assignments: audit / implementation / review
                | Check and start | One-click sample
                | Active task: actual executor + stage progress
                | Edit subsequent assignments
                | Verification, review, evidence disclosure
```

Required checks: affected package race tests, adapter tests, go vet, package golangci-lint, browser desktop/mobile and persisted selection reload, and one real Codex/DeepSeek sample. No commits, pushes, project application or model calls outside disposable validation projects are part of development.
