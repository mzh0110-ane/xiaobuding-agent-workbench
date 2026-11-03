package reliable

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
)

type SharedContextStatus struct {
	Prepared           bool   `json:"prepared"`
	Complete           bool   `json:"complete"`
	TaskID             string `json:"task_id"`
	SavedBatches       int    `json:"saved_batches"`
	SavedMessages      int    `json:"saved_messages"`
	ArchivedMessages   int    `json:"archived_messages"`
	PreparedForMessage string `json:"prepared_for_message,omitempty"`
	IndexSHA256        string `json:"index_sha256,omitempty"`
	Summary            string `json:"summary"`
}

// Status never creates archives or updates task records. A prepared archive is
// a saved snapshot, not a claim that an Agent read it or that it is still fresh.
func (e *Engine) SharedContextStatus(id string) (*SharedContextStatus, error) {
	task, err := e.Store.Get(id)
	if err != nil {
		return nil, err
	}
	batches, err := e.sharedContextBatches(context.Background(), task, "")
	if err != nil {
		return nil, err
	}
	status := &SharedContextStatus{TaskID: task.ID, SavedBatches: len(batches), Summary: "项目记录已保存；下次派发前会准备共享交接。原生工具私有会话不互通。"}
	for _, batch := range batches {
		status.SavedMessages += len(batch.Messages)
	}
	root, err := e.sharedContextRoot(false)
	if os.IsNotExist(err) {
		return status, nil
	}
	if err != nil {
		return nil, err
	}
	raw, err := readSharedContextFile(root, filepath.Join(sharedContextHeadsName, id+".json"), 16<<10)
	_ = root.Close()
	if os.IsNotExist(err) {
		return status, nil
	}
	if err != nil {
		return nil, err
	}
	var ref SharedContextReference
	if err = json.Unmarshal(raw, &ref); err != nil {
		return nil, err
	}
	if ref.TaskID != task.ID {
		return nil, os.ErrInvalid
	}
	if err = e.validateSharedContextReference(ref, ref.IndexSHA256, nil); err != nil {
		return nil, err
	}
	status.Prepared, status.Complete, status.ArchivedMessages = true, ref.Complete, ref.MessageCount
	status.PreparedForMessage, status.IndexSHA256 = ref.MessageID, ref.IndexSHA256
	status.Summary = "已有可校验的完整交接快照；新消息和回复会在下次派发前更新。可用记录不代表模型已经阅读或理解。"
	return status, nil
}
