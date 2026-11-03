package reliable

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeLoginThenRefreshActivatesRunnerAndCurrentRedaction(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node unavailable")
	}
	home := t.TempDir()
	settings := Settings{DataRoot: filepath.Join(home, "data"), CodexPath: node, CodexHome: home, Model: "first", Connections: []ConnectionSettings{{ID: "codex", Kind: "codex", Home: home, Provider: "openai", Model: "first"}}}
	raw, _ := json.Marshal(settings)
	for name, data := range map[string][]byte{"reliable-settings.json": raw, "config.toml": []byte("model='first'"), "models_cache.json": []byte(`{"models":[{"slug":"first","visibility":"list"}]}`)} {
		if err = os.WriteFile(filepath.Join(home, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	h, err := FromHome(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Engine.Close() })
	if _, err = h.Engine.provider("codex"); err == nil {
		t.Fatal("positive control: connection unexpectedly started authenticated")
	}
	if err = os.WriteFile(filepath.Join(home, "auth.json"), []byte(`{"auth_mode":"chatgpt"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(home, "config.toml"), []byte("model='first'\nexperimental_bearer_token='fresh-private-test-token'"), 0600); err != nil {
		t.Fatal(err)
	}
	response := httpRequest(h, "POST", "/api/v1/reliable/models/refresh", `{}`, h.Token)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	preload := filepath.Join(t.TempDir(), "native-login.cjs")
	script := `const fs=require('node:fs'),a=process.argv;fs.writeFileSync(a[a.indexOf('-o')+1],'{}');console.log(JSON.stringify({type:'thread.started',thread_id:'logged-in-after-refresh'}));console.log(JSON.stringify({type:'turn.completed'}));process.exit(0);`
	if err = os.WriteFile(preload, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NODE_OPTIONS", `--require="`+filepath.ToSlash(preload)+`"`)
	result, err := h.Engine.Runner.Run(context.Background(), RoleRequest{Role: roleAuditor, ProviderID: "codex", Directory: t.TempDir(), OutDir: t.TempDir(), Prompt: "local protocol fixture"})
	if err != nil || result.Session != "logged-in-after-refresh" {
		t.Fatalf("refreshed authenticated connection cannot execute: %+v %v", result, err)
	}
	if strings.Contains(string(h.Engine.Redact([]byte("fresh-private-test-token"))), "fresh-private-test-token") {
		t.Fatal("refreshed credential was absent from redaction")
	}
}
