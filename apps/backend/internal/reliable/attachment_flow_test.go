package reliable

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAttachmentFlowEveryRoleAndAcceptanceBindsOriginalBytes(t *testing.T) {
	base := &fixtureAgent{}
	var requests []RoleRequest
	probe := agentFunc(func(ctx context.Context, request RoleRequest) (*AgentResult, error) {
		requests = append(requests, request)
		return base.Run(ctx, request)
	})
	e, task := fixtureEngine(t, probe, passingCheck)
	h := &HTTP{Engine: e, Token: "attachment-integrity"}
	metadata := flowUploadAttachment(t, h, "项目资料.dat", []byte("immutable file input"))
	attachment := flowSetInitialAttachment(t, e, task.ID, metadata)
	require.NoError(t, e.RunTask(context.Background(), task.ID))
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, stateAwaitingAcceptance, current.State)
	require.Len(t, requests, 3)
	for _, request := range requests {
		require.Len(t, request.Attachments, 1)
		require.Equal(t, attachment, request.Attachments[0].Attachment)
		require.Contains(t, request.Prompt, attachment.SHA256)
	}
	_, err = e.acceptanceEvidence(context.Background(), current)
	require.NoError(t, err, "the untouched candidate must be eligible before the tamper")
	require.NoError(t, os.WriteFile(requests[0].Attachments[0].Path, bytes.Repeat([]byte("x"), int(attachment.Size)), 0600))
	_, err = e.acceptanceEvidence(context.Background(), current)
	require.Error(t, err, "same-size attachment byte changes must invalidate acceptance")
	require.Error(t, e.validateRecoveryHistory(current), "recovery must not adopt modified input")
	original, err := os.ReadFile(filepath.Join(current.Project, "app.txt"))
	require.NoError(t, err)
	require.NotEqual(t, "reviewed\n", string(original), "checking attachments must not apply source changes")
}

func TestAttachmentOnlyQuestionCanCiteUploadedInputs(t *testing.T) {
	probe := agentFunc(func(_ context.Context, request RoleRequest) (*AgentResult, error) {
		raw, err := json.Marshal(map[string]any{"summary": "Only the attached document was read", "facts": []string{"the supplied document is available"}, "inferences": []string{}, "unknowns": []string{}, "read_files": []string{}, "read_attachments": []string{request.Attachments[0].ID}, "required_read_files": []string{}, "allowed_paths": []string{}, "plan": []string{}})
		return &AgentResult{Session: "attachment-only-question", Output: raw}, err
	})
	e, task := fixtureEngine(t, probe, passingCheck)
	_, err := e.Store.Update(task.ID, func(saved *Task) error { saved.State = stateInspected; return nil })
	require.NoError(t, err)
	h := &HTTP{Engine: e, Token: "attachment-only-report"}
	metadata := flowUploadAttachment(t, h, "only.txt", []byte("document input"))
	request := messageRequest{ID: "only-input", Kind: messageQuestion, AttachmentIDs: []string{metadata["id"].(string)}}
	message, err := e.submitMessage(task.ID, request)
	require.NoError(t, err)
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.NoError(t, e.answerQuestion(context.Background(), current, message), "reading a file must not require an invented project-source citation")
	current, err = e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, "answered", current.Messages[0].Status)
}

func TestAttachmentFlowRenewalInheritsFilesAndRejectsNonceReuse(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, `python -c "raise SystemExit(1)"`)
	h := &HTTP{Engine: e, Token: "attachment-renewal"}
	first := flowUploadAttachment(t, h, "initial.txt", []byte("initial input"))
	flowSetInitialAttachment(t, e, task.ID, first)
	require.ErrorContains(t, e.RunTask(context.Background(), task.ID), "three rounds exhausted")
	parent, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, stateExhausted, parent.State)
	question := flowUploadAttachment(t, h, "analysed-earlier.txt", []byte("previously analysed input"))
	_, err = e.submitMessage(parent.ID, messageRequest{ID: "earlier-question", Kind: messageQuestion, Content: "Analyse this file", AttachmentIDs: []string{question["id"].(string)}})
	require.NoError(t, err)
	_, err = e.Store.Update(parent.ID, func(saved *Task) error { saved.Messages[0].Status = "answered"; return nil })
	require.NoError(t, err)
	second := flowUploadAttachment(t, h, "next.txt", []byte("next input"))
	request := messageRequest{ID: "next-files", Kind: messageRequirement, Content: "Continue the same source", AttachmentIDs: []string{second["id"].(string)}}
	message, err := e.StartExhaustedBatch(parent.ID, request)
	require.NoError(t, err)
	child, err := e.Store.Get(message.LinkedTask)
	require.NoError(t, err)
	require.Len(t, child.Attachments, 3)
	require.Zero(t, child.Round)
	require.NoError(t, e.prepareExhaustedSeed(context.Background(), child))
	request.AttachmentIDs = []string{first["id"].(string)}
	_, err = e.StartExhaustedBatch(parent.ID, request)
	require.ErrorContains(t, err, "message_conflict", "a reused request ID cannot silently change files")
	unchanged, err := e.Store.Get(parent.ID)
	require.NoError(t, err)
	require.Equal(t, 3, unchanged.Round)
	require.Equal(t, message.LinkedTask, unchanged.SuccessorTaskID)
}

func TestAttachmentQuestionReferencesSurviveInspectionToModification(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	_, err := e.Store.Update(task.ID, func(saved *Task) error { saved.ReadOnly, saved.State = true, stateInspected; return nil })
	require.NoError(t, err)
	h := &HTTP{Engine: e, Token: "inspection-attachment-context"}
	metadata := flowUploadAttachment(t, h, "spec-from-question.txt", []byte("inspected specification"))
	_, err = e.submitMessage(task.ID, messageRequest{ID: "question", Kind: messageQuestion, Content: "Analyse the specification", AttachmentIDs: []string{metadata["id"].(string)}})
	require.NoError(t, err)
	request := messageRequest{ID: "implement", Kind: messageRequirement, Content: "Implement the specification from the previous question"}
	message, err := e.startLinkedBatch(task.ID, request)
	require.NoError(t, err)
	child, err := e.Store.Get(message.LinkedTask)
	require.NoError(t, err)
	require.Len(t, child.Attachments, 1)
	require.Equal(t, metadata["id"], child.Attachments[0].ID)
	require.NoError(t, e.validateAttachments(child.Attachments))
}

func TestAttachmentQuestionReferencesFreezeWhenWorkIsRequested(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "attachment-question-context"}
	first := flowUploadAttachment(t, h, "earlier-spec.txt", []byte("earlier specification"))
	_, err := e.submitMessage(task.ID, messageRequest{ID: "analyse-file", Kind: messageQuestion, Content: "Analyse this specification", AttachmentIDs: []string{first["id"].(string)}})
	require.NoError(t, err)
	request := messageRequest{ID: "implement-spec", Kind: messageRequirement, Content: "Implement the specification from my previous question"}
	_, err = e.submitMessage(task.ID, request)
	require.NoError(t, err)
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Empty(t, workAttachments(current), "new requirements must not leak into an already frozen invocation")
	current, err = e.Store.Update(task.ID, func(saved *Task) error { saved.AuditedRevision = saved.RequirementRevision; return nil })
	require.NoError(t, err)
	refs := workAttachments(current)
	require.Len(t, refs, 1, "an explicit modification request must retain the earlier question's file")
	require.Equal(t, first["id"], refs[0].ID)
	second := flowUploadAttachment(t, h, "later-question.txt", []byte("later question material"))
	_, err = e.submitMessage(task.ID, messageRequest{ID: "later-question", Kind: messageQuestion, Content: "Read this later", AttachmentIDs: []string{second["id"].(string)}})
	require.NoError(t, err)
	_, err = e.submitMessage(task.ID, request)
	require.NoError(t, err, "retrying the same requirement must retain its original context")
	current, err = e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, refs, workAttachments(current), "a later read-only question cannot change frozen work inputs")
}

func flowSetInitialAttachment(t *testing.T, e *Engine, id string, metadata map[string]any) Attachment {
	t.Helper()
	raw, err := json.Marshal(metadata)
	require.NoError(t, err)
	var attachment Attachment
	require.NoError(t, json.Unmarshal(raw, &attachment))
	_, err = e.Store.Update(id, func(task *Task) error { task.Attachments = []Attachment{attachment}; return nil })
	require.NoError(t, err)
	return attachment
}

// @covers AC-RELIABLE-ATTACHMENTS-001.1, AC-RELIABLE-ATTACHMENTS-001.4
func TestAttachmentFlowNewInspectionAndFollowupReceiveRealFiles(t *testing.T) {
	probe := &conversationProbe{}
	e, previous := fixtureEngine(t, probe, passingCheck)
	t.Cleanup(e.wg.Wait)
	_, err := e.Store.Update(previous.ID, func(task *Task) error { task.State = stateInspected; return nil })
	require.NoError(t, err)
	h := &HTTP{Engine: e, Token: "attachment-flow"}
	first := flowUploadAttachment(t, h, "需求说明.custom", []byte("original document bytes\x00"))
	body, err := json.Marshal(map[string]any{"project": previous.Project, "read_only": true, "attachment_ids": []string{first["id"].(string)}})
	require.NoError(t, err)
	response := httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks", string(body), h.Token)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	var created Task
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &created))
	require.Eventually(t, func() bool {
		task, loadErr := e.Store.Get(created.ID)
		return loadErr == nil && task.State == stateInspected
	}, 15*time.Second, 10*time.Millisecond)

	second := flowUploadAttachment(t, h, "后续说明.txt", []byte("follow-up file sentinel"))
	body, err = json.Marshal(map[string]any{"id": "question-with-file", "kind": messageQuestion, "attachment_ids": []string{second["id"].(string)}})
	require.NoError(t, err)
	response = httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks/"+created.ID+"/messages", string(body), h.Token)
	require.Equal(t, http.StatusAccepted, response.Code, response.Body.String())
	require.Eventually(t, func() bool {
		task, loadErr := e.Store.Get(created.ID)
		return loadErr == nil && len(task.Messages) == 1 && task.Messages[0].Status == "answered"
	}, 15*time.Second, 10*time.Millisecond)
	e.wg.Wait()
	probe.mu.Lock()
	defer probe.mu.Unlock()
	require.Len(t, probe.requests, 2)
	for i, expected := range []map[string]any{first, second} {
		require.Contains(t, probe.requests[i].Prompt, expected["sha256"])
		found := false
		for _, input := range probe.requests[i].Attachments {
			if input.ID == expected["id"] {
				found = true
				require.True(t, filepath.IsAbs(input.Path))
				require.True(t, pathInside(filepath.Join(e.Root, "attachments"), input.Path))
			}
		}
		require.True(t, found, "uploaded bytes were lost before the native runner")
	}
}

func flowUploadAttachment(t *testing.T, h *HTTP, name string, data []byte) map[string]any {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:38429/api/v1/reliable/attachments", &body)
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-XBD-Token", h.Token)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var attachment map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &attachment))
	return attachment
}
