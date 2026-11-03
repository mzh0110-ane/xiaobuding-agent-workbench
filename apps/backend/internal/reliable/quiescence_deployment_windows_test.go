package reliable

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"golang.org/x/sys/windows"
)

func TestRetainedCancelledCandidateDeploymentPreflight(t *testing.T) {
	dbPath, root, id := os.Getenv("XIAOBUDING_QUIESCENCE_COPY_DB"), os.Getenv("XIAOBUDING_QUIESCENCE_ROOT"), os.Getenv("XIAOBUDING_PREFLIGHT_CANCELLED_TASK")
	if dbPath == "" || root == "" || id == "" {
		t.Skip("explicit retained-candidate deployment preflight is disabled")
	}
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	task, err := store.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	engine := &Engine{Store: store, Root: root}
	for _, validate := range []func(*Task) error{recoveryEligible, engine.recoveryQuiescence, engine.validateRecoveryHistory} {
		if err = validate(task); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = engine.validateRecoveryCandidate(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	t.Logf("validated retained task %s at round %d without recovery, dispatch, receipt publication or source edits", id, task.Round)
}

// This opt-in deployment check requires a stopped owner's handle to remain held
// by the installer, preventing reuse of its PID until receipts are published.
func TestStoppedServiceQuiescenceReceipts(t *testing.T) {
	dbPath, root := os.Getenv("XIAOBUDING_QUIESCENCE_COPY_DB"), os.Getenv("XIAOBUDING_QUIESCENCE_ROOT")
	if dbPath == "" || root == "" {
		t.Skip("explicit stopped-service migration is disabled")
	}
	pid, err := strconv.Atoi(os.Getenv("XIAOBUDING_STOPPED_SERVICE_PID"))
	if err != nil || pid <= 0 {
		t.Fatal("a verified stopped service PID is required")
	}
	owner, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = windows.CloseHandle(owner) }()
	status, waitErr := windows.WaitForSingleObject(owner, 0)
	if waitErr != nil || status != windows.WAIT_OBJECT_0 {
		t.Fatal("previous service exit is not proven; no process proof was published")
	}
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	tasks, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	engine := &Engine{Root: root}
	for _, task := range tasks {
		name, receipt, receiptErr := quiescenceReceipt(task)
		if receiptErr != nil {
			t.Fatal(receiptErr)
		}
		receipt.Method = quiescenceServiceExit
		target := filepath.Join(root, name)
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err = exclusiveJSON(target, receipt); err != nil && !os.IsExist(err) {
			t.Fatal(err)
		}
		if err = engine.readQuiescenceReceipt(name, receipt); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("published stopped Windows service proof for %d retained task histories without executing or editing a task", len(tasks))
}
