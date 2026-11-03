package reliable

import "testing"

func TestDefaultPlanUsesBothNativeToolsWhenAvailable(t *testing.T) {
	e := Engine{Providers: testProviders()}
	want := ModelPlan{Auditor: "deepseek", Implementer: "codex", Reviewer: "deepseek"}
	if got := e.defaultPlan(); got != want {
		t.Fatalf("novice default does not start mixed native collaboration: %+v", got)
	}
	e.Providers = []Provider{{ID: "codex", Available: true}}
	if got := e.defaultPlan(); got != (ModelPlan{Auditor: "codex", Implementer: "codex", Reviewer: "codex"}) {
		t.Fatalf("single-tool fallback selected an unavailable connection: %+v", got)
	}
}
