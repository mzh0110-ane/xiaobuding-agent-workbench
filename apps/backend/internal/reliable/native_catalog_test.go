package reliable

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCodexConnectionProjectsItsVisibleNativeModelsAndEfforts(t *testing.T) {
	home := t.TempDir()
	for name, data := range map[string]string{
		"config.toml":       "model = \"first\"\n",
		"auth.json":         `{"auth_mode":"chatgpt"}`,
		"models_cache.json": `{"identity":"private-cache-identity","models":[{"slug":"second","display_name":"Second","visibility":"list","priority":2,"default_reasoning_level":"medium","supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"}]},{"slug":"hidden","visibility":"hide","priority":0},{"slug":"first","display_name":"First","visibility":"list","priority":1,"default_reasoning_level":"low","supported_reasoning_levels":[{"effort":"low"},{"effort":"high"}]}]}`,
	} {
		if err := os.WriteFile(filepath.Join(home, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	settings := Settings{CodexPath: "unused-cli", Connections: []ConnectionSettings{{ID: "codex", Kind: "codex", Home: home, Provider: "openai", Model: "first"}}}
	providers, _, _, err := buildConnections(settings)
	if err != nil {
		t.Fatal(err)
	}
	if len(providers[0].Models) != 2 || providers[0].Models[0].ID != "first" || providers[0].Models[1].ID != "second" || providers[0].Models[0].DefaultEffort != "low" {
		t.Fatalf("native catalog was missing or incorrectly projected: %+v", providers)
	}
	raw, _ := json.Marshal(providers)
	for _, secret := range []string{"private-cache-identity", "auth_mode", "supported_reasoning_levels"} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatalf("native metadata leaked instead of projecting options: %s", secret)
		}
	}
}
