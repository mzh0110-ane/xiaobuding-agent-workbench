package reliable

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNativeActivityProjectsToolsWithoutPrivateReasoning(t *testing.T) {
	for _, tc := range []struct{ provider, raw, kind, title, detail string }{
		{"codex", `{"type":"item.started","item":{"type":"command_execution","command":"read README.md"}}`, "command", "正在执行命令", "read README.md"},
		{"codex", `{"type":"item.completed","item":{"type":"reasoning","text":"PRIVATE_REASONING"}}`, "thinking", "模型正在推理", ""},
		{"harness", `{"type":"thinking","text":"PRIVATE_REASONING"}`, "thinking", "模型正在推理", ""},
		{"harness", `{"type":"tool_call","tool":"read_file","input":{"path":"README.md"}}`, "tool", "调用工具 · read_file", `{"path":"README.md"}`},
		{"harness", `{"type":"tool_result","status":"completed","result":"source text"}`, "output", "工具执行结果 · completed", "source text"},
		{"codex", `{"type":"unknown","secret":"PRIVATE_REASONING"}`, "", "", ""},
	} {
		t.Run(tc.provider+tc.title, func(t *testing.T) {
			kind, title, detail := nativeActivity(tc.provider, []byte(tc.raw))
			require.Equal(t, tc.kind, kind)
			require.Equal(t, tc.title, title)
			require.Equal(t, tc.detail, detail)
		})
	}
}

func TestActivityArrivesBeforeProcessExitAndReplaysAfterRestart(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node runtime unavailable")
	}
	dir := t.TempDir()
	ready, release := filepath.Join(dir, "ready"), filepath.Join(dir, "release")
	script := filepath.Join(dir, "native.cjs")
	code := `const fs=require('node:fs');const [ready,release]=process.argv.slice(2);const raw=Buffer.from(JSON.stringify({type:'item.started',item:{type:'command_execution',command:'读取中文 credential-spanning-chunks'}})+'\n');process.stdout.write(raw.subarray(0,21));process.stdout.write(raw.subarray(21));console.log(JSON.stringify({type:'item.completed',item:{type:'reasoning',text:'PRIVATE_REASONING'}}));fs.writeFileSync(ready,'ready');const timer=setInterval(()=>{if(fs.existsSync(release)){clearInterval(timer);console.log(JSON.stringify({type:'turn.completed'}));}},5);`
	require.NoError(t, os.WriteFile(script, []byte(code), 0600))
	e := &Engine{Root: dir, Redact: func(raw []byte) []byte {
		return bytes.ReplaceAll(raw, []byte("credential-spanning-chunks"), []byte("[REDACTED]"))
	}}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	ctx = observeActivity(ctx, func(event Activity) { e.appendActivity("task", event) })
	done := make(chan error, 1)
	go func() {
		_, _, _, runErr := runProcess(observeNativeProcess(ctx, "codex", e.Redact), dir, os.Environ(), "", node, script, ready, release)
		done <- runErr
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("fixture did not stop")
		}
	})
	require.Eventually(t, func() bool { _, statErr := os.Stat(ready); return statErr == nil }, 5*time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		page, readErr := e.readActivity("task", 0)
		return readErr == nil && len(page.Events) == 2
	}, 5*time.Second, 10*time.Millisecond)
	first, err := e.readActivity("task", 0)
	require.NoError(t, err)
	require.Len(t, first.Events, 2)
	require.Equal(t, "读取中文 [REDACTED]", first.Events[0].Detail)
	require.Empty(t, first.Events[1].Detail)
	select {
	case <-done:
		t.Fatal("fixture exited before the incremental assertion")
	default:
	}
	require.NoError(t, os.WriteFile(release, []byte("release"), 0600))
	require.Eventually(t, func() bool {
		page, readErr := e.readActivity("task", first.Cursor)
		return readErr == nil && len(page.Events) == 1
	}, 5*time.Second, 10*time.Millisecond)
	var restarted Engine
	restarted.Root = dir
	page, err := restarted.readActivity("task", 0)
	require.NoError(t, err)
	require.Len(t, page.Events, 3)
	raw, err := json.Marshal(page)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "PRIVATE_REASONING")
	require.NotContains(t, string(raw), "credential-spanning-chunks")
}

func TestReadOnlyNativePolicies(t *testing.T) {
	runner := &CodexRunner{}
	args := runner.arguments(RoleRequest{ReadOnly: true}, "schema.json", "response.json")
	require.Equal(t, "read-only", args[5])
	require.Contains(t, args, "--skip-git-repo-check")
	require.NotContains(t, runner.arguments(RoleRequest{}, "schema.json", "response.json"), "--skip-git-repo-check")
	patch := harnessPatch(RoleRequest{ReadOnly: true}, "native", "model")
	for _, item := range patch {
		if item["id"] == "sandbox-policy" {
			require.Equal(t, "read-only", item["config"].(map[string]any)["mode"])
		}
	}
}
