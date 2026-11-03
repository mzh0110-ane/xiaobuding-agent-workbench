package reliable

import (
	"context"
	"testing"
)

func TestPassingReviewAtomicallyClosesModelPlan(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	e.Providers = testProviders()
	plan := ModelPlan{Auditor: "deepseek", Implementer: "codex", Reviewer: "deepseek"}
	if _, err := e.UpdateModelPlan(task.ID, 0, plan); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, stage := range []func(context.Context, *Task) error{e.prepare, e.audit} {
		if err := stage(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	var err error
	task, err = e.Store.StartRound(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = e.implement(ctx, task); err != nil {
		t.Fatal(err)
	}
	decision, _, err := e.verifyAndReview(ctx, task)
	if err != nil || decision != Pass {
		t.Fatalf("candidate did not pass: %s %v", decision, err)
	}
	// This is precisely the boundary after evidence hashing and before
	// runRounds receives the review result. A plan must already be frozen.
	if _, err = e.UpdateModelPlan(task.ID, 1, plan); err == nil {
		t.Fatal("sealed review still allows new evidence to invalidate acceptance")
	}
	if err = e.Accept(ctx, task.ID); err != nil {
		t.Fatalf("atomically sealed review was not ready for acceptance: %v", err)
	}
}
