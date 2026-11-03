package reliable

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelConnectionsReuseSeparateHomesAndExposeNoConfigurationSecrets(t *testing.T) {
	root := t.TempDir()
	cli, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	homes := []string{filepath.Join(root, "official"), filepath.Join(root, "custom")}
	config := []byte("model = 'fixture'\nexperimental_bearer_token = 'fixture-private-credential'\n")
	for _, home := range homes {
		if err = os.MkdirAll(home, 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(home, "config.toml"), config, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err = os.WriteFile(filepath.Join(homes[0], "auth.json"), []byte(`{"auth_mode":"chatgpt"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(homes[1], ".credentials.yaml"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(root, "harness.js")
	if err = os.WriteFile(entry, []byte("// native fixture entry"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := map[string]any{"data_root": filepath.Join(root, "state"), "codex_path": cli, "codex_home": homes[1], "model": "custom-model", "connections": []map[string]string{
		{"id": "codex", "name": "Codex（ChatGPT 登录）", "kind": "codex", "home": homes[0], "provider": "openai", "model": "official-model"},
		{"id": "deepseek", "name": "DeepSeek Harness", "kind": "harness", "executable": cli, "entry": entry, "home": homes[1], "provider": "deepseek-official", "model": "custom-model"},
	}}
	raw, _ := json.Marshal(settings)
	if err = os.WriteFile(filepath.Join(root, "reliable-settings.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	h, err := FromHome(root)
	if err != nil {
		t.Fatalf("independent model connections were not accepted: %v", err)
	}
	t.Cleanup(func() { _ = h.Engine.Close() })
	for i, id := range []string{"codex", "deepseek"} {
		p, lookupErr := h.Engine.provider(id)
		if lookupErr != nil || p.Model != []string{"official-model", "custom-model"}[i] {
			t.Fatalf("connection identity was lost: %+v %v", p, lookupErr)
		}
		after, readErr := os.ReadFile(filepath.Join(homes[i], "config.toml"))
		if readErr != nil || !bytes.Equal(config, after) {
			t.Fatal("connection discovery rewrote an existing configuration")
		}
	}
	w := httpRequest(h, "GET", "/api/v1/reliable/config", "", "")
	if strings.Contains(w.Body.String(), "fixture-private-credential") || strings.Contains(w.Body.String(), "config.toml") {
		t.Fatal("private connection configuration escaped through the browser API")
	}
}
