package reliable

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func sharedContextFixture(t *testing.T) (*Engine, *Task) {
	t.Helper()
	root := t.TempDir()
	store, err := OpenStore(filepath.Join(root, "state.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	project := filepath.Join(root, "project")
	require.NoError(t, os.Mkdir(project, 0700))
	task := &Task{ID: "context-task", Project: project, Goal: "Original complete requirement", State: stateInspected}
	require.NoError(t, store.Create(task))
	return &Engine{Root: root, Store: store}, task
}

func sharedContextIndexFor(t *testing.T, shared *SharedContext) sharedContextIndex {
	t.Helper()
	raw, err := os.ReadFile(shared.Reference.IndexPath)
	require.NoError(t, err)
	var index sharedContextIndex
	require.NoError(t, json.Unmarshal(raw, &index))
	return index
}

func sharedContextContents(t *testing.T, e *Engine, index sharedContextIndex) string {
	t.Helper()
	var result strings.Builder
	for _, batch := range index.Batches {
		for _, hash := range append([]sharedContextChunk{{SHA256: batch.HeaderSHA256}}, batch.Chunks...) {
			raw, err := os.ReadFile(filepath.Join(e.Root, "context", "blobs", hash.SHA256+".json"))
			require.NoError(t, err)
			result.Write(raw)
		}
	}
	return result.String()
}

func sharedContextFileCount(t *testing.T, dir string) int {
	t.Helper()
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	return len(files)
}

func TestSharedContextExcerptLabelsOmittedHistoryAndRequirements(t *testing.T) {
	task := &Task{ID: "task", Goal: "original requirement"}
	for i := range 12 {
		task.Messages = append(task.Messages, TaskMessage{ID: fmt.Sprint(i), Kind: messageQuestion, Content: "question", Status: "answered", Reply: &Audit{Summary: "model claim"}})
	}
	task.Messages = append(task.Messages, TaskMessage{ID: "requirement", Kind: messageRequirement, Content: "unanswered requirement", Status: stateQueued})
	var compact map[string]any
	require.NoError(t, json.Unmarshal(conversationContext(task, ""), &compact))
	require.Equal(t, float64(13), compact["recorded_messages_total"])
	require.Equal(t, true, compact["dialogue_excerpted"], "a bounded prompt must not pretend it contains the complete history")
	require.Equal(t, float64(1), compact["requirements_total"])
}

func TestSharedContextRetainsCompleteEarlyHistoryRequirementsAndAttachments(t *testing.T) {
	e, task := sharedContextFixture(t)
	attachment, err := e.storeAttachment(context.Background(), "earlier-notes.txt", bytes.NewBufferString("original reference bytes"), func() error { return nil })
	require.NoError(t, err)
	for i := range 15 {
		task.Messages = append(task.Messages, TaskMessage{ID: fmt.Sprint(i), Kind: messageQuestion, Content: fmt.Sprintf("question-%d", i), Status: "answered", Reply: &Audit{Summary: fmt.Sprintf("unverified-claim-%d", i), Facts: []string{"reported, not independently confirmed"}}})
	}
	task.Messages[0].Content = strings.Repeat("earlier detailed constraint ", 1000) + "EARLIEST-SUFFIX"
	task.Messages[0].Attachments = []Attachment{*attachment}
	task.Messages = append(task.Messages, TaskMessage{ID: "unanswered", Kind: messageRequirement, Content: "UNANSWERED-REQUIREMENT", Status: stateQueued}, TaskMessage{ID: "current", Kind: messageQuestion, Content: "Continue with Codex", Status: stateQueued})
	before, _ := json.Marshal(task)
	shared, err := e.buildSharedContext(context.Background(), task, "current")
	require.NoError(t, err)
	require.True(t, shared.Reference.Complete)
	require.Equal(t, 17, shared.Reference.MessageCount)
	require.Less(t, len(shared.Prompt), 24<<10)
	index := sharedContextIndexFor(t, shared)
	contents := sharedContextContents(t, e, index)
	for _, text := range []string{"EARLIEST-SUFFIX", "UNANSWERED-REQUIREMENT", "unverified-claim-0", "Continue with Codex", attachment.ID} {
		require.Contains(t, contents, text)
	}
	require.Contains(t, index.Trust, "not contain native private")
	require.Len(t, index.Attachments, 1)
	require.Equal(t, filepath.Join(e.Root, "attachments", attachment.ID, "content"), index.Attachments[0].Path)
	require.Equal(t, attachment.SHA256, index.Attachments[0].SHA256)
	after, _ := json.Marshal(task)
	require.Equal(t, string(before), string(after), "context preparation must not mutate the caller's task")
	count := sharedContextFileCount(t, filepath.Join(e.Root, "context", "blobs"))
	repeated, err := e.buildSharedContext(context.Background(), task, "current")
	require.NoError(t, err)
	require.Equal(t, shared.Reference, repeated.Reference)
	require.Equal(t, count, sharedContextFileCount(t, filepath.Join(e.Root, "context", "blobs")), "unchanged history must reuse content-addressed files")
}

func TestSharedContextParentHistoryIsCompleteAndInvalidChainsFailClosed(t *testing.T) {
	for _, kind := range []string{"valid", "missing", "cycle", "different-project"} {
		t.Run(kind, func(t *testing.T) {
			e, task := sharedContextFixture(t)
			parent := &Task{ID: "parent", Project: task.Project, Goal: "EARLY-PARENT-GOAL", State: stateAccepted, Messages: []TaskMessage{{ID: "parent-message", Kind: messageRequirement, Content: "EARLY-PARENT-CONSTRAINT", Status: "applied"}}}
			if kind == "different-project" {
				parent.Project = t.TempDir()
			}
			require.NoError(t, e.Store.Create(parent))
			task.ParentTaskID = parent.ID
			if kind == "missing" {
				task.ParentTaskID = "absent"
			}
			if kind == "cycle" {
				_, err := e.Store.Update(parent.ID, func(saved *Task) error { saved.ParentTaskID = task.ID; return nil })
				require.NoError(t, err)
			}
			shared, err := e.buildSharedContext(context.Background(), task, "")
			if kind != "valid" {
				require.ErrorContains(t, err, "shared_context_history")
				require.Nil(t, shared)
				_, statErr := os.Stat(filepath.Join(e.Root, "context"))
				require.True(t, os.IsNotExist(statErr))
				return
			}
			require.NoError(t, err)
			require.Equal(t, 2, shared.Reference.BatchCount)
			require.Contains(t, sharedContextContents(t, e, sharedContextIndexFor(t, shared)), "EARLY-PARENT-CONSTRAINT")
		})
	}
}

func TestSharedContextCutoffAndReadOnlyStatusNeverInventPreparation(t *testing.T) {
	e, task := sharedContextFixture(t)
	task.Messages = []TaskMessage{{ID: "current", Kind: messageQuestion, Content: "CURRENT", Status: stateQueued}, {ID: "future", Kind: messageQuestion, Content: "FUTURE-MESSAGE", Status: stateQueued}}
	_, err := e.Store.Update(task.ID, func(saved *Task) error { saved.Messages = task.Messages; return nil })
	require.NoError(t, err)
	status, err := e.SharedContextStatus(task.ID)
	require.NoError(t, err)
	require.False(t, status.Prepared)
	require.Equal(t, 2, status.SavedMessages)
	_, err = os.Stat(filepath.Join(e.Root, "context"))
	require.True(t, os.IsNotExist(err), "a status request must not create archive directories")
	_, err = e.buildSharedContext(context.Background(), task, "missing")
	require.ErrorContains(t, err, "cutoff")
	shared, err := e.buildSharedContext(context.Background(), task, "current")
	require.NoError(t, err)
	require.NotContains(t, sharedContextContents(t, e, sharedContextIndexFor(t, shared)), "FUTURE-MESSAGE")
	status, err = e.SharedContextStatus(task.ID)
	require.NoError(t, err)
	require.True(t, status.Prepared)
	require.Equal(t, 1, status.ArchivedMessages)
	require.Equal(t, 2, status.SavedMessages)
}

func TestSharedContextRejectsUnavailableHistoricalFilesMalformedRecordsAndUnsafeStorage(t *testing.T) {
	t.Run("missing-attachment", func(t *testing.T) {
		e, task := sharedContextFixture(t)
		task.Attachments = []Attachment{{ID: "att-aaaaaaaaaaaaaaaaaaaaaaaaaa", SHA256: strings.Repeat("a", 64)}}
		_, err := e.buildSharedContext(context.Background(), task, "")
		require.ErrorContains(t, err, "shared_context_attachments")
	})
	t.Run("malformed-history", func(t *testing.T) {
		e, task := sharedContextFixture(t)
		task.PreviousContext = json.RawMessage(`{not-json`)
		_, err := e.buildSharedContext(context.Background(), task, "")
		require.ErrorContains(t, err, "malformed retained context")
	})
	t.Run("unsafe-storage", func(t *testing.T) {
		e, task := sharedContextFixture(t)
		require.NoError(t, os.WriteFile(filepath.Join(e.Root, "context"), []byte("preserve me"), 0600))
		_, err := e.buildSharedContext(context.Background(), task, "")
		require.ErrorContains(t, err, "shared_context_storage")
		raw, readErr := os.ReadFile(filepath.Join(e.Root, "context"))
		require.NoError(t, readErr)
		require.Equal(t, "preserve me", string(raw))
	})
}
