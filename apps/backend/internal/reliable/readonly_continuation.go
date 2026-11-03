package reliable

// Read-only presets select a planner; a later modification batch activates its writers.
func modificationBatchPlan(parent *Task) ModelPlan {
	plan := continuationPlan(parent)
	if !parent.ReadOnly {
		return plan
	}
	switch plan.AssistantMode {
	case connectionCodex, connectionHarness, connectionZcode:
	default:
		return plan
	}
	if plan.Auditor != plan.AssistantMode {
		return plan
	}
	id, model, effort := plan.selection(roleAuditor, "")
	plan.Implementer, plan.ImplementerModel, plan.ImplementerEffort = id, model, effort
	plan.Frontend, plan.FrontendModel, plan.FrontendEffort = id, model, effort
	plan.Backend, plan.BackendModel, plan.BackendEffort = id, model, effort
	if plan.Reviewer != connectionCodex {
		plan.ReviewerModel, plan.ReviewerEffort = "", ""
	}
	plan.Reviewer = connectionCodex
	if id == connectionZcode {
		plan.Auditor = connectionCodex
		plan.AuditorModel, plan.AuditorEffort = "", ""
	}
	return plan
}
