package reliable

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Engine struct {
	activityMu sync.Mutex
	catalogMu  sync.RWMutex
	Providers  []Provider
	Store      *Store
	Root       string
	Previews   *PreviewManager
	Runner     AgentRunner
	Redact     func([]byte) []byte
	mu         sync.Mutex
	active     map[string]context.CancelFunc
	questions  map[string]bool
	wg         sync.WaitGroup
	closing    bool
}

func (e *Engine) reserve(ctx context.Context, id string) (context.Context, func(), error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.reserveLocked(ctx, id)
}

func (e *Engine) reserveLocked(ctx context.Context, id string) (context.Context, func(), error) {
	if e.closing {
		return nil, nil, errors.New("application is shutting down")
	}
	if e.active == nil {
		e.active = make(map[string]context.CancelFunc)
	}
	if _, ok := e.active[id]; ok {
		return nil, nil, errors.New("task is already running")
	}
	if e.Previews != nil {
		if err := e.Previews.StopTask(id); err != nil {
			return nil, nil, err
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	ctx = storageContext(ctx, e.Root)
	ctx = pinCodexExecutables(ctx, e.Runner)
	e.active[id] = cancel
	e.wg.Add(1)
	return ctx, func() {
		cancel()
		e.mu.Lock()
		delete(e.active, id)
		delete(e.questions, id)
		e.mu.Unlock()
		e.kickQuestions(id)
		e.kickProjectNext(id)
		e.wg.Done()
	}, nil
}

func (e *Engine) Close() error {
	e.mu.Lock()
	e.closing = true
	for _, cancel := range e.active {
		cancel()
	}
	e.mu.Unlock()
	e.wg.Wait()
	var previewErr error
	if e.Previews != nil {
		previewErr = e.Previews.Close()
	}
	return errors.Join(previewErr, e.Store.Close())
}

func (e *Engine) RunTask(ctx context.Context, id string) (runErr error) {
	ctx, release, err := e.reserve(ctx, id)
	if err != nil {
		return err
	}
	defer release()
	return e.runTask(ctx, id)
}

func (e *Engine) runTask(ctx context.Context, id string) (runErr error) {
	ctx = observeActivity(ctx, func(event Activity) { e.appendActivity(id, event) })
	t, err := e.Store.Get(id)
	if err != nil {
		return err
	}
	if t.ArchivedAt != "" {
		return errors.New("task_archived: restore the task before continuing")
	}
	if acceptancePending(t) {
		err = e.finishAcceptance(ctx, t)
		e.recordRunFailure(id, err)
		return err
	}
	if t.State != stateQueued && t.State != stateBlocked && t.State != stateInterrupted {
		return fmt.Errorf("cannot run task from %s", t.State)
	}
	resumeDomain := domainStageResumeEligible(t)
	if t.Round >= MaxRounds && !resumeDomain {
		return errors.New("three implementation rounds have been used")
	}
	ctx, cleanupProof := observeOwnedCleanup(ctx)
	priorQuiescent := e.processQuiescent(t) == nil
	defer e.finishRunProcessCleanup(ctx, t, cleanupProof, priorQuiescent)
	defer func() { e.recordRunFailure(id, runErr) }()
	emitActivity(ctx, "state", "开始处理任务", t.Goal)
	if resumeDomain {
		return e.resumeDomainStage(ctx, t)
	}
	if t.ReadOnly {
		return e.inspectProject(ctx, t)
	}
	return e.prepareAndRunTask(ctx, t)
}

func (e *Engine) prepareAndRunTask(ctx context.Context, t *Task) error {
	if err := e.CheckStorageBudget(ctx); err != nil {
		return err
	}
	id := t.ID
	seedPrepared := t.SeedPending != nil
	var err error
	if seedPrepared {
		if err = e.prepareExhaustedSeed(ctx, t); err != nil {
			return err
		}
	}
	if t.RecoveryPending != nil || len(t.RecoveryEvidence) > 0 {
		if err = e.beginRecoveryValidation(t); err == nil {
			emitActivity(ctx, "phase", "正在检查保留的副本", "保留已有修改和已用轮次；通过检查后重新调查。")
			err = e.validateRecovery(ctx, t)
		}
		if err != nil {
			e.failRecovery(id, err)
			return err
		}
	}
	if len(t.Commands) == 0 && !t.AutoChecks {
		return errors.New("no verification commands were configured; automatic acceptance is disabled")
	}
	if !seedPrepared && len(t.SeedEvidence) > 0 {
		if err = e.prepareExhaustedSeed(ctx, t); err != nil {
			return err
		}
	}
	if err = e.prepare(ctx, t); err != nil {
		return err
	}
	return e.runRounds(ctx, t)
}

func (e *Engine) finishRunProcessCleanup(ctx context.Context, t *Task, proof *ownedCleanupProof, priorQuiescent bool) {
	if !priorQuiescent || t.ReadOnly {
		if proof.verified() {
			return
		}
	}
	current, err := e.Store.Get(t.ID)
	if err != nil {
		return
	}
	if err = e.recordProcessQuiescence(current, proof); err != nil {
		emitActivity(ctx, "error", "进程收尾记录未完成", err.Error())
	}
}

func (e *Engine) recordRunFailure(id string, runErr error) {
	if runErr == nil {
		return
	}
	_, _ = e.Store.Update(id, func(t *Task) error {
		if t.State == stateCancelling || t.State == stateCancelled {
			t.State = stateCancelled
		} else if t.State != stateExhausted {
			t.State = stateBlocked
		}
		t.Reason = runErr.Error()
		return nil
	})
}

func (e *Engine) runRounds(ctx context.Context, t *Task) error {
	var err error
	id := t.ID
	for t.Round < MaxRounds {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = e.audit(ctx, t); err != nil {
			return err
		}
		t, err = e.Store.StartRound(id)
		if errors.Is(err, errRequirementsChanged) {
			t, err = e.Store.Get(id)
			if err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if err = e.implement(ctx, t); err != nil {
			return err
		}
		decision, reason, err := e.verifyAndReview(ctx, t)
		if errors.Is(err, errRequirementsChanged) {
			decision, reason, err = Rework, "New user requirements require a fresh audit and verification", nil
		}
		if err != nil {
			return err
		}
		if decision == Pass {
			return nil // The passing review atomically sealed evidence and state.
		}
		if decision == Block {
			return errors.New(reason)
		}
		if t.Round == MaxRounds {
			_ = e.setState(id, stateExhausted, reason)
			return errors.New("three rounds exhausted: " + reason)
		}
		if err = e.setState(id, stateAuditing, reason); err != nil {
			return err
		}
		t, err = e.Store.Get(id)
		if err != nil {
			return err
		}
	}
	return errors.New("three implementation rounds have been used")
}

func (e *Engine) setState(id, state, reason string) error {
	_, err := e.Store.Update(id, func(t *Task) error {
		if t.State == stateCancelling || t.State == stateCancelled {
			return context.Canceled
		}
		t.State = state
		t.Reason = reason
		return nil
	})
	return err
}

func (e *Engine) prepare(ctx context.Context, t *Task) error {
	if t.Worktree != "" {
		if err := e.restoreWorktreeContract(ctx, t); err != nil {
			return err
		}
		return e.setState(t.ID, stateAuditing, t.Reason)
	}
	if err := preflightWorktree(ctx, t.Project); err != nil {
		return err
	}
	point, err := e.prepareLocalRecoveryPoint(ctx, t)
	if err != nil {
		return err
	}
	dir, base, err := PrepareWorktree(ctx, e.Root, t.Project, t.ID)
	if err != nil {
		return err
	}
	if point != nil && point.Head != base {
		return errors.New("local_snapshot_changed: original project changed during preparation")
	}
	baseline, err := TakeSnapshot(ctx, dir)
	if err != nil {
		return err
	}
	t.Worktree = dir
	t.Base = base
	t.Baseline = baseline
	if _, err = e.Store.Update(t.ID, func(saved *Task) error {
		saved.Worktree = dir
		saved.Base = base
		saved.Baseline = baseline
		saved.State = stateAuditing
		return nil
	}); err != nil {
		return err
	}
	return e.saveJSON(t.ID, "contract.json", t)
}

func (e *Engine) restoreWorktreeContract(ctx context.Context, t *Task) error {
	if t.Baseline == nil {
		return errors.New("initial source manifest is missing")
	}
	if err := requireClean(ctx, t.Project, t.Base); err != nil {
		return err
	}
	_, err := os.Stat(filepath.Join(e.Root, "evidence", t.ID, "contract.json"))
	if os.IsNotExist(err) {
		return e.saveJSON(t.ID, "contract.json", t)
	}
	return err
}

func (e *Engine) saveJSON(id, name string, value any) error {
	dir := filepath.Join(e.Root, "evidence", id)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if e.Redact != nil {
		raw = e.Redact(raw)
	}
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(raw)
	syncErr := f.Sync()
	closeErr := f.Close()
	return errors.Join(writeErr, syncErr, closeErr)
}

func digest(raw []byte) string { hash := sha256.Sum256(raw); return hex.EncodeToString(hash[:]) }

func (e *Engine) Cancel(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.questions[id] {
		task, err := e.Store.Get(id)
		if err != nil {
			return err
		}
		if task.SuccessorTaskID != "" {
			return fmt.Errorf("batch_successor_exists: continue or stop the linked batch %s", task.SuccessorTaskID)
		}
		if acceptancePending(task) {
			return errors.New("local_checkpoint_pending: finish the previously accepted local checkpoint before stopping")
		}
	}
	if cancel, ok := e.active[id]; ok {
		if e.questions[id] {
			if err := e.cancelQuestions(id); err != nil {
				return err
			}
			cancel()
			return nil
		}
		t, err := e.Store.Get(id)
		if err != nil {
			return err
		}
		if t.State == stateApplying {
			return errors.New("application of accepted changes is in progress")
		}
		if _, err = e.Store.Update(id, func(t *Task) error {
			t.State = stateCancelling
			t.ContinuePending = false
			cancelPendingQuestions(t)
			return nil
		}); err != nil {
			return err
		}
		cancel()
		return nil
	}
	_, err := e.Store.Update(id, func(t *Task) error {
		if t.State == stateAccepted {
			return errors.New("accepted tasks cannot be cancelled")
		}
		t.State = stateCancelled
		t.ContinuePending = false
		cancelPendingQuestions(t)
		t.Reason = "Cancelled by the user; evidence and worktree have been retained"
		return nil
	})
	return err
}
