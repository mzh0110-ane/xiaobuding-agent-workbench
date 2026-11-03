package reliable

import "net/http"

func (h *HTTP) sharedContextStatusRoute(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	status, err := h.Engine.SharedContextStatus(id)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.reply(w, http.StatusOK, status)
}
