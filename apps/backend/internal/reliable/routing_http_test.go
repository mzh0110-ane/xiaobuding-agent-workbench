package reliable

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestModelPlanUpdatesAreVersionedAndKeepTheSameTask(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "capability", Model: "deepseek-flash"}
	path := "/api/v1/reliable/tasks/" + task.ID + "/models"
	body := `{"expected_revision":0,"plan":{"auditor":"deepseek","implementer":"deepseek","reviewer":"deepseek"}}`
	w := httpRequest(h, "POST", path, body, h.Token)
	if w.Code != http.StatusOK {
		t.Fatalf("stage model plan could not be saved: %d %s", w.Code, w.Body)
	}
	var saved map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved["id"] != task.ID || saved["state"] != "queued" || saved["round"] != float64(0) || saved["plan_revision"] != float64(1) {
		t.Fatalf("model selection replaced task progress: %v", saved)
	}
	if got := httpRequest(h, "POST", path, body, h.Token); got.Code != http.StatusConflict {
		t.Fatalf("stale browser overwrote the plan: %d", got.Code)
	}
	bad := `{"expected_revision":1,"plan":{"auditor":"invented","implementer":"deepseek","reviewer":"deepseek"}}`
	if got := httpRequest(h, "POST", path, bad, h.Token); got.Code != http.StatusConflict {
		t.Fatalf("unavailable provider accepted: %d", got.Code)
	}
}
