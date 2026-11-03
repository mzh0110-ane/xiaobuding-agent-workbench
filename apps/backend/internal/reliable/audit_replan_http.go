package reliable

import "net/http"

type auditReplanRequest struct {
	SourceHash string `json:"source_hash"`
	Confirm    bool   `json:"confirm"`
	Note       string `json:"note,omitempty"`
}

func (h *HTTP) auditReplanRoute(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		result, err := h.Engine.InspectAuditReplan(r.Context(), id)
		if err != nil {
			h.fail(w, err)
			return
		}
		h.reply(w, http.StatusOK, result)
	case http.MethodPost:
		var request auditReplanRequest
		if err := requestJSON(r, &request); err != nil {
			h.reply(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		result, err := h.Engine.ConfirmAuditReplan(r.Context(), id, request)
		if err != nil {
			h.fail(w, err)
			return
		}
		h.reply(w, http.StatusAccepted, result)
	default:
		http.NotFound(w, r)
	}
}
