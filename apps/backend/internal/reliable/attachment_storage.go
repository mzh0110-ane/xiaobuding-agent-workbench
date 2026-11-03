package reliable

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (e *Engine) openAttachment(id string) (*Attachment, *os.File, error) {
	if !validAttachmentID(id) {
		return nil, nil, errors.New("attachment_invalid: invalid file identifier")
	}
	root, err := e.attachmentRoot()
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = root.Close() }()
	info, err := root.Lstat(id)
	if err != nil || !info.IsDir() || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return nil, nil, errors.New("attachment_unavailable: file directory is missing or unsafe")
	}
	dir, err := root.OpenRoot(id)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = dir.Close() }()
	meta, err := readAttachmentMetadata(dir, id)
	if err != nil {
		return nil, nil, err
	}
	file, err := openAttachmentRegular(dir, "content", meta.Size)
	if err != nil {
		return nil, nil, err
	}
	if err = verifyAttachmentContent(file, meta); err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	return meta, file, nil
}

func openAttachmentRegular(root *os.Root, name string, size int64) (*os.File, error) {
	info, err := root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() != size {
		return nil, errors.New("attachment_unavailable: original file is missing or unsafe")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		_ = file.Close()
		return nil, errors.New("attachment_changed: file identity changed while opening")
	}
	return file, nil
}

func readAttachmentMetadata(root *os.Root, id string) (*Attachment, error) {
	info, err := root.Lstat("metadata.json")
	if err != nil || !info.Mode().IsRegular() || info.Size() > 16<<10 {
		return nil, errors.New("attachment_unavailable: upload was not completed")
	}
	file, err := openAttachmentRegular(root, "metadata.json", info.Size())
	if err != nil {
		return nil, err
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, (16<<10)+1))
	closeErr := file.Close()
	var meta Attachment
	if readErr != nil || closeErr != nil || decodeStrict(raw, &meta) != nil || meta.ID != id || meta.Size < 0 || meta.Size > attachmentByteLimit || len(meta.SHA256) != 64 || meta.Created == "" {
		return nil, errors.New("attachment_changed: file receipt is missing or invalid")
	}
	name, err := attachmentName(meta.Name)
	if err != nil || name != meta.Name {
		return nil, errors.New("attachment_changed: file receipt name changed")
	}
	return &meta, nil
}

func verifyAttachmentContent(file *os.File, meta *Attachment) error {
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(file, attachmentByteLimit+1))
	if err != nil || size != meta.Size || hex.EncodeToString(hash.Sum(nil)) != meta.SHA256 {
		return errors.New("attachment_changed: saved bytes do not match the upload receipt")
	}
	mediaType, image, err := classifyAttachment(file, size)
	if err != nil || mediaType != meta.MediaType || image != meta.Image {
		return errors.New("attachment_changed: saved file type differs from its receipt")
	}
	_, err = file.Seek(0, io.SeekStart)
	return err
}

func (e *Engine) resolveAttachments(ids []string) ([]Attachment, error) {
	if len(ids) > attachmentTurnLimit {
		return nil, errors.New("attachment_limit: a submission may contain at most ten files")
	}
	seen := map[string]bool{}
	refs := make([]Attachment, 0, len(ids))
	var total int64
	for _, id := range ids {
		if seen[id] {
			return nil, errors.New("attachment_invalid: a submission contains duplicate file identifiers")
		}
		seen[id] = true
		meta, file, err := e.openAttachment(id)
		if err != nil {
			return nil, err
		}
		_ = file.Close()
		total += meta.Size
		if total > attachmentByteLimit {
			return nil, errors.New("attachment_too_large: submitted files exceed 100 MiB in total")
		}
		refs = append(refs, *meta)
	}
	return refs, nil
}

func (e *Engine) validateAttachments(refs []Attachment) error {
	_, err := e.attachmentInputs(refs)
	return err
}

func (e *Engine) attachmentInputs(refs []Attachment) ([]AttachmentInput, error) {
	seen := map[string]Attachment{}
	inputs := make([]AttachmentInput, 0, len(refs))
	for _, ref := range refs {
		if old, exists := seen[ref.ID]; exists {
			if old != ref {
				return nil, errors.New("attachment_changed: conflicting references to the same file")
			}
			continue
		}
		meta, file, err := e.openAttachment(ref.ID)
		if err != nil {
			return nil, err
		}
		_ = file.Close()
		if *meta != ref {
			return nil, fmt.Errorf("attachment_changed: saved receipt differs from the authorized file %s", ref.ID)
		}
		path, err := filepath.Abs(filepath.Join(e.Root, "attachments", ref.ID, "content"))
		if err != nil {
			return nil, err
		}
		seen[ref.ID] = ref
		inputs = append(inputs, AttachmentInput{Attachment: ref, Path: path})
	}
	return inputs, nil
}
