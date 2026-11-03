package reliable

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCodexRunnerUsesPerCallModelAndEffortWithoutMutatingItsDefault(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node unavailable")
	}
	preload := filepath.Join(t.TempDir(), "codex-wire.cjs")
	script := `const fs=require('node:fs');const a=process.argv;if(a[a.indexOf('-m')+1]!=='chosen-codex'||!a.includes('model_reasoning_effort="high"'))process.exit(3);fs.writeFileSync(a[a.indexOf('-o')+1],JSON.stringify({summary:'ok',read_files:['app.txt'],docs_impact:'none',residual_risks:[]}));console.log(JSON.stringify({type:'thread.started',thread_id:'native-chosen-thread'}));console.log(JSON.stringify({type:'turn.completed'}));process.exit(0);`
	if err = os.WriteFile(preload, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NODE_OPTIONS", `--require="`+filepath.ToSlash(preload)+`"`)
	runner := &CodexRunner{Executable: node, Home: t.TempDir(), Model: "default-codex", Provider: "openai"}
	evidence := t.TempDir()
	result, err := runner.Run(context.Background(), RoleRequest{Role: roleImplementer, Directory: t.TempDir(), OutDir: evidence, Model: "chosen-codex", Effort: "high", Prompt: "Read source"})
	if err != nil || result == nil || result.Session != "native-chosen-thread" {
		diagnostic, _ := os.ReadFile(filepath.Join(evidence, "stderr.txt"))
		t.Fatalf("actual Codex wire ignored selection: %+v %v; %s", result, err, diagnostic)
	}
	if runner.Model != "default-codex" {
		t.Fatal("per-call model changed the shared runner")
	}
}
