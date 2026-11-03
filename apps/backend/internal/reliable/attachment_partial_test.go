package reliable

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// @covers AC-RELIABLE-ATTACHMENTS-001.3
func TestAttachmentUploadPartialOrMultipleFilesNeverPublishes(t *testing.T) {
	for _, multiple := range []bool{false, true} {
		t.Run(map[bool]string{false: "truncated", true: "multiple-files"}[multiple], func(t *testing.T) {
			e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
			h := &HTTP{Engine: e, Token: "upload-action"}
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			for range map[bool]int{false: 1, true: 2}[multiple] {
				part, err := writer.CreateFormFile("file", "a.bin")
				if err != nil {
					t.Fatal(err)
				}
				if _, err = part.Write([]byte("partial bytes")); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			data := body.Bytes()
			if !multiple {
				data = data[:len(data)-20]
			}
			r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:38429/api/v1/reliable/attachments", bytes.NewReader(data))
			r.RemoteAddr = "127.0.0.1:4321"
			r.Header.Set("X-XBD-Token", h.Token)
			r.Header.Set("Content-Type", writer.FormDataContentType())
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("incomplete multipart was published: %d %s", w.Code, w.Body)
			}
			if entries, err := os.ReadDir(filepath.Join(e.Root, "attachments")); err != nil || len(entries) != 0 {
				t.Fatalf("failed multipart left ready/partial files: entries=%d error=%v", len(entries), err)
			}
		})
	}
}

// @covers AC-RELIABLE-ATTACHMENTS-001.3, AC-RELIABLE-ATTACHMENTS-001.6
func TestAttachmentUploadCancellationAndMissingMetadataFailClosed(t *testing.T) {
	e, _ := fixtureEngine(t, &fixtureAgent{}, passingCheck)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.storeAttachment(ctx, "cancelled.bin", bytes.NewReader([]byte("bytes")), nil); err == nil {
		t.Fatal("cancelled stream became a ready upload")
	}
	meta := storedAttachment(t, e, "original.bin", []byte("original"))
	name := filepath.Join(e.Root, "attachments", meta.ID, "metadata.json")
	if err := os.WriteFile(name, []byte(`{"id":"forged"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := e.validateAttachments([]Attachment{meta}); err == nil {
		t.Fatal("changed metadata remained authorized")
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	if _, err := e.resolveAttachments([]string{meta.ID}); err == nil {
		t.Fatal("content without its completed receipt was accepted")
	}
}
