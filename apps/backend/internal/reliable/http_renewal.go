package reliable

import "net/http"

func (h *HTTP) startExhaustedBatch(w http.ResponseWriter, r *http.Request, id string) {
	var request messageRequest
	if err := requestJSON(r, &request); err != nil {
		h.reply(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	message, err := h.Engine.StartExhaustedBatch(id, request)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.Engine.kickQuestions(message.LinkedTask)
	h.reply(w, http.StatusAccepted, message)
}
