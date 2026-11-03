package reliable

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	bolt "go.etcd.io/bbolt"
)

type TaskSummary struct {
	ID              string    `json:"id"`
	Project         string    `json:"project"`
	ProjectID       string    `json:"project_id"`
	Goal            string    `json:"goal"`
	State           string    `json:"state"`
	Round           int       `json:"round"`
	Worktree        string    `json:"worktree,omitempty"`
	ReadOnly        bool      `json:"read_only,omitempty"`
	ArchivedAt      string    `json:"archived_at,omitempty"`
	ParentTaskID    string    `json:"parent_task_id,omitempty"`
	SuccessorTaskID string    `json:"successor_task_id,omitempty"`
	Created         string    `json:"created"`
	Updated         string    `json:"updated"`
	ActiveRun       *Dispatch `json:"active_run,omitempty"`
	SummaryOnly     bool      `json:"summary_only"`
	SearchMatch     bool      `json:"search_match,omitempty"`
}

func (s *Store) taskSummaries(search string) ([]TaskSummary, error) {
	search = strings.ToLower(strings.TrimSpace(search))
	items := []TaskSummary{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(tasksBucket).ForEach(func(_, raw []byte) error {
			var projected struct {
				TaskSummary
				Runs []Dispatch `json:"runs"`
			}
			if err := json.Unmarshal(raw, &projected); err != nil {
				return err
			}
			item := projected.TaskSummary
			if search != "" && !strings.Contains(strings.ToLower(item.Goal+" "+item.Project), search) {
				return nil
			}
			item.SearchMatch = search != ""
			item.Goal = projectExcerpt(strings.Join(strings.Fields(item.Goal), " "), 88)
			item.ProjectID = projectLedgerID(item.Project)
			item.SummaryOnly = true
			if len(projected.Runs) > 0 {
				last := projected.Runs[len(projected.Runs)-1]
				if last.Status == "running" || last.Status == "preparing" {
					item.ActiveRun = &last
				}
			}
			items = append(items, item)
			return nil
		})
	})
	sort.SliceStable(items, func(i, j int) bool { return items[i].Created > items[j].Created })
	return items, err
}

func (h *HTTP) conditionalReply(w http.ResponseWriter, r *http.Request, value any) {
	raw, err := json.Marshal(value)
	if err != nil {
		h.fail(w, err)
		return
	}
	if h.Engine.Redact != nil {
		raw = h.Engine.Redact(raw)
	}
	etag := `"` + digest(raw) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, no-cache")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(raw)
}

func (h *HTTP) summaryList(w http.ResponseWriter, r *http.Request) {
	items, err := h.Engine.Store.taskSummaries(r.URL.Query().Get("q"))
	if err != nil {
		h.fail(w, err)
		return
	}
	offset, limit, err := pageRange(r, len(items), 200, "offset")
	if err != nil {
		h.reply(w, http.StatusBadRequest, map[string]string{"error": "pagination_invalid"})
		return
	}
	end := min(offset+limit, len(items))
	next := -1
	if end < len(items) {
		next = end
	}
	h.conditionalReply(w, r, map[string]any{"tasks": items[offset:end], "total": len(items), "next_offset": next})
}

func (h *HTTP) workspaceDetail(w http.ResponseWriter, r *http.Request, task *Task) {
	h.Engine.mu.Lock()
	active := h.Engine.active[task.ID] != nil
	h.Engine.mu.Unlock()
	copyTask := *task
	if task.Baseline != nil {
		copyTask.Baseline = &Snapshot{Hash: task.Baseline.Hash, Head: task.Baseline.Head, ResourceHash: task.Baseline.ResourceHash}
	}
	copyTask.PreviousContext = nil
	before := max(0, len(task.Messages)-25)
	copyTask.Messages = task.Messages[before:]
	h.conditionalReply(w, r, struct {
		*Task
		MessagesBefore  int              `json:"messages_before"`
		MessagesTotal   int              `json:"messages_total"`
		Recovery        RecoveryGuidance `json:"recovery"`
		ExecutionActive bool             `json:"execution_active"`
	}{&copyTask, before, len(task.Messages), RecoveryGuidanceFor(task), active})
}

func pageRange(r *http.Request, total, defaultLimit int, key string) (int, int, error) {
	start, limit := 0, defaultLimit
	var err error
	if raw := r.URL.Query().Get(key); raw != "" {
		start, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, err
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
	}
	if err != nil || start < 0 || start > total || limit < 1 || limit > 200 {
		return 0, 0, strconv.ErrRange
	}
	return start, limit, nil
}

func (h *HTTP) messagePage(w http.ResponseWriter, r *http.Request, task *Task) {
	if r.URL.Query().Get("before") == "" {
		query := r.URL.Query()
		query.Set("before", strconv.Itoa(len(task.Messages)))
		r.URL.RawQuery = query.Encode()
	}
	before, limit, err := pageRange(r, len(task.Messages), 25, "before")
	if err != nil {
		h.reply(w, http.StatusBadRequest, map[string]string{"error": "pagination_invalid"})
		return
	}
	start := max(0, before-limit)
	h.conditionalReply(w, r, map[string]any{"messages": task.Messages[start:before], "before": start, "total": len(task.Messages), "updated": task.Updated})
}
