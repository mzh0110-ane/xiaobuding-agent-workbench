package reliable

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf8"

	bolt "go.etcd.io/bbolt"
)

type ProjectEvidence struct {
	TaskID string `json:"task_id"`
	Path   string `json:"path"`
	Label  string `json:"label"`
}

type ProjectItem struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	ShortTitle   string            `json:"short_title,omitempty"`
	Acceptance   string            `json:"acceptance"`
	Milestone    string            `json:"milestone,omitempty"`
	DependsOn    []string          `json:"depends_on,omitempty"`
	StatusReason string            `json:"status_reason,omitempty"`
	Status       string            `json:"status"`
	Source       string            `json:"source"`
	TaskID       string            `json:"task_id,omitempty"`
	Confirmed    bool              `json:"confirmed"`
	Evidence     []ProjectEvidence `json:"evidence"`
}

type ProjectMemory struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Text       string `json:"text"`
	SourceTask string `json:"source_task,omitempty"`
	Source     string `json:"source"`
	Commit     string `json:"commit,omitempty"`
	Verified   bool   `json:"verified"`
}

type ProjectBatch struct {
	ID    string `json:"id"`
	State string `json:"state"`
	Round int    `json:"round"`
	Goal  string `json:"goal"`
}

type ProjectRecord struct {
	ID           string          `json:"id"`
	Project      string          `json:"project"`
	Title        string          `json:"title"`
	Revision     int             `json:"revision"`
	AutoContinue bool            `json:"auto_continue"`
	Items        []ProjectItem   `json:"items"`
	Memories     []ProjectMemory `json:"memories"`
	Batches      []ProjectBatch  `json:"batches"`
	Handoff      *ProjectHandoff `json:"handoff,omitempty"`
}

var projectLedgerBucket = []byte("reliable-project-ledgers-v1")

const projectPlanned = "planned"

func projectLedgerID(project string) string {
	return digest([]byte(projectKey(project)))[:26]
}

func emptyProject(project string) *ProjectRecord {
	return &ProjectRecord{ID: projectLedgerID(project), Project: project, Title: filepath.Base(project), Items: []ProjectItem{}, Memories: []ProjectMemory{}, Batches: []ProjectBatch{}}
}

func readProject(tx *bolt.Tx, project string) (*ProjectRecord, error) {
	record := emptyProject(project)
	if bucket := tx.Bucket(projectLedgerBucket); bucket != nil {
		if raw := bucket.Get([]byte(record.ID)); raw != nil {
			if err := json.Unmarshal(raw, record); err != nil {
				return nil, err
			}
		}
	}
	return record, nil
}

func (s *Store) updateProject(project string, revision int, change func(*ProjectRecord) error) (*ProjectRecord, error) {
	var record *ProjectRecord
	err := s.db.Update(func(tx *bolt.Tx) error {
		var err error
		record, err = readProject(tx, project)
		if err != nil {
			return err
		}
		if revision >= 0 && record.Revision != revision {
			return errors.New("project_revision_conflict: reload the latest project record before saving")
		}
		if err = change(record); err != nil {
			return err
		}
		record.Revision++
		raw, err := json.Marshal(record)
		if err != nil {
			return err
		}
		if len(raw) > 256<<10 {
			return errors.New("project_limit: project knowledge exceeds 256 KiB")
		}
		bucket, err := tx.CreateBucketIfNotExists(projectLedgerBucket)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(record.ID), raw)
	})
	return record, err
}

func validProjectText(text string, limit int) bool {
	return strings.TrimSpace(text) != "" && len(text) <= limit && utf8.ValidString(text) && !strings.ContainsRune(text, 0)
}

func validOptionalProjectText(text string, limit int) bool {
	return text == "" || validProjectText(text, limit)
}

func (e *Engine) Project(id string) (*ProjectRecord, error) {
	task, err := e.Store.Get(id)
	if err != nil {
		return nil, err
	}
	var record *ProjectRecord
	if err = e.Store.db.View(func(tx *bolt.Tx) error {
		record, err = readProject(tx, task.Project)
		return err
	}); err != nil {
		return nil, err
	}
	tasks, err := e.Store.List()
	if err != nil {
		return nil, err
	}
	record = assembleProject(record, tasks)
	record.Handoff = projectHandoff(record, task)
	return record, nil
}

func assembleProject(record *ProjectRecord, tasks []*Task) *ProjectRecord {
	linked := make(map[string]bool)
	for i := range record.Items {
		linked[record.Items[i].TaskID] = true
	}
	for _, task := range tasks {
		if projectKey(task.Project) != projectKey(record.Project) {
			continue
		}
		record.Batches = append(record.Batches, ProjectBatch{ID: task.ID, State: task.State, Round: task.Round, Goal: task.Goal})
		projectTaskItems(record, task, linked[task.ID])
	}
	return record
}

func projectTaskItems(record *ProjectRecord, task *Task, linked bool) {
	status := projectTaskStatus(task)
	proof := projectTaskEvidence(task)
	for i := range record.Items {
		if record.Items[i].TaskID == task.ID {
			record.Items[i].Status, record.Items[i].Evidence = status, proof
		}
	}
	if !linked && !task.ReadOnly {
		record.Items = append(record.Items, ProjectItem{ID: "batch-" + task.ID, Title: task.Goal, Status: status, Source: "task", TaskID: task.ID, Confirmed: true, Evidence: proof})
	}
}

func projectTaskStatus(task *Task) string {
	switch task.State {
	case stateAccepted:
		return stateAccepted
	case stateAwaitingAcceptance:
		if projectTaskVerified(task) {
			return "verified"
		}
		return stateBlocked
	case stateBlocked, stateInterrupted, stateCancelled, stateExhausted:
		return stateBlocked
	default:
		return "in_progress"
	}
}

func projectTaskVerified(task *Task) bool {
	if task.RequirementRevision != task.AuditedRevision || task.EvidenceHash == "" || task.PatchHash == "" || task.Snapshot == "" {
		return false
	}
	commands := verificationCommands(task)
	if len(commands) == 0 || len(task.Checks) != len(commands) {
		return false
	}
	for i, check := range task.Checks {
		if check.Command != commands[i] || check.ExitCode != 0 || check.VerificationProblem != "" {
			return false
		}
	}
	decision, _ := Decide(task.Review, true, true, task.RequiredReads)
	return decision == Pass
}

func projectTaskEvidence(task *Task) []ProjectEvidence {
	proof := []ProjectEvidence{}
	if task.State == stateAccepted {
		proof = append(proof, ProjectEvidence{TaskID: task.ID, Path: "acceptance.json", Label: "人工验收记录"})
	}
	if task.EvidenceHash != "" && task.Round > 0 {
		proof = append(proof, ProjectEvidence{TaskID: task.ID, Path: roundFile(task.Round, "review"), Label: "独立审查"}, ProjectEvidence{TaskID: task.ID, Path: roundFile(task.Round, "checks"), Label: "实际检查"})
	}
	return proof
}
