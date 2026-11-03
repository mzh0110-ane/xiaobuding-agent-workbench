package reliable

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSharedContextProviderSwitchKeepsOriginalConversationAndAnchorsInput(t *testing.T) {
	probe := &conversationProbe{}
	e, task := fixtureEngine(t, probe, passingCheck)
	e.Providers = []Provider{{ID: connectionHarness, Model: "deepseek-fixture", Available: true}, {ID: connectionCodex, Model: "codex-fixture", Available: true}}
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.State, saved.ReadOnly = stateInspected, true
		saved.ModelPlan = ModelPlan{Auditor: connectionHarness}
		for i := range 12 {
			saved.Messages = append(saved.Messages, TaskMessage{ID: string(rune('a' + i)), Kind: messageQuestion, Content: "EARLY-USER-CONSTRAINT", Status: "answered", Reply: &Audit{Summary: "prior model opinion"}})
		}
		saved.Messages = append(saved.Messages, TaskMessage{ID: "first", Kind: messageQuestion, Content: "Please explain using DeepSeek", Status: stateQueued})
		return nil
	})
	require.NoError(t, err)
	task, err = e.Store.Get(task.ID)
	require.NoError(t, err)
	require.NoError(t, e.answerQuestion(context.Background(), task, &task.Messages[len(task.Messages)-1]))
	_, err = e.Store.Update(task.ID, func(saved *Task) error {
		saved.ModelPlan.Auditor = connectionCodex
		saved.Messages = append(saved.Messages, TaskMessage{ID: "second", Kind: messageQuestion, Content: "Continue using Codex", Status: stateQueued})
		return nil
	})
	require.NoError(t, err)
	task, err = e.Store.Get(task.ID)
	require.NoError(t, err)
	require.NoError(t, e.answerQuestion(context.Background(), task, &task.Messages[len(task.Messages)-1]))
	require.Len(t, probe.requests, 2)
	require.Equal(t, connectionHarness, probe.requests[0].ProviderID)
	require.Equal(t, connectionCodex, probe.requests[1].ProviderID)
	require.NotEqual(t, probe.requests[0].OutDir, probe.requests[1].OutDir)
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	message := current.Messages[len(current.Messages)-1]
	raw, err := os.ReadFile(filepath.Join(message.Evidence, "input.json"))
	require.NoError(t, err)
	var input struct {
		SharedContext SharedContextReference `json:"shared_context"`
	}
	require.NoError(t, json.Unmarshal(raw, &input))
	require.Equal(t, message.ContextHash, input.SharedContext.IndexSHA256)
	index := sharedContextIndexFor(t, &SharedContext{Reference: input.SharedContext})
	contents := sharedContextContents(t, e, index)
	require.Contains(t, contents, "EARLY-USER-CONSTRAINT")
	require.Contains(t, contents, "Please explain using DeepSeek")
	require.Contains(t, contents, "Source-based answer")
	require.Contains(t, contents, "Continue using Codex")
	require.Contains(t, probe.requests[1].Prompt, "not be listed as project-relative read_files")
	require.Equal(t, stateInspected, current.State)
	require.Zero(t, current.Round)
}

func TestSharedContextStorageFailureStopsQuestionBeforeAgentRuns(t *testing.T) {
	probe := &conversationProbe{}
	e, task := fixtureEngine(t, probe, passingCheck)
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.State, saved.ReadOnly = stateInspected, true
		saved.Messages = []TaskMessage{{ID: "question", Kind: messageQuestion, Content: "continue", Status: stateQueued}}
		return nil
	})
	require.NoError(t, err)
	task, err = e.Store.Get(task.ID)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(e.Root, "context"), []byte("foreign file"), 0600))
	require.Error(t, e.answerQuestion(context.Background(), task, &task.Messages[0]))
	require.Empty(t, probe.requests, "missing context cannot be replaced by a context-free model call")
}

func TestSharedContextImmutableBytesBlockBothSealingAndAcceptance(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	require.NoError(t, e.RunTask(context.Background(), task.ID))
	task, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Equal(t, stateAwaitingAcceptance, task.State)
	require.NotEmpty(t, task.Runs[0].ContextHash)
	ref := SharedContextReference{IndexPath: filepath.Join(e.Root, "context", "indices", task.Runs[0].ContextHash+".json")}
	index := sharedContextIndexFor(t, &SharedContext{Reference: ref})
	file := filepath.Join(e.Root, "context", "blobs", index.Batches[0].HeaderSHA256+".json")
	require.NoError(t, os.WriteFile(file, []byte(`{"forged_context":true}`), 0600))
	_, err = e.evidenceDigestSnapshot(task)
	require.ErrorContains(t, err, "shared_context_changed")
	require.ErrorContains(t, e.sealReview(task.ID, Pass, "must not reseal modified context", task.RequirementRevision), "shared_context_changed")
	require.Error(t, e.Accept(context.Background(), task.ID))
	original, err := os.ReadFile(filepath.Join(task.Project, "app.txt"))
	require.NoError(t, err)
	require.Equal(t, "before\n", string(original))
}

func TestSharedContextDispatchDatabaseAnchorRejectsAReplacedValidIndex(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	require.NoError(t, e.RunTask(context.Background(), task.ID))
	task, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	first, last := task.Runs[0], task.Runs[len(task.Runs)-1]
	require.NotEqual(t, first.ContextHash, last.ContextHash)
	readRef := func(name string) map[string]json.RawMessage {
		raw, readErr := os.ReadFile(filepath.Join(e.Root, "evidence", task.ID, name))
		require.NoError(t, readErr)
		var data map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw, &data))
		return data
	}
	data, replacement := readRef(first.InputEvidence), readRef(last.InputEvidence)
	data["shared_context"] = replacement["shared_context"]
	raw, err := json.Marshal(data)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(e.Root, "evidence", task.ID, first.InputEvidence), raw, 0600))
	_, err = e.evidenceDigestSnapshot(task)
	require.ErrorContains(t, err, "controller's dispatch record")
}
