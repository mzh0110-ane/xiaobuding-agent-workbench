# Domain and model selection implementation

Status: complete. Implements REQ-RELIABLE-DOMAIN-001 in the existing reliable extension. Version 1.2.0 deployed to E:\小布丁 on 2026-10-19.

1. Extend the versioned plan with explicit domain participation, tool/model/effort selections and scope constraints. Preserve legacy plans.
2. Read native model catalogs and validate selections; pass immutable per-call overrides to separate native runners.
3. Implement sequential domain writers within one round, per-domain scope/evidence, legacy digest compatibility and recovery.
4. Update the Chinese setup/running UI and first-run sample to demonstrate frontend work.
5. Run focused regression/race, routing, static, browser and real native collaboration checks. Back up and deploy into E:\小布丁 after checks pass.

## ASCII UI preview

UI-01: Setup at /reliable, with project and goal already present.

```text
This task: [Frontend] [Backend] [Both]
Frontend    [Tool] [Model] [Effort: Auto]    [Scope settings]
Backend     [Tool] [Model] [Effort: Auto]    [Scope settings]
Planning    [Codex] [Model] [Effort: Auto]
Review      [Codex, fresh session] [Model] [Effort: Auto]
[Check and start] [Try frontend example] [Refresh models]
```

Only participating domains appear. On phones, each assignment stacks its labeled controls. Running tasks lock mode/scope, preserve unsaved choices and show current actual execution separately from future choices. Catalog errors stay visible beside model refresh. These structures implement AC-RELIABLE-DOMAIN-001.1, .2 and .4; spacing is illustrative.

See task-01-domain-models.md.

## Completed validation

The race-enabled reliable package run passed 88 test/subtest items with no failures; real invocation tests were opt-in. The backend route test, vet and golangci-lint passed. Four state tests, six domain browser scenarios and five legacy browser scenarios passed. A real native sample used four distinct sessions (Codex planning, Harness frontend, Codex backend, Codex review), passed in round one, and passed eight checks after applying only to the disposable sample. Actual candidate and deployed pages passed desktop/mobile/model-selection checks.

Deployment preserved all five production task states, rounds and evidence seals. Windows PowerShell 5.1 cold start and repeated launch used the same process. A read-only check of the copied closed task database verified the one pre-existing awaiting-acceptance candidate against its original evidence without applying its patch. Documentation catalogs validated 350 decisions and 1348 specifications; no reliable-collaboration findings were added to the upstream spec lint backlog.

Evidence: E:\小布丁\data\validation\upgrade-1.2.0. Rollback backup: E:\小布丁\backups\upgrade-1.2.0-20261019-142745. Existing user tasks were not accepted, resumed or cancelled during deployment.
