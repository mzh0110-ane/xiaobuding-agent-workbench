package reliable

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAuthenticatedCatalogRefreshLoadsNewNativeOptions(t *testing.T) {
	home := t.TempDir()
	cli, _ := os.Executable()
	settings := Settings{DataRoot: filepath.Join(home, "data"), CodexPath: cli, CodexHome: home, Model: "first", Connections: []ConnectionSettings{{ID: "codex", Kind: "codex", Home: home, Provider: "openai", Model: "first"}}}
	raw, _ := json.Marshal(settings)
	for name, data := range map[string][]byte{"reliable-settings.json": raw, "config.toml": []byte("model='first'"), "auth.json": []byte(`{"auth_mode":"chatgpt"}`), "models_cache.json": []byte(`{"models":[{"slug":"first","visibility":"list"}]}`)} {
		if err := os.WriteFile(filepath.Join(home, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	h, err := FromHome(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Engine.Close() })
	if err = os.WriteFile(filepath.Join(home, "models_cache.json"), []byte(`{"models":[{"slug":"first","visibility":"list"},{"slug":"second","visibility":"list"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	response := httpRequest(h, "POST", "/api/v1/reliable/models/refresh", `{}`, h.Token)
	if response.Code != 200 {
		t.Fatalf("native catalog refresh is unavailable: %d %s", response.Code, response.Body.String())
	}
	p, _ := h.Engine.provider("codex")
	if len(p.Models) != 2 {
		t.Fatalf("refresh left stale native options: %+v", p.Models)
	}
	if forbidden := httpRequest(h, "POST", "/api/v1/reliable/models/refresh", `{}`, ""); forbidden.Code != 403 {
		t.Fatal("catalog refresh bypassed explicit local action")
	}
}
