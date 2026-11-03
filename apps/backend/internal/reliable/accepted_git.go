package reliable

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AcceptanceCheckpoint journals only the controller's reviewed patch. It never
// grants permission to commit arbitrary edits in an existing repository.
type AcceptanceCheckpoint struct {
	Project      string `json:"project"`
	Base         string `json:"base"`
	Tree         string `json:"tree,omitempty"`
	Commit       string `json:"commit,omitempty"`
	Reference    string `json:"reference"`
	Nonce        string `json:"nonce"`
	PatchHash    string `json:"patch_hash"`
	IndexHash    string `json:"index_hash,omitempty"`
	NewIndexHash string `json:"new_index_hash,omitempty"`
	Complete     bool   `json:"complete,omitempty"`
}

type acceptedGit struct {
	ctx                 context.Context
	project, temporary  string
	index, lock, staged string
	marker              []byte
	intent              *AcceptanceCheckpoint
	persist             func(*AcceptanceCheckpoint) error
}

// checkpointAcceptedPatch is restartable: persist must durably save the intent
// before returning. allowApply is false for already-applied legacy acceptances.
// All patch and acceptance evidence must be authenticated by the caller first.
func checkpointAcceptedPatch(ctx context.Context, project, base, patch, dataDir string, intent *AcceptanceCheckpoint, persist func(*AcceptanceCheckpoint) error, allowApply bool) (*AcceptanceCheckpoint, error) {
	intent, err := acceptedIntent(ctx, project, base, patch, intent, persist)
	if err != nil || intent.Complete {
		return intent, err
	}
	a, err := newAcceptedGit(ctx, project, dataDir, intent, persist)
	if err != nil {
		return nil, err
	}
	defer a.removeTemporary()
	patch, err = a.freezePatch(patch)
	if err != nil {
		return nil, err
	}
	if err = a.acquire(); err != nil {
		return nil, err
	}
	defer a.release()
	if err = a.checkReference(); err != nil {
		return nil, err
	}
	if err = a.checkOperation(); err != nil {
		return nil, err
	}
	if err = a.prepareExpected(patch); err != nil {
		return nil, err
	}
	if err = a.finish(patch, allowApply); err != nil {
		return nil, err
	}
	return intent, nil
}

func acceptedIntent(ctx context.Context, project, base, patch string, intent *AcceptanceCheckpoint, persist func(*AcceptanceCheckpoint) error) (*AcceptanceCheckpoint, error) {
	if persist == nil {
		return nil, errors.New("local_checkpoint: missing durable journal")
	}
	raw, err := os.ReadFile(patch)
	if err != nil {
		return nil, err
	}
	if !acceptedOID(base) {
		return nil, errors.New("local_checkpoint: invalid reviewed patch or base")
	}
	patchHash := digest(raw)
	if intent != nil && intent.Nonce == "" {
		if *intent != (AcceptanceCheckpoint{Base: base}) {
			return nil, errors.New("local_checkpoint: incomplete journal has unexpected fields")
		}
		intent = nil // Engine has durably marked this acceptance as pending.
	}
	if intent == nil {
		ref, refErr := acceptedReference(ctx, project)
		if refErr != nil {
			return nil, refErr
		}
		intent = &AcceptanceCheckpoint{Project: project, Base: base, PatchHash: patchHash, Reference: ref, Nonce: randomID()}
		return intent, persist(intent)
	}
	return validateAcceptedJournal(project, base, patchHash, intent)
}

func validateAcceptedJournal(project, base, patchHash string, intent *AcceptanceCheckpoint) (*AcceptanceCheckpoint, error) {
	copy := *intent
	if copy.Base != base || copy.PatchHash != patchHash || projectKey(copy.Project) != projectKey(project) || !safeRelative(copy.Nonce) || strings.Contains(copy.Nonce, "/") || len(copy.Nonce) > 64 {
		return nil, errors.New("local_checkpoint: saved journal does not match this acceptance")
	}
	return &copy, nil
}

func newAcceptedGit(ctx context.Context, project, dataDir string, intent *AcceptanceCheckpoint, persist func(*AcceptanceCheckpoint) error) (*acceptedGit, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}
	indexRaw, err := localGitEnvironment(ctx, project, nil, []string{"GIT_OPTIONAL_LOCKS=0"}, "rev-parse", "--path-format=absolute", "--git-path", "index")
	if err != nil {
		return nil, err
	}
	index := strings.TrimSpace(string(indexRaw))
	a := &acceptedGit{ctx: ctx, project: project, index: index, lock: index + ".lock", temporary: filepath.Join(dataDir, ".accepted-"+intent.Nonce+".index"), staged: filepath.Join(filepath.Dir(index), "xiaobuding-accepted-"+intent.Nonce+".index"), marker: []byte("xiaobuding-accepted-checkpoint:" + intent.Nonce + "\n"), intent: intent, persist: persist}
	return a, nil
}

func acceptedOID(value string) bool {
	return (len(value) == 40 || len(value) == 64) && strings.Trim(value, "0123456789abcdef") == ""
}

func acceptedReference(ctx context.Context, project string) (string, error) {
	raw, err := localGitEnvironment(ctx, project, nil, []string{"GIT_OPTIONAL_LOCKS=0"}, "symbolic-ref", "-q", "HEAD")
	if err == nil {
		ref := strings.TrimSpace(string(raw))
		if strings.HasPrefix(ref, "refs/heads/") {
			return ref, nil
		}
		return "", errors.New("local_checkpoint: unsupported HEAD reference")
	}
	if _, headErr := localGitEnvironment(ctx, project, nil, []string{"GIT_OPTIONAL_LOCKS=0"}, "rev-parse", "--verify", "HEAD"); headErr != nil {
		return "", headErr
	}
	return "HEAD", nil // Detached HEAD is preserved as detached.
}

func (a *acceptedGit) git(index string, input []byte, args ...string) ([]byte, error) {
	env := []string{"GIT_OPTIONAL_LOCKS=0"}
	if index != "" {
		env = append(env, "GIT_INDEX_FILE="+index)
	}
	return localGitEnvironment(a.ctx, a.project, input, env, args...)
}

func (a *acceptedGit) scalar(index string, args ...string) (string, error) {
	raw, err := a.git(index, nil, args...)
	return strings.TrimSpace(string(raw)), err
}

func (a *acceptedGit) checkReference() error {
	ref, err := acceptedReference(a.ctx, a.project)
	if err != nil {
		return err
	}
	if ref != a.intent.Reference {
		return errors.New("local_checkpoint_drift: the original project changed branches; no files were reset")
	}
	return nil
}

func (a *acceptedGit) checkOperation() error {
	for _, name := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "BISECT_LOG", "BISECT_START", "rebase-merge", "rebase-apply", "sequencer"} {
		if _, err := os.Lstat(filepath.Join(filepath.Dir(a.index), name)); err == nil {
			return errors.New("local_checkpoint_git_operation: finish the existing merge, rebase, cherry-pick, revert or bisect first; its state was preserved")
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (a *acceptedGit) acquire() error {
	file, err := os.OpenFile(a.lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if os.IsExist(err) {
		raw, readErr := os.ReadFile(a.lock)
		if readErr == nil && bytes.Equal(raw, a.marker) {
			return nil // Exact nonce proves this is our interrupted lock.
		}
		return errors.New("local_checkpoint_locked: another Git operation holds the index lock; its files were preserved")
	}
	if err != nil {
		return err
	}
	_, err = file.Write(a.marker)
	return errors.Join(err, file.Sync(), file.Close())
}

func (a *acceptedGit) release() {
	raw, err := os.ReadFile(a.lock)
	if err == nil && bytes.Equal(raw, a.marker) {
		_ = os.Remove(a.lock)
	}
}

func (a *acceptedGit) removeTemporary() {
	_ = os.Remove(a.temporary)
	_ = os.Remove(a.temporary + ".lock")
	_ = os.Remove(a.temporary + ".patch")
	if raw, err := os.ReadFile(a.staged); err == nil && a.intent.NewIndexHash != "" && digest(raw) == a.intent.NewIndexHash {
		_ = os.Remove(a.staged)
	}
}

func (a *acceptedGit) freezePatch(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if digest(raw) != a.intent.PatchHash {
		return "", errors.New("local_checkpoint: reviewed patch changed before local checkpoint")
	}
	path = a.temporary + ".patch"
	if err = writeAcceptedIndex(path, raw); err != nil {
		return "", err
	}
	return path, nil
}

func writeAcceptedIndex(path string, raw []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(raw)
	return errors.Join(writeErr, file.Sync(), file.Close())
}

func (a *acceptedGit) prepareExpected(patch string) error {
	actual, err := os.ReadFile(a.index)
	if err != nil {
		return err
	}
	actualHash := digest(actual)
	head, err := a.scalar("", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if err = writeAcceptedIndex(a.temporary, actual); err != nil {
		return err
	}
	indexTree, err := a.scalar(a.temporary, "write-tree")
	if err != nil {
		return fmt.Errorf("local_checkpoint_staged: cannot preserve the original index: %w", err)
	}
	baseTree, err := a.scalar("", "rev-parse", a.intent.Base+"^{tree}")
	if err != nil {
		return err
	}
	tree, err := a.expectedTree(patch, indexTree, baseTree)
	if err != nil {
		return err
	}
	if a.intent.Tree != "" && a.intent.Tree != tree {
		return errors.New("local_checkpoint: reviewed tree no longer matches its journal")
	}
	a.intent.Tree = tree
	if reused, reuseErr := a.reuseExistingCommit(head, indexTree); reused || reuseErr != nil {
		return reuseErr
	}
	if err = a.validateOriginalIndex(head, indexTree, baseTree, actualHash); err != nil {
		return err
	}
	if a.intent.IndexHash == "" {
		a.intent.IndexHash = actualHash
	}
	prepared, err := os.ReadFile(a.temporary)
	if err != nil {
		return err
	}
	a.intent.NewIndexHash = digest(prepared)
	return a.persist(a.intent)
}

func (a *acceptedGit) reuseExistingCommit(head, indexTree string) (bool, error) {
	headTree, err := a.scalar("", "rev-parse", "HEAD^{tree}")
	if err != nil {
		return false, err
	}
	if headTree != a.intent.Tree || indexTree != a.intent.Tree {
		return false, nil
	}
	// Reuse a user-created local version when it is exactly the accepted result.
	if err = a.cleanExpected(); err != nil {
		return false, err
	}
	a.intent.Commit, a.intent.Complete = head, true
	return true, a.persist(a.intent)
}

func (a *acceptedGit) expectedTree(patch, indexTree, baseTree string) (string, error) {
	// Starting from the existing index retains unchanged entries' stat caches.
	if indexTree != baseTree {
		if _, err := a.git(a.temporary, nil, "read-tree", a.intent.Base); err != nil {
			return "", err
		}
	}
	info, err := os.Stat(patch)
	if err != nil {
		return "", err
	}
	if info.Size() != 0 {
		if _, err = a.git(a.temporary, nil, "apply", "--cached", "--binary", "--", patch); err != nil {
			return "", err
		}
	}
	return a.scalar(a.temporary, "write-tree")
}

func (a *acceptedGit) validateOriginalIndex(head, indexTree, baseTree, actualHash string) error {
	if head != a.intent.Base && (a.intent.Commit == "" || head != a.intent.Commit) {
		return errors.New("local_checkpoint_drift: original HEAD changed beyond the accepted result; no changes were committed")
	}
	if indexTree != baseTree {
		if head != a.intent.Commit || indexTree != a.intent.Tree || actualHash != a.intent.NewIndexHash {
			return errors.New("local_checkpoint_staged: manually staged changes were preserved; they were not included in the local checkpoint")
		}
	}
	if a.intent.IndexHash != "" && actualHash != a.intent.IndexHash && actualHash != a.intent.NewIndexHash {
		return errors.New("local_checkpoint_staged: the Git index changed during checkpoint recovery; it was preserved")
	}
	return nil
}

func (a *acceptedGit) cleanExpected() error {
	// Git's plumbing diff-files can report stat-dirty entries without hashing
	// their content. Refresh only our disposable index; never refresh or stage
	// the user's actual index while comparing the reviewed result.
	_, refreshErr := a.git(a.temporary, nil, "update-index", "--refresh")
	raw, err := a.git(a.temporary, nil, "diff-files", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "--")
	if err != nil {
		return err
	}
	if len(raw) != 0 {
		return errors.New("local_checkpoint_drift: original files differ from the accepted result; manual edits were preserved")
	}
	if refreshErr != nil {
		return refreshErr
	}
	raw, err = a.git(a.temporary, nil, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return err
	}
	if len(raw) != 0 {
		return errors.New("local_checkpoint_drift: unrelated new files were preserved; they were not included in the checkpoint")
	}
	// Deleted paths are absent from the expected index; check them explicitly,
	// including paths now ignored by an accepted .gitignore change.
	raw, err = a.git("", nil, "diff", "--name-only", "--diff-filter=D", "-z", a.intent.Base, a.intent.Tree, "--")
	if err != nil {
		return err
	}
	for _, path := range bytes.Split(raw, []byte{0}) {
		if len(path) == 0 {
			continue
		}
		if !safeRelative(string(path)) {
			return errors.New("local_checkpoint: unsafe deleted path")
		}
		if _, statErr := os.Lstat(filepath.Join(a.project, filepath.FromSlash(string(path)))); !os.IsNotExist(statErr) {
			return errors.New("local_checkpoint_drift: a reviewed deletion is no longer present; files were preserved")
		}
	}
	return nil
}

func (a *acceptedGit) finish(patch string, allowApply bool) error {
	if a.intent.Complete {
		return nil
	}
	if err := a.ensureApplied(patch, allowApply); err != nil {
		return err
	}
	if err := a.prepareCommit(); err != nil {
		return err
	}
	if err := a.installCheckpoint(); err != nil {
		return err
	}
	if err := a.verifyInstalled(); err != nil {
		return err
	}
	a.intent.Complete = true
	emitActivity(a.ctx, "phase", "已建立本地版本存档", "仅存档本批已验收修改，不上传 GitHub，也不复制整个项目。")
	return a.persist(a.intent)
}

func (a *acceptedGit) ensureApplied(patch string, allowApply bool) error {
	err := a.cleanExpected()
	if err == nil {
		return nil
	}
	if !allowApply {
		return err
	}
	head, headErr := a.scalar("", "rev-parse", "HEAD")
	if headErr != nil || head != a.intent.Base {
		return err
	}
	// The only alternative eligible for automatic application is the clean
	// original Base. Never merge or stage arbitrary original edits.
	status, statusErr := a.git("", nil, "status", "--porcelain=v1", "--untracked-files=all")
	if statusErr != nil {
		return statusErr
	}
	if len(status) != 0 {
		return err
	}
	if _, applyErr := a.git("", nil, "apply", "--check", "--", patch); applyErr != nil {
		return applyErr
	}
	if _, applyErr := a.git("", nil, "apply", "--", patch); applyErr != nil {
		return applyErr
	}
	return a.cleanExpected()
}

func (a *acceptedGit) prepareCommit() error {
	prepared, err := os.ReadFile(a.temporary)
	if err != nil {
		return err
	}
	a.intent.NewIndexHash = digest(prepared)
	if err = a.persist(a.intent); err != nil {
		return err
	}
	if a.intent.Commit == "" {
		message := []byte("chore: save accepted Xiaobuding changes\n\nCheckpoint: " + a.intent.Nonce + "\nPatch: " + a.intent.PatchHash + "\n")
		raw, err := a.git("", message, "-c", "user.name=Xiaobuding Local", "-c", "user.email=local@xiaobuding.invalid", "commit-tree", a.intent.Tree, "-p", a.intent.Base)
		if err != nil {
			return err
		}
		a.intent.Commit = strings.TrimSpace(string(raw))
		if err = a.persist(a.intent); err != nil {
			return err
		}
	}
	return nil
}

func (a *acceptedGit) installCheckpoint() error {
	if err := a.checkReference(); err != nil {
		return err
	}
	if err := a.checkOperation(); err != nil {
		return err
	}
	current, err := os.ReadFile(a.index)
	if err != nil {
		return err
	}
	if digest(current) != a.intent.IndexHash && digest(current) != a.intent.NewIndexHash {
		return errors.New("local_checkpoint_staged: the Git index changed; it was preserved")
	}
	head, err := a.scalar("", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if head == a.intent.Base {
		if _, err = a.git("", nil, "update-ref", "--no-deref", "-m", "xiaobuding: accepted local checkpoint", a.intent.Reference, a.intent.Commit, a.intent.Base); err != nil {
			return err
		}
	} else if head != a.intent.Commit {
		return errors.New("local_checkpoint_drift: HEAD changed before local checkpoint; no files were reset")
	}
	prepared, err := os.ReadFile(a.temporary)
	if err != nil || digest(prepared) != a.intent.NewIndexHash {
		return errors.New("local_checkpoint: prepared index changed")
	}
	if digest(current) != a.intent.NewIndexHash {
		if err = writeAcceptedIndex(a.staged, prepared); err != nil {
			return err
		}
		if err = os.Rename(a.staged, a.index); err != nil {
			return err
		}
	}
	return nil
}

func (a *acceptedGit) verifyInstalled() error {
	if err := a.checkReference(); err != nil {
		return err
	}
	head, err := a.scalar("", "rev-parse", "HEAD")
	if err != nil || head != a.intent.Commit {
		return errors.New("local_checkpoint_drift: HEAD moved during checkpoint; existing files were preserved")
	}
	if err = a.cleanExpected(); err != nil {
		return err
	}
	stagedDiff, err := a.git("", nil, "diff-index", "--cached", "--name-only", "-z", a.intent.Tree, "--")
	if err != nil || len(stagedDiff) != 0 {
		return errors.New("local_checkpoint: actual Git index does not match the saved version")
	}
	return nil
}
