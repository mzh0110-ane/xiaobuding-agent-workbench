package reliable

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestProjectLedgerPreservesRequirementsAcrossBatches(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "project-ledger-fixture"}
	url := "/api/v1/reliable/tasks/" + task.ID + "/project"
	response := httpRequest(h, http.MethodGet, url, "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("long project has no persistent ledger: %d %s", response.Code, response.Body.String())
	}
	var before map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &before); err != nil {
		t.Fatal(err)
	}
	response = httpRequest(h, http.MethodPost, url, `{"expected_revision":0,"items":[{"title":"Export a PDF","acceptance":"A real PDF opens and includes the selected questions"}],"memory":{"kind":"constraint","text":"Keep all original question resources"}}`, h.Token)
	if response.Code != http.StatusOK {
		t.Fatalf("could not add a project requirement: %d %s", response.Code, response.Body.String())
	}
	response = httpRequest(h, http.MethodGet, url, "", "")
	var after struct {
		Revision int                                          `json:"revision"`
		Items    []struct{ Title, Status, Acceptance string } `json:"items"`
		Memories []struct {
			Text     string
			Verified bool
		} `json:"memories"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &after); err != nil {
		t.Fatal(err)
	}
	if after.Revision != 1 || len(after.Items) < 2 || len(after.Memories) != 1 {
		t.Fatalf("project knowledge was not retained: %+v", after)
	}
	found := false
	for _, item := range after.Items {
		if item.Title == "Export a PDF" {
			found = item.Status == "planned" && item.Acceptance != ""
		}
	}
	if !found || after.Memories[0].Verified {
		t.Fatalf("user requirements were incorrectly represented as verified work: %+v", after)
	}
	response = httpRequest(h, http.MethodPost, url, `{"expected_revision":0,"items":[{"title":"Stale write"}]}`, h.Token)
	if response.Code < 400 {
		t.Fatal("a stale project edit silently replaced the latest requirements")
	}
}
