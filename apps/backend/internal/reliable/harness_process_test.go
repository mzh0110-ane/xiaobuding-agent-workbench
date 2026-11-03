package reliable

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHarnessUsesNativeHomeFullAccessAndNoCodexKeyFallback(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node runtime is unavailable")
	}
	dir, home, evidence := t.TempDir(), t.TempDir(), t.TempDir()
	entry := filepath.Join(t.TempDir(), "native-fixture.cjs")
	script := `const fs=require('node:fs');const args=process.argv.slice(2);const overlay=JSON.parse(fs.readFileSync(args[args.indexOf('--patch')+1],'utf8'));const cfg=id=>overlay.find(x=>x.id===id)?.config;let prompt='';process.stdin.setEncoding('utf8');process.stdin.on('data',x=>prompt+=x);process.stdin.on('end',()=>{if(process.env.DEEPSEEK_API_KEY||!process.env.DSH_HOME||!args.includes('--json')||cfg('sandbox-policy').mode!=='danger-full-access'||cfg('permission').presets[cfg('permission').defaultPreset].sandbox!=='danger-full-access'||cfg('permission').presets[cfg('permission').defaultPreset].approval!=='never'||cfg('approval').policy!=='never'||cfg('agent-default-model').model!=='selected-native'||cfg('agent-default-model').reasoningEffort!=='low'||!prompt.includes('additionalProperties'))process.exit(2);console.info('native account model catalog loaded');const report={summary:'native fixture completed',read_files:['app.txt'],docs_impact:'none',residual_risks:[]};for(const event of [{type:'session',sessionId:'session-native-env',cwd:process.cwd()},{type:'status',phase:'turn_end',reason:{kind:'completed'}},{type:'final',text:JSON.stringify(report)}])console.log(JSON.stringify(event));});`
	if err = os.WriteFile(entry, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEEPSEEK_API_KEY", "test-only-poisoned-codex-key")
	runner := &HarnessRunner{Executable: node, Entry: entry, Home: home, Provider: "deepseek-official", Model: "native-fixture"}
	for _, role := range []string{roleAuditor, roleImplementer, roleReviewer} {
		out := filepath.Join(evidence, role)
		result, err := runner.Run(context.Background(), RoleRequest{Role: role, Directory: dir, OutDir: out, Model: "selected-native", Effort: "low", Prompt: "Read source and return JSON"})
		if err != nil || result == nil || result.Session != "session-native-env" {
			t.Fatalf("native Harness %s connection is incorrect: %+v %v", role, result, err)
		}
		diagnostics, err := os.ReadFile(filepath.Join(out, "stderr.txt"))
		if err != nil || !strings.Contains(string(diagnostics), "native account model catalog loaded") {
			t.Fatalf("native diagnostic was not preserved on stderr: %q %v", diagnostics, err)
		}
	}
}
