package reliable

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type conversationProbe struct {
	mu       sync.Mutex
	requests []RoleRequest
	started  chan struct{}
	release  chan struct{}
}

func (p *conversationProbe) Run(ctx context.Context, r RoleRequest) (*AgentResult, error) {
	p.mu.Lock()
	p.requests = append(p.requests, r)
	n := len(p.requests)
	p.mu.Unlock()
	if n == 1 && p.started != nil {
		close(p.started)
		select {
		case <-p.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	raw := `{"summary":"Source-based answer","facts":["app.txt exists"],"inferences":[],"unknowns":[],"read_files":["app.txt"],"required_read_files":[],"allowed_paths":[],"plan":[]}`
	return &AgentResult{Session: fmt.Sprintf("conversation-%d", n), Output: json.RawMessage(raw)}, nil
}

func TestHTTPMultipleReadOnlyQuestionsRetainContextAndIdentity(t *testing.T) {
	probe := &conversationProbe{started: make(chan struct{}), release: make(chan struct{})}
	e, task := fixtureEngine(t, probe, passingCheck)
	t.Cleanup(e.wg.Wait)
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.ReadOnly, saved.State = true, stateInspected
		saved.Audit = &Audit{Summary: "Existing project report", ReadFiles: []string{"app.txt"}}
		return nil
	})
	require.NoError(t, err)
	h := &HTTP{Engine: e, Token: "conversation-test"}
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"question-1","kind":"question","content":"Explain the current architecture"}`, h.Token)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	select {
	case <-probe.started:
	case <-time.After(5 * time.Second):
		t.Fatal("question did not start")
	}
	response = httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"question-2","kind":"question","content":"Which source supports that?"}`, h.Token)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	close(probe.release)
	require.Eventually(t, func() bool {
		current, loadErr := e.Store.Get(task.ID)
		raw, _ := json.Marshal(current)
		return loadErr == nil && strings.Count(string(raw), `"status":"answered"`) == 2
	}, 5*time.Second, 10*time.Millisecond)
	e.wg.Wait()
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, stateInspected, current.State)
	require.Zero(t, current.Round)
	require.Empty(t, current.Runs, "questions must not alter the sealed dispatch history")
	probe.mu.Lock()
	defer probe.mu.Unlock()
	require.Len(t, probe.requests, 2)
	for _, request := range probe.requests {
		require.True(t, request.ReadOnly)
		require.Equal(t, task.Project, request.Directory)
		require.True(t, pathInside(filepath.Join(e.Root, "conversations", task.ID), request.OutDir))
	}
	require.Contains(t, probe.requests[1].Prompt, "Existing project report")
	require.Contains(t, probe.requests[1].Prompt, "Explain the current architecture")
	require.Contains(t, probe.requests[1].Prompt, "Source-based answer")
}

type requirementProbe struct {
	base      fixtureAgent
	mu        sync.Mutex
	prompts   []RoleRequest
	started   chan struct{}
	release   chan struct{}
	blockRole string
	blocked   bool
}

func (p *requirementProbe) Run(ctx context.Context, request RoleRequest) (*AgentResult, error) {
	p.mu.Lock()
	p.prompts = append(p.prompts, request)
	p.mu.Unlock()
	role := p.blockRole
	if role == "" {
		role = roleReviewer
	}
	if request.Role == role && !p.blocked {
		p.blocked = true
		close(p.started)
		select {
		case <-p.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return p.base.Run(ctx, request)
}

func TestRequirementDuringReviewCannotSealAnOutdatedGoal(t *testing.T) {
	probe := &requirementProbe{started: make(chan struct{}), release: make(chan struct{})}
	e, task := fixtureEngine(t, probe, passingCheck)
	t.Cleanup(e.wg.Wait)
	done := make(chan error, 1)
	go func() { done <- e.RunTask(context.Background(), task.ID) }()
	select {
	case <-probe.started:
	case <-time.After(15 * time.Second):
		t.Fatal("review did not start")
	}
	h := &HTTP{Engine: e, Token: "requirements-test"}
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"requirement-1","kind":"requirement","content":"Also preserve the existing public interface"}`, h.Token)
	close(probe.release)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	require.NoError(t, <-done)
	e.wg.Wait()
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, stateAwaitingAcceptance, current.State)
	require.Equal(t, 2, current.Round)
	probe.mu.Lock()
	defer probe.mu.Unlock()
	for _, request := range probe.prompts[len(probe.prompts)-3:] {
		require.Contains(t, request.Prompt, "Also preserve the existing public interface")
	}
	_, err = e.acceptanceEvidence(context.Background(), current)
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(task.Project, "app.txt"))
	require.NoError(t, err)
	require.Equal(t, "before\n", string(raw))
	for _, name := range current.RequirementEvidence {
		require.NoError(t, os.WriteFile(filepath.Join(e.Root, "evidence", task.ID, name), []byte("altered requirement"), 0600))
	}
	_, err = e.acceptanceEvidence(context.Background(), current)
	require.Error(t, err, "changed requirement evidence must invalidate acceptance")
}

func TestQuestionPreservesSealedCandidateAndIdempotentSubmission(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	t.Cleanup(e.wg.Wait)
	require.NoError(t, e.RunTask(context.Background(), task.ID))
	sealed, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	probe := &conversationProbe{}
	e.Runner = probe
	h := &HTTP{Engine: e, Token: "sealed-question-test"}
	body := `{"id":"question-1","kind":"question","content":"What changed?"}`
	for i := 0; i < 2; i++ {
		response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", body, h.Token)
		require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	}
	e.wg.Wait()
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, sealed.EvidenceHash, current.EvidenceHash)
	require.Equal(t, sealed.Round, current.Round)
	require.Equal(t, sealed.Runs, current.Runs)
	_, err = e.acceptanceEvidence(context.Background(), current)
	require.NoError(t, err)
	require.Len(t, probe.requests, 1)
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"question-1","kind":"question","content":"Different content"}`, h.Token)
	require.Equal(t, http.StatusConflict, response.Code)
	require.Contains(t, response.Body.String(), "message_conflict")
}

func TestInspectionStartsLinkedModificationBatchWithoutResettingHistory(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	t.Cleanup(e.wg.Wait)
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.ReadOnly, saved.State = true, stateInspected
		saved.Audit = &Audit{Summary: "Existing source report", ReadFiles: []string{"app.txt"}}
		return nil
	})
	require.NoError(t, err)
	h := &HTTP{Engine: e, Token: "linked-batch-test"}
	body := `{"id":"new-batch-1","kind":"requirement","content":"Update app.txt and retain the public interface"}`
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", body, h.Token)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	var message struct {
		LinkedTask string `json:"linked_task"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &message))
	require.NotEmpty(t, message.LinkedTask)
	require.NotEqual(t, task.ID, message.LinkedTask)
	e.wg.Wait()
	child, err := e.Store.Get(message.LinkedTask)
	require.NoError(t, err)
	require.Equal(t, task.Project, child.Project)
	require.False(t, child.ReadOnly)
	require.Equal(t, 1, child.Round)
	parent, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, stateInspected, parent.State)
	require.Zero(t, parent.Round)
	response = httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", body, h.Token)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	all, err := e.Store.List()
	require.NoError(t, err)
	require.Len(t, all, 2, "retry created an extra work batch")
}

func TestRestartInterruptsConversationWithoutReplayingCalls(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	raw := `{"id":"task","state":"inspected","messages":[{"id":"q1","kind":"question","content":"Question","status":"reading"},{"id":"q2","kind":"question","content":"Queued question","status":"queued"}]}`
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		project := saved.Project
		if err := json.Unmarshal([]byte(raw), saved); err != nil {
			return err
		}
		saved.Project = project
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, e.Store.Recover())
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	data, err := json.Marshal(current)
	require.NoError(t, err)
	require.Equal(t, 2, strings.Count(string(data), `"status":"interrupted"`))
	require.Equal(t, stateInspected, current.State)
}

func TestCancelFollowupPreservesCompletedInspection(t *testing.T) {
	probe := &conversationProbe{started: make(chan struct{}), release: make(chan struct{})}
	e, task := fixtureEngine(t, probe, passingCheck)
	t.Cleanup(e.wg.Wait)
	_, err := e.Store.Update(task.ID, func(saved *Task) error { saved.State, saved.ReadOnly = stateInspected, true; return nil })
	require.NoError(t, err)
	h := &HTTP{Engine: e, Token: "cancel-question-test"}
	for _, id := range []string{"question-1", "question-2"} {
		response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"`+id+`","kind":"question","content":"Read current source"}`, h.Token)
		require.Equal(t, http.StatusAccepted, response.Code)
	}
	select {
	case <-probe.started:
	case <-time.After(5 * time.Second):
		t.Fatal("question did not start")
	}
	require.NoError(t, e.Cancel(task.ID))
	e.wg.Wait()
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, stateInspected, current.State)
	raw, err := json.Marshal(current)
	require.NoError(t, err)
	require.Equal(t, 2, strings.Count(string(raw), `"status":"cancelled"`))
}

func TestMessageGuardsRetainTheWorkBatchLimit(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	_, err := e.Store.Update(task.ID, func(saved *Task) error { saved.State, saved.Round = stateExhausted, MaxRounds; return nil })
	require.NoError(t, err)
	h := &HTTP{Engine: e, Token: "message-guards-test"}
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"r1","kind":"requirement","content":"Try again"}`, h.Token)
	require.Equal(t, http.StatusConflict, response.Code)
	require.Contains(t, response.Body.String(), "batch_exhausted")
	require.NoError(t, e.ArchiveTask(task.ID, true))
	response = httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"q1","kind":"question","content":"Explain the failure"}`, h.Token)
	require.Equal(t, http.StatusConflict, response.Code)
	require.Contains(t, response.Body.String(), "task_archived")
	response = httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"q2","kind":"question","content":"Explain the failure"}`, "wrong-token")
	require.Equal(t, http.StatusForbidden, response.Code)
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, MaxRounds, current.Round)
	require.Empty(t, current.Messages)
}

func TestRequirementDuringAuditDoesNotSpendAnExtraImplementationRound(t *testing.T) {
	probe := &requirementProbe{started: make(chan struct{}), release: make(chan struct{}), blockRole: roleAuditor}
	e, task := fixtureEngine(t, probe, passingCheck)
	t.Cleanup(e.wg.Wait)
	done := make(chan error, 1)
	go func() { done <- e.RunTask(context.Background(), task.ID) }()
	select {
	case <-probe.started:
	case <-time.After(10 * time.Second):
		t.Fatal("audit did not start")
	}
	h := &HTTP{Engine: e, Token: "audit-requirement-test"}
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"r1","kind":"requirement","content":"Retain the public interface"}`, h.Token)
	close(probe.release)
	require.Equal(t, http.StatusAccepted, response.Code)
	require.NoError(t, <-done)
	e.wg.Wait()
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, 1, current.Round)
	require.Equal(t, stateAwaitingAcceptance, current.State)
}

type continuationWriter struct{ fixtureAgent }

func (w *continuationWriter) Run(ctx context.Context, request RoleRequest) (*AgentResult, error) {
	result, err := w.fixtureAgent.Run(ctx, request)
	if err == nil && request.Role == roleImplementer && w.implement > 1 {
		err = os.WriteFile(filepath.Join(request.Directory, "app.txt"), []byte("reviewed again\n"), 0600)
	}
	return result, err
}

func TestManagedProjectContinuesAfterAcceptanceInTheSameDirectory(t *testing.T) {
	e, task := fixtureEngine(t, &continuationWriter{}, `python -c "from pathlib import Path; assert Path('app.txt').read_text().startswith('reviewed')"`)
	t.Cleanup(e.wg.Wait)
	project := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(project, "app.txt"), []byte("before\n"), 0600))
	_, err := e.Store.Update(task.ID, func(saved *Task) error { saved.Project, saved.AutoSnapshot = project, true; return nil })
	require.NoError(t, err)
	require.NoError(t, e.RunTask(context.Background(), task.ID))
	require.NoError(t, e.Accept(context.Background(), task.ID))
	h := &HTTP{Engine: e, Token: "next-managed-batch"}
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"next-1","kind":"requirement","content":"Extend app.txt to reviewed again while keeping its verified prefix"}`, h.Token)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	e.wg.Wait()
	all, err := e.Store.List()
	require.NoError(t, err)
	require.Len(t, all, 2)
	child := all[0]
	require.Equal(t, project, child.Project)
	require.Equal(t, task.ID, child.ParentTaskID)
	require.Equal(t, stateAwaitingAcceptance, child.State, child.Reason)
	require.Equal(t, 1, child.Round)
	require.Contains(t, string(child.PreviousContext), "Update app.txt")
	parent, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, stateAccepted, parent.State)
}

func TestCancellingWorkflowDoesNotStartQueuedFollowupQuestions(t *testing.T) {
	probe := &requirementProbe{started: make(chan struct{}), release: make(chan struct{})}
	e, task := fixtureEngine(t, probe, passingCheck)
	t.Cleanup(e.wg.Wait)
	done := make(chan error, 1)
	go func() { done <- e.RunTask(context.Background(), task.ID) }()
	select {
	case <-probe.started:
	case <-time.After(15 * time.Second):
		t.Fatal("review did not start")
	}
	h := &HTTP{Engine: e, Token: "cancel-workflow-questions"}
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"q1","kind":"question","content":"Explain the result after work"}`, h.Token)
	require.Equal(t, http.StatusAccepted, response.Code)
	require.NoError(t, e.Cancel(task.ID))
	require.Error(t, <-done)
	e.wg.Wait()
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, stateCancelled, current.State)
	raw, _ := json.Marshal(current)
	require.Contains(t, string(raw), `"status":"cancelled"`)
}
