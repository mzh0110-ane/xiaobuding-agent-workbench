package reliable

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Check struct {
	VerificationProblem string `json:"verification_problem,omitempty"`
	Command             string `json:"command"`
	ExitCode            int    `json:"exit_code"`
	Log                 string `json:"log"`
}

type Audit struct {
	ReadAttachments   []string `json:"read_attachments,omitempty"`
	FrontendPaths     []string `json:"frontend_paths,omitempty"`
	BackendPaths      []string `json:"backend_paths,omitempty"`
	Summary           string   `json:"summary"`
	Facts             []string `json:"facts"`
	Inferences        []string `json:"inferences"`
	Unknowns          []string `json:"unknowns"`
	ReadFiles         []string `json:"read_files"`
	RequiredReadFiles []string `json:"required_read_files"`
	AllowedPaths      []string `json:"allowed_paths"`
	Plan              []string `json:"plan"`
}

type Task struct {
	AuditReplan         *AuditReplanApproval  `json:"audit_replan,omitempty"`
	StageRecovery       *StageRecovery        `json:"stage_recovery,omitempty"`
	LocalCheckpoint     *AcceptanceCheckpoint `json:"local_checkpoint,omitempty"`
	Attachments         []Attachment          `json:"attachments,omitempty"`
	SeedPending         *ExhaustedSeed        `json:"seed_pending,omitempty"`
	SeedEvidence        []string              `json:"seed_evidence,omitempty"`
	SuccessorTaskID     string                `json:"successor_task_id,omitempty"`
	ProcessCleanupEpoch string                `json:"process_cleanup_epoch,omitempty"`
	RecoveryPending     *RecoveryAttempt      `json:"recovery_pending,omitempty"`
	RecoveryEvidence    []string              `json:"recovery_evidence,omitempty"`
	FuturePlan          *FutureModelPlan      `json:"future_plan,omitempty"`
	ParentTaskID        string                `json:"parent_task_id,omitempty"`
	PreviousContext     json.RawMessage       `json:"previous_context,omitempty"`
	Messages            []TaskMessage         `json:"messages,omitempty"`
	RequirementRevision int                   `json:"requirement_revision,omitempty"`
	AuditedRevision     int                   `json:"audited_revision,omitempty"`
	RequirementEvidence []string              `json:"requirement_evidence,omitempty"`
	ContinuePending     bool                  `json:"continue_pending,omitempty"`
	ReadOnly            bool                  `json:"read_only,omitempty"`
	ArchivedAt          string                `json:"archived_at,omitempty"`
	AutoChecks          bool                  `json:"auto_checks,omitempty"`
	DiscoveredCommands  []string              `json:"discovered_commands,omitempty"`
	CheckPlanEvidence   string                `json:"check_plan_evidence,omitempty"`
	LocalInitEvidence   string                `json:"local_init_evidence,omitempty"`
	AutoSnapshot        bool                  `json:"auto_snapshot,omitempty"`
	SetupEvidence       []string              `json:"setup_evidence,omitempty"`
	DomainEvidence      []string              `json:"domain_evidence,omitempty"`
	ModelPlan           ModelPlan             `json:"model_plan"`
	PlanRevision        int                   `json:"plan_revision"`
	PlanEvidence        []string              `json:"plan_evidence"`
	Runs                []Dispatch            `json:"runs"`
	ID                  string                `json:"id"`
	Project             string                `json:"project"`
	Goal                string                `json:"goal"`
	Commands            []string              `json:"commands"`
	AllowedPaths        []string              `json:"allowed_paths"`
	ApprovedPaths       []string              `json:"approved_paths"`
	State               string                `json:"state"`
	Round               int                   `json:"round"`
	Worktree            string                `json:"worktree"`
	Base                string                `json:"base"`
	Snapshot            string                `json:"snapshot"`
	PatchHash           string                `json:"patch_hash"`
	EvidenceHash        string                `json:"evidence_hash"`
	RequiredReads       []string              `json:"required_reads"`
	Baseline            *Snapshot             `json:"baseline,omitempty"`
	Reason              string                `json:"reason"`
	Audit               *Audit                `json:"audit,omitempty"`
	Review              json.RawMessage       `json:"review,omitempty"`
	Checks              []Check               `json:"checks"`
	Sessions            []string              `json:"sessions"`
	Created             string                `json:"created"`
	Updated             string                `json:"updated"`
}

type Store struct{ db *bolt.DB }

var tasksBucket = []byte("tasks")

func OpenStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error { _, e := tx.CreateBucketIfNotExists(tasksBucket); return e })
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func projectKey(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = resolved
	}
	return strings.ToLower(filepath.Clean(path))
}

func (s *Store) Create(t *Task) error {
	return s.db.Update(func(tx *bolt.Tx) error { return createTaskTx(tx, t) })
}

func createTaskTx(tx *bolt.Tx, t *Task) error {
	b := tx.Bucket(tasksBucket)
	if b.Get([]byte(t.ID)) != nil {
		return errors.New("task already exists")
	}
	if err := b.ForEach(func(_, v []byte) error {
		var old Task
		if err := json.Unmarshal(v, &old); err != nil {
			return err
		}
		if projectsOverlap(old.Project, t.Project) && old.State != stateAccepted && old.State != stateCancelled && old.State != stateInspected && !supersededProjectTask(tx, &old) {
			return errors.New("this project already has an unfinished task")
		}
		return nil
	}); err != nil {
		return err
	}
	t.Created = time.Now().UTC().Format(time.RFC3339Nano)
	t.Updated = t.Created
	raw, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return b.Put([]byte(t.ID), raw)
}

func projectsOverlap(a, b string) bool {
	a, b = projectKey(a), projectKey(b)
	return pathInside(a, b) || pathInside(b, a)
}

func (s *Store) Get(id string) (*Task, error) {
	var task Task
	err := s.db.View(func(tx *bolt.Tx) error {
		raw := tx.Bucket(tasksBucket).Get([]byte(id))
		if raw == nil {
			return os.ErrNotExist
		}
		return json.Unmarshal(raw, &task)
	})
	return &task, err
}

func (s *Store) List() ([]*Task, error) {
	out := make([]*Task, 0)
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(tasksBucket).ForEach(func(_, raw []byte) error {
			var t Task
			if err := json.Unmarshal(raw, &t); err != nil {
				return err
			}
			out = append(out, &t)
			return nil
		})
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Created > out[j].Created })
	return out, err
}

func (s *Store) Update(id string, fn func(*Task) error) (*Task, error) {
	var t Task
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(tasksBucket)
		raw := b.Get([]byte(id))
		if raw == nil {
			return os.ErrNotExist
		}
		if err := json.Unmarshal(raw, &t); err != nil {
			return err
		}
		if err := fn(&t); err != nil {
			return err
		}
		t.Updated = time.Now().UTC().Format(time.RFC3339Nano)
		encoded, err := json.Marshal(t)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), encoded)
	})
	return &t, err
}

func (s *Store) StartRound(id string) (*Task, error) {
	return s.Update(id, func(t *Task) error {
		if t.RequirementRevision != t.AuditedRevision {
			return errRequirementsChanged
		}
		if t.State != stateAuditing {
			return fmt.Errorf("cannot start an implementation from %s", t.State)
		}
		if t.Round >= MaxRounds {
			return errors.New("three implementation rounds have been used")
		}
		t.Round++
		t.State = stateImplementing
		t.Snapshot = ""
		t.EvidenceHash = ""
		t.Review = nil
		t.Checks = nil
		return nil
	})
}

func (s *Store) Recover() error {
	all, err := s.List()
	if err != nil {
		return err
	}
	for _, t := range all {
		if err = s.interruptMessages(t.ID); err != nil {
			return err
		}
		switch t.State {
		case stateQueued, statePreparing, stateAuditing, stateImplementing, stateChecking, stateReviewing, stateApplying, stateCancelling:
			if _, err = s.Update(t.ID, func(t *Task) error {
				recoverDispatches(t)
				if t.State == stateCancelling {
					t.State = stateCancelled
					t.Reason = "Cancellation completed after restart; evidence and round counter have been retained."
				} else {
					t.State = stateInterrupted
					t.Reason = "The application stopped during execution. Review the evidence before continuing; the round counter has been preserved."
				}
				return nil
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) interruptMessages(id string) error {
	current, err := s.Get(id)
	if err != nil {
		return err
	}
	pending := current.ContinuePending
	for _, message := range current.Messages {
		pending = pending || (message.Kind == messageQuestion && (message.Status == stateQueued || message.Status == messageReading))
	}
	if !pending {
		return nil
	}
	_, err = s.Update(id, func(task *Task) error {
		task.ContinuePending = false
		for i := range task.Messages {
			message := &task.Messages[i]
			if message.Kind == messageQuestion && (message.Status == stateQueued || message.Status == messageReading) {
				message.Status = "interrupted"
				message.Error = "Application restarted before delivery completed; resend to continue"
			}
		}
		return nil
	})
	return err
}
