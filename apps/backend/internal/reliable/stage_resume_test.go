package reliable

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type stageResumeFixture struct {
	base            *domainFixture
	failures        int
	backendCalls    int
	partialFrontend bool
	failAfter       int
	rejectReviews   int
}

func (a *stageResumeFixture) Run(ctx context.Context, req RoleRequest) (*AgentResult, error) {
	if req.Role == roleImplementer && req.Domain == domainBackend {
		a.backendCalls++
		if a.failures > 0 && a.backendCalls > a.failAfter {
			return nil, a.failBackend(req)
		}
	}
	result, err := a.base.Run(ctx, req)
	if err == nil && req.Role == roleReviewer && a.rejectReviews > 0 {
		a.rejectReviews--
		result.Output = []byte(`{"verdict":"rework","summary":"Behavior still needs correction","read_files":["frontend.txt","backend.txt"],"findings":[{"severity":"P1","file":"backend.txt","reason":"Required behavior incomplete"}]}`)
	}
	return result, err
}

func (a *stageResumeFixture) failBackend(req RoleRequest) error {
	a.failures--
	path := "backend.txt"
	if a.partialFrontend {
		path = "frontend.txt"
	}
	if err := os.WriteFile(filepath.Join(req.Directory, path), []byte("partial\n"), 0600); err != nil {
		return err
	}
	return context.DeadlineExceeded
}

func TestStageResumeTransientBackendDoesNotRepeatFrontendOrSpendRound(t *testing.T) {
	base := &domainFixture{}
	e, task := domainEngine(t, base, domainCheck)
	agent := &stageResumeFixture{base: base, failures: 1}
	e.Runner = agent
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	got, err := e.Store.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != stateAwaitingAcceptance || got.Round != 1 || base.frontend != 1 || base.backend != 1 || agent.backendCalls != 2 {
		t.Fatalf("operational retry lost the completed writer or consumed a quality round: state=%s round=%d fixture=%+v calls=%d", got.State, got.Round, base, agent.backendCalls)
	}
	if outcomes, recoverErr := e.prepareDomainOutcomes(context.Background(), got); recoverErr != nil || len(outcomes) != 2 {
		t.Fatalf("completed retry evidence cannot be reconstructed after restart: outcomes=%d error=%v", len(outcomes), recoverErr)
	}
	if err := e.Accept(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
}

func TestStageResumeManualRetryCanRecoverLegacyBoundedFailure(t *testing.T) {
	base := &domainFixture{}
	e, task := domainEngine(t, base, domainCheck)
	agent := &stageResumeFixture{base: base, failures: 3}
	e.Runner = agent
	if err := e.RunTask(context.Background(), task.ID); err == nil {
		t.Fatal("expected a bounded operational pause")
	}
	got, _ := e.Store.Get(task.ID)
	if got.Round != 1 || agent.backendCalls != 3 {
		t.Fatalf("unbounded retries or round spending: %+v calls=%d", got, agent.backendCalls)
	}
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		raw, marshalErr := json.Marshal(saved)
		if marshalErr != nil {
			return marshalErr
		}
		var fields map[string]json.RawMessage
		if marshalErr = json.Unmarshal(raw, &fields); marshalErr != nil {
			return marshalErr
		}
		delete(fields, "stage_recovery")
		raw, marshalErr = json.Marshal(fields)
		if marshalErr != nil {
			return marshalErr
		}
		*saved = Task{}
		return json.Unmarshal(raw, saved)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = e.Store.Get(task.ID)
	if got.State != stateAwaitingAcceptance || got.Round != 1 || base.frontend != 1 || base.backend != 1 {
		t.Fatalf("legacy saved frontend was not reused: %+v %+v", got, base)
	}
	if got.StageRecovery == nil || got.StageRecovery.Attempts != 4 || got.StageRecovery.ManualResumes != 1 {
		t.Fatalf("legacy attempts were not retained transparently: %+v", got.StageRecovery)
	}
}

func TestStageResumeRejectsPartialBackendEditsToCompletedFrontend(t *testing.T) {
	base := &domainFixture{}
	e, task := domainEngine(t, base, domainCheck)
	agent := &stageResumeFixture{base: base, failures: 1, partialFrontend: true}
	e.Runner = agent
	err := e.RunTask(context.Background(), task.ID)
	if err == nil {
		t.Fatal("failed backend overwrote the frontend and was retried")
	}
	if agent.backendCalls != 1 || base.reviews != 0 {
		t.Fatalf("scope-invalid source was dispatched: %v %+v", err, agent)
	}
}

func TestStageResumeRejectsChangedCompletedSourceAndReport(t *testing.T) {
	for _, target := range []string{"source", "report"} {
		t.Run(target, func(t *testing.T) {
			base := &domainFixture{}
			e, task := domainEngine(t, base, domainCheck)
			agent := &stageResumeFixture{base: base, failures: 3}
			e.Runner = agent
			if err := e.RunTask(context.Background(), task.ID); err == nil {
				t.Fatal("expected bounded pause")
			}
			got, _ := e.Store.Get(task.ID)
			path := filepath.Join(got.Worktree, "frontend.txt")
			raw := []byte("manually changed completed work\n")
			if target == "report" {
				path = filepath.Join(e.Root, "evidence", task.ID, "round-1-frontend-implementation.json")
				var outcome domainOutcome
				if err := e.readStageEvidence(got, "round-1-frontend-implementation.json", &outcome); err != nil {
					t.Fatal(err)
				}
				outcome.Report.Summary = "Changed after the identified session"
				raw, _ = json.Marshal(outcome)
			}
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if err := e.RunTask(context.Background(), task.ID); err == nil {
				t.Fatal("invalid completed stage evidence was reused")
			}
			next, _ := e.Store.Get(task.ID)
			if next.Round != 1 || base.frontend != 1 || agent.backendCalls != 3 || base.reviews != 0 {
				t.Fatalf("invalidated work was dispatched or consumed a round: %+v %+v", next, agent)
			}
		})
	}
}

func TestStageResumeOperationalFailureOnThirdRoundStillResumes(t *testing.T) {
	base := &domainFixture{}
	e, task := domainEngine(t, base, domainCheck)
	agent := &stageResumeFixture{base: base, failures: 3, failAfter: 2, rejectReviews: 2}
	e.Runner = agent
	if err := e.RunTask(context.Background(), task.ID); err == nil {
		t.Fatal("expected third-round operational pause")
	}
	got, _ := e.Store.Get(task.ID)
	if got.Round != 3 || got.State != stateBlocked || !domainStageResumeEligible(got) {
		t.Fatalf("third-round operational retry is unavailable: %+v", got)
	}
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = e.Store.Get(task.ID)
	if got.Round != 3 || got.State != stateAwaitingAcceptance || base.frontend != 3 || agent.backendCalls != 6 || base.reviews != 3 {
		t.Fatalf("operational recovery bypassed actual third-round review: %+v %+v", got, agent)
	}
}

func TestStageResumeOperationalClassification(t *testing.T) {
	if operationalStageError(context.Canceled) {
		t.Fatal("user cancellation must not automatically retry")
	}
	if !operationalStageError(context.DeadlineExceeded) {
		t.Fatal("deadline interruption must remain resumable")
	}
}

func TestStageResumeLiveLegacyReadOnlyProof(t *testing.T) {
	endpoint := os.Getenv("XIAOBUDING_VALIDATE_STAGE_TASK_URL")
	if endpoint == "" {
		t.Skip("explicit read-only live validation is not configured")
	}
	root := os.Getenv("XIAOBUDING_VALIDATE_STAGE_ROOT")
	if root == "" {
		t.Fatal("explicit validation evidence root is required")
	}
	response, err := (&http.Client{Timeout: 15 * time.Second}).Get(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("task GET failed: %s", response.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		t.Fatal(err)
	}
	var task Task
	if err = json.Unmarshal(raw, &task); err != nil {
		t.Fatal(err)
	}
	if task.State != stateBlocked || task.Round != 1 || task.StageRecovery != nil || !domainStageResumeEligible(&task) {
		t.Fatalf("expected the legacy failed first-round writer: state=%s round=%d stage=%+v", task.State, task.Round, task.StageRecovery)
	}
	engine := &Engine{Root: root}
	for _, name := range []string{fmt.Sprintf("round-%d-audit.json", task.Round), fmt.Sprintf("round-%d-before.json", task.Round)} {
		if _, err = os.Stat(filepath.Join(root, "evidence", task.ID, name)); err != nil {
			t.Fatal(err)
		}
	}
	before, err := TakeSnapshot(context.Background(), task.Worktree)
	if err != nil {
		t.Fatal(err)
	}
	outcomes, err := engine.prepareDomainOutcomes(context.Background(), &task)
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 || outcomes[0].Domain != domainFrontend {
		t.Fatalf("legacy completed frontend was not reconstructed: %+v", outcomes)
	}
	after, err := TakeSnapshot(context.Background(), task.Worktree)
	if err != nil {
		t.Fatal(err)
	}
	if before.Hash != after.Hash {
		t.Fatal("read-only validation changed the candidate")
	}
	t.Logf("task=%s round=%d legacy_frontend_reused=%t next_domain=%s candidate_hash=%s partial_backend_files=%v", task.ID, task.Round, true, task.ModelPlan.domains()[len(outcomes)], after.Hash, changedFiles(outcomes[0].After, after))
}
