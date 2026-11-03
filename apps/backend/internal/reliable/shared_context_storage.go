package reliable

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type sharedContextWriter struct {
	root  *os.Root
	bytes int
}

func (e *Engine) sharedContextRoot(create bool) (*os.Root, error) {
	if !filepath.IsAbs(e.Root) {
		return nil, errors.New("shared_context_storage: archive root must be absolute")
	}
	controller, err := os.OpenRoot(e.Root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = controller.Close() }()
	if create {
		if err = controller.Mkdir(sharedContextDirectoryName, 0700); err != nil && !os.IsExist(err) {
			return nil, err
		}
	}
	if err = sharedContextDirectory(controller, sharedContextDirectoryName); err != nil {
		return nil, err
	}
	root, err := controller.OpenRoot(sharedContextDirectoryName)
	if err != nil {
		return nil, err
	}
	for _, name := range []string{sharedContextBlobsName, sharedContextIndicesName, sharedContextHeadsName} {
		if create {
			if err = root.Mkdir(name, 0700); err != nil && !os.IsExist(err) {
				_ = root.Close()
				return nil, err
			}
		}
		if err = sharedContextDirectory(root, name); err != nil {
			_ = root.Close()
			return nil, err
		}
	}
	return root, nil
}

func sharedContextDirectory(root *os.Root, name string) error {
	info, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return errors.New("shared_context_storage: archive directory is unsafe")
	}
	return nil
}

func (writer *sharedContextWriter) marshal(value any) ([]byte, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("shared_context_storage: malformed retained context: %w", err)
	}
	writer.bytes += len(raw)
	if len(raw) > sharedContextBlobLimit || writer.bytes > sharedContextByteLimit {
		return nil, errors.New("shared_context_limit: archive exceeds 4 MiB per record or 64 MiB total; original data remains retained")
	}
	return raw, nil
}

func (writer *sharedContextWriter) blob(value any) (string, error) {
	raw, err := writer.marshal(value)
	if err != nil {
		return "", err
	}
	return publishSharedContextBlob(writer.root, sharedContextBlobsName, raw)
}

func (writer *sharedContextWriter) index(value sharedContextIndex) (string, error) {
	raw, err := writer.marshal(value)
	if err != nil {
		return "", err
	}
	return publishSharedContextBlob(writer.root, sharedContextIndicesName, raw)
}

func (writer *sharedContextWriter) batch(task *Task) (sharedContextBatch, error) {
	entry := sharedContextBatch{TaskID: task.ID, Chunks: []sharedContextChunk{}}
	header := *task
	header.Messages, header.Baseline = nil, nil
	var err error
	entry.HeaderSHA256, err = writer.blob(map[string]any{"saved_batch": header, "trust": "Historical model reports and recorded verification states are not current source facts."})
	if err != nil {
		return entry, err
	}
	for start := 0; start < len(task.Messages); {
		end, err := sharedContextChunkEnd(task.Messages, start)
		if err != nil {
			return entry, err
		}
		hash, err := writer.blob(map[string]any{"task_id": task.ID, "start": start, "messages": task.Messages[start:end]})
		if err != nil {
			return entry, err
		}
		entry.Chunks = append(entry.Chunks, sharedContextChunk{SHA256: hash, Messages: end - start})
		start = end
	}
	return entry, nil
}

func sharedContextChunkEnd(messages []TaskMessage, start int) (int, error) {
	end, total := start, 0
	for end < len(messages) {
		raw, err := json.Marshal(messages[end])
		if err != nil {
			return 0, err
		}
		if end > start && total+len(raw) > sharedContextChunkTarget {
			break
		}
		total += len(raw)
		end++
	}
	return end, nil
}

func validSharedContextHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(hash)
	return err == nil && len(decoded) == 32 && hash == fmt.Sprintf("%x", decoded)
}

func readSharedContextFile(root *os.Root, name string, limit int64) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("shared_context_storage: archive file is unsafe or oversized")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("shared_context_storage: archive file identity changed")
	}
	raw, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return nil, errors.New("shared_context_storage: archive read failed")
	}
	return raw, nil
}

func publishSharedContextBlob(root *os.Root, kind string, raw []byte) (string, error) {
	hash := digest(raw)
	name := filepath.Join(kind, hash+".json")
	old, err := readSharedContextFile(root, name, sharedContextBlobLimit)
	if err == nil {
		if !bytes.Equal(old, raw) {
			return "", errors.New("shared_context_changed: existing archive bytes differ from their hash")
		}
		return hash, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	temporary, err := writeSharedContextTemporary(root, kind, raw)
	if err != nil {
		return "", err
	}
	defer func() { _ = root.Remove(temporary) }()
	if err = root.Link(temporary, name); err != nil && !os.IsExist(err) {
		return "", err
	}
	stored, err := readSharedContextFile(root, name, sharedContextBlobLimit)
	if err != nil || digest(stored) != hash {
		return "", errors.New("shared_context_changed: archive publication did not retain the expected bytes")
	}
	return hash, nil
}

func writeSharedContextTemporary(root *os.Root, kind string, raw []byte) (string, error) {
	name := filepath.Join(kind, ".pending-"+randomID()+".tmp")
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	_, writeErr := file.Write(raw)
	syncErr, closeErr := file.Sync(), file.Close()
	if err = errors.Join(writeErr, syncErr, closeErr); err != nil {
		_ = root.Remove(name)
		return "", err
	}
	return name, nil
}

func publishSharedContextHead(root *os.Root, ref SharedContextReference) error {
	raw, err := json.Marshal(ref)
	if err != nil {
		return err
	}
	temporary, err := writeSharedContextTemporary(root, sharedContextHeadsName, raw)
	if err != nil {
		return err
	}
	defer func() { _ = root.Remove(temporary) }()
	return root.Rename(temporary, filepath.Join(sharedContextHeadsName, ref.TaskID+".json"))
}
