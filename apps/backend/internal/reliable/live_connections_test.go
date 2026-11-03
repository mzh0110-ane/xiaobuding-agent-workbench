package reliable

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLiveNativeCodexAndHarnessCollaboration(t *testing.T) {
	home := os.Getenv("XIAOBUDING_LIVE_SETTINGS_HOME")
	if home == "" {
		t.Skip("explicit native-model live verification is disabled")
	}
	h, err := FromHome(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Engine.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	project := filepath.Join(h.Engine.Root, "live-sample-"+randomID())
	if _, err = git(ctx, h.Engine.Root, "clone", "--no-hardlinks", "--config", "core.autocrlf=false", "--", h.DemoProject, project); err != nil {
		t.Fatal(err)
	}
	task := &Task{ID: randomID(), Project: project, Goal: "修复 calculator.py 的 average：空列表返回 0，其他平均值保持正确，保留现有测试，只修改 calculator.py。", Commands: []string{"python -m unittest discover -s tests -v"}, AllowedPaths: []string{"calculator.py"}, State: stateQueued}
	if err = h.Engine.Store.Create(task); err != nil {
		t.Fatal(err)
	}
	plan := ModelPlan{Auditor: "deepseek", Implementer: "codex", Reviewer: "deepseek"}
	if _, err = h.Engine.UpdateModelPlan(task.ID, 0, plan); err != nil {
		t.Fatal(err)
	}
	t.Logf("live task %s; evidence: %s", task.ID, filepath.Join(h.Engine.Root, "evidence", task.ID))
	if err = h.Engine.RunTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := h.Engine.Store.Get(task.ID)
	if got.State != stateAwaitingAcceptance || len(got.Runs) != 3 || got.Runs[0].ProviderID != "deepseek" || got.Runs[1].ProviderID != "codex" {
		t.Fatalf("native tools did not reach a shared verified candidate: %+v", got)
	}
	if err = requireClean(ctx, project, ""); err != nil {
		t.Fatal("original changed before human acceptance")
	}
	if err = h.Engine.Accept(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	check, err := runCheck(ctx, project, task.Commands[0], filepath.Join(h.Engine.Root, "live-applied-check.log"), h.Engine.Redact)
	if err != nil || check.ExitCode != 0 {
		t.Fatalf("applied native result did not pass real checks: %+v %v", check, err)
	}
	got, _ = h.Engine.Store.Get(task.ID)
	raw, _ := json.MarshalIndent(got, "", "  ")
	if err = os.WriteFile(filepath.Join(h.Engine.Root, "live-proof.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}
