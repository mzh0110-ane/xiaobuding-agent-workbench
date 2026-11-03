---
system: reliable-collaboration
title: Reliable Collaboration
status: active
specification_version: 1
migration: complete
---

# Reliable Collaboration

This local extension owns evidence-based collaboration tasks, managed local recovery points, real-check bootstrap, role isolation, bounded rework, and application of verified changes. It does not own ordinary Kandev task transitions, remote execution, or deployment.

The project-centered workbench extends this boundary with conditional navigation,
editable backlog and handoff views, actionable recovery, bounded candidate
inspection/preview, and explicit storage pressure controls. Its contracts are in
[workbench requirements](requirements/workbench.md) and the
[workbench design](system-design/workbench.md). The staged delivery record is the
[workbench implementation plan](../../plans/reliable-workbench/plan.md).

- [Requirements](requirements/local-loop.md)
- [System design](system-design/local-loop.md)
- [Stage handoff requirements](requirements/model-handoff.md)
- [Native tool connections and stage handoff](system-design/model-handoff.md)
- [Frontend/backend assignments and native models](requirements/domain-assignment.md)
- [Sequential domain execution and catalogs](system-design/domain-assignment.md)
- [Automatic local project requirements](requirements/local-snapshots.md)
- [Managed recovery points and real-check bootstrap](system-design/local-snapshots.md)
- [Desktop workspace and task organization](requirements/desktop-workspace.md)
- [Task archive, folder actions, and workspace projection](system-design/desktop-workspace.md)
- [Long-project knowledge, recovery and storage requirements](requirements/long-projects.md)
- [Project ledger, writer recovery and storage boundaries](system-design/long-projects.md)
- [Decision](../../decisions/2026-10-18-local-reliable-collaboration.md)
