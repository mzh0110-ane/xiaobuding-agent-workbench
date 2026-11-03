package backendapp

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestReliableRoutesAreOptInAndRejectUngatedAcceptance(t *testing.T) {
	home := t.TempDir()
	router := gin.New()
	if err := registerReliableRoutes(router, home, func(func() error) {}); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "http://127.0.0.1/reliable", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Fatalf("default route unexpectedly enabled: %d", w.Code)
	}
	config := filepath.Join(home, "config.toml")
	if err := os.WriteFile(config, []byte("model = 'fixture'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	settings := map[string]string{"data_root": filepath.Join(home, "state"), "codex_path": executable, "codex_home": home, "model": "fixture"}
	raw, _ := json.Marshal(settings)
	if err = os.WriteFile(filepath.Join(home, "reliable-settings.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	router = gin.New()
	if err = registerReliableRoutes(router, home, func(close func() error) { t.Cleanup(func() { _ = close() }) }); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/reliable/icon.png", "/reliable/background.png"} {
		imageRequest := httptest.NewRequest("GET", "http://127.0.0.1"+path, nil)
		imageRequest.RemoteAddr = "127.0.0.1:1234"
		imageResponse := httptest.NewRecorder()
		router.ServeHTTP(imageResponse, imageRequest)
		if imageResponse.Code != 200 || imageResponse.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("desktop image route %s is not mounted: %d %s", path, imageResponse.Code, imageResponse.Header().Get("Content-Type"))
		}
	}
	req = httptest.NewRequest("POST", "http://127.0.0.1/api/v1/reliable/tasks/fake/accept", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("acceptance bypassed gate: %d", w.Code)
	}
}
