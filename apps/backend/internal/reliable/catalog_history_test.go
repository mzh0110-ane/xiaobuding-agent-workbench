package reliable

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func historySettings(t *testing.T) (string, Settings) {
	t.Helper()
	home := t.TempDir()
	cli, _ := os.Executable()
	settings := Settings{DataRoot: filepath.Join(home, "data"), CodexPath: cli, CodexHome: home, Model: "first", Connections: []ConnectionSettings{{ID: "codex", Kind: "codex", Home: home, Provider: "openai", Model: "first"}}}
	for name, data := range map[string]string{"config.toml": "model='first'", "auth.json": `{"auth_mode":"chatgpt"}`, "models_cache.json": `{"private_identity":"never-copy-this","models":[{"slug":"first","display_name":"Native first","visibility":"list","default_reasoning_level":"low","supported_reasoning_levels":[{"effort":"low"},{"effort":"high"}]}]}`} {
		if err := os.WriteFile(filepath.Join(home, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return home, settings
}

func openHistoryFixture(t *testing.T, home string, settings Settings) *HTTP {
	t.Helper()
	raw, _ := json.Marshal(settings)
	if err := os.WriteFile(filepath.Join(home, "reliable-settings.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	h, err := FromHome(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Engine.Close() })
	return h
}

func TestNativeModelHistorySurvivesCatalogOmissionAndYieldsToFreshMetadata(t *testing.T) {
	home, settings := historySettings(t)
	h := openHistoryFixture(t, home, settings)
	if err := h.Engine.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "models_cache.json"), []byte(`{"models":[{"slug":"other","visibility":"list"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	h = openHistoryFixture(t, home, settings)
	p, _, effort, err := h.Engine.resolveSelection("codex", "first", "high")
	if err != nil || effort != "high" || p.CatalogSource != "codex_cache_history" || !strings.Contains(p.CatalogStatus, "历史") {
		t.Fatalf("native metadata was lost or mislabeled after restart: %+v %s %v", p, effort, err)
	}
	_ = h.Engine.Close()
	raw, err := os.ReadFile(filepath.Join(settings.DataRoot, "native-model-metadata.json"))
	if err != nil || bytes.Contains(raw, []byte("never-copy-this")) {
		t.Fatal("native metadata cache is missing or contains unrelated private data")
	}
	if err = os.WriteFile(filepath.Join(home, "models_cache.json"), []byte(`{"models":[{"slug":"first","visibility":"list","supported_reasoning_levels":[{"effort":"ultra"}]}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	h = openHistoryFixture(t, home, settings)
	defer func() { _ = h.Engine.Close() }()
	if _, _, _, err = h.Engine.resolveSelection("codex", "first", "high"); err == nil {
		t.Fatal("historical capability overrode newer native metadata")
	}
	if _, _, _, err = h.Engine.resolveSelection("codex", "first", "ultra"); err != nil {
		t.Fatal(err)
	}
	_ = h.Engine.Close()
	if err = os.WriteFile(filepath.Join(home, "models_cache.json"), []byte(`{"models":[{"slug":"other","visibility":"list"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	h = openHistoryFixture(t, home, settings)
	if _, _, _, err = h.Engine.resolveSelection("codex", "first", "ultra"); err != nil {
		t.Fatal("updated native capabilities were not preserved across another restart")
	}
	if _, _, _, err = h.Engine.resolveSelection("codex", "first", "high"); err == nil {
		t.Fatal("an old capability survived replacement of its native observation")
	}
}

func TestNativeModelHistoryDoesNotCrossHomesOrInventMissingCapabilities(t *testing.T) {
	home, settings := historySettings(t)
	h := openHistoryFixture(t, home, settings)
	_ = h.Engine.Close()
	other, _ := historySettings(t)
	if err := os.WriteFile(filepath.Join(other, "models_cache.json"), []byte(`{"models":[{"slug":"other","visibility":"list"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	settings.Connections[0].Home = other
	h = openHistoryFixture(t, home, settings)
	defer func() { _ = h.Engine.Close() }()
	if _, _, _, err := h.Engine.resolveSelection("codex", "first", "high"); err == nil {
		t.Fatal("capabilities crossed native home identities")
	}
}
