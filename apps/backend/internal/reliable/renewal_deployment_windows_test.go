package reliable

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExistingExhaustedCandidateDeploymentPreflight(t *testing.T) {
	path, root, id := os.Getenv("XIAOBUDING_PREFLIGHT_TASKS_JSON"), os.Getenv("XIAOBUDING_PREFLIGHT_EVIDENCE_ROOT"), os.Getenv("XIAOBUDING_PREFLIGHT_TASK_ID")
	if path == "" || root == "" || id == "" {
		t.Skip("explicit read-only exhausted-candidate deployment preflight is disabled")
	}
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var tasks []*Task
	require.NoError(t, json.Unmarshal(raw, &tasks))
	var parent *Task
	for _, task := range tasks {
		if task.ID == id {
			parent = task
		}
	}
	require.NotNil(t, parent)
	require.NoError(t, exhaustedBatchEligible(parent))
	engine := &Engine{Root: root}
	require.NoError(t, engine.recoveryQuiescence(parent))
	require.NoError(t, engine.validateRecoveryHistory(parent))
	snap, err := engine.validateRecoveryCandidate(context.Background(), parent)
	require.NoError(t, err)
	if parent.Snapshot != "" {
		require.Equal(t, parent.Snapshot, snap.Hash)
	}
	t.Logf("verified exhausted task %s at %d/3 for explicit renewal without creating a child, changing source or invoking a model", parent.ID, parent.Round)
}
