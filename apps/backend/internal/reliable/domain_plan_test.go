package reliable

import (
	"context"
	"encoding/json"
	"testing"
)

func TestFrontendPlanAcceptsSeparateNativeModelAssignments(t *testing.T) {
	e := Engine{Providers: testProviders()}
	var plan ModelPlan
	if err := json.Unmarshal([]byte(`{"mode":"frontend","auditor":"codex","auditor_model":"codex-fixture","reviewer":"codex","reviewer_model":"codex-fixture","frontend":"deepseek","frontend_model":"deepseek-fixture"}`), &plan); err != nil {
		t.Fatal(err)
	}
	if err := e.validatePlan(plan); err != nil {
		t.Fatalf("frontend-only plan wrongly requires a generic or backend writer: %v", err)
	}
	raw, _ := json.Marshal(plan)
	var saved map[string]any
	_ = json.Unmarshal(raw, &saved)
	if saved["mode"] != "frontend" || saved["frontend_model"] != "deepseek-fixture" {
		t.Fatalf("plan lost its work domain or native model: %s", raw)
	}
}

func TestLegacyStartedTaskCannotBeConvertedToDomainWorkflow(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	e.Providers = testProviders()
	if err := e.prepare(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	var plan ModelPlan
	_ = json.Unmarshal([]byte(`{"mode":"fullstack","auditor":"codex","reviewer":"codex","frontend":"deepseek","backend":"codex"}`), &plan)
	if _, err := e.UpdateModelPlan(task.ID, 0, plan); err == nil {
		t.Fatal("legacy in-flight task changed workflow contract on its first plan update")
	}
}

func TestPlanRejectsModelFromAnotherNativeCatalog(t *testing.T) {
	e := Engine{Providers: testProviders()}
	var plan ModelPlan
	_ = json.Unmarshal([]byte(`{"auditor":"codex","implementer":"codex","reviewer":"deepseek","implementer_model":"deepseek-fixture"}`), &plan)
	if err := e.validatePlan(plan); err == nil {
		t.Fatal("Codex assignment accepted a model belonging to the Harness connection")
	}
}

func TestDomainWorkflowKeepsIndependentReviewOnCodex(t *testing.T) {
	e := Engine{Providers: testProviders()}
	var plan ModelPlan
	_ = json.Unmarshal([]byte(`{"mode":"frontend","auditor":"codex","reviewer":"deepseek","frontend":"deepseek"}`), &plan)
	if err := e.validatePlan(plan); err == nil {
		t.Fatal("domain workflow moved required Codex review to another tool")
	}
}
