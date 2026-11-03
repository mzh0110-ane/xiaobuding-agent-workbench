package reliable

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
)

func (h *HTTP) attachmentRoute(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 && r.Method == http.MethodPost {
		h.uploadAttachment(w, r)
		return
	}
	if r.Method == http.MethodGet && (len(parts) == 1 || len(parts) == 2 && parts[1] == "preview") {
		h.downloadAttachment(w, r, parts[0], len(parts) == 2)
		return
	}
	http.NotFound(w, r)
}

func (h *HTTP) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, attachmentByteLimit+(1<<20))
	reader, err := r.MultipartReader()
	if err != nil {
		h.reply(w, http.StatusBadRequest, map[string]string{"error": "attachment_invalid: multipart file required"})
		return
	}
	part, err := reader.NextPart()
	if err != nil || part.FormName() != "file" || part.FileName() == "" {
		h.reply(w, http.StatusBadRequest, map[string]string{"error": "attachment_invalid: upload exactly one file field"})
		return
	}
	defer func() { _ = part.Close() }()
	finish := func() error {
		if err := r.Context().Err(); err != nil {
			return err
		}
		_, nextErr := reader.NextPart()
		if !errors.Is(nextErr, io.EOF) {
			return errors.New("attachment_invalid: upload exactly one file per request")
		}
		return nil
	}
	meta, err := h.Engine.storeAttachment(r.Context(), part.FileName(), part, finish)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "attachment_too_large") {
			status = http.StatusRequestEntityTooLarge
		}
		h.reply(w, status, map[string]string{"error": err.Error()})
		return
	}
	h.reply(w, http.StatusCreated, meta)
}

func (h *HTTP) downloadAttachment(w http.ResponseWriter, r *http.Request, id string, preview bool) {
	meta, file, err := h.Engine.openAttachment(id)
	if err != nil {
		h.reply(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	defer func() { _ = file.Close() }()
	if preview && (!meta.Image || !nativeImageMediaType(meta.MediaType)) {
		h.reply(w, http.StatusUnsupportedMediaType, map[string]string{"error": "attachment_preview_unavailable: this format is available as a download"})
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": meta.Name}))
	if preview {
		w.Header().Set("Content-Type", meta.MediaType)
		w.Header().Set("Content-Disposition", "inline")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	}
	_, _ = io.Copy(w, file)
}
