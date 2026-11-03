package reliable

import (
	"encoding/json"
	"unicode/utf8"
)

type HandoffNote struct {
	Kind       string `json:"kind"`
	Text       string `json:"text"`
	SourceTask string `json:"source_task"`
	Source     string `json:"source"`
	Commit     string `json:"commit,omitempty"`
	Verified   bool   `json:"verified"`
}

type HandoffRun struct {
	Role     string `json:"role"`
	Domain   string `json:"domain,omitempty"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Status   string `json:"status"`
	Round    int    `json:"round"`
}

type ProjectHandoff struct {
	TaskID              string            `json:"task_id"`
	ProjectRevision     int               `json:"project_revision"`
	RequirementRevision int               `json:"requirement_revision"`
	AuditedRevision     int               `json:"audited_revision"`
	SourceCommit        string            `json:"source_commit,omitempty"`
	SourceSnapshot      string            `json:"source_snapshot,omitempty"`
	State               string            `json:"state"`
	Round               int               `json:"round"`
	GoalExcerpt         string            `json:"goal_excerpt"`
	VerifiedBatch       bool              `json:"verified_batch"`
	RequirementsCurrent bool              `json:"requirements_current"`
	Facts               []HandoffNote     `json:"facts"`
	Inferences          []HandoffNote     `json:"inferences"`
	OpenIssues          []HandoffNote     `json:"open_issues"`
	Evidence            []ProjectEvidence `json:"evidence"`
	RecentRuns          []HandoffRun      `json:"recent_runs"`
	Warnings            []string          `json:"warnings"`
	Truncated           bool              `json:"truncated"`
}

func projectHandoff(record *ProjectRecord, task *Task) *ProjectHandoff {
	h := &ProjectHandoff{
		TaskID: task.ID, ProjectRevision: record.Revision,
		RequirementRevision: task.RequirementRevision, AuditedRevision: task.AuditedRevision,
		SourceCommit: projectReferenceCommit(task), SourceSnapshot: task.Snapshot, State: task.State, Round: task.Round,
		GoalExcerpt: projectExcerpt(task.Goal, 360), RequirementsCurrent: task.RequirementRevision == task.AuditedRevision,
		VerifiedBatch: projectTaskVerified(task) && (task.State == stateAwaitingAcceptance || task.State == stateAccepted),
		Facts:         []HandoffNote{}, Inferences: []HandoffNote{}, OpenIssues: []HandoffNote{},
		Evidence: projectTaskEvidence(task), RecentRuns: []HandoffRun{}, Warnings: []string{},
		Truncated: utf8.RuneCountInString(task.Goal) > 360 || len(task.Runs) > 4,
	}
	appendHandoffClaims(h, task)
	appendHandoffMemories(h, record)
	appendHandoffRuns(h, task)
	if !h.RequirementsCurrent {
		h.Warnings = append(h.Warnings, "需求已有更新，历史检查和审查不能覆盖当前要求。")
	}
	if !h.VerifiedBatch {
		h.Warnings = append(h.Warnings, "本批尚无完整核验结果，记录内容需要结合当前源码检查。")
	}
	h.Warnings = append(h.Warnings, "交接卡来自已保存记录，不代表实时源码或整个项目均已验证。")
	boundProjectHandoff(h)
	return h
}

func handoffNote(task *Task, kind, source, text string) HandoffNote {
	return HandoffNote{Kind: kind, Text: projectExcerpt(text, 220), SourceTask: task.ID, Source: source, Commit: task.Base}
}

func appendHandoffClaims(h *ProjectHandoff, task *Task) {
	if task.Audit != nil {
		appendHandoffTexts(h, task, task.Audit.Facts, "reported_fact", "audit", 4, &h.Facts)
		appendHandoffTexts(h, task, task.Audit.Inferences, "inference", "audit", 4, &h.Inferences)
		appendHandoffTexts(h, task, task.Audit.Unknowns, "unknown", "audit", 6, &h.OpenIssues)
	}
	if task.Reason != "" && task.State != stateAccepted && task.State != stateAwaitingAcceptance {
		appendHandoffTexts(h, task, []string{task.Reason}, "execution", "controller", 7, &h.OpenIssues)
	}
	var review Review
	if json.Unmarshal(task.Review, &review) == nil {
		texts := make([]string, 0, len(review.Findings))
		for _, finding := range review.Findings {
			texts = append(texts, finding.Reason)
		}
		appendHandoffTexts(h, task, texts, "review_finding", "review", 10, &h.OpenIssues)
	}
}

func appendHandoffTexts(h *ProjectHandoff, task *Task, texts []string, kind, source string, limit int, target *[]HandoffNote) {
	for _, text := range texts {
		if len(*target) >= limit {
			h.Truncated = true
			return
		}
		if utf8.RuneCountInString(text) > 220 {
			h.Truncated = true
		}
		*target = append(*target, handoffNote(task, kind, source, text))
	}
}

func appendHandoffMemories(h *ProjectHandoff, record *ProjectRecord) {
	stale := false
	for i := len(record.Memories) - 1; i >= 0; i-- {
		memory := record.Memories[i]
		if memory.Commit != "" && h.SourceCommit != "" && memory.Commit != h.SourceCommit {
			stale = true
		}
		if memory.Kind != "open_issue" {
			continue
		}
		if len(h.OpenIssues) >= 12 {
			h.Truncated = true
			continue
		}
		if utf8.RuneCountInString(memory.Text) > 220 {
			h.Truncated = true
		}
		h.OpenIssues = append(h.OpenIssues, HandoffNote{Kind: memory.Kind, Text: projectExcerpt(memory.Text, 220), SourceTask: memory.SourceTask, Source: memory.Source, Commit: memory.Commit, Verified: memory.Verified && memory.Commit != "" && memory.Commit == h.SourceCommit})
	}
	if stale {
		h.Warnings = append(h.Warnings, "部分项目记忆关联旧版本，接续前需要重新核对。")
	}
}

func appendHandoffRuns(h *ProjectHandoff, task *Task) {
	start := len(task.Runs) - 4
	if start < 0 {
		start = 0
	}
	for _, run := range task.Runs[start:] {
		h.RecentRuns = append(h.RecentRuns, HandoffRun{Role: projectExcerpt(run.Role, 30), Domain: projectExcerpt(run.Domain, 30), Provider: projectExcerpt(run.ProviderID, 60), Model: projectExcerpt(run.Model, 60), Status: projectExcerpt(run.Status, 30), Round: run.Round})
	}
}

func boundProjectHandoff(h *ProjectHandoff) {
	for {
		raw, _ := json.Marshal(h)
		if len(raw) <= 14000 {
			return
		}
		h.Truncated = true
		switch {
		case len(h.OpenIssues) > 0:
			h.OpenIssues = h.OpenIssues[:len(h.OpenIssues)-1]
		case len(h.Inferences) > 0:
			h.Inferences = h.Inferences[:len(h.Inferences)-1]
		case len(h.Facts) > 0:
			h.Facts = h.Facts[:len(h.Facts)-1]
		case len(h.RecentRuns) > 0:
			h.RecentRuns = h.RecentRuns[:len(h.RecentRuns)-1]
		default:
			return
		}
	}
}
