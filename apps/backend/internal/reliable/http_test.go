package reliable

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func httpRequest(h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://127.0.0.1:38429"+path, strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-XBD-Token", token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestHTTPRequiresHumanActionAndRejectsForgedState(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "browser-capability"}
	if got := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/accept", `{}`, ""); got.Code != http.StatusForbidden {
		t.Fatalf("unauthorized acceptance: %d", got.Code)
	}
	if got := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/accept", `{}`, h.Token); got.Code != http.StatusConflict {
		t.Fatalf("unverified acceptance: %d", got.Code)
	}
	if got := httpRequest(h, "POST", "/api/v1/reliable/tasks", `{"project":"x","goal":"x","state":"accepted"}`, h.Token); got.Code != http.StatusBadRequest {
		t.Fatalf("forged state accepted: %d", got.Code)
	}
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	if got := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/accept", `{}`, h.Token); got.Code != http.StatusOK {
		t.Fatalf("valid human acceptance failed: %d %s", got.Code, got.Body)
	}
}

func TestHTTPRejectsCrossOriginAndTraversal(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "token"}
	req := httptest.NewRequest("POST", "http://127.0.0.1:38429/api/v1/reliable/tasks", strings.NewReader(`{}`))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Origin", "https://attacker.example")
	req.Header.Set("X-XBD-Token", h.Token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-origin request admitted: %d", w.Code)
	}
	got := httpRequest(h, "GET", "/api/v1/reliable/tasks/task/evidence?path=../../state.db", "", "")
	if got.Code != http.StatusBadRequest {
		t.Fatalf("evidence traversal admitted: %d", got.Code)
	}
}
