package reliable

import (
	"strings"
	"testing"
)

func TestConfiguredModelSurvivesAnOutdatedNativeCatalog(t *testing.T) {
	p := Provider{Model: "configured-new-model"}
	models := []ModelOption{{ID: "another-model", Name: "Another", Efforts: []string{"low"}}}
	setModelCatalog(&p, models, "codex_cache", nil)
	if p.Model != "configured-new-model" {
		t.Fatalf("an outdated catalog silently replaced the configured model: %s", p.Model)
	}
	if len(p.Models) != 2 || p.Models[0].ID != p.Model || len(p.Models[0].Efforts) != 0 || p.Models[0].DefaultEffort != "" {
		t.Fatalf("the configured option was missing or had invented capabilities: %+v", p.Models)
	}
	if !strings.Contains(p.CatalogStatus, "未列出") || !strings.Contains(p.Models[0].Name, "未验证") {
		t.Fatal("the unlisted model was presented as a verified catalog entry")
	}
	setModelCatalog(&p, append([]ModelOption{{ID: p.Model, Name: "Now listed", Efforts: []string{"high"}}}, models...), "codex_cache", nil)
	if len(p.Models) != 2 || p.Models[0].Name != "Now listed" || len(p.Models[0].Efforts) != 1 {
		t.Fatal("a refreshed native catalog retained the stale placeholder")
	}
}
