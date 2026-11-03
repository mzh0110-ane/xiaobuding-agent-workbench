package reliable

import (
	"strings"
	"testing"
)

// @covers AC-RELIABLE-RENEWAL-001.5
func TestExhaustedParentActiveModelSettingsRedirectToSuccessor(t *testing.T) {
	e, parent := preparedExhaustedTask(t, &fixtureAgent{})
	parent, err := e.Store.Update(parent.ID, func(saved *Task) error {
		saved.State = stateBlocked
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	plan := e.defaultPlan()
	before, err := e.UpdateModelPlan(parent.ID, parent.PlanRevision, plan)
	if err != nil {
		t.Fatalf("blocked-parent model update positive control failed: %v", err)
	}
	child := claimExhaustedChild(t, e, before, "active-settings-successor")
	if _, err = e.UpdateModelPlan(parent.ID, before.PlanRevision, plan); err == nil || !strings.Contains(err.Error(), "batch_successor_exists") {
		t.Fatalf("old batch silently changed the captured assignment: %v", err)
	}
	current, err := e.Store.Get(parent.ID)
	if err != nil || current.PlanRevision != before.PlanRevision || len(current.PlanEvidence) != len(before.PlanEvidence) {
		t.Fatalf("old batch assignment history changed: %+v %v", current, err)
	}
	successor, err := e.Store.Get(child.ID)
	if err != nil || successor.ModelPlan != child.ModelPlan || successor.Round != 0 {
		t.Fatalf("old assignment update changed the successor: %+v %v", successor, err)
	}
}
