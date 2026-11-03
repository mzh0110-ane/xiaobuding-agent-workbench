package reliable

import "net/http"

func (h *HTTP) workbenchTaskRoute(w http.ResponseWriter, r *http.Request, id, action string) bool {
	switch action {
	case "replan":
		h.auditReplanRoute(w, r, id)
	case "preview":
		h.previewRoute(w, r, id)
	case "files":
		h.resultFilesRoute(w, r, id)
	case "recovery":
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return true
		}
		task, _ := h.Engine.Store.Get(id)
		h.reply(w, http.StatusOK, RecoveryGuidanceFor(task))
	default:
		return false
	}
	return true
}

func (h *HTTP) storageBudgetRoute(w http.ResponseWriter, r *http.Request) {
	var report *StorageBudgetReport
	var err error
	switch r.Method {
	case http.MethodGet:
		report, err = h.Engine.StorageBudget(r.Context())
	case http.MethodPost:
		var edit StorageBudgetEdit
		if err = requestJSON(r, &edit); err == nil {
			report, err = h.Engine.EditStorageBudget(r.Context(), edit)
		}
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	h.reply(w, http.StatusOK, report)
}
