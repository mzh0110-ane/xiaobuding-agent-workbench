package reliable

import "net/http"

func (h *HTTP) updateModels(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		ExpectedRevision int       `json:"expected_revision"`
		Plan             ModelPlan `json:"plan"`
	}
	if err := requestJSON(r, &body); err != nil {
		h.reply(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	task, err := h.Engine.UpdateModelPlan(id, body.ExpectedRevision, body.Plan)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.reply(w, http.StatusOK, task)
}
