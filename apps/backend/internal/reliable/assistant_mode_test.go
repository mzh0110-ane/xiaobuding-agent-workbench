package reliable

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"testing"
	"time"
)

func assistantTestPlan(t *testing.T, mode, provider string) ModelPlan {
	t.Helper()
	raw, err := json.Marshal(map[string]string{"assistant_mode": mode, "auditor": provider, "implementer": provider, "reviewer": connectionCodex})
	if err != nil {
		t.Fatal(err)
	}
	var plan ModelPlan
	if err = json.Unmarshal(raw, &plan); err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestAssistantModesRejectMisleadingSelectionsAndWeakenedReview(t *testing.T) {
	e := &Engine{Providers: testProviders()}
	for _, name := range []string{"unknown", "wrong_planner", "mixed_writer", "changed_reviewer"} {
		t.Run(name, func(t *testing.T) {
			plan := assistantTestPlan(t, connectionCodex, connectionCodex)
			switch name {
			case "unknown":
				plan = assistantTestPlan(t, "invisible-fallback", connectionCodex)
			case "wrong_planner":
				plan = assistantTestPlan(t, connectionCodex, connectionHarness)
			case "mixed_writer":
				plan.Implementer = connectionHarness
			case "changed_reviewer":
				plan = assistantTestPlan(t, "custom", connectionCodex)
				plan.Reviewer = connectionHarness
			}
			if err := e.validatePlan(plan); err == nil {
				t.Fatal("misleading assistant mode was accepted")
			}
		})
	}
}

func TestAssistantModesPreserveLegacyAndResolveNativeDefaults(t *testing.T) {
	e := &Engine{Providers: testProviders()}
	legacy := ModelPlan{Auditor: connectionHarness, Implementer: connectionCodex, Reviewer: connectionHarness}
	if err := e.validatePlan(legacy); err != nil {
		t.Fatalf("legacy plan changed: %v", err)
	}
	plan := assistantTestPlan(t, connectionCodex, connectionCodex)
	plan.ImplementerModel = "codex-fixture"
	if err := e.validatePlan(plan); err != nil {
		t.Fatalf("equivalent native defaults rejected: %v", err)
	}
	raw, _ := json.Marshal(plan)
	var value map[string]any
	_ = json.Unmarshal(raw, &value)
	if value["assistant_mode"] != connectionCodex {
		t.Fatal("assistant choice disappeared after persistence")
	}
}

func TestAssistantModesValidateReadOnlySelectionWithoutActivatingWriters(t *testing.T) {
	e := &Engine{Providers: testProviders()}
	invalid := assistantTestPlan(t, "unsupported-mode", connectionCodex)
	if err := e.validateTaskPlan(invalid, true); err == nil {
		t.Fatal("read-only plan bypassed assistant-mode validation")
	}
	invalid = assistantTestPlan(t, connectionCodex, connectionHarness)
	if err := e.validateTaskPlan(invalid, true); err == nil {
		t.Fatal("read-only selection contradicts the visible assistant")
	}
	valid := assistantTestPlan(t, connectionHarness, connectionHarness)
	valid.Implementer, valid.Reviewer = "unused", "unused"
	if err := e.validateTaskPlan(valid, true); err != nil {
		t.Fatalf("unused writers blocked a read-only question: %v", err)
	}
}

func TestAssistantSwitchPreservesFrozenDispatchAndProjectConversation(t *testing.T) {
	agent := &gatedAgent{started: make(chan RoleRequest, 1), release: make(chan struct{})}
	e, task := fixtureEngine(t, agent, passingCheck)
	e.Providers = testProviders()
	if _, err := e.UpdateModelPlan(task.ID, 0, assistantTestPlan(t, connectionHarness, connectionHarness)); err != nil {
		t.Fatal(err)
	}
	wantMessages := []TaskMessage{{ID: "early-context", Kind: messageQuestion, Content: "Preserve the original requirements when changing native tools", Status: "answered"}}
	if _, err := e.Store.Update(task.ID, func(saved *Task) error { saved.Messages = wantMessages; return nil }); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	done := make(chan error, 1)
	finished := make(chan struct{})
	var release sync.Once
	go func() { defer close(finished); done <- e.RunTask(ctx, task.ID) }()
	t.Cleanup(func() { cancel(); release.Do(func() { close(agent.release) }); <-finished })
	select {
	case <-agent.started:
	case <-ctx.Done():
		t.Fatal("initial auditor did not start")
	}
	if _, err := e.UpdateModelPlan(task.ID, 1, assistantTestPlan(t, connectionCodex, connectionCodex)); err != nil {
		t.Fatal(err)
	}
	release.Do(func() { close(agent.release) })
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	saved, err := e.Store.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Round != 1 || saved.State != stateAwaitingAcceptance || !reflect.DeepEqual(saved.Messages, wantMessages) {
		t.Fatal("assistant switch lost project state or conversation")
	}
	want := []string{connectionHarness, connectionCodex, connectionCodex}
	if len(saved.Runs) != len(want) {
		t.Fatalf("unexpected native dispatches: %d", len(saved.Runs))
	}
	for index, provider := range want {
		if saved.Runs[index].ProviderID != provider || agent.seen[index].ProviderID != provider {
			t.Fatalf("dispatch %d changed identity", index)
		}
	}
	if saved.Runs[0].PlanRevision != 1 || saved.Runs[1].PlanRevision != 2 {
		t.Fatal("active execution was rewritten rather than preserving its frozen assignment")
	}
	if err := e.Accept(context.Background(), task.ID); err != nil {
		t.Fatalf("independent evidence became unusable: %v", err)
	}
}
