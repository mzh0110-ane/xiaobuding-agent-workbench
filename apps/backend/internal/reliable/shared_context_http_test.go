package reliable

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSharedContextHTTPStatusReadsSavedHistoryWithoutPreparingOrRunning(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.Messages = []TaskMessage{{ID: "earlier-message", Kind: messageQuestion, Content: "Keep this early project context", Status: "answered"}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := e.Store.Get(task.ID)
	h := &HTTP{Engine: e, Token: "context-status-test"}
	url := "/api/v1/reliable/tasks/" + task.ID + "/context/status"
	response := httpRequest(h, http.MethodGet, url, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("context status unavailable: %d %s", response.Code, response.Body.String())
	}
	var status SharedContextStatus
	if err = json.Unmarshal(response.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.TaskID != task.ID || status.Prepared || status.Complete || status.SavedBatches != 1 || status.SavedMessages != 1 {
		t.Fatalf("status misrepresented retained history: %+v", status)
	}
	if _, err = os.Stat(filepath.Join(e.Root, "context")); !os.IsNotExist(err) {
		t.Fatal("reading context status created archive files")
	}
	after, _ := e.Store.Get(task.ID)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("reading shared context changed task state")
	}
	response = httpRequest(h, http.MethodPost, url, "{}", h.Token)
	if response.Code != http.StatusNotFound {
		t.Fatalf("unsupported mutation exposed: %d", response.Code)
	}
}
