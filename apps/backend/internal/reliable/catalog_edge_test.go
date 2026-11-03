package reliable

import (
	"fmt"
	"testing"
)

func TestHarnessCatalogRejectsExplicitNullRatherThanTreatingItAsOmitted(t *testing.T) {
	for _, field := range []string{"input_modalities", "output_modalities", "context_window", "max_output_tokens", "client_requirements"} {
		t.Run(field, func(t *testing.T) {
			raw := fmt.Sprintf(`{"id":"deepseek-flash","api_capabilities":{"anthropic_messages":{}},%q:null}`, field)
			if _, ok := parseHarnessModel([]byte(raw)); ok {
				t.Fatalf("native catalog accepted explicit null for %s", field)
			}
		})
	}
}

func TestUnverifiedNativeAutomaticEffortDoesNotInventMax(t *testing.T) {
	patch := harnessPatch(RoleRequest{Role: roleImplementer, WorkMode: "frontend"}, "deepseek-account", "deepseek-flash")
	for _, item := range patch {
		if item["id"] == "agent-default-model" {
			config := item["config"].(map[string]any)
			if _, exists := config["reasoningEffort"]; exists {
				t.Fatalf("unverified automatic setting invented an explicit effort: %v", config)
			}
		}
	}
}
