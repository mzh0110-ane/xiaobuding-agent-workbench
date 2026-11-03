package reliable

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const activityLimit = 8 << 20

const (
	activityAgent    = "agent"
	activityThinking = "thinking"
	activityTool     = "tool"
)

type Activity struct {
	Sequence int64  `json:"sequence"`
	Time     string `json:"time"`
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Detail   string `json:"detail,omitempty"`
	Role     string `json:"role,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
}

type ActivityPage struct {
	Events  []Activity `json:"events"`
	Cursor  int64      `json:"cursor"`
	Active  bool       `json:"active"`
	Limited bool       `json:"limited"`
}

type activityContextKey struct{}

type activityObserver func(Activity)

func observeActivity(ctx context.Context, observer activityObserver) context.Context {
	return context.WithValue(ctx, activityContextKey{}, observer)
}

func emitActivity(ctx context.Context, kind, title, detail string) {
	if observer, ok := ctx.Value(activityContextKey{}).(activityObserver); ok {
		observer(Activity{Time: time.Now().UTC().Format(time.RFC3339Nano), Kind: kind, Title: title, Detail: detail})
	}
}

func scopeActivity(ctx context.Context, d *Dispatch) context.Context {
	observer, _ := ctx.Value(activityContextKey{}).(activityObserver)
	return observeActivity(ctx, func(event Activity) {
		if observer == nil {
			return
		}
		event.Role, event.Domain, event.Provider, event.Model = d.Role, d.Domain, d.ProviderName, d.Model
		observer(event)
	})
}

func boundedActivityText(text string) string {
	text = strings.ToValidUTF8(text, "")
	if len(text) <= 4096 {
		return text
	}
	text = text[:4096]
	for !utf8.ValidString(text) {
		text = text[:len(text)-1]
	}
	return text + "\n…（内容已截取）"
}

func (e *Engine) appendActivity(id string, event Activity) {
	e.activityMu.Lock()
	defer e.activityMu.Unlock()
	dir := filepath.Join(e.Root, "activity")
	if os.MkdirAll(dir, 0700) != nil {
		return
	}
	path := filepath.Join(dir, id+".jsonl")
	if info, err := os.Lstat(path); err == nil && (!info.Mode().IsRegular() || info.Size() >= activityLimit) {
		return
	}
	if e.Redact != nil {
		event.Detail = string(e.Redact([]byte(event.Detail)))
		event.Title = string(e.Redact([]byte(event.Title)))
	}
	event.Detail, event.Title = boundedActivityText(event.Detail), boundedActivityText(event.Title)
	raw, err := json.Marshal(event)
	if err != nil {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()
	_, _ = file.Write(append(raw, '\n'))
}

func (e *Engine) readActivity(id string, after int64) (ActivityPage, error) {
	e.activityMu.Lock()
	page, err := e.readActivityFile(id, after)
	e.activityMu.Unlock()
	e.mu.Lock()
	_, page.Active = e.active[id]
	e.mu.Unlock()
	return page, err
}

func (e *Engine) readActivityFile(id string, after int64) (ActivityPage, error) {
	page := ActivityPage{Events: []Activity{}, Cursor: after}
	path := filepath.Join(e.Root, "activity", id+".jsonl")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		page.Cursor = 0
		return page, nil
	}
	if err != nil {
		return page, err
	}
	if !info.Mode().IsRegular() || info.Size() > activityLimit+64<<10 {
		return page, errors.New("activity log is unavailable")
	}
	if after < 0 || after > info.Size() {
		return page, errors.New("invalid activity cursor")
	}
	page.Limited = info.Size() >= activityLimit
	file, err := os.Open(path)
	if err != nil {
		return page, err
	}
	defer func() { _ = file.Close() }()
	if _, err = file.Seek(after, io.SeekStart); err != nil {
		return page, err
	}
	reader := bufio.NewReader(file)
	for len(page.Events) < 100 {
		line, readErr := reader.ReadBytes('\n')
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return page, readErr
		}
		var event Activity
		if json.Unmarshal(line, &event) != nil {
			return page, errors.New("activity log is malformed")
		}
		page.Cursor += int64(len(line))
		event.Sequence = page.Cursor
		page.Events = append(page.Events, event)
	}
	return page, nil
}

func activityCursor(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0, errors.New("invalid activity cursor")
	}
	return value, nil
}
