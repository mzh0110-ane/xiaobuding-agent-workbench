package reliable

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type activityProtocolFixture struct{ node, script, release string }

func (r *activityProtocolFixture) Run(ctx context.Context, req RoleRequest) (*AgentResult, error) {
	if !req.ReadOnly {
		return nil, errors.New("fixture requires explicit inspection mode")
	}
	if err := os.MkdirAll(req.OutDir, 0700); err != nil {
		return nil, err
	}
	response := filepath.Join(req.OutDir, "response.json")
	stdout, _, _, err := runProcess(observeNativeProcess(ctx, "codex", nil), req.Directory, os.Environ(), "", r.node, r.script, r.release, response)
	if err != nil {
		return nil, err
	}
	session, err := parseSession(stdout)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(response)
	return &AgentResult{Session: session, Output: raw}, err
}

func TestLiveActivityBrowserWithProtocolFixture(t *testing.T) {
	probe := os.Getenv("XIAOBUDING_ACTIVITY_BROWSER_PROBE")
	if probe == "" {
		t.Skip("explicit local browser integration is disabled")
	}
	node, err := exec.LookPath("node")
	require.NoError(t, err)
	dir := t.TempDir()
	script, release := filepath.Join(dir, "fixture.cjs"), filepath.Join(dir, "release")
	code := `const fs=require('node:fs');const [release,response]=process.argv.slice(2);const emit=x=>console.log(JSON.stringify(x));emit({type:'thread.started',thread_id:'local-protocol-fixture'});emit({type:'item.started',item:{type:'command_execution',command:'read app.txt'}});const source=fs.readFileSync('app.txt','utf8');emit({type:'item.completed',item:{type:'command_execution',command:'read app.txt',status:'completed',aggregated_output:source}});emit({type:'item.completed',item:{type:'reasoning',text:'PRIVATE_REASONING_MUST_NOT_DISPLAY'}});const timer=setInterval(()=>{if(fs.existsSync(release)){clearInterval(timer);const report={summary:'协议夹具读取完成（非真实模型推理）',facts:['app.txt: '+source.trim()],inferences:[],unknowns:[],read_files:['app.txt'],required_read_files:[],allowed_paths:[],plan:[]};fs.writeFileSync(response,JSON.stringify(report));emit({type:'item.completed',item:{type:'agent_message',text:JSON.stringify(report)}});emit({type:'turn.completed'});}},10);`
	require.NoError(t, os.WriteFile(script, []byte(code), 0600))
	runner := &activityProtocolFixture{node: node, script: script, release: release}
	e, _ := fixtureEngine(t, runner, passingCheck)
	t.Cleanup(func() { require.NoError(t, e.Close()) })
	project := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(project, "app.txt"), []byte("original source\n"), 0600))
	server := httptest.NewServer(&HTTP{Engine: e, Token: "local-protocol-fixture"})
	t.Cleanup(server.Close)
	cmd := exec.Command(node, probe, server.URL, project, release)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	t.Log(string(output))
	_, err = os.Stat(filepath.Join(project, ".git"))
	require.True(t, os.IsNotExist(err))
}
