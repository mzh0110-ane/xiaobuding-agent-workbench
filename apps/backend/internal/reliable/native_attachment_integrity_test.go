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

func TestNativeInitialImagesRespectByteBudgetAndKeepUnreadReferences(t *testing.T) {
	refs := []map[string]any{}
	for i := range 4 {
		refs = append(refs, map[string]any{"id": string(rune('a' + i)), "name": "image.png", "path": "unused", "image": true, "size": 8 << 20, "media_type": "image/png"})
	}
	// The helper never opens ordinary file references, including unsupported media.
	refs = append(refs, map[string]any{"name": "vector.svg", "path": "unused.svg", "image": true, "size": 100, "media_type": "image/svg+xml"})
	raw, err := json.Marshal(map[string]any{"attachments": refs})
	if err != nil {
		t.Fatal(err)
	}
	var req RoleRequest
	if json.Unmarshal(raw, &req) != nil {
		t.Fatal("attachment fixture did not decode")
	}
	images, references := nativeImageInputs(req.Attachments)
	if len(images) != 2 || images[0].ID != "c" || images[1].ID != "d" || len(references) != 3 {
		t.Fatalf("native image budget lost or misselected lifetime file references: %+v %+v", images, references)
	}
}

// @covers AC-RELIABLE-ATTACHMENTS-001.6
func TestNativeImageMutationStopsBeforeStartingEitherNativeProcess(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is unavailable")
	}
	for _, provider := range []string{"codex", "harness"} {
		t.Run(provider, func(t *testing.T) {
			req, paths := nativeAttachmentRequest(t, 1)
			data, err := os.ReadFile(paths[0])
			if err != nil {
				t.Fatal(err)
			}
			data[0] ^= 1
			if err := os.WriteFile(paths[0], data, 0600); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(t.TempDir(), "started.txt")
			entry := filepath.Join(t.TempDir(), "native-start.cjs")
			if err := os.WriteFile(entry, []byte(`require('node:fs').writeFileSync(process.env.ATTACHMENT_STARTED,'started');process.exit(4);`), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("ATTACHMENT_STARTED", marker)
			var runner AgentRunner = &HarnessRunner{Executable: node, Entry: entry, Home: t.TempDir()}
			if provider == "codex" {
				t.Setenv("NODE_OPTIONS", `--require="`+filepath.ToSlash(entry)+`"`)
				runner = &CodexRunner{Executable: node, Home: t.TempDir(), Provider: "openai"}
			}
			if _, err := runner.Run(context.Background(), req); err == nil || !strings.Contains(err.Error(), "attachment_changed") {
				t.Fatalf("changed bytes did not refuse native dispatch: %v", err)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("native model started after attachment integrity failed: %v", err)
			}
		})
	}
}
