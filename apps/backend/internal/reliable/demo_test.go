package reliable

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestOneClickSampleUsesFreshProjectsAndRetainsModelPlan(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	t.Cleanup(func() { _ = e.Close() })
	h := &HTTP{Engine: e, Token: "capability", DemoProject: fixtureRepo(t)}
	body := `{"model_plan":{"auditor":"deepseek","implementer":"deepseek","reviewer":"deepseek"}}`
	projects := make(map[string]bool)
	for range 2 {
		w := httpRequest(h, "POST", "/api/v1/reliable/demo", body, h.Token)
		if w.Code != http.StatusAccepted {
			t.Fatalf("one-click sample did not create a fresh task: %d %s", w.Code, w.Body)
		}
		var task Task
		if err := json.Unmarshal(w.Body.Bytes(), &task); err != nil {
			t.Fatal(err)
		}
		if task.Project == h.DemoProject || projects[task.Project] || task.PlanRevision != 1 || task.ModelPlan.Implementer != "deepseek" {
			t.Fatalf("sample reused its template or lost assignments: %+v", task)
		}
		projects[task.Project] = true
	}
}
