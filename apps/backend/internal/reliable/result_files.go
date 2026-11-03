package reliable

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

const resultTextLimit = 512 << 10

type ResultFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	Status string `json:"status"`
}

type ResultFiles struct {
	Files      []ResultFile `json:"files"`
	Total      int          `json:"total"`
	NextOffset int          `json:"next_offset"`
	SourceHash string       `json:"source_hash"`
}

func (e *Engine) resultCandidate(ctx context.Context, id string) (*Task, *Snapshot, error) {
	task, err := e.Store.Get(id)
	if err != nil {
		return nil, nil, err
	}
	if task.ReadOnly || task.Worktree == "" || task.Baseline == nil || task.Base == "" || task.Baseline.Head != task.Base {
		return nil, nil, errors.New("preview_unavailable: no owned candidate source is available")
	}
	if err = e.resultCandidateIdle(task); err != nil {
		return nil, nil, err
	}
	if err = e.recoveryOwnership(ctx, task); err != nil {
		return nil, nil, fmt.Errorf("preview_ownership: %w", err)
	}
	if err = e.processQuiescent(task); err != nil {
		return nil, nil, err
	}
	snapshot, err := TakeSnapshot(ctx, task.Worktree)
	return task, snapshot, err
}

func (e *Engine) resultCandidateIdle(task *Task) error {
	e.mu.Lock()
	busy := e.closing || e.active[task.ID] != nil
	e.mu.Unlock()
	switch task.State {
	case stateAccepted, stateAwaitingAcceptance, stateBlocked, stateInterrupted, stateCancelled, stateExhausted:
		if !busy {
			return nil
		}
	}
	return errors.New("preview_busy: wait for candidate execution to stop before viewing files")
}

func resultFilePage(task *Task, snapshot *Snapshot, offset, limit int) *ResultFiles {
	files := make([]ResultFile, 0)
	for _, name := range changedFiles(task.Baseline, snapshot) {
		if !resultPathAllowed(name) {
			continue
		}
		state := "modified"
		if snapshot.Files[name] == "" || snapshot.Files[name] == deletedFingerprint {
			state = deletedFingerprint
		} else if task.Baseline.Files[name] == "" || task.Baseline.Files[name] == deletedFingerprint {
			state = "new"
		}
		var size int64
		if state != deletedFingerprint {
			if info, err := os.Lstat(task.Worktree + string(os.PathSeparator) + name); err == nil && info.Mode().IsRegular() {
				size = info.Size()
			}
		}
		files = append(files, ResultFile{Path: name, Size: size, Status: state})
	}
	result := &ResultFiles{Files: []ResultFile{}, Total: len(files), NextOffset: -1, SourceHash: snapshot.Hash}
	if offset < len(files) {
		end := min(offset+limit, len(files))
		result.Files = files[offset:end]
		if end < len(files) {
			result.NextOffset = end
		}
	}
	return result
}

func (h *HTTP) resultFilesRoute(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	task, snapshot, err := h.Engine.resultCandidate(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	if name := r.URL.Query().Get("path"); name != "" {
		h.resultFileContent(w, r, task, snapshot, name)
		return
	}
	offset, limit, err := resultFilePagination(r)
	if err != nil {
		h.reply(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.reply(w, http.StatusOK, resultFilePage(task, snapshot, offset, limit))
}

func resultFilePagination(r *http.Request) (int, int, error) {
	parse := func(name string, fallback, ceiling int) (int, error) {
		if r.URL.Query().Get(name) == "" {
			return fallback, nil
		}
		value, err := strconv.Atoi(r.URL.Query().Get(name))
		if err != nil || value < 0 || value > ceiling {
			return 0, errors.New("result_files_pagination: invalid result page")
		}
		return value, nil
	}
	offset, err := parse("offset", 0, 30000)
	if err != nil {
		return 0, 0, err
	}
	limit, err := parse("limit", 100, 200)
	if err == nil && limit == 0 {
		err = errors.New("result_files_pagination: page size must be positive")
	}
	return offset, limit, err
}

func (h *HTTP) resultFileContent(w http.ResponseWriter, _ *http.Request, task *Task, snapshot *Snapshot, name string) {
	if !resultPathAllowed(name) || snapshot.Files[name] == "" || snapshot.Files[name] == deletedFingerprint || snapshot.Files[name] == task.Baseline.Files[name] {
		h.reply(w, http.StatusNotFound, map[string]string{"error": "result_file_unavailable: select an available changed source file"})
		return
	}
	root, err := os.OpenRoot(task.Worktree)
	if err != nil {
		h.fail(w, err)
		return
	}
	defer func() { _ = root.Close() }()
	file, err := openResultFile(root, name, resultTextLimit)
	if err != nil {
		h.fail(w, err)
		return
	}
	defer func() { _ = file.Close() }()
	raw, err := io.ReadAll(io.LimitReader(file, resultTextLimit+1))
	if err != nil || len(raw) > resultTextLimit || !utf8.Valid(raw) || bytes.IndexByte(raw, 0) >= 0 {
		h.reply(w, http.StatusUnsupportedMediaType, map[string]string{"error": "result_file_format: only UTF-8 text up to 512 KiB can be read here"})
		return
	}
	if digest(raw) != snapshot.Files[name] {
		h.fail(w, errors.New("result_file_changed: candidate changed while reading"))
		return
	}
	h.reply(w, http.StatusOK, map[string]any{"path": name, "content": string(raw), "size": len(raw), "sha256": digest(raw), "source_hash": snapshot.Hash, "truncated": false})
}

func resultPathAllowed(name string) bool {
	if !safeRelative(name) || protectedPath(name) {
		return false
	}
	for _, component := range strings.Split(strings.ToLower(name), "/") {
		if !resultComponentAllowed(component) {
			return false
		}
	}
	return true
}
