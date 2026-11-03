package reliable

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type activityHistoryResponse struct {
	Events       []Activity `json:"events"`
	Cursor       int64      `json:"cursor"`
	BeforeCursor int64      `json:"before_cursor"`
	HasEarlier   bool       `json:"has_earlier"`
}

func activityHistoryFixture(t *testing.T) (*Engine, *HTTP, string) {
	t.Helper()
	root := t.TempDir()
	store, err := OpenStore(filepath.Join(root, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{Root: root, Store: store}
	t.Cleanup(func() { _ = e.Close() })
	if err = store.Create(&Task{ID: "history", Project: "history-project", State: stateBlocked}); err != nil {
		t.Fatal(err)
	}
	return e, &HTTP{Engine: e, Token: "history"}, "/api/v1/reliable/tasks/history/activity"
}

func fetchActivityHistoryFixture(t *testing.T, h *HTTP, endpoint string) activityHistoryResponse {
	t.Helper()
	response := httpRequest(h, http.MethodGet, endpoint, "", "")
	var page activityHistoryResponse
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &page) != nil {
		t.Fatalf("activity history unavailable: %d %s", response.Code, response.Body)
	}
	return page
}

func TestActivityHistoryTailAndEarlierPagesUseAbsoluteNonoverlappingCursors(t *testing.T) {
	e, h, endpoint := activityHistoryFixture(t)
	for i := 0; i < 350; i++ {
		e.appendActivity("history", Activity{Kind: "message", Title: fmt.Sprintf("event-%03d", i), Detail: strings.Repeat("x", 4096)})
	}
	latest := fetchActivityHistoryFixture(t, h, endpoint+"?tail=1")
	if len(latest.Events) != 100 || latest.Events[0].Title != "event-250" || latest.Events[99].Title != "event-349" || !latest.HasEarlier || latest.BeforeCursor <= 0 {
		t.Fatalf("tail must contain only latest100: count=%d first=%v before=%d", len(latest.Events), latest.Events[0], latest.BeforeCursor)
	}
	info, err := os.Stat(filepath.Join(e.Root, "activity", "history.jsonl"))
	if err != nil || latest.Cursor != info.Size() || latest.Events[99].Sequence != info.Size() {
		t.Fatalf("tail cursor lost absolute byte identity: %+v %v", latest, err)
	}
	earlier := fetchActivityHistoryFixture(t, h, endpoint+"?before="+strconv.FormatInt(latest.BeforeCursor, 10))
	if len(earlier.Events) != 100 || earlier.Events[0].Title != "event-150" || earlier.Events[99].Title != "event-249" || earlier.Events[99].Sequence != latest.BeforeCursor || earlier.BeforeCursor >= latest.BeforeCursor {
		t.Fatalf("history page overlapped or skipped events: count=%d first=%v before=%d", len(earlier.Events), earlier.Events[0], earlier.BeforeCursor)
	}
	e.appendActivity("history", Activity{Kind: "message", Title: "new-after-tail"})
	forward := fetchActivityHistoryFixture(t, h, endpoint+"?after="+strconv.FormatInt(latest.Cursor, 10))
	if len(forward.Events) != 1 || forward.Events[0].Title != "new-after-tail" || forward.Events[0].Sequence <= latest.Cursor {
		t.Fatalf("legacy forward cursor changed: %+v", forward)
	}
}

func TestActivityHistoryBoundedTailIgnoresOldMalformedPrefix(t *testing.T) {
	e, h, endpoint := activityHistoryFixture(t)
	if err := os.MkdirAll(filepath.Join(e.Root, "activity"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.Root, "activity", "history.jsonl"), []byte(strings.Repeat("old malformed prefix ", 60000)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 250; i++ {
		e.appendActivity("history", Activity{Title: fmt.Sprintf("recent-%03d", i), Detail: strings.Repeat("x", 4096)})
	}
	page := fetchActivityHistoryFixture(t, h, endpoint+"?tail=1")
	if len(page.Events) != 100 || page.Events[0].Title != "recent-150" || page.BeforeCursor < 1<<20 {
		t.Fatalf("tail read old data instead of bounded recent page: %+v", page)
	}
	for _, suffix := range []string{"?before=-1", "?before=bad", "?before=1", "?tail=1&before=0", "?tail=2"} {
		response := httpRequest(h, http.MethodGet, endpoint+suffix, "", "")
		if response.Code == http.StatusOK {
			t.Fatalf("invalid activity history cursor accepted: %s", suffix)
		}
	}
}

func TestActivityHistoryHandlesEmptyMissingAndZeroBoundary(t *testing.T) {
	_, h, endpoint := activityHistoryFixture(t)
	for _, suffix := range []string{"?tail=1", "?before=0"} {
		page := fetchActivityHistoryFixture(t, h, endpoint+suffix)
		if len(page.Events) != 0 || page.Cursor != 0 || page.BeforeCursor != 0 || page.HasEarlier {
			t.Fatalf("missing history was invented: %+v", page)
		}
	}
}
