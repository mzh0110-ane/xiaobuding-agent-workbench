package reliable

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func (e *Engine) sharedContextReferenceMatches(ref SharedContextReference, expected string) bool {
	return ref.Version == sharedContextVersion && ref.Complete && validSharedContextID(ref.TaskID) && validSharedContextHash(expected) && ref.IndexSHA256 == expected && ref.IndexPath == filepath.Join(e.Root, sharedContextDirectoryName, sharedContextIndicesName, expected+".json")
}

func (e *Engine) sharedContextIndexMatches(index sharedContextIndex, ref SharedContextReference) bool {
	return index.Version == sharedContextVersion && index.Complete && index.TaskID == ref.TaskID && index.MessageID == ref.MessageID && len(index.Batches) == ref.BatchCount && index.MessageCount == ref.MessageCount && index.ArchiveRoot == filepath.Join(e.Root, sharedContextDirectoryName)
}

func (e *Engine) validateSharedContextReference(ref SharedContextReference, expected string, hashes map[string]string) error {
	if !e.sharedContextReferenceMatches(ref, expected) {
		return errors.New("shared_context_changed: archive reference differs from the controller's dispatch record")
	}
	root, err := e.sharedContextRoot(false)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	raw, err := readSharedContextFile(root, filepath.Join(sharedContextIndicesName, expected+".json"), sharedContextBlobLimit)
	if err != nil || digest(raw) != expected {
		return errors.New("shared_context_changed: complete context index is missing or changed")
	}
	var index sharedContextIndex
	if err = decodeStrict(raw, &index); err != nil || !e.sharedContextIndexMatches(index, ref) {
		return errors.New("shared_context_changed: complete context index does not match its controller reference")
	}
	if hashes != nil {
		hashes["shared-context/index/"+expected] = expected
	}
	if err = validateSharedContextBlobs(root, index, hashes); err != nil {
		return err
	}
	return e.validateSharedContextAttachments(index.Attachments, hashes)
}

func sharedContextBlobReferences(index sharedContextIndex) ([]string, error) {
	if len(index.Batches) == 0 || len(index.Batches) > sharedContextBatchLimit || index.MessageCount > sharedContextMessageLimit {
		return nil, errors.New("shared_context_changed: context index limits are invalid")
	}
	refs, count := []string{index.LedgerSHA256}, 0
	for _, batch := range index.Batches {
		refs = append(refs, batch.HeaderSHA256)
		for _, chunk := range batch.Chunks {
			if chunk.Messages <= 0 {
				return nil, errors.New("shared_context_changed: message chunk count is invalid")
			}
			refs, count = append(refs, chunk.SHA256), count+chunk.Messages
		}
	}
	if count != index.MessageCount || len(refs) > sharedContextMessageLimit+sharedContextBatchLimit+1 {
		return nil, errors.New("shared_context_changed: full message count differs from its index")
	}
	return refs, nil
}

func validateSharedContextBlobs(root *os.Root, index sharedContextIndex, hashes map[string]string) error {
	refs, err := sharedContextBlobReferences(index)
	if err != nil {
		return err
	}
	seen, total := map[string]bool{}, 0
	for _, hash := range refs {
		if seen[hash] {
			continue
		}
		if !validSharedContextHash(hash) {
			return errors.New("shared_context_changed: invalid content hash")
		}
		raw, err := readSharedContextFile(root, filepath.Join(sharedContextBlobsName, hash+".json"), sharedContextBlobLimit)
		if err != nil || digest(raw) != hash {
			return errors.New("shared_context_changed: a retained context record is missing or changed")
		}
		total += len(raw)
		if total > sharedContextByteLimit {
			return errors.New("shared_context_limit: context verification exceeds 64 MiB")
		}
		if hashes != nil {
			hashes["shared-context/blob/"+hash] = hash
		}
		seen[hash] = true
	}
	return nil
}

func (e *Engine) validateSharedContextAttachments(inputs []AttachmentInput, hashes map[string]string) error {
	refs, total := make([]Attachment, 0, len(inputs)), int64(0)
	for _, input := range inputs {
		if input.Path != filepath.Join(e.Root, "attachments", input.ID, "content") {
			return errors.New("shared_context_changed: historical attachment path is not controller-owned")
		}
		total += input.Size
		if total > 2<<30 || len(refs) >= sharedContextAttachmentLimit {
			return errors.New("shared_context_limit: historical attachment budget exceeded")
		}
		refs = append(refs, input.Attachment)
		if hashes != nil {
			hashes["shared-context/attachment/"+input.ID] = input.SHA256
		}
	}
	if err := e.validateAttachments(refs); err != nil {
		return fmt.Errorf("shared_context_changed: historical attachment is unavailable: %w", err)
	}
	return nil
}

func (e *Engine) readSharedContextInput(taskID, name string, expected string, hashes map[string]string) error {
	if !validSharedContextID(taskID) || !safeRelative(name) {
		return errors.New("shared_context_changed: invalid input evidence name")
	}
	root, err := os.OpenRoot(filepath.Join(e.Root, "evidence", taskID))
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	raw, err := readSharedContextFile(root, name, 16<<20)
	if err != nil {
		return err
	}
	if hashes != nil && hashes[name] != digest(raw) {
		return errors.New("shared_context_changed: input evidence changed during sealing")
	}
	var input struct {
		SharedContext *SharedContextReference `json:"shared_context"`
	}
	if err = json.Unmarshal(raw, &input); err != nil || input.SharedContext == nil || input.SharedContext.TaskID != taskID {
		return errors.New("shared_context_changed: dispatch context reference is missing")
	}
	return e.validateSharedContextReference(*input.SharedContext, expected, hashes)
}

func (e *Engine) validateContextBoundEvidence(task *Task, hashes map[string]string) error {
	if err := e.validateAuditReplanEvidence(task, hashes); err != nil {
		return err
	}
	return e.validateSharedContextEvidence(task, hashes)
}

// Final sealing may hold a Store write transaction. This validates only the
// supplied dispatch snapshot and its immutable files; it opens no database.
func (e *Engine) validateSharedContextEvidence(task *Task, hashes map[string]string) error {
	for _, dispatch := range task.Runs {
		if dispatch.ContextHash == "" || dispatch.Status == dispatchNotStarted {
			continue
		}
		if err := e.readSharedContextInput(task.ID, dispatch.InputEvidence, dispatch.ContextHash, hashes); err != nil {
			return err
		}
	}
	return nil
}
