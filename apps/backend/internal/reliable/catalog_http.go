package reliable

import (
	"errors"
	"net/http"
)

func (h *HTTP) refreshCatalog(w http.ResponseWriter, r *http.Request) {
	var body struct{}
	if err := requestJSON(r, &body); err != nil {
		h.fail(w, err)
		return
	}
	if h.ReloadCatalog == nil {
		h.fail(w, errors.New("native catalog reload is unavailable"))
		return
	}
	h.catalogMu.Lock()
	defer h.catalogMu.Unlock()
	providers, err := h.ReloadCatalog()
	if err != nil {
		h.fail(w, errors.New("native catalog reload failed; check the original tools"))
		return
	}
	h.Engine.catalogMu.Lock()
	h.Engine.Providers = providers
	h.Engine.catalogMu.Unlock()
	h.reply(w, 200, map[string]any{"providers": providers, "default_plan": h.Engine.defaultTeamPlan()})
}
