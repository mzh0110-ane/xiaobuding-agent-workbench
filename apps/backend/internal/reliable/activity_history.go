package reliable

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const activityHistoryReadLimit = 1 << 20

type ActivityHistoryPage struct {
	ActivityPage
	BeforeCursor int64 `json:"before_cursor"`
	HasEarlier   bool  `json:"has_earlier"`
}

func (h *HTTP) activityHistoryRoute(w http.ResponseWriter, r *http.Request, id string) bool {
	query := r.URL.Query()
	if !query.Has("tail") && !query.Has("before") {
		return false
	}
	if query.Has("after") || query.Has("tail") && (query.Get("tail") != "1" || query.Has("before")) {
		h.reply(w, http.StatusBadRequest, map[string]string{"error": "invalid activity history cursor"})
		return true
	}
	before := int64(-1)
	var err error
	if query.Has("before") {
		before, err = activityCursor(query.Get("before"))
	}
	if err != nil {
		h.reply(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return true
	}
	page, err := h.Engine.readActivityHistory(id, before)
	if err != nil {
		h.reply(w, http.StatusConflict, map[string]string{"error": err.Error()})
	} else {
		h.reply(w, http.StatusOK, page)
	}
	return true
}

func (e *Engine) readActivityHistory(id string, before int64) (ActivityHistoryPage, error) {
	e.activityMu.Lock()
	page, err := e.readActivityHistoryFile(id, before)
	e.activityMu.Unlock()
	e.mu.Lock()
	_, page.Active = e.active[id]
	e.mu.Unlock()
	return page, err
}

func (e *Engine) openActivityHistory(id string) (*os.File, int64, error) {
	if !safeRelative(id) || strings.Contains(id, "/") {
		return nil, 0, errors.New("invalid activity task identity")
	}
	root, err := openStorageRoot(e.Root)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = root.Close() }()
	info, err := root.Lstat("activity")
	if err != nil {
		return nil, 0, err
	}
	if !info.IsDir() || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return nil, 0, errors.New("activity log is unavailable")
	}
	name := filepath.Join("activity", id+".jsonl")
	info, err = root.Lstat(name)
	if err != nil {
		return nil, 0, err
	}
	if !info.Mode().IsRegular() || info.Size() > activityLimit+64<<10 {
		return nil, 0, errors.New("activity log is unavailable")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, 0, err
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		_ = file.Close()
		return nil, 0, errors.New("activity log identity changed")
	}
	return file, info.Size(), nil
}

func (e *Engine) readActivityHistoryFile(id string, before int64) (ActivityHistoryPage, error) {
	page := ActivityHistoryPage{ActivityPage: ActivityPage{Events: []Activity{}}}
	file, size, err := e.openActivityHistory(id)
	if os.IsNotExist(err) {
		return page, nil
	}
	if err != nil {
		return page, err
	}
	defer func() { _ = file.Close() }()
	if before < 0 {
		before = size
	}
	if before > size {
		return page, errors.New("invalid activity cursor")
	}
	if before == 0 {
		return page, nil
	}
	if before < size {
		var boundary [1]byte
		if _, err = file.ReadAt(boundary[:], before-1); err != nil || boundary[0] != '\n' {
			return page, errors.New("invalid activity cursor boundary")
		}
	}
	start := max(int64(0), before-activityHistoryReadLimit)
	raw := make([]byte, before-start)
	n, err := file.ReadAt(raw, start)
	if err != nil && !errors.Is(err, io.EOF) {
		return page, err
	}
	raw = raw[:n]
	if start > 0 {
		index := bytes.IndexByte(raw, '\n')
		if index < 0 {
			return page, errors.New("activity history record exceeds read limit")
		}
		start += int64(index + 1)
		raw = raw[index+1:]
	}
	last := bytes.LastIndexByte(raw, '\n')
	if last < 0 {
		return page, nil
	}
	raw = raw[:last+1]
	page.Cursor, page.Limited = start+int64(len(raw)), size >= activityLimit
	page.Events, page.BeforeCursor, err = decodeActivityHistory(raw, start)
	page.HasEarlier = page.BeforeCursor > 0
	return page, err
}

func decodeActivityHistory(raw []byte, base int64) ([]Activity, int64, error) {
	events := make([]Activity, 0, 100)
	end := len(raw)
	for len(events) < 100 && end > 0 {
		start := bytes.LastIndexByte(raw[:end-1], '\n') + 1
		var event Activity
		if json.Unmarshal(raw[start:end], &event) != nil {
			return nil, 0, errors.New("activity log is malformed")
		}
		event.Sequence = base + int64(end)
		events = append(events, event)
		end = start
	}
	slices.Reverse(events)
	return events, base + int64(end), nil
}
