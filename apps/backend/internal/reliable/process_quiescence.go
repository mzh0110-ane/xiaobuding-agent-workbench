package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type ownedCleanupKey struct{}

const quiescenceServiceExit = "windows-service-exit"

type ownedCleanupProof struct {
	mu     sync.Mutex
	active int
	failed bool
}

func observeOwnedCleanup(ctx context.Context) (context.Context, *ownedCleanupProof) {
	proof := &ownedCleanupProof{}
	return context.WithValue(ctx, ownedCleanupKey{}, proof), proof
}

func (p *ownedCleanupProof) verified() bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active == 0 && !p.failed
}

func trackOwnedCleanup(ctx context.Context) func(error) {
	proof, ok := ctx.Value(ownedCleanupKey{}).(*ownedCleanupProof)
	if !ok {
		return func(error) {}
	}
	proof.mu.Lock()
	proof.active++
	proof.mu.Unlock()
	return func(err error) {
		proof.mu.Lock()
		proof.active--
		proof.failed = proof.failed || err != nil
		proof.mu.Unlock()
	}
}

type processQuiescenceReceipt struct {
	TaskID   string `json:"task_id"`
	Base     string `json:"base"`
	RunsHash string `json:"runs_hash"`
	Method   string `json:"method"`
	Epoch    string `json:"cleanup_epoch,omitempty"`
}

func quiescenceReceipt(t *Task) (string, processQuiescenceReceipt, error) {
	var receipt processQuiescenceReceipt
	if !safeRelative(t.ID) || strings.Contains(t.ID, "/") {
		return "", receipt, errors.New("invalid task identity")
	}
	raw, err := json.Marshal(t.Runs)
	if err != nil {
		return "", receipt, err
	}
	receipt = processQuiescenceReceipt{TaskID: t.ID, Base: t.Base, RunsHash: digest(raw), Method: "owned-process-cleanup", Epoch: t.ProcessCleanupEpoch}
	key := receipt.RunsHash
	if receipt.Epoch != "" {
		key += "-" + digest([]byte(receipt.Epoch))
	}
	return filepath.Join("quiescence", t.ID, key+".json"), receipt, nil
}

func hasNativeRuns(t *Task) bool {
	for _, dispatch := range t.Runs {
		if dispatch.Status != dispatchNotStarted {
			return true
		}
	}
	return false
}

func (e *Engine) processQuiescent(t *Task) error {
	if !hasNativeRuns(t) && t.ProcessCleanupEpoch == "" {
		return nil
	}
	name, want, err := quiescenceReceipt(t)
	if err == nil {
		err = e.readQuiescenceReceipt(name, want)
	}
	if err != nil {
		return fmt.Errorf("recovery_process_unknown: previous owned process cleanup cannot be verified: %w", err)
	}
	return nil
}

func (e *Engine) readQuiescenceReceipt(name string, want processQuiescenceReceipt) error {
	root, err := os.OpenRoot(e.Root)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	file, err := root.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 4096 {
		return errors.New("unsafe process cleanup receipt")
	}
	raw, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil {
		return err
	}
	var saved processQuiescenceReceipt
	if decodeStrict(raw, &saved) != nil || saved.TaskID != want.TaskID || saved.Base != want.Base || saved.RunsHash != want.RunsHash || saved.Epoch != want.Epoch {
		return errors.New("process cleanup receipt does not match this task history")
	}
	if saved.Method != "owned-process-cleanup" && (saved.Method != quiescenceServiceExit || runtime.GOOS != "windows") {
		return errors.New("unknown process cleanup method")
	}
	return nil
}

func (e *Engine) recordProcessQuiescence(t *Task, proof *ownedCleanupProof) error {
	needsReceipt := hasNativeRuns(t) || t.ProcessCleanupEpoch != ""
	if !needsReceipt && proof == nil {
		return nil
	}
	if !proof.verified() {
		return errors.Join(errors.New("recovery_process_unknown: owned process cleanup was not confirmed"), e.invalidateProcessQuiescence(t.ID))
	}
	if !needsReceipt {
		return nil
	}
	name, receipt, err := quiescenceReceipt(t)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(e.Root)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	if err = root.MkdirAll(filepath.Dir(name), 0700); err != nil {
		return err
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if os.IsExist(err) {
		return e.readQuiescenceReceipt(name, receipt)
	}
	if err != nil {
		return err
	}
	_, writeErr := file.Write(raw)
	return errors.Join(writeErr, file.Sync(), file.Close())
}

func (e *Engine) invalidateProcessQuiescence(id string) error {
	if e.Store == nil {
		return nil
	}
	_, err := e.Store.Update(id, func(saved *Task) error {
		saved.ProcessCleanupEpoch = randomID()
		return nil
	})
	return err
}
