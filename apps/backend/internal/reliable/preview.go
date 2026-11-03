package reliable

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	previewStopping       = "preview_stopping"
	previewCleanupUnknown = "preview_cleanup_unknown"
	previewKindVite       = "vite"
	previewViteBasic      = "preview_vite_basic"
)

type PreviewStatus struct {
	TaskID            string   `json:"task_id"`
	Supported         bool     `json:"supported"`
	Kind              string   `json:"kind"`
	Entry             string   `json:"entry"`
	Candidates        []string `json:"candidates,omitempty"`
	URL               string   `json:"url,omitempty"`
	Running           bool     `json:"running"`
	SourceHash        string   `json:"source_hash"`
	CurrentSourceHash string   `json:"current_source_hash,omitempty"`
	Stale             bool     `json:"stale"`
	Reason            string   `json:"reason"`
	LogTail           string   `json:"log_tail,omitempty"`
	BuildUnverified   bool     `json:"build_unverified"`
}

type managedPreview struct {
	mu       sync.RWMutex
	status   PreviewStatus
	server   *http.Server
	root     *os.Root
	done     chan struct{}
	stop     func() error
	proxy    http.Handler
	log      *previewLog
	stopping bool
	stopDone chan struct{}
	stopErr  error
}

type PreviewManager struct {
	mu      sync.Mutex
	entries map[string]*managedPreview
	closed  bool
}

func (e *Engine) previewManager() *PreviewManager {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.Previews == nil {
		if e.closing {
			return &PreviewManager{entries: make(map[string]*managedPreview), closed: true}
		}
		e.Previews = &PreviewManager{entries: make(map[string]*managedPreview)}
	}
	return e.Previews
}

func (manager *PreviewManager) status(id string) *PreviewStatus {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if entry := manager.entries[id]; entry != nil {
		status := entry.status
		if entry.log != nil {
			status.LogTail = entry.log.Tail()
		}
		return &status
	}
	return nil
}

func (manager *PreviewManager) StopTask(id string) error {
	manager.mu.Lock()
	entry := manager.entries[id]
	if entry == nil {
		manager.mu.Unlock()
		return nil
	}
	if entry.stopping {
		done := entry.stopDone
		manager.mu.Unlock()
		<-done
		return entry.stopErr
	}
	entry.stopping, entry.stopDone = true, make(chan struct{})
	entry.status.Running, entry.status.Reason = false, previewStopping
	manager.mu.Unlock()
	err := closeManagedPreview(entry)
	manager.mu.Lock()
	entry.stopErr = err
	if manager.entries[id] == entry {
		if err != nil {
			entry.status.Reason = previewCleanupUnknown
		} else {
			delete(manager.entries, id)
		}
	}
	close(entry.stopDone)
	manager.mu.Unlock()
	return err
}

func (manager *PreviewManager) Close() error {
	manager.mu.Lock()
	manager.closed = true
	ids := make([]string, 0, len(manager.entries))
	for id := range manager.entries {
		ids = append(ids, id)
	}
	manager.mu.Unlock()
	var result error
	for _, id := range ids {
		result = errors.Join(result, manager.StopTask(id))
	}
	return result
}

func closeManagedPreview(entry *managedPreview) error {
	if entry == nil {
		return nil
	}
	err := entry.server.Close()
	if entry.stop != nil {
		err = errors.Join(err, entry.stop())
	}
	<-entry.done
	return errors.Join(err, entry.root.Close())
}

func detectStaticPreview(task *Task, snapshot *Snapshot) PreviewStatus {
	status := PreviewStatus{TaskID: task.ID, Kind: "static", Reason: "preview_no_static_entry"}
	root, err := os.OpenRoot(task.Worktree)
	if err != nil {
		return status
	}
	defer func() { _ = root.Close() }()
	names := []string{"dist/index.html", "build/index.html", "index.html", "public/index.html"}
	if snapshot != nil {
		for _, name := range changedFiles(task.Baseline, snapshot) {
			if strings.EqualFold(path.Ext(name), ".html") && len(names) < 24 {
				names = append(names, name)
			}
		}
	}
	seen := make(map[string]bool)
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		file, openErr := openResultFile(root, name, 16<<20)
		if openErr != nil {
			continue
		}
		_ = file.Close()
		status.Candidates = append(status.Candidates, name)
	}
	if len(status.Candidates) > 0 {
		status.Supported, status.Entry, status.Reason = true, status.Candidates[0], "preview_static_ready"
		if previewBuildEntry(status.Entry) {
			status.BuildUnverified, status.Reason = true, "preview_build_unverified"
		}
	}
	return status
}

func (e *Engine) Preview(ctx context.Context, id string) (*PreviewStatus, error) {
	manager := e.previewManager()
	running := manager.status(id)
	task, snapshot, err := e.resultCandidate(ctx, id)
	if err != nil {
		if running != nil {
			running.Stale = true
			if !previewCleanupPending(running) {
				running.Reason = err.Error()
			}
			return running, nil
		}
		return &PreviewStatus{TaskID: id, Reason: err.Error()}, nil
	}
	if running != nil {
		running.CurrentSourceHash = snapshot.Hash
		running.Stale = running.SourceHash != snapshot.Hash
		if running.Stale {
			if !previewCleanupPending(running) {
				running.Reason = "preview_source_changed"
			}
		}
		return running, nil
	}
	status := detectPreview(task, snapshot)
	status.SourceHash, status.CurrentSourceHash = snapshot.Hash, snapshot.Hash
	return &status, nil
}

func previewCleanupPending(status *PreviewStatus) bool {
	return status.Reason == previewStopping || status.Reason == previewCleanupUnknown
}

func (e *Engine) StartPreview(ctx context.Context, id, selected string) (*PreviewStatus, error) {
	task, snapshot, err := e.resultCandidate(ctx, id)
	if err != nil {
		return nil, err
	}
	status, err := selectPreviewStatus(detectPreview(task, snapshot), selected)
	if err != nil {
		return nil, err
	}
	status.SourceHash, status.CurrentSourceHash = snapshot.Hash, snapshot.Hash
	manager := e.previewManager()
	current, err := manager.existingPreview(id, status)
	if err != nil || current != nil {
		return current, err
	}
	if err = e.CheckStorageBudget(ctx); err != nil {
		return nil, err
	}
	if status.Kind == previewKindVite {
		return e.startVitePreview(ctx, task, status)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	fresh, err := e.Store.Get(id)
	if err != nil || e.closing || e.active[id] != nil || fresh.Updated != task.Updated {
		return nil, errors.New("preview_changed: candidate task changed before preview admission")
	}
	return manager.startStatic(e, task, status)
}

func selectPreviewStatus(status PreviewStatus, selected string) (PreviewStatus, error) {
	if selected == "" {
		return requireSupportedPreview(status)
	}
	status.Supported = false
	for _, name := range status.Candidates {
		if selected != name || (name == previewKindVite && status.Reason != previewViteBasic) {
			continue
		}
		status.Entry, status.Supported = name, true
		if name == previewKindVite {
			continue
		}
		status.Kind, status.Reason = "static", "preview_static_ready"
		status.BuildUnverified = previewBuildEntry(name)
		if status.BuildUnverified {
			status.Reason = "preview_build_unverified"
		}
	}
	return requireSupportedPreview(status)
}

func requireSupportedPreview(status PreviewStatus) (PreviewStatus, error) {
	if !status.Supported {
		return status, errors.New("preview_unsupported: no supported candidate preview entry is available")
	}
	return status, nil
}

func (manager *PreviewManager) existingPreview(id string, status PreviewStatus) (*PreviewStatus, error) {
	if current := manager.status(id); current != nil {
		switch current.Reason {
		case previewStopping, previewCleanupUnknown:
			return nil, errors.New(current.Reason + ": previous runtime cleanup has not completed")
		case "preview_process_exited":
			if err := manager.StopTask(id); err != nil {
				return nil, err
			}
		default:
			if current.SourceHash == status.SourceHash && current.Entry == status.Entry {
				return current, nil
			}
			return nil, errors.New("preview_changed: stop the previous preview before restarting changed source")
		}
	}
	return nil, nil
}

func previewBuildEntry(name string) bool {
	return strings.HasPrefix(name, "dist/") || strings.HasPrefix(name, "build/")
}

func (manager *PreviewManager) startStatic(engine *Engine, task *Task, status PreviewStatus) (*PreviewStatus, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.closed {
		return nil, errors.New("preview_unavailable: preview manager has closed")
	}
	if manager.entries == nil {
		manager.entries = make(map[string]*managedPreview)
	}
	if entry := manager.entries[task.ID]; entry != nil {
		if entry.status.SourceHash != status.SourceHash || entry.status.Entry != status.Entry {
			return nil, errors.New("preview_changed: stop the previous preview before restarting changed source")
		}
		copy := entry.status
		return &copy, nil
	}
	if len(manager.entries) >= 2 {
		return nil, errors.New("preview_limit: stop an existing preview before opening another")
	}
	return manager.listenStatic(engine, task, status)
}

func (manager *PreviewManager) listenStatic(engine *Engine, task *Task, status PreviewStatus) (*PreviewStatus, error) {
	entry, err := manager.newListener(engine, task, status)
	if err != nil {
		return nil, err
	}
	manager.entries[task.ID] = entry
	copy := entry.status
	return &copy, nil
}

func (manager *PreviewManager) newListener(engine *Engine, task *Task, status PreviewStatus) (*managedPreview, error) {
	root, err := os.OpenRoot(filepath.Join(task.Worktree, filepath.FromSlash(path.Dir(status.Entry))))
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = root.Close()
		return nil, err
	}
	status.URL, status.Running = "http://"+listener.Addr().String()+"/", status.Kind != previewKindVite
	if status.Kind == previewKindVite {
		status.Reason = "preview_starting"
	}
	entry := &managedPreview{status: status, root: root, done: make(chan struct{})}
	entry.server = &http.Server{Handler: staticPreviewHandler(engine, task, entry, listener.Addr().String()), ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() { _ = entry.server.Serve(listener); close(entry.done) }()
	return entry, nil
}

func previewRequestLocal(r *http.Request, host string) bool {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !net.ParseIP(peer).IsLoopback() || r.Host != host || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	origin := r.Header.Get("Origin")
	return origin == "" || origin == "http://"+host
}

func previewAssetType(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".html", ".htm", ".css", ".js", ".mjs", ".json", ".svg", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif", ".ico", ".woff", ".woff2", ".ttf", ".otf", ".wasm", ".mp3", ".wav", ".mp4", ".webm", ".pdf":
		return true
	}
	return false
}
