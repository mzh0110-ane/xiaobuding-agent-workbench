package reliable

import (
	"bytes"
	"net/http"
	"testing"
)

func TestDesktopAvatarIsServedAsAnImage(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "desktop-action"}
	for _, path := range []string{"/reliable/icon.png", "/reliable/background.png", "/api/v1/reliable/brand/codex.png", "/api/v1/reliable/brand/deepseek.png"} {
		w := httpRequest(h, "GET", path, "", "")
		if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/png" || !bytes.HasPrefix(w.Body.Bytes(), []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
			t.Fatalf("desktop image %s is unavailable as a PNG: %d %s", path, w.Code, w.Header().Get("Content-Type"))
		}
	}
}
