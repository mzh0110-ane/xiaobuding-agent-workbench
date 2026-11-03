package reliable

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenewalPendingRejectsNewRequirements(t *testing.T) {
	e, task := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	_, err := e.Store.Update(task.ID, func(saved *Task) error {
		return json.Unmarshal([]byte(`{"seed_pending":{"parent_id":"prior-batch"}}`), saved)
	})
	require.NoError(t, err)
	h := &HTTP{Engine: e, Token: "renewal-pending-test"}
	response := httpRequest(h, http.MethodPost, "/api/v1/reliable/tasks/"+task.ID+"/messages", `{"id":"new-requirement","kind":"requirement","content":"Change the pending seed"}`, h.Token)
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), "renewal_active")
	current, err := e.Store.Get(task.ID)
	require.NoError(t, err)
	require.Empty(t, current.Messages)
	require.Zero(t, current.RequirementRevision)
}
