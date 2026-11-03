package reliable

import (
	"encoding/json"
	"testing"
)

func TestRoleSchemasAreValidProviderJSON(t *testing.T) {
	for _, role := range []string{"auditor", "implementer", "reviewer"} {
		if !json.Valid([]byte(schemaFor(role))) {
			t.Errorf("%s schema is invalid JSON at the provider boundary", role)
		}
	}
}
