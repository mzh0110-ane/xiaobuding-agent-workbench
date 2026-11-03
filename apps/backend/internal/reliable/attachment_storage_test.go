package reliable

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func storedAttachment(t *testing.T, e *Engine, name string, data []byte) Attachment {
	t.Helper()
	meta, err := e.storeAttachment(context.Background(), name, bytes.NewReader(data), nil)
	if err != nil {
		t.Fatal(err)
	}
	return *meta
}

func attachmentUploadRequest(t *testing.T, h *HTTP, name string, data []byte, edit func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:38429/api/v1/reliable/attachments", &body)
	r.RemoteAddr = "127.0.0.1:4321"
	r.Header.Set("X-XBD-Token", h.Token)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	if edit != nil {
		edit(r)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// @covers AC-RELIABLE-ATTACHMENTS-001.1, AC-RELIABLE-ATTACHMENTS-001.3
func TestAttachmentUploadSafeDownloadAndPreview(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "file-action"}
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name  string
		data  []byte
		image bool
	}{{"需求.任意格式", []byte("document\x00bytes"), false}, {"pretends.png", []byte("<html><script>alert(1)</script></html>"), false}, {"drawing.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), false}, {"real-image.custom", imageBytes.Bytes(), true}} {
		t.Run(test.name, func(t *testing.T) {
			upload := attachmentUploadRequest(t, h, test.name, test.data, nil)
			if upload.Code != http.StatusCreated {
				t.Fatalf("upload failed: %d %s", upload.Code, upload.Body)
			}
			var meta Attachment
			if err := json.Unmarshal(upload.Body.Bytes(), &meta); err != nil {
				t.Fatal(err)
			}
			if meta.Image != test.image || meta.Size != int64(len(test.data)) || meta.Name != test.name || !validAttachmentID(meta.ID) {
				t.Fatalf("incorrect canonical upload metadata: %+v", meta)
			}
			download := httpRequest(h, http.MethodGet, "/api/v1/reliable/attachments/"+meta.ID, "", "")
			if download.Code != http.StatusOK || !bytes.Equal(download.Body.Bytes(), test.data) || download.Header().Get("Content-Type") != "application/octet-stream" || download.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.HasPrefix(download.Header().Get("Content-Disposition"), "attachment;") {
				t.Fatalf("unsafe or changed download: %d %v", download.Code, download.Header())
			}
			preview := httpRequest(h, http.MethodGet, "/api/v1/reliable/attachments/"+meta.ID+"/preview", "", "")
			if test.image {
				if preview.Code != http.StatusOK || preview.Header().Get("Content-Type") != "image/png" || !bytes.Equal(preview.Body.Bytes(), test.data) {
					t.Fatalf("real raster preview failed: %d %v", preview.Code, preview.Header())
				}
			} else if preview.Code != http.StatusUnsupportedMediaType {
				t.Fatalf("active or unknown content received an inline preview: %d", preview.Code)
			}
		})
	}
}

// @covers AC-RELIABLE-ATTACHMENTS-001.3
func TestAttachmentUploadRequiresExactProtectedRoute(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	h := &HTTP{Engine: e, Token: "file-action"}
	for _, test := range []struct {
		name   string
		edit   func(*http.Request)
		status int
	}{{"missing-token", func(r *http.Request) { r.Header.Del("X-XBD-Token") }, http.StatusForbidden}, {"cross-origin", func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, http.StatusForbidden}, {"cross-site", func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, http.StatusForbidden}, {"wrong-route", func(r *http.Request) { r.URL.Path += "/extra" }, http.StatusBadRequest}} {
		t.Run(test.name, func(t *testing.T) {
			result := attachmentUploadRequest(t, h, "file.bin", []byte("safe"), test.edit)
			if result.Code != test.status {
				t.Fatalf("upload authorization changed: got %d want %d", result.Code, test.status)
			}
		})
	}
	if entries, err := os.ReadDir(filepath.Join(e.Root, "attachments")); err == nil && len(entries) > 0 {
		t.Fatal("rejected upload published a file")
	}
}

// @covers AC-RELIABLE-ATTACHMENTS-001.3, AC-RELIABLE-ATTACHMENTS-001.6
func TestAttachmentStorageSurvivesRestartAndRejectsTamper(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	ref := storedAttachment(t, e, "original.custom", []byte("saved bytes"))
	restarted := &Engine{Root: e.Root}
	inputs, err := restarted.attachmentInputs([]Attachment{ref, ref})
	if err != nil || len(inputs) != 1 || !filepath.IsAbs(inputs[0].Path) || !pathInside(filepath.Join(e.Root, "attachments"), inputs[0].Path) {
		t.Fatalf("canonical bytes did not survive restart: %+v %v", inputs, err)
	}
	changed := ref
	changed.Name = "forged.txt"
	if err = restarted.validateAttachments([]Attachment{changed}); err == nil {
		t.Fatal("client metadata replaced the canonical receipt")
	}
	if _, err = restarted.resolveAttachments([]string{ref.ID, ref.ID}); err == nil {
		t.Fatal("one submission accepted duplicate file identifiers")
	}
	for _, id := range []string{"../secret", "C:/secret", "att-not-an-id", strings.Repeat("a", 30)} {
		if _, err = restarted.resolveAttachments([]string{id}); err == nil {
			t.Fatalf("unsafe identifier resolved: %s", id)
		}
	}
	if err = os.WriteFile(inputs[0].Path, []byte("other bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = restarted.validateAttachments([]Attachment{ref}); err == nil {
		t.Fatal("same-length changed bytes were trusted")
	}
	if err = os.Remove(inputs[0].Path); err != nil {
		t.Fatal(err)
	}
	if _, err = restarted.attachmentInputs([]Attachment{ref}); err == nil {
		t.Fatal("missing content was handed to a native agent")
	}
}

type attachmentZeroReader struct{}

func (attachmentZeroReader) Read(buffer []byte) (int, error) {
	clear(buffer)
	return len(buffer), nil
}

// @covers AC-RELIABLE-ATTACHMENTS-001.3
func TestAttachmentStorageByteLimitsAndHistoryLimits(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	large, err := e.storeAttachment(context.Background(), "large.bin", io.LimitReader(attachmentZeroReader{}, attachmentByteLimit), nil)
	if err != nil || large.Size != attachmentByteLimit {
		t.Fatalf("exact 100 MiB file was rejected: %+v %v", large, err)
	}
	if _, err = e.storeAttachment(context.Background(), "too-large.bin", io.LimitReader(attachmentZeroReader{}, attachmentByteLimit+1), nil); err == nil {
		t.Fatal("oversized file was published")
	}
	refs := []Attachment{*large}
	ids := []string{large.ID}
	for range 10 {
		ref := storedAttachment(t, e, "small.bin", []byte("x"))
		refs, ids = append(refs, ref), append(ids, ref.ID)
	}
	if _, err = e.resolveAttachments(ids[:2]); err == nil {
		t.Fatal("a submission exceeded 100 MiB aggregate bytes")
	}
	if _, err = e.resolveAttachments(ids); err == nil {
		t.Fatal("a submission exceeded ten files")
	}
	if err = e.validateAttachments(refs); err != nil {
		t.Fatalf("turn limits incorrectly truncated cumulative history: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(e.Root, "attachments"))
	if err != nil || len(entries) != 11 {
		t.Fatalf("oversized upload left a published/partial file: entries=%d error=%v", len(entries), err)
	}
}
