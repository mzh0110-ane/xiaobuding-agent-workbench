# Local project admission

Status: complete. Implements AC-RELIABLE-LOOP-001.7 in the existing reliable controller. Patch 1.2.1 deployed locally on 2026-10-19.

A tools collection may contain multiple projects and local configuration. Select a specific source directory for modification, and use read-only inspection for a broader directory.

Raise the fixed limits to 30000 entries and 256 MiB total while preserving the 16 MiB per-file ceiling, full coverage and the existing hash/serialization contracts. Do not initialize the tools directory, exclude tracked source to make a snapshot fit, upload to GitHub, or start an unspecified agent task.

Verification: reproduce the old failure; exercise file-count and byte boundaries, source drift and patch protection with race checking; snapshot the actual repository; run vet and lint; build and deploy the small patch; save the tested local source using the repository commit workflow. Preserve task state and record the final local commit and project check instructions.

See task-01-snapshot-admission.md.

## Results

The original ceilings failed both boundary regressions and the actual project admission check. After the fix, ten focused tests passed with race checking, vet passed and golangci-lint reported zero issues. The actual project included all 23140 entries in a read-only snapshot, completed in 232.33 seconds. The first combined race run exceeded its five-minute test deadline during filesystem reads; its log is retained separately, and the passing actual-source run used a longer deadline without race instrumentation.

The exact prefilled source check passed the complete reliable package in 87.688 seconds, followed by four JavaScript state tests. Documentation catalogs validated 350 decisions and 1348 specifications; the extension has zero spec findings and the upstream baseline remains 107. Deployment preserved all five task states, rounds and evidence seals. Backup: E:\小布丁\backups\upgrade-1.2.1-20261019-150034.

The tested 1.2 source was saved in local commit 865cbf950a1c4f2783b8ef2f66bf48169f8eab6e. The patch follows in a separate normal local commit; its receipt is stored in E:\小布丁\data\validation\upgrade-1.2.1. No remote upload or unspecified agent task is part of this preparation. The tools directory remains outside Git.
