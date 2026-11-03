package reliable

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAcceptedExternalProjectIsCheckpointedBeforeTheNextBatch(t *testing.T) {
	agent := &checkpointContinuationAgent{}
	e, task := fixtureEngine(t, agent, checkpointPassingCheck)
	e.Providers = testProviders()
	t.Cleanup(e.wg.Wait)
	base := strings.TrimSpace(gitTest(t, task.Project, "rev-parse", "HEAD"))
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.Accept(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	if status := gitTest(t, task.Project, "status", "--porcelain=v1", "--untracked-files=all"); status != "" {
		t.Fatalf("accepted changes have no completed local checkpoint: %s", status)
	}
	acceptedHead := strings.TrimSpace(gitTest(t, task.Project, "rev-parse", "HEAD"))
	if acceptedHead == base || strings.TrimSpace(gitTest(t, task.Project, "rev-parse", "HEAD^")) != base {
		t.Fatal("acceptance must create one local commit directly after the reviewed original base")
	}
	if content := gitTest(t, task.Project, "show", "HEAD:app.txt"); content != "reviewed\n" {
		t.Fatalf("local checkpoint does not contain the accepted bytes: %q", content)
	}
	if remotes := gitTest(t, task.Project, "remote"); remotes != "" {
		t.Fatal("local acceptance configured an upload remote")
	}
	h := &HTTP{Engine: e, Token: "accepted-checkpoint-next-batch"}
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"next-checkpoint","kind":"requirement","content":"Continue improving the accepted project"}`, h.Token)
	if response.Code != http.StatusAccepted {
		t.Fatalf("next batch was not admitted: %d %s", response.Code, response.Body.String())
	}
	var message TaskMessage
	if err := json.Unmarshal(response.Body.Bytes(), &message); err != nil {
		t.Fatal(err)
	}
	e.wg.Wait()
	child, err := e.Store.Get(message.LinkedTask)
	if err != nil {
		t.Fatal(err)
	}
	if child.State != stateAwaitingAcceptance || child.Round != 1 || child.Base != acceptedHead {
		t.Fatalf("next batch did not continue the accepted local version: state=%s round=%d base=%s reason=%s", child.State, child.Round, child.Base, child.Reason)
	}
	if len(agent.auditedSources) != 2 || agent.auditedSources[1] != "reviewed\n" {
		t.Fatalf("next agent did not begin with accepted source: %q", agent.auditedSources)
	}
	if content, err := os.ReadFile(filepath.Join(child.Worktree, "app.txt")); err != nil || string(content) != "reviewed again\n" {
		t.Fatalf("next candidate did not continue the inherited source: %q %v", content, err)
	}
}

func TestAcceptanceCheckpointIsIdempotentAndDoesNotCreateEmptyCommits(t *testing.T) {
	for _, unchanged := range []bool{false, true} {
		t.Run(map[bool]string{false: "changed result", true: "unchanged result"}[unchanged], func(t *testing.T) {
			e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
			if unchanged {
				if err := os.WriteFile(filepath.Join(task.Project, "app.txt"), []byte("reviewed\n"), 0600); err != nil {
					t.Fatal(err)
				}
				gitTest(t, task.Project, "add", "app.txt")
				gitTest(t, task.Project, "-c", "user.name=Fixture", "-c", "user.email=fixture@localhost", "commit", "-m", "already reviewed")
			}
			base := gitTest(t, task.Project, "rev-parse", "HEAD")
			if err := e.RunTask(context.Background(), task.ID); unchanged {
				if err == nil || !strings.Contains(err.Error(), "implementation produced no changes") {
					t.Fatalf("unchanged implementation must stop without creating an acceptance: %v", err)
				}
				if base != gitTest(t, task.Project, "rev-parse", "HEAD") {
					t.Fatal("unchanged implementation created an empty checkpoint")
				}
				if err = e.Accept(context.Background(), task.ID); err == nil {
					t.Fatal("unchanged unreviewed implementation was accepted")
				}
				return
			} else if err != nil {
				t.Fatal(err)
			}
			if err := e.Accept(context.Background(), task.ID); err != nil {
				t.Fatal(err)
			}
			accepted := gitTest(t, task.Project, "rev-parse", "HEAD")
			if unchanged && accepted != base {
				t.Fatal("accepting an unchanged result created an empty commit")
			}
			if !unchanged && accepted == base {
				t.Fatal("changed result was not checkpointed")
			}
			if err := e.Accept(context.Background(), task.ID); err != nil {
				t.Fatalf("repeated acceptance was not idempotent: %v", err)
			}
			if accepted != gitTest(t, task.Project, "rev-parse", "HEAD") {
				t.Fatal("repeated acceptance created another commit")
			}
			entries, err := os.ReadDir(task.Project)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 2 || entries[0].Name() != ".git" || entries[1].Name() != "app.txt" {
				t.Fatalf("acceptance created extra project files or folder backups: %v", entries)
			}
		})
	}
}

func TestLegacyAcceptedPatchIsCheckpointedWhenTheNextBatchStarts(t *testing.T) {
	e, parent := legacyAcceptedCheckpointFixture(t)
	base := parent.Base
	message, err := e.startLinkedBatch(parent.ID, messageRequest{ID: "legacy-next", Kind: messageRequirement, Content: "Continue this same project"})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.RunTask(context.Background(), message.LinkedTask); err != nil {
		t.Fatalf("legacy accepted result did not recover automatically: %v", err)
	}
	head := strings.TrimSpace(gitTest(t, parent.Project, "rev-parse", "HEAD"))
	if head == base || strings.TrimSpace(gitTest(t, parent.Project, "rev-parse", "HEAD^")) != base {
		t.Fatal("legacy acceptance must add exactly one local checkpoint")
	}
	if status := gitTest(t, parent.Project, "status", "--porcelain=v1", "--untracked-files=all"); status != "" {
		t.Fatalf("legacy checkpoint did not leave a clean original: %s", status)
	}
	child, err := e.Store.Get(message.LinkedTask)
	if err != nil {
		t.Fatal(err)
	}
	if child.State != stateAwaitingAcceptance || child.Base != head || child.Round != 1 {
		t.Fatalf("legacy next batch did not use the recovered accepted source: %+v", child)
	}
	if content := gitTest(t, parent.Project, "show", "HEAD:app.txt"); content != "reviewed\n" {
		t.Fatalf("legacy checkpoint saved different bytes: %q", content)
	}
}

func TestLegacyAcceptedCheckpointPreservesManualChangesAndInvalidReceipts(t *testing.T) {
	for _, scenario := range []string{"untracked", "changed accepted file", "staged accepted file", "receipt nested object", "changed sealed evidence"} {
		t.Run(scenario, func(t *testing.T) {
			e, parent := legacyAcceptedCheckpointFixture(t)
			mutated := filepath.Join(parent.Project, "app.txt")
			switch scenario {
			case "untracked":
				mutated = filepath.Join(parent.Project, "manual.txt")
				checkpointWriteFile(t, mutated, []byte("manual work must remain\n"))
			case "changed accepted file":
				checkpointWriteFile(t, mutated, []byte("manual edit after acceptance\n"))
			case "staged accepted file":
				gitTest(t, parent.Project, "add", "app.txt")
			case "receipt nested object":
				checkpointWriteFile(t, filepath.Join(e.Root, "evidence", parent.ID, "acceptance.json"), []byte(`{"source_hash":{},"patch_hash":[],"round":1,"human_action":"accept_and_apply"}`))
			case "changed sealed evidence":
				checkpointWriteFile(t, filepath.Join(e.Root, "evidence", parent.ID, "round-1-review.json"), []byte(`{"verdict":"pass","summary":"tampered"}`))
			}
			before, err := os.ReadFile(mutated)
			if err != nil {
				t.Fatal(err)
			}
			gitTest(t, parent.Project, "status", "--porcelain=v1", "--untracked-files=all")
			indexPath := filepath.Join(parent.Project, ".git", "index")
			indexBefore, err := os.ReadFile(indexPath)
			if err != nil {
				t.Fatal(err)
			}
			head := gitTest(t, parent.Project, "rev-parse", "HEAD")
			message, err := e.startLinkedBatch(parent.ID, messageRequest{ID: "legacy-reject", Kind: messageRequirement, Content: "Continue this project"})
			if err != nil {
				t.Fatal(err)
			}
			if err = e.RunTask(context.Background(), message.LinkedTask); err == nil {
				t.Fatal("unrelated work or invalid acceptance evidence was silently checkpointed")
			}
			after, err := os.ReadFile(mutated)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("refused checkpoint changed the user's files: %v", err)
			}
			indexAfter, err := os.ReadFile(indexPath)
			if err != nil || !bytes.Equal(indexBefore, indexAfter) || head != gitTest(t, parent.Project, "rev-parse", "HEAD") {
				t.Fatalf("refused checkpoint changed the original index or HEAD: %v", err)
			}
			child, err := e.Store.Get(message.LinkedTask)
			if err != nil || child.Round != 0 || child.State != stateBlocked {
				t.Fatalf("refused local preparation spent a modification round: %+v %v", child, err)
			}
		})
	}
}

type checkpointThirdRoundAgent struct{ fixtureAgent }

func (a *checkpointThirdRoundAgent) Run(ctx context.Context, request RoleRequest) (*AgentResult, error) {
	result, err := a.fixtureAgent.Run(ctx, request)
	if err == nil && request.Role == roleReviewer && a.review < MaxRounds {
		result.Output = json.RawMessage(`{"verdict":"rework","summary":"Another independent verification is needed","read_files":["app.txt"],"findings":[{"severity":"P1","file":"app.txt","reason":"Improve the verified behavior"}]}`)
	}
	return result, err
}

func TestPendingAcceptedCheckpointResumesAtTheThreeRoundLimit(t *testing.T) {
	agent := &checkpointThirdRoundAgent{}
	e, task := fixtureEngine(t, agent, passingCheck)
	t.Cleanup(e.wg.Wait)
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	ready, err := e.Store.Get(task.ID)
	if err != nil || ready.Round != MaxRounds || ready.State != stateAwaitingAcceptance {
		t.Fatalf("fixture did not reach a reviewed third round: %+v %v", ready, err)
	}
	ref := strings.TrimSpace(gitTest(t, task.Project, "symbolic-ref", "HEAD"))
	refLock := filepath.Join(task.Project, ".git", filepath.FromSlash(ref)+".lock")
	checkpointWriteFile(t, refLock, []byte("foreign branch lock\n"))
	if err = e.Accept(context.Background(), task.ID); err == nil {
		t.Fatal("branch-ref lock did not interrupt checkpoint completion")
	}
	pending, err := e.Store.Get(task.ID)
	if err != nil || pending.State != stateBlocked || pending.LocalCheckpoint == nil || pending.LocalCheckpoint.Complete {
		t.Fatalf("interrupted acceptance lost the recoverable checkpoint: %+v %v", pending, err)
	}
	if content, err := os.ReadFile(filepath.Join(task.Project, "app.txt")); err != nil || string(content) != "reviewed\n" {
		t.Fatalf("failure boundary did not occur after actual patch application: %q %v", content, err)
	}
	if lock, err := os.ReadFile(refLock); err != nil || string(lock) != "foreign branch lock\n" {
		t.Fatalf("checkpoint modified another Git operation's lock: %q %v", lock, err)
	}
	if err = os.Remove(refLock); err != nil {
		t.Fatal(err)
	}
	calls := agent.total
	h := &HTTP{Engine: e, Token: "pending-checkpoint-resume"}
	response := httpRequest(h, "POST", "/api/v1/reliable/tasks/"+task.ID+"/resume", "{}", h.Token)
	if response.Code != http.StatusOK {
		t.Fatalf("third-round pending acceptance cannot resume: %d %s", response.Code, response.Body.String())
	}
	awaitCheckpointAccepted(t, e, task.ID)
	e.wg.Wait()
	completed, err := e.Store.Get(task.ID)
	if err != nil || completed.State != stateAccepted || completed.LocalCheckpoint == nil || !completed.LocalCheckpoint.Complete || completed.Round != MaxRounds {
		t.Fatalf("resumed acceptance did not finish without another implementation round: %+v %v", completed, err)
	}
	if agent.total != calls {
		t.Fatal("checkpoint recovery reran the agents")
	}
	if status := gitTest(t, task.Project, "status", "--porcelain=v1", "--untracked-files=all"); status != "" {
		t.Fatalf("recovered checkpoint left an unclean original: %s", status)
	}
	if commits := strings.TrimSpace(gitTest(t, task.Project, "rev-list", "--count", "HEAD")); commits != "2" {
		t.Fatalf("recovery reapplied or duplicated the accepted commit: %s", commits)
	}
}

func TestCompletedCheckpointBeforeTerminalStateResumesWithoutAnotherCommit(t *testing.T) {
	agent := &fixtureAgent{}
	e, task := fixtureEngine(t, agent, passingCheck)
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.Accept(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	head := gitTest(t, task.Project, "rev-parse", "HEAD")
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		saved.State = stateInterrupted
		saved.Reason = "Process stopped before persisting the terminal accepted state"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := agent.total
	if err = e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatalf("completed checkpoint was not recognized after restart: %v", err)
	}
	completed, err := e.Store.Get(task.ID)
	if err != nil || completed.State != stateAccepted || agent.total != calls || head != gitTest(t, task.Project, "rev-parse", "HEAD") {
		t.Fatalf("checkpoint finalization duplicated work or lost acceptance: %+v %v", completed, err)
	}
}

func legacyAcceptedCheckpointFixture(t *testing.T) (*Engine, *Task) {
	t.Helper()
	e, task := fixtureEngine(t, &checkpointContinuationAgent{}, checkpointPassingCheck)
	e.Providers = testProviders()
	if err := e.RunTask(context.Background(), task.ID); err != nil {
		t.Fatal(err)
	}
	current, err := e.Store.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	patch, err := e.acceptanceEvidence(context.Background(), current)
	if err != nil {
		t.Fatal(err)
	}
	if err = ApplyPatch(context.Background(), current.Project, current.Base, patch); err != nil {
		t.Fatal(err)
	}
	if err = e.setState(current.ID, stateAccepted, "Human accepted; verified changes applied to the original project"); err != nil {
		t.Fatal(err)
	}
	if err = e.saveJSON(current.ID, "acceptance.json", map[string]any{"source_hash": current.Snapshot, "patch_hash": current.PatchHash, "round": current.Round, "human_action": "accept_and_apply"}); err != nil {
		t.Fatal(err)
	}
	current, err = e.Store.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	return e, current
}

const checkpointPassingCheck = `python -c "from pathlib import Path; assert Path('app.txt').read_text().startswith('reviewed')"`

type checkpointContinuationAgent struct {
	fixtureAgent
	auditedSources []string
}

func (a *checkpointContinuationAgent) Run(ctx context.Context, request RoleRequest) (*AgentResult, error) {
	if request.Role == roleAuditor {
		content, err := os.ReadFile(filepath.Join(request.Directory, "app.txt"))
		if err != nil {
			return nil, err
		}
		a.auditedSources = append(a.auditedSources, string(content))
	}
	result, err := a.fixtureAgent.Run(ctx, request)
	if err == nil && request.Role == roleImplementer && a.implement > 1 {
		err = os.WriteFile(filepath.Join(request.Directory, "app.txt"), []byte("reviewed again\n"), 0600)
	}
	return result, err
}

func checkpointWriteFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
}

func awaitCheckpointAccepted(t *testing.T, e *Engine, id string) {
	t.Helper()
	deadline := time.NewTimer(60 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		current, err := e.Store.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if current.State == stateAccepted {
			return
		}
		select {
		case <-deadline.C:
			t.Fatalf("resumed checkpoint did not reach accepted state: %s %s", current.State, current.Reason)
		case <-tick.C:
		}
	}
}
