package reliable

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCancelledRecoveryEvidence(t *testing.T) {
	agent := &fixtureAgent{}
	e, task := preparedCancelledTask(t, agent, 1)
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.PlanRevision = 1
		saved.PlanEvidence = []string{"model-plan-1-missing.json"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if code, body := recoveryHTTP(e, task.ID); code != http.StatusAccepted {
		t.Fatalf("claim rejected: %d %s", code, body)
	}
	e.wg.Wait()
	saved, _ := e.Store.Get(task.ID)
	if saved.State != stateCancelled || !strings.Contains(saved.Reason, "recovery_evidence_invalid") || agent.total != 0 {
		t.Fatal("missing plan evidence admitted work")
	}
}

func TestCancelledRecoveryKnownGitCleanupFailure(t *testing.T) {
	agent := &fixtureAgent{}
	e, task := preparedCancelledTask(t, agent, 1)
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.Reason = "owned_process_cleanup: wait for job processes: context deadline exceeded"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for attempt := range 3 {
		if code, body := recoveryHTTP(e, task.ID); code != http.StatusAccepted {
			t.Fatalf("claim rejected: %d %s", code, body)
		}
		e.wg.Wait()
		saved, _ := e.Store.Get(task.ID)
		if saved.State != stateCancelled || !strings.Contains(saved.Reason, "recovery_process_unknown") || agent.total != 0 {
			t.Fatalf("known failed Git cleanup was ignored on attempt %d: state=%s calls=%d reason=%s", attempt+1, saved.State, agent.total, saved.Reason)
		}
		if attempt == 0 {
			if err = e.Cancel(task.ID); err != nil {
				t.Fatal(err)
			}
			if _, err = e.Store.Update(task.ID, func(saved *Task) error { saved.State = stateCancelling; return nil }); err != nil {
				t.Fatal(err)
			}
			if err = e.Store.Recover(); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestCancelledRecoveryKeepsModelAndRequirementHistory(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	t.Cleanup(func() { _ = e.Close() })
	plan := e.defaultPlan()
	var err error
	if task, err = e.UpdateModelPlan(task.ID, 0, plan); err != nil {
		t.Fatal(err)
	}
	if err = e.prepare(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	plan.ImplementerModel = "existing"
	if _, err = e.UpdateModelPlan(task.ID, 1, plan); err != nil {
		t.Fatal(err)
	}
	if _, err = e.submitMessage(task.ID, messageRequest{ID: "req", Kind: messageRequirement, Content: "Keep prior behavior"}); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Store.Update(task.ID, func(saved *Task) error {
		saved.State, saved.Round = stateCancelled, 1
		saved.ApprovedPaths = []string{"app.txt"}
		saved.Messages[0].Status = "applied"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	plan.ReviewerModel = "existing"
	if _, err = e.UpdateFutureModelPlan(task.ID, 0, plan); err != nil {
		t.Fatal(err)
	}
	if code, body := recoveryHTTP(e, task.ID); code != http.StatusAccepted {
		t.Fatalf("claim rejected: %d %s", code, body)
	}
	e.wg.Wait()
	saved, _ := e.Store.Get(task.ID)
	if saved.State != stateAwaitingAcceptance || saved.PlanRevision != 3 || saved.ModelPlan != plan || saved.FuturePlan != nil || saved.RequirementRevision != 1 {
		t.Fatalf("legitimate history rejected or lost: state=%s plan=%d reason=%s", saved.State, saved.PlanRevision, saved.Reason)
	}
}

func TestCancelledRecoveryUnknownLegacyProcesses(t *testing.T) {
	agent := &fixtureAgent{}
	e, task := preparedCancelledTask(t, agent, 1)
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.Runs = []Dispatch{{ID: "legacy-failed", Role: roleReviewer, Status: "failed"}}
		saved.Review = json.RawMessage(`{"verdict":"pass"}`)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if code, body := recoveryHTTP(e, task.ID); code != http.StatusAccepted {
		t.Fatalf("claim rejected: %d %s", code, body)
	}
	e.wg.Wait()
	saved, _ := e.Store.Get(task.ID)
	if saved.State != stateCancelled || !strings.Contains(saved.Reason, "recovery_process_unknown") || agent.total != 0 {
		t.Fatal("raw legacy pass or absent owner was treated as completed cleanup")
	}
	if _, err = e.acceptanceEvidence(context.Background(), saved); err == nil {
		t.Fatal("raw old review established acceptance")
	}
}

func TestCancelledRecoveryAdmissionRacesNewProjectOwner(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	if err := e.Cancel(task.ID); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		_, err := e.Store.claimCancelled(task.ID, &RecoveryAttempt{ID: "claim", Intent: "intent.json"})
		results <- err
	}()
	go func() {
		<-start
		results <- e.Store.Create(&Task{ID: "competing", Project: task.Project, State: stateQueued})
	}()
	close(start)
	winners := 0
	for range 2 {
		if <-results == nil {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("concurrent project writers admitted: %d", winners)
	}
}

func TestCancelledRecoveryReadHasNoSideEffects(t *testing.T) {
	agent := &fixtureAgent{}
	e, task := preparedCancelledTask(t, agent, 1)
	before, _ := json.Marshal(task)
	h := &HTTP{Engine: e, Token: "test-action"}
	for range 2 {
		if r := httpRequest(h, http.MethodGet, "/api/v1/reliable/tasks/"+task.ID, "", ""); r.Code != http.StatusOK {
			t.Fatal(r.Code)
		}
		if r := httpRequest(h, http.MethodGet, "/api/v1/reliable/tasks/"+task.ID+"/recover-cancelled", "", ""); r.Code != http.StatusNotFound {
			t.Fatal("GET performed recovery")
		}
	}
	after, _ := e.Store.Get(task.ID)
	raw, _ := json.Marshal(after)
	if string(raw) != string(before) || agent.total != 0 {
		t.Fatal("passive read modified the stopped task")
	}
}

func TestCancelledRecoveryRejectsRequirementWhilePending(t *testing.T) {
	e, task := preparedCancelledTask(t, &fixtureAgent{}, 1)
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.State = stateQueued
		saved.RecoveryPending = &RecoveryAttempt{ID: "pending", Intent: "pending.json"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.submitMessage(task.ID, messageRequest{ID: "supplement", Kind: messageRequirement, Content: "Change more"})
	if err == nil || !strings.Contains(err.Error(), "recovery_active") {
		t.Fatal("unvalidated candidate received new requirements")
	}
}

func TestCancelledRecoveryBeforePreparation(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	t.Cleanup(func() { _ = e.Close() })
	if err := e.Cancel(task.ID); err != nil {
		t.Fatal(err)
	}
	if code, body := recoveryHTTP(e, task.ID); code != http.StatusAccepted {
		t.Fatalf("claim rejected: %d %s", code, body)
	}
	e.wg.Wait()
	saved, _ := e.Store.Get(task.ID)
	if saved.State != stateAwaitingAcceptance || saved.Round != 1 || saved.Worktree == "" {
		t.Fatalf("unprepared recovery failed: state=%s round=%d reason=%s", saved.State, saved.Round, saved.Reason)
	}
}

func TestCancelledRecoveryResourceReferences(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	task.Project = resourceFixture(t)
	if _, err := e.Store.Update(task.ID, func(saved *Task) error { saved.Project = task.Project; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := e.prepare(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if err := e.recoveryResources(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(e.Root, "evidence", task.ID, "resource-paths.txt")
	old, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	mustRecoveryWrite(t, index, "changed")
	if err = e.recoveryResources(context.Background(), task); err == nil {
		t.Fatal("altered resource-path index accepted")
	}
	if err = os.WriteFile(index, old, 0600); err != nil {
		t.Fatal(err)
	}
	marker, err := profileMarkerPath(context.Background(), task.Worktree)
	if err != nil {
		t.Fatal(err)
	}
	mustRecoveryWrite(t, marker+".ready", `{"ready":false}`)
	if err = e.recoveryResources(context.Background(), task); err == nil {
		t.Fatal("incomplete sparse checkout accepted")
	}
}
