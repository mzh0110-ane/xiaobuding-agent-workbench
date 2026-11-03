package reliable

import "strings"

type RecoveryGuidance struct {
	Code       string `json:"code"`
	Summary    string `json:"summary"`
	NextAction string `json:"next_action"`
	Detail     string `json:"detail"`
}

const recoveryActionNone = "none"

func RecoveryGuidanceFor(task *Task) RecoveryGuidance {
	if task == nil {
		return RecoveryGuidance{Code: "unavailable", Summary: "任务暂不可用。", NextAction: recoveryActionNone}
	}
	guidance := classifyRecovery(task.Reason)
	if task.State == stateExhausted {
		guidance = classifyRecovery("three rounds exhausted")
	} else if !recoveryTaskPaused(task.State) {
		guidance = classifyRecovery("")
	}
	guidance.Detail = task.Reason
	if task.SuccessorTaskID != "" {
		guidance.Code, guidance.Summary, guidance.NextAction = "superseded", "本批已交接给下一批，请打开下一批继续。", recoveryActionNone
	}
	if task.ArchivedAt != "" {
		guidance.Code, guidance.Summary, guidance.NextAction = "archived", "请先恢复到任务列表，再继续处理。", recoveryActionNone
	}
	return guidance
}

func recoveryTaskPaused(state string) bool {
	switch state {
	case stateBlocked, stateInterrupted, stateCancelled:
		return true
	default:
		return false
	}
}

func classifyRecovery(reason string) RecoveryGuidance {
	raw := strings.ToLower(reason)
	if reason == "" {
		return RecoveryGuidance{Code: "none", Summary: "当前没有需要处理的阻碍。", NextAction: recoveryActionNone}
	}
	for _, rule := range recoveryRules() {
		if containsRecoveryMarker(raw, rule.markers) {
			return RecoveryGuidance{Code: rule.code, Summary: rule.summary, NextAction: rule.action}
		}
	}
	return RecoveryGuidance{Code: "needs_attention", Summary: "本批需要处理，请展开诊断记录核对原因后继续。", NextAction: recoveryActionNone}
}

type recoveryRule struct {
	code, summary, action string
	markers               []string
}

func recoveryRules() []recoveryRule {
	return []recoveryRule{
		{"stage_inputs_updated", "需求或规划已经更新，旧断点不能直接续用。保留已有修改，按最新需求重新规划后继续。", "replan", []string{"frozen stage inputs changed", "requirements or the frozen audit changed"}},
		{"source_changed", "源码或授权范围发生变化。请查看具体文件差异，明确范围后重新规划；已有修改保留。", "replan", []string{"stage_resume_source_changed", "source changed", "source drift", "outside audited scope", "outside allowed", "unauthorized scope", "scope exceeded"}},
		{"scope_invalid", "规划的修改范围需要调整。请核对具体目录与前后端分工后重新规划。", "replan", []string{"unsafe scope", "unsafe path", "scopes overlap", "audit exceeded", "scope_invalid", "audit_retry_invalidated"}},
		{"login_required", "原生工具的登录或授权需要处理。请回到对应的 Codex 或 Harness 登录，再继续本批。", "login", []string{"login expired", "authentication failed", "unauthorized", "http 401", "invalid api key", "not logged in", "token expired", "authentication required"}},
		{"storage_pressure", "E 盘数据目录可用空间不足。请查看空间统计，处理允许清理的临时文件后继续。", "resume", []string{"storage_pressure", "disk full", "no space left"}},
		{"round_limit", "本批首次实施与两次自动返工已结束。请查看遗留问题，再保留修改开启下一批。", "new_batch", []string{"three rounds exhausted", "three implementation rounds", "batch_exhausted"}},
		{"evidence_invalid", "执行证据或输入记录不完整。请展开诊断核对，不能跳过审查或强行应用。", recoveryActionNone, []string{"stage_resume_evidence", "resource_request", "resource_inventory", "evidence", "invalid schema", "decode", "unknown field"}},
		{"checks_failed", "实际检查未通过。请查看失败日志和对应要求，补充说明后再继续修改。", "checks", []string{"verification command failed", "verification failed", "checks failed", "automatic_checks_missing", "zero tests"}},
		{"execution_timeout", "执行超时或连接暂时中断，已有记录保留。可继续本批；运行重试不占代码返工轮次。", "resume", []string{"deadline exceeded", "timed out", "timeout", "connection reset", "connection refused", "http 429", "http 502", "http 503", "http 504"}},
		{"cancelled", "本批已停止，已有修改保留。请先检查副本，再决定是否继续。", "resume", []string{"context canceled", "context cancelled", "cancelled by the user"}},
	}
}

func containsRecoveryMarker(reason string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(reason, marker) {
			return true
		}
	}
	return false
}
