package reliable

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type catalogTransport func(*http.Request) (*http.Response, error)

func (f catalogTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestHarnessCatalogUsesNativeGrantAndFiltersUnsupportedModels(t *testing.T) {
	home := t.TempDir()
	executable, _ := os.Executable()
	credential := "version: 1\nrecords:\n  deepseek-account-platform/default:\n    kind: grant\n    payload:\n      version: 1\n      token: fixture-native-private-token\n      issuer: https://platform.deepseek.com\n"
	if err := os.WriteFile(filepath.Join(home, ".credentials.yaml"), []byte(credential), 0600); err != nil {
		t.Fatal(err)
	}
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	called := false
	http.DefaultTransport = catalogTransport(func(req *http.Request) (*http.Response, error) {
		called = true
		if req.URL.String() != "https://api.deepseek.com/models" || req.Header.Get("x-dsh-auth-token") != "fixture-native-private-token" {
			t.Fatal("catalog request did not preserve the native account boundary")
		}
		body := `{"data":[{"id":"deepseek-flash","name":"Native Flash","api_capabilities":{"anthropic_messages":{}}},{"id":"deepseek-v4-pro","api_capabilities":{"anthropic_messages":{}}},{"id":"unknown-no-version","api_capabilities":{"anthropic_messages":{}}},{"id":"future","api_capabilities":{"anthropic_messages":{}},"client_requirements":{"deepseek-harness":{"min_version":"999.0.0"}}},{"id":"compatible-new","api_capabilities":{"anthropic_messages":{}},"client_requirements":{"deepseek-harness":{"min_version":"0.2.0"}}},{"id":"deepseek-flash-no-capabilities"}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	settings := Settings{Connections: []ConnectionSettings{{ID: "deepseek", Kind: "harness", Home: home, Provider: "deepseek-account", Executable: executable, Entry: executable, Model: "deepseek-flash"}}}
	providers, _, _, err := buildConnections(settings)
	if err != nil {
		t.Fatal(err)
	}
	if !called || len(providers[0].Models) != 3 || providers[0].Models[2].ID != "compatible-new" || providers[0].Models[0].DefaultEffort != "high" {
		t.Fatalf("native account models were not loaded and filtered: %+v", providers)
	}
	raw, _ := json.Marshal(providers)
	if strings.Contains(string(raw), "fixture-native-private-token") {
		t.Fatal("native grant escaped into the browser catalog")
	}
}
