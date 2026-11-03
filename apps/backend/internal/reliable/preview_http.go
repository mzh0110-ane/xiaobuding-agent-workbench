package reliable

import (
	"errors"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func (h *HTTP) previewRoute(w http.ResponseWriter, r *http.Request, id string) {
	var result *PreviewStatus
	var err error
	switch r.Method {
	case http.MethodGet:
		result, err = h.Engine.Preview(r.Context(), id)
	case http.MethodPost:
		var body struct {
			Action string `json:"action"`
			Entry  string `json:"entry,omitempty"`
		}
		if err = requestJSON(r, &body); err == nil {
			switch body.Action {
			case "start":
				result, err = h.Engine.StartPreview(r.Context(), id, body.Entry)
			case "stop":
				err = h.Engine.previewManager().StopTask(id)
				result = &PreviewStatus{TaskID: id, Reason: "preview_stopped"}
			default:
				err = errors.New("preview_action: explicitly select start or stop")
			}
		}
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	h.reply(w, http.StatusOK, result)
}

func staticPreviewHandler(engine *Engine, task *Task, entry *managedPreview, host string) http.Handler {
	identity, _ := entry.root.Stat(".")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'; base-uri 'self'")
		if !previewRequestLocal(r, host) || r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Local read-only preview required", http.StatusForbidden)
			return
		}
		if !previewCandidateAvailable(engine, task, entry, identity) {
			http.Error(w, "Candidate is no longer available for preview", http.StatusConflict)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if entry.status.Kind == previewKindVite {
			serveVitePreview(w, r, task, entry, name)
			return
		}
		if name == "" {
			name = path.Base(entry.status.Entry)
		} else if strings.HasSuffix(name, "/") {
			name += "index.html"
		}
		if !previewAssetType(name) {
			http.NotFound(w, r)
			return
		}
		serveStaticPreviewFile(w, r, entry, name)
	})
}

func previewCandidateAvailable(engine *Engine, task *Task, entry *managedPreview, identity os.FileInfo) bool {
	fresh, err := engine.Store.Get(task.ID)
	if err != nil || fresh.Worktree != task.Worktree || fresh.Base != task.Base || engine.resultCandidateIdle(fresh) != nil {
		return false
	}
	info, err := os.Lstat(filepath.Join(task.Worktree, filepath.FromSlash(path.Dir(entry.status.Entry))))
	return err == nil && identity != nil && os.SameFile(identity, info)
}

func serveVitePreview(w http.ResponseWriter, r *http.Request, task *Task, entry *managedPreview, name string) {
	if !vitePreviewPathAllowed(task.Worktree, name) {
		http.NotFound(w, r)
		return
	}
	entry.mu.RLock()
	proxy := entry.proxy
	entry.mu.RUnlock()
	if proxy == nil {
		http.Error(w, "Preview runtime is starting", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("X-XBD-Preview-Source", entry.status.SourceHash)
	proxy.ServeHTTP(w, r)
}

func serveStaticPreviewFile(w http.ResponseWriter, r *http.Request, entry *managedPreview, name string) {
	file, err := openResultFile(entry.root, name, 16<<20)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = file.Close() }()
	w.Header().Set("X-XBD-Preview-Source", entry.status.SourceHash)
	http.ServeContent(w, r, path.Base(name), time.Time{}, file)
}
