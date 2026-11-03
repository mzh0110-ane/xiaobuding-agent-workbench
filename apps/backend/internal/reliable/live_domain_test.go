package reliable

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLiveNativeFrontendBackendModelsAndCodexReview(t *testing.T) {
	home := os.Getenv("XIAOBUDING_DOMAIN_LIVE_HOME")
	if home == "" {
		t.Skip("explicit native domain verification is disabled")
	}
	h, err := FromHome(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Engine.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	project := filepath.Join(h.Engine.Root, "domain-sample-"+randomID())
	if _, err = git(ctx, h.Engine.Root, "clone", "--no-hardlinks", "--config", "core.autocrlf=false", "--", h.TeamDemoProject, project); err != nil {
		t.Fatal(err)
	}
	plan := h.Engine.defaultTeamPlan()
	plan.Mode = modeFullstack
	plan.FrontendModel = "deepseek-flash"
	plan.FrontendEffort = "high"
	plan.BackendModel = "gpt-6.1-sol"
	plan.BackendEffort = "high"
	plan.AuditorModel = "gpt-6.1-sol"
	plan.AuditorEffort = "low"
	plan.ReviewerModel = "gpt-6.1-sol"
	plan.ReviewerEffort = "high"
	req := teamDemoRequest(project, plan)
	task := &Task{ID: randomID(), Project: project, Goal: req.Goal, Commands: req.Commands, AllowedPaths: req.AllowedPaths, State: stateQueued}
	if err = h.Engine.Store.Create(task); err != nil {
		t.Fatal(err)
	}
	if _, err = h.Engine.UpdateModelPlan(task.ID, 0, req.ModelPlan); err != nil {
		t.Fatal(err)
	}
	t.Logf("native domain task %s; evidence %s", task.ID, filepath.Join(h.Engine.Root, "evidence", task.ID))
	if err = h.Engine.RunTask(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := h.Engine.Store.Get(task.ID)
	if got.State != stateAwaitingAcceptance || len(got.Runs) != 4 || got.Runs[1].Domain != domainFrontend || got.Runs[1].ProviderID != "deepseek" || got.Runs[2].Domain != domainBackend || got.Runs[2].ProviderID != "codex" || got.Runs[3].ProviderID != "codex" {
		t.Fatalf("domain handoff was not completed: %+v", got)
	}
	if err = requireClean(ctx, project, ""); err != nil {
		t.Fatal("original changed before explicit sample acceptance")
	}
	if err = h.Engine.Accept(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	for i, command := range task.Commands {
		check, checkErr := runCheck(ctx, project, command, filepath.Join(h.Engine.Root, "domain-applied-check-"+string(rune('1'+i))+".log"), h.Engine.Redact)
		if checkErr != nil || check.ExitCode != 0 {
			t.Fatalf("applied domain result failed: %+v %v", check, checkErr)
		}
	}
	got, _ = h.Engine.Store.Get(task.ID)
	raw, _ := json.MarshalIndent(got, "", "  ")
	if err = os.WriteFile(filepath.Join(h.Engine.Root, "domain-live-proof.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}
