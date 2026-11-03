package reliable

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCodexRunnerUsesReservedExecutableSnapshot(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node unavailable")
	}
	preload := filepath.Join(t.TempDir(), "pinned-codex.cjs")
	script := `const fs=require('node:fs'),a=process.argv;fs.writeFileSync(a[a.indexOf('-o')+1],'{}');console.log(JSON.stringify({type:'thread.started',thread_id:'pinned-executable'}));console.log(JSON.stringify({type:'turn.completed'}));process.exit(0);`
	if err = os.WriteFile(preload, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NODE_OPTIONS", `--require="`+filepath.ToSlash(preload)+`"`)
	runner := &CodexRunner{Executable: filepath.Join(t.TempDir(), "missing.exe"), Home: t.TempDir(), Provider: "openai"}
	snapshots := map[*CodexRunner]codexExecutableSnapshot{runner: {path: node}}
	ctx := context.WithValue(context.Background(), codexExecutableContextKey{}, snapshots)
	result, err := runner.Run(ctx, RoleRequest{Role: roleAuditor, Directory: t.TempDir(), OutDir: t.TempDir()})
	if err != nil || result == nil || result.Session != "pinned-executable" {
		t.Fatalf("runner ignored its immutable executable snapshot: %+v %v", result, err)
	}
	if runner.Executable == node {
		t.Fatal("executing a snapshot mutated the runner's configured path")
	}
}
