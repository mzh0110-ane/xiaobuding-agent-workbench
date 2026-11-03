package reliable

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestStorageBudgetHTTPPersistsLimitsAndRejectsStaleEdits(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "token"}
	response := httpRequest(h, http.MethodGet, "/api/v1/reliable/storage/budget", "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("storage budget unavailable: %d %s", response.Code, response.Body)
	}
	var initial map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	if initial["revision"] != float64(0) || initial["cache_limit_bytes"] != float64(8<<30) {
		t.Fatalf("invalid safe defaults: %+v", initial)
	}
	body := `{"expected_revision":0,"cache_limit_bytes":1073741824,"temp_limit_bytes":536870912,"minimum_free_bytes":268435456}`
	response = httpRequest(h, http.MethodPost, "/api/v1/reliable/storage/budget", body, h.Token)
	if response.Code != http.StatusOK {
		t.Fatalf("limit save failed: %d %s", response.Code, response.Body)
	}
	response = httpRequest(h, http.MethodGet, "/api/v1/reliable/storage/budget", "", "")
	var saved map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved["revision"] != float64(1) || saved["cache_limit_bytes"] != float64(1<<30) {
		t.Fatalf("limit not persisted: %+v", saved)
	}
	response = httpRequest(h, http.MethodPost, "/api/v1/reliable/storage/budget", body, h.Token)
	if response.Code != http.StatusConflict {
		t.Fatalf("stale limits replaced latest: %d %s", response.Code, response.Body)
	}
}

func TestStorageBudgetRejectsInvalidLimitsAndCancellationBeforeSaving(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "token"}
	for _, body := range []string{
		`{"expected_revision":0,"cache_limit_bytes":0,"temp_limit_bytes":536870912,"minimum_free_bytes":268435456}`,
		`{"expected_revision":0,"cache_limit_bytes":1073741824,"temp_limit_bytes":-1,"minimum_free_bytes":268435456}`,
		`{"expected_revision":0,"cache_limit_bytes":1073741824,"temp_limit_bytes":536870912,"minimum_free_bytes":1099511627777}`,
	} {
		response := httpRequest(h, http.MethodPost, "/api/v1/reliable/storage/budget", body, h.Token)
		if response.Code == http.StatusOK {
			t.Fatalf("unsafe limit admitted: %s", body)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := e.EditStorageBudget(ctx, StorageBudgetEdit{CacheLimitBytes: 1 << 30, TempLimitBytes: 1 << 30, MinimumFreeBytes: 1 << 30})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled setting changed state: %v", err)
	}
	policy, err := e.storageBudgetPolicy()
	if err != nil || policy.Revision != 0 {
		t.Fatalf("invalid save persisted: %+v %v", policy, err)
	}
}

func TestStorageAdmissionChecksOnlyDiskPressureAndLeavesDataIntact(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	_, err := e.EditStorageBudget(context.Background(), StorageBudgetEdit{CacheLimitBytes: 64 << 20, TempLimitBytes: 64 << 20, MinimumFreeBytes: 1 << 40})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.CheckStorageBudget(context.Background()); err == nil {
		t.Fatal("low disk admission was allowed")
	}
	if _, err := e.Store.Get("task"); err != nil {
		t.Fatal("storage pressure removed task")
	}
}

func TestStorageBudgetScanReportsPressureWithoutDeletingProtectedData(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	for _, category := range []string{"cache", "temp", "evidence", "attachments", "worktrees"} {
		if err := os.MkdirAll(filepath.Join(e.Root, category), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(e.Root, category, "preserve.txt"), []byte("preserve"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	h := &HTTP{Engine: e, Token: "token"}
	response := httpRequest(h, http.MethodGet, "/api/v1/reliable/storage/budget", "", "")
	if response.Code != http.StatusOK {
		t.Fatalf("budget scan unavailable: %d", response.Code)
	}
	var got struct {
		CacheBytes int64 `json:"cache_bytes"`
		TempBytes  int64 `json:"temp_bytes"`
		FreeKnown  bool  `json:"free_known"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.CacheBytes != 8 || got.TempBytes != 8 {
		t.Fatalf("wrong cache/temp accounting: %+v", got)
	}
	for _, category := range []string{"cache", "temp", "evidence", "attachments", "worktrees"} {
		if _, err := os.Stat(filepath.Join(e.Root, category, "preserve.txt")); err != nil {
			t.Fatalf("read-only budget deleted %s: %v", category, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.StorageUsage(ctx); err == nil {
		t.Fatal("cancelled scan continued")
	}
}
