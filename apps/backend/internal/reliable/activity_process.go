package reliable

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"regexp"
	"strings"
)

type outputContextKey struct{}

const harnessToolKind = "harness"

type outputObserver func(string, []byte)

type activityLines struct {
	buffer  []byte
	dropped bool
	line    func([]byte)
}

func (s *activityLines) write(data []byte) {
	for _, b := range data {
		switch {
		case b == '\n' || b == '\r':
			if !s.dropped && len(s.buffer) > 0 {
				s.line(s.buffer)
			}
			s.buffer = nil
			s.dropped = false
		case len(s.buffer) < 1<<20 && !s.dropped:
			s.buffer = append(s.buffer, b)
		default:
			s.buffer = nil
			s.dropped = true
		}
	}
}

func (s *activityLines) flush() {
	if len(s.buffer) > 0 && !s.dropped {
		s.line(s.buffer)
	}
	s.buffer = nil
}

func observeProcessOutput(ctx context.Context, stdout, stderr *limitedBuffer) func() {
	observer, ok := ctx.Value(outputContextKey{}).(outputObserver)
	if !ok {
		return func() {}
	}
	out := &activityLines{line: func(line []byte) { observer("stdout", line) }}
	err := &activityLines{line: func(line []byte) { observer("stderr", line) }}
	stdout.observe, stderr.observe = out.write, err.write
	return func() { out.flush(); err.flush() }
}

type gitProgressWriter struct {
	target io.Writer
	lines  *activityLines
}

func (w *gitProgressWriter) Write(data []byte) (int, error) {
	n, err := w.target.Write(data)
	w.lines.write(data[:n])
	return n, err
}

func gitActivityWriter(ctx context.Context, target io.Writer) io.Writer {
	return &gitProgressWriter{target: target, lines: &activityLines{line: func(line []byte) {
		if match := gitProgressPattern.FindStringSubmatch(string(line)); len(match) == 4 {
			emitActivity(ctx, "progress", "本地复制 · "+match[1]+"%（"+match[2]+" / "+match[3]+" 文件）", "")
			return
		}
		emitActivity(ctx, "progress", "本地文件准备进度", string(line))
	}}}
}

var gitProgressPattern = regexp.MustCompile(`Updating files:\s+(\d+)% \((\d+)/(\d+)\)`)

func observeNativeProcess(ctx context.Context, provider string, redact func([]byte) []byte) context.Context {
	return context.WithValue(ctx, outputContextKey{}, outputObserver(func(stream string, line []byte) {
		if redact != nil {
			line = redact(line)
		}
		if stream == "stderr" {
			emitActivity(ctx, "diagnostic", "运行诊断", string(line))
			return
		}
		kind, title, detail := nativeActivity(provider, line)
		if kind != "" {
			emitActivity(ctx, kind, title, detail)
		}
	}))
}

type nativeActivityEvent struct {
	Type    string          `json:"type"`
	Text    string          `json:"text"`
	Summary string          `json:"summary"`
	Phase   string          `json:"phase"`
	Tool    string          `json:"tool"`
	Input   json.RawMessage `json:"input"`
	Result  string          `json:"result"`
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Item    struct {
		Type    string          `json:"type"`
		Text    string          `json:"text"`
		Command string          `json:"command"`
		Output  string          `json:"aggregated_output"`
		Status  string          `json:"status"`
		Summary json.RawMessage `json:"summary"`
	} `json:"item"`
}

func nativeActivity(provider string, raw []byte) (string, string, string) {
	var event nativeActivityEvent
	if json.Unmarshal(raw, &event) != nil {
		return "", "", ""
	}
	if provider == harnessToolKind {
		return harnessActivity(event)
	}
	switch event.Type {
	case "reasoning.summary":
		return activityThinking, "模型思考摘要", event.Summary
	case "thread.started":
		return activityAgent, "Codex 会话已建立", ""
	case "turn.started":
		return activityThinking, "模型开始处理任务", ""
	case "error", "turn.failed":
		return "error", "模型执行遇到错误", event.Message
	case "item.started", "item.updated", "item.completed":
		return codexItemActivity(event)
	case "turn.completed":
		return activityAgent, "Codex 本轮执行结束", ""
	}
	return "", "", ""
}

func codexItemActivity(event nativeActivityEvent) (string, string, string) {
	item := event.Item
	switch item.Type {
	case "reasoning":
		return activityThinking, "模型正在推理", ""
	case "agent_message":
		return "message", "Agent 的说明", publicReportText(item.Text)
	case "command_execution":
		if event.Type == "item.completed" {
			return "output", "命令执行结果 · " + item.Status, item.Command + "\n" + item.Output
		}
		return "command", "正在执行命令", item.Command
	case "file_change":
		return activityTool, "文件修改 · " + item.Status, "详细变更保留在实际交接记录中。"
	case "mcp_tool_call", "web_search":
		return activityTool, "正在调用工具", item.Type
	}
	return "", "", ""
}

func harnessActivity(event nativeActivityEvent) (string, string, string) {
	switch event.Type {
	case "session":
		return activityAgent, "Harness 会话已建立", ""
	case "thinking":
		return activityThinking, "模型正在推理", ""
	case "text", "final":
		return "message", "Agent 的说明", publicReportText(event.Text)
	case "tool_call":
		return activityTool, "调用工具 · " + event.Tool, string(event.Input)
	case "tool_result":
		return "output", "工具执行结果 · " + event.Status, event.Result
	case "status":
		return activityAgent, "Harness 执行状态", event.Phase
	case "error":
		return "error", "Harness 执行遇到错误", event.Message
	}
	return "", "", ""
}

func publicReportText(text string) string {
	var report struct {
		Summary string `json:"summary"`
	}
	if json.Unmarshal([]byte(text), &report) == nil && report.Summary != "" {
		return report.Summary
	}
	return strings.TrimSpace(text)
}

func observeCheckProcess(ctx context.Context, command string) context.Context {
	emitActivity(ctx, "command", "正在运行实际检查", command)
	return context.WithValue(ctx, outputContextKey{}, outputObserver(func(stream string, line []byte) {
		if len(bytes.TrimSpace(line)) > 0 {
			emitActivity(ctx, "output", "检查输出 · "+stream, string(line))
		}
	}))
}
