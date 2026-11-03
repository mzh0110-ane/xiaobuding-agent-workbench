package reliable

import "testing"

func TestRequirementsStageRecoveryGuidanceExplainsReplanning(t *testing.T) {
	for _, reason := range []string{
		"stage_resume_invalidated: frozen stage inputs changed",
		"stage_resume_invalidated: requirements or the frozen audit changed",
	} {
		task := &Task{State: stateBlocked, Round: 1, Reason: reason}
		guidance := RecoveryGuidanceFor(task)
		if guidance.Code != "stage_inputs_updated" || guidance.NextAction != "replan" || guidance.Detail != reason || guidance.Summary == "" {
			t.Fatalf("stale stage recovery did not expose a guarded replan action: %+v", guidance)
		}
		if task.Reason != reason || task.Round != 1 || task.State != stateBlocked {
			t.Fatal("recovery guidance mutated the retained task")
		}
	}
}
