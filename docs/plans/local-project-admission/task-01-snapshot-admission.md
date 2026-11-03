# Snapshot admission work order

Implement AC-RELIABLE-LOOP-001.7 from [the owning requirement](../../specs/reliable-collaboration/requirements/local-loop.md). Design: [source coverage and limits](../../specs/reliable-collaboration/system-design/local-loop.md#security-and-limits).

The affected boundary is TakeSnapshot and fileFingerprint. The owning regression file is workspace_limits_test.go: accept a source manifest above the former count limit, reject 30001 entries, accept totals above the former byte limit through the new exact ceiling, reject overflow, and retain the per-file ceiling. An opt-in read-only snapshot validates the user's actual repository. No new UI surface or task state is introduced.

Run the affected reliable tests with race checking, vet and golangci-lint, and preserve red/green logs. Deploy only the validated executable, preserve all existing task states/rounds/seals, and keep a rollback backup. Commit only the existing extension scope and these related documents after checks pass.

Completed. See the parent plan for boundary, real-source, prefilled check and preservation results. Commit receipts and validation logs remain outside the source repository.
