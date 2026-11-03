package reliable

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const attachmentByteLimit int64 = 100 << 20
const attachmentTurnLimit = 10

func validAttachmentID(id string) bool {
	if len(id) != 30 || !strings.HasPrefix(id, "att-") {
		return false
	}
	for _, char := range id[4:] {
		if char < 'a' || char > 'z' {
			if char < '0' || char > '9' {
				return false
			}
		}
	}
	return true
}

func attachmentName(name string) (string, error) {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name))
	if name == "" || name == "." || name == "/" || len(name) > 1024 || !utf8.ValidString(name) {
		return "", errors.New("attachment_invalid: provide a valid filename")
	}
	return name, nil
}

func (e *Engine) attachmentRoot() (*os.Root, error) {
	controller, err := os.OpenRoot(e.Root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = controller.Close() }()
	if err = controller.Mkdir("attachments", 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	info, err := controller.Lstat("attachments")
	if err != nil || !info.IsDir() || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return nil, errors.New("attachment_invalid: storage directory is not an ordinary directory")
	}
	return controller.OpenRoot("attachments")
}

func (e *Engine) storeAttachment(ctx context.Context, name string, reader io.Reader, finish func() error) (*Attachment, error) {
	name, err := attachmentName(name)
	if err != nil {
		return nil, err
	}
	root, err := e.attachmentRoot()
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	id := "att-" + randomID()
	if err = root.Mkdir(id, 0700); err != nil {
		return nil, err
	}
	ready := false
	defer func() {
		if !ready {
			_ = root.Remove(filepath.Join(id, "content"))
			_ = root.Remove(filepath.Join(id, "metadata.json"))
			_ = root.Remove(id)
		}
	}()
	file, err := root.OpenFile(filepath.Join(id, "content"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	meta, err := receiveAttachment(ctx, id, name, file, reader)
	if err != nil {
		return nil, err
	}
	if finish != nil {
		if err = finish(); err != nil {
			return nil, err
		}
	}
	if err = file.Sync(); err != nil {
		return nil, err
	}
	if err = publishAttachmentMetadata(root, meta); err != nil {
		return nil, err
	}
	ready = true
	return meta, nil
}

func receiveAttachment(ctx context.Context, id, name string, file *os.File, reader io.Reader) (*Attachment, error) {
	hash := sha256.New()
	length, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(attachmentContextReader{ctx, reader}, attachmentByteLimit+1))
	if err != nil {
		return nil, fmt.Errorf("attachment_upload_failed: %w", err)
	}
	if length > attachmentByteLimit {
		return nil, errors.New("attachment_too_large: each file and submission must not exceed 100 MiB")
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	mediaType, image, err := classifyAttachment(file, length)
	if err != nil {
		return nil, err
	}
	return &Attachment{ID: id, Name: name, MediaType: mediaType, SHA256: hex.EncodeToString(hash.Sum(nil)), Created: time.Now().UTC().Format(time.RFC3339Nano), Size: length, Image: image}, nil
}

type attachmentContextReader struct {
	ctx context.Context
	in  io.Reader
}

func (r attachmentContextReader) Read(data []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.in.Read(data)
}

func classifyAttachment(file *os.File, size int64) (string, bool, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", false, err
	}
	header := make([]byte, min(size, 512))
	if _, err := io.ReadFull(file, header); err != nil {
		return "", false, err
	}
	mediaType := http.DetectContentType(header)
	return mediaType, nativeImageMediaType(mediaType), nil
}

func publishAttachmentMetadata(root *os.Root, meta *Attachment) error {
	raw, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	file, err := root.OpenFile(filepath.Join(meta.ID, "metadata.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(raw)
	return errors.Join(writeErr, file.Sync(), file.Close())
}
