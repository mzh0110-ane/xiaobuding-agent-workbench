package reliable

import (
	"os"
	"testing"
)

func TestInstalledNativeCodexDiscovery(t *testing.T) {
	configured := os.Getenv("XIAOBUDING_DISCOVERY_CONFIGURED")
	expected := os.Getenv("XIAOBUDING_DISCOVERY_EXPECTED")
	if configured == "" || expected == "" {
		t.Skip("explicit installed Codex discovery fixture is not configured")
	}
	path, err := resolveCodexExecutable(configured)
	if err != nil || path != expected {
		t.Fatalf("installed native update discovery: got %q, want %q, error %v", path, expected, err)
	}
	t.Log("verified installed native Codex selection without starting a model or changing credentials")
}
