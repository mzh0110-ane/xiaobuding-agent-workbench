package reliable

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFrontendDemoUsesFrontendGoalAndChecks(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	e.Providers = testProviders()
	t.Cleanup(func() { _ = e.Close() })
	h := &HTTP{Engine: e, Token: "cap", DemoProject: fixtureRepo(t)}
	body := `{"model_plan":{"mode":"frontend","auditor":"codex","reviewer":"codex","frontend":"deepseek","backend":"codex"}}`
	response := httpRequest(h, "POST", "/api/v1/reliable/demo", body, h.Token)
	if response.Code != 202 {
		t.Fatalf("domain demo not accepted: %d %s", response.Code, response.Body.String())
	}
	var task Task
	_ = json.Unmarshal(response.Body.Bytes(), &task)
	if !strings.Contains(task.Goal, "前端") || len(task.Commands) != 1 || !strings.Contains(task.Commands[0], "node --test") || len(task.AllowedPaths) != 1 || task.AllowedPaths[0] != "frontend/app.js" {
		t.Fatalf("frontend demo still points at the old backend-only example: %+v", task)
	}
}
