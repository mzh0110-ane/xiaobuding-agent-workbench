package reliable

import "testing"

func TestCodexDefaultsToFullAccessForEveryRole(t *testing.T) {
	runner := &CodexRunner{Model: "configured-model", Provider: "openai"}
	for _, role := range []string{roleAuditor, roleImplementer, roleReviewer} {
		args := runner.arguments(RoleRequest{Role: role, Directory: "project"}, "schema.json", "response.json")
		values := map[string]string{}
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "-s" || args[i] == "-c" || args[i] == "-C" {
				values[args[i]] = args[i+1]
			}
		}
		if values["-s"] != "danger-full-access" || values["-C"] != "project" {
			t.Fatalf("%s does not receive the native full-access policy: %v", role, args)
		}
		foundApproval := false
		for _, arg := range args {
			foundApproval = foundApproval || arg == `approval_policy="never"`
		}
		if !foundApproval {
			t.Fatalf("%s unexpectedly requires interactive approvals", role)
		}
	}
}
