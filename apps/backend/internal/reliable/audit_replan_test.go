package reliable

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type replanAuditAgent struct {
	base            fixtureAgent
	calls           atomic.Int32
	started         chan struct{}
	release         chan struct{}
	extraReviewFile string
}

func (agent *replanAuditAgent) Run(ctx context.Context, request RoleRequest) (*AgentResult, error) {
	if request.Role == roleAuditor {
		count := agent.calls.Add(1)
		if count == 1 {
			if err := os.WriteFile(filepath.Join(request.Directory, "app.txt"), []byte("retained-human-review\n"), 0600); err != nil {
				return nil, err
			}
			return nil, context.DeadlineExceeded
		}
		if count == 2 && agent.started != nil {
			close(agent.started)
			select {
			case <-agent.release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	result, err := agent.base.Run(ctx, request)
	if err == nil && request.Role == roleReviewer && agent.extraReviewFile != "" {
		var report Review
		if err = json.Unmarshal(result.Output, &report); err != nil {
			return nil, err
		}
		report.ReadFiles = append(report.ReadFiles, agent.extraReviewFile)
		result.Output, err = json.Marshal(report)
	}
	return result, err
}

func TestAuditReplanConfirmsExistingChangesWithoutExpandingWriterScope(t *testing.T) {
	agent := &replanAuditAgent{extraReviewFile: "docs.md"}
	engine, task, handler := auditReplanFixture(t, agent, ".")
	if _, err := engine.Store.Update(task.ID, func(saved *Task) error { saved.ApprovedPaths = []string{"app.txt"}; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(task.Worktree, "docs.md"), []byte("Human-confirmed documentation\n"), 0600); err != nil {
		t.Fatal(err)
	}
	inspection := replanGet(t, handler, task)
	body, _ := json.Marshal(map[string]any{"source_hash": inspection["source_hash"], "confirm": true})
	response := httpRequest(handler, http.MethodPost, "/api/v1/reliable/tasks/"+task.ID+"/replan", string(body), handler.Token)
	if response.Code != http.StatusAccepted {
		t.Fatalf("human confirmation rejected: %d %s", response.Code, response.Body)
	}
	engine.wg.Wait()
	saved, err := engine.Store.Get(task.ID)
	if err != nil || saved.State != stateAwaitingAcceptance || saved.Round != 1 {
		t.Fatalf("confirmed changes remained stuck outside the next minimal audit: %v %+v", err, saved)
	}
	if len(saved.Audit.AllowedPaths) != 1 || saved.Audit.AllowedPaths[0] != "app.txt" || !containsPath(saved.RequiredReads, "docs.md") || !containsPath(saved.ApprovedPaths, "docs.md") {
		t.Fatal("human confirmation broadened writer scope or omitted the change from independent review")
	}
	raw, err := engine.readAuditReplanReceipt(saved.ID, saved.AuditReplan.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		ConfirmedPaths []string `json:"confirmed_paths"`
	}
	if err = json.Unmarshal(raw, &receipt); err != nil || !containsPath(receipt.ConfirmedPaths, "docs.md") {
		t.Fatalf("exact confirmed paths were not recorded: %v", err)
	}
}

func auditReplanFixture(t *testing.T, agent *replanAuditAgent, scope ...string) (*Engine, *Task, *HTTP) {
	t.Helper()
	engine, task := fixtureEngine(t, agent, passingCheck)
	if len(scope) == 0 {
		scope = []string{"app.txt"}
	}
	if _, err := engine.Store.Update(task.ID, func(saved *Task) error { saved.AllowedPaths = scope; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := engine.RunTask(context.Background(), task.ID); err == nil {
		t.Fatal("mutating auditor positive control did not block")
	}
	task, err := engine.Store.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	return engine, task, &HTTP{Engine: engine, Token: "explicit-replan"}
}

func replanGet(t *testing.T, handler *HTTP, task *Task) map[string]any {
	t.Helper()
	response := httpRequest(handler, http.MethodGet, "/api/v1/reliable/tasks/"+task.ID+"/replan", "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("replan inspection unavailable: %d %s", response.Code, response.Body)
	}
	var result map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestAuditReplanRequiresExplicitSourceConfirmationAndQueuesOnce(t *testing.T) {
	agent := &replanAuditAgent{started: make(chan struct{}), release: make(chan struct{})}
	engine, task, handler := auditReplanFixture(t, agent)
	var release sync.Once
	defer func() { release.Do(func() { close(agent.release) }); engine.wg.Wait() }()
	inspection := replanGet(t, handler, task)
	if inspection["eligible"] != true || inspection["source_hash"] == "" || inspection["total_changes"] != float64(1) {
		t.Fatalf("retained candidate was not inspectable: %+v", inspection)
	}
	body, _ := json.Marshal(map[string]any{"source_hash": inspection["source_hash"], "confirm": true, "note": "Keep this change and independently replan"})
	endpoint := "/api/v1/reliable/tasks/" + task.ID + "/replan"
	denied := httpRequest(handler, http.MethodPost, endpoint, string(body), "")
	if denied.Code != http.StatusForbidden {
		t.Fatal("human confirmation did not require the existing browser capability")
	}
	accepted := httpRequest(handler, http.MethodPost, endpoint, string(body), handler.Token)
	if accepted.Code != http.StatusAccepted {
		t.Fatalf("confirmed replan rejected: %d %s", accepted.Code, accepted.Body)
	}
	select {
	case <-agent.started:
	case <-time.After(10 * time.Second):
		t.Fatal("confirmed candidate did not reach a fresh auditor")
	}
	duplicate := httpRequest(handler, http.MethodPost, endpoint, string(body), handler.Token)
	if duplicate.Code != http.StatusAccepted && duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate action had unexpected status: %d", duplicate.Code)
	}
	if agent.calls.Load() != 2 {
		t.Fatal("duplicate confirmation dispatched another auditor")
	}
	release.Do(func() { close(agent.release) })
	engine.wg.Wait()
	saved, err := engine.Store.Get(task.ID)
	if err != nil || saved.Round != 1 || saved.State != stateAwaitingAcceptance || len(saved.RequirementEvidence) != 1 {
		t.Fatalf("replan bypassed ordinary implementation/verification or duplicated its receipt: %v %+v", err, saved)
	}
}

func TestAuditReplanRejectsStaleFingerprintAndMissingConfirmation(t *testing.T) {
	agent := &replanAuditAgent{}
	engine, task, handler := auditReplanFixture(t, agent)
	inspection := replanGet(t, handler, task)
	endpoint := "/api/v1/reliable/tasks/" + task.ID + "/replan"
	for _, confirm := range []bool{false, true} {
		if confirm {
			if err := os.WriteFile(filepath.Join(task.Worktree, "app.txt"), []byte("changed-after-inspection\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		body, _ := json.Marshal(map[string]any{"source_hash": inspection["source_hash"], "confirm": confirm})
		response := httpRequest(handler, http.MethodPost, endpoint, string(body), handler.Token)
		if response.Code != http.StatusConflict {
			t.Fatalf("unconfirmed or stale candidate was admitted: %d %s", response.Code, response.Body)
		}
	}
	saved, err := engine.Store.Get(task.ID)
	if err != nil || agent.calls.Load() != 1 || saved.Round != 0 || len(saved.RequirementEvidence) != 0 {
		t.Fatal("rejected replan changed task history")
	}
}

func TestAuditReplanNeverExpandsOriginalScopeOrProtectedFiles(t *testing.T) {
	for _, name := range []string{"outside.txt", "AGENTS.md"} {
		t.Run(name, func(t *testing.T) {
			agent := &replanAuditAgent{}
			scope := "app.txt"
			if name == "AGENTS.md" {
				scope = "."
			}
			_, task, handler := auditReplanFixture(t, agent, scope)
			if err := os.WriteFile(filepath.Join(task.Worktree, name), []byte("unauthorized\n"), 0600); err != nil {
				t.Fatal(err)
			}
			inspection := replanGet(t, handler, task)
			if inspection["eligible"] != false || !strings.Contains(inspection["reason"].(string), "audit_replan_scope") {
				t.Fatalf("unsafe candidate was offered for confirmation: %+v", inspection)
			}
		})
	}
}

func TestAuditReplanDoesNotResetExhaustedOrArchivedTasks(t *testing.T) {
	agent := &replanAuditAgent{}
	engine, task, handler := auditReplanFixture(t, agent)
	for _, change := range []func(*Task){
		func(saved *Task) { saved.Round = MaxRounds },
		func(saved *Task) { saved.Round, saved.ArchivedAt = 0, "archived" },
	} {
		if _, err := engine.Store.Update(task.ID, func(saved *Task) error { change(saved); return nil }); err != nil {
			t.Fatal(err)
		}
		inspection := replanGet(t, handler, task)
		if inspection["eligible"] != false || agent.calls.Load() != 1 {
			t.Fatalf("ended task was offered automatic new attempts: %+v", inspection)
		}
	}
}
