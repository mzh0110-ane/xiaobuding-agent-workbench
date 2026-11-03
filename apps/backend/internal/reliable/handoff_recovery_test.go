package reliable

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInterruptedAuditResumesWithoutInventingMissingResults(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(map[bool]string{false: "after-dispatch", true: "before-dispatch"}[partial], func(t *testing.T) {
			e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
			ctx := context.Background()
			if err := e.prepare(ctx, task); err != nil {
				t.Fatal(err)
			}
			if partial {
				_, err := e.Store.Update(task.ID, func(saved *Task) error {
					saved.Runs = append(saved.Runs, Dispatch{ID: "not-started", Role: roleAuditor, Round: 1, Status: "preparing", InputEvidence: "missing-input.json", StartEvidence: "missing-start.json", EndEvidence: "missing-result.json"})
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			} else if _, _, err := e.beginDispatch(ctx, task, roleAuditor, 1); err != nil {
				t.Fatal(err)
			}
			if err := e.Store.Recover(); err != nil {
				t.Fatal(err)
			}
			if err := e.RunTask(ctx, task.ID); err != nil {
				t.Fatalf("interrupted audit could not resume safely: %v", err)
			}
			got, _ := e.Store.Get(task.ID)
			if got.State != stateAwaitingAcceptance || got.Round != 1 || len(got.Sessions) != 3 {
				t.Fatalf("recovery changed progress or invented a session: %+v", got)
			}
			if err := e.Accept(ctx, task.ID); err != nil {
				t.Fatalf("recovered task cannot pass intact evidence: %v", err)
			}
		})
	}
}

func TestOrphanedPlanExportDoesNotPermanentlyLockModelSelection(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	dir := filepath.Join(e.Root, "evidence", task.ID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "model-plan-1.json"), []byte(`{"stale":"uncommitted export"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.UpdateModelPlan(task.ID, 0, e.defaultPlan()); err != nil {
		t.Fatalf("orphaned pre-commit export locked the task: %v", err)
	}
}
