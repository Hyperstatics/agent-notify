// Package kimihooks parses Kimi Code CLI hook stdin payloads into notify.Message values.
//
// Kimi Code writes [[hooks]] into ~/.kimi-code/config.toml (or $KIMI_CODE_HOME/config.toml)
// and delivers snake_case JSON on stdin. Event names in the config are PascalCase
// (SessionStart, Stop, …); hook_event_name in the payload matches those names.
//
// Notification body strings (e.g. "未知错误", "等待您的操作") are hardcoded Chinese to
// match claudehooks, grokhooks, and notify.DefaultBody / FormatTitle.
package kimihooks

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/notify"
)

// payload 描述 Kimi Code hooks 通过 stdin 投递的事件 JSON。
// 官方文档字段为 snake_case；同时兼容 camelCase，避免 CLI 版本差异。
type payload struct {
	HookEventName         string          `json:"hook_event_name"`
	HookEventNameCamel    string          `json:"hookEventName"`
	SessionID             string          `json:"session_id"`
	SessionIDCamel        string          `json:"sessionId"`
	CWD                   string          `json:"cwd"`
	SessionTitle          string          `json:"session_title"`
	NotificationType      string          `json:"notification_type"`
	NotificationTypeCamel string          `json:"notificationType"`
	Title                 string          `json:"title"`
	Body                  string          `json:"body"`
	Message               string          `json:"message"`
	ToolName              string          `json:"tool_name"`
	ToolNameCamel         string          `json:"toolName"`
	ToolInput             json.RawMessage `json:"tool_input"`
	ToolInputCamel        json.RawMessage `json:"toolInput"`
	Kind                  string          `json:"kind"`
	TaskKind              string          `json:"task_kind"`
	Description           string          `json:"description"`
	AgentName             string          `json:"agent_name"`
	AgentNameCamel        string          `json:"agentName"`
	AgentType             string          `json:"agent_type"`
	AgentTypeCamel        string          `json:"agentType"`
	SubagentType          string          `json:"subagent_type"`
	SubagentTypeCamel     string          `json:"subagentType"`
	Error                 string          `json:"error"`
	ErrorType             string          `json:"error_type"`
	ErrorMessage          string          `json:"error_message"`
	ErrorMessageCamel     string          `json:"errorMessage"`
}

func (p payload) eventOf() string {
	if p.HookEventName != "" {
		return p.HookEventName
	}
	return p.HookEventNameCamel
}

func (p payload) sessionOf() string {
	if p.SessionID != "" {
		return p.SessionID
	}
	return p.SessionIDCamel
}

func (p payload) toolNameOf() string {
	if p.ToolName != "" {
		return p.ToolName
	}
	return p.ToolNameCamel
}

func (p payload) notificationTypeOf() string {
	if p.NotificationType != "" {
		return p.NotificationType
	}
	return p.NotificationTypeCamel
}

func (p payload) errorMessageOf() string {
	if p.ErrorMessage != "" {
		return p.ErrorMessage
	}
	if p.ErrorMessageCamel != "" {
		return p.ErrorMessageCamel
	}
	if p.Error != "" {
		return p.Error
	}
	if p.ErrorType != "" {
		return p.ErrorType
	}
	return ""
}

func (p payload) kindOf() string {
	if p.Kind != "" {
		return p.Kind
	}
	return p.TaskKind
}

func (p payload) toolInputOf() map[string]any {
	if m := common.LenientObject(p.ToolInput); m != nil {
		return m
	}
	return common.LenientObject(p.ToolInputCamel)
}

func ParseMessage(stdin io.Reader) (notify.Message, error) {
	var p payload
	if err := common.DecodeHookPayload(stdin, &p); err != nil {
		return notify.Message{}, err
	}

	event := normalizeEventName(p.eventOf())
	switch event {
	case "session_start":
		return notify.Message{
			Agent:     "kimi",
			Event:     "session_start",
			SessionID: p.sessionOf(),
			Workspace: p.CWD,
			Title:     notify.FormatTitle("kimi", "session_start"),
			Body:      notify.DefaultBody("session_start"),
		}, nil
	case "permission_request":
		return notify.Message{
			Agent:     "kimi",
			Event:     "permission_required",
			SessionID: p.sessionOf(),
			Workspace: p.CWD,
			Title:     notify.FormatTitle("kimi", "permission_required"),
			Body:      permissionBody(firstNonEmpty(p.Body, p.Message, p.Title), p.toolNameOf()),
		}, nil
	case "notification":
		return parseNotification(p)
	case "pre_tool_use":
		if !isAskUserQuestionTool(p.toolNameOf()) {
			return notify.Message{}, fmt.Errorf("unsupported hook event: %s tool=%s", p.eventOf(), p.toolNameOf())
		}
		return inputRequiredMessage(p, extractQuestionHint(p), "AskUserQuestion"), nil
	case "task_started":
		if !isQuestionTaskKind(p.kindOf()) {
			return notify.Message{}, fmt.Errorf("unsupported hook event: %s kind=%s", p.eventOf(), p.kindOf())
		}
		return inputRequiredMessage(p, firstNonEmpty(p.Description, p.Title, p.Message, "请选择一个选项"), "question"), nil
	case "subagent_start":
		return notify.Message{
			Agent:     "kimi",
			Event:     "subagent_start",
			SessionID: p.sessionOf(),
			Workspace: p.CWD,
		}, nil
	case "subagent_stop":
		return notify.Message{
			Agent:     "kimi",
			Event:     "subagent_stop",
			SessionID: p.sessionOf(),
			Workspace: p.CWD,
		}, nil
	case "stop":
		if isSubagentPayload(p) {
			return notify.Message{}, fmt.Errorf("skip subagent stop")
		}
		body := notify.DefaultBody("run_completed")
		if hint := firstNonEmpty(p.Body, p.Message, p.SessionTitle); hint != "" {
			body = common.TruncateRunes(hint, 200)
		}
		return notify.Message{
			Agent:     "kimi",
			Event:     "run_completed",
			SessionID: p.sessionOf(),
			Workspace: p.CWD,
			Title:     notify.FormatTitle("kimi", "run_completed"),
			Body:      body,
		}, nil
	case "stop_failure", "post_tool_use_failure":
		errMsg := p.errorMessageOf()
		if errMsg == "" {
			errMsg = "未知错误"
		}
		errMsg = common.TruncateRunes(errMsg, 200)
		tool := strings.TrimSpace(p.toolNameOf())
		body := fmt.Sprintf("错误: %s", errMsg)
		if tool != "" {
			body = fmt.Sprintf("工具: %s\n错误: %s", tool, errMsg)
		}
		return notify.Message{
			Agent:     "kimi",
			Event:     "run_failed",
			SessionID: p.sessionOf(),
			Workspace: p.CWD,
			Title:     notify.FormatTitle("kimi", "run_failed"),
			Body:      body,
		}, nil
	default:
		return notify.Message{}, fmt.Errorf("unsupported hook event: %s", p.eventOf())
	}
}

func parseNotification(p payload) (notify.Message, error) {
	notifType := strings.ToLower(strings.TrimSpace(p.notificationTypeOf()))
	msg := firstNonEmpty(p.Body, p.Message, p.Title)

	if isPermissionNotificationType(notifType) || isPermissionNotificationMessage(strings.ToLower(msg)) {
		return notify.Message{
			Agent:     "kimi",
			Event:     "permission_required",
			SessionID: p.sessionOf(),
			Workspace: p.CWD,
			Title:     notify.FormatTitle("kimi", "permission_required"),
			Body:      permissionBody(msg, p.toolNameOf()),
		}, nil
	}
	if isInputRequiredNotificationType(notifType) || isInputRequiredNotificationMessage(strings.ToLower(msg)) {
		return inputRequiredMessage(p, msg, notifType), nil
	}
	if isTaskCompletedNotificationType(notifType) || isTaskFailedNotificationType(notifType) {
		return notify.Message{}, fmt.Errorf("skip background task notification type=%s", notifType)
	}

	// Unknown notification with any payload → input_required, matching Grok.
	if msg != "" || notifType != "" {
		return inputRequiredMessage(p, msg, notifType), nil
	}
	return notify.Message{}, fmt.Errorf("unsupported notification: message=%q type=%q", msg, notifType)
}

func inputRequiredMessage(p payload, msg, notifType string) notify.Message {
	hint := extractInputHint(msg)
	if hint == "" {
		hint = notifType
	}
	if hint == "" {
		hint = "等待您的操作"
	}
	return notify.Message{
		Agent:     "kimi",
		Event:     "input_required",
		SessionID: p.sessionOf(),
		Workspace: p.CWD,
		Title:     notify.FormatTitle("kimi", "input_required"),
		Body:      fmt.Sprintf("提示: %s", hint),
	}
}

func permissionBody(msg, toolName string) string {
	tool := strings.TrimSpace(toolName)
	if strings.TrimSpace(msg) != "" {
		body := common.TruncateRunes(strings.TrimSpace(msg), 200)
		if tool != "" {
			return fmt.Sprintf("工具: %s\n%s", tool, body)
		}
		return body
	}
	if tool != "" {
		return fmt.Sprintf("工具: %s\n操作需要您的授权许可", tool)
	}
	return "操作需要您的授权许可"
}

func normalizeEventName(name string) string {
	name = strings.TrimSpace(name)
	switch name {
	case "SessionStart", "session_start":
		return "session_start"
	case "PermissionRequest", "permission_request":
		return "permission_request"
	case "Notification", "notification":
		return "notification"
	case "Stop", "stop":
		return "stop"
	case "StopFailure", "stop_failure":
		return "stop_failure"
	case "PostToolUseFailure", "post_tool_use_failure":
		return "post_tool_use_failure"
	case "PreToolUse", "pre_tool_use":
		return "pre_tool_use"
	case "TaskStarted", "task_started":
		return "task_started"
	case "SubagentStart", "subagent_start":
		return "subagent_start"
	case "SubagentStop", "subagent_stop":
		return "subagent_stop"
	default:
		return strings.ToLower(name)
	}
}

func isSubagentPayload(p payload) bool {
	return strings.TrimSpace(p.AgentName) != "" ||
		strings.TrimSpace(p.AgentNameCamel) != "" ||
		strings.TrimSpace(p.AgentType) != "" ||
		strings.TrimSpace(p.AgentTypeCamel) != "" ||
		strings.TrimSpace(p.SubagentType) != "" ||
		strings.TrimSpace(p.SubagentTypeCamel) != ""
}

func isAskUserQuestionTool(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return n == "askuserquestion" || n == "ask_user_question"
}

func isQuestionTaskKind(kind string) bool {
	return strings.EqualFold(strings.TrimSpace(kind), "question")
}

func extractQuestionHint(p payload) string {
	input := p.toolInputOf()
	if qs, ok := input["questions"].([]any); ok && len(qs) > 0 {
		if q, ok := qs[0].(map[string]any); ok {
			if s, ok := q["question"].(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return firstNonEmpty(p.Description, p.Title, p.Message, p.Body, "请选择一个选项")
}

func isPermissionNotificationType(t string) bool {
	switch t {
	case "permission_prompt",
		"permission",
		"permission_request",
		"approval",
		"approval_prompt",
		"approval_request":
		return true
	default:
		return false
	}
}

func isInputRequiredNotificationType(t string) bool {
	switch t {
	case "idle_prompt",
		"input_required",
		"waiting_input",
		"needs_input":
		return true
	default:
		return false
	}
}

func isTaskCompletedNotificationType(t string) bool {
	switch t {
	case "task.completed", "task_completed", "task.complete":
		return true
	default:
		return false
	}
}

func isTaskFailedNotificationType(t string) bool {
	switch t {
	case "task.failed", "task_failed", "task.error", "task_error":
		return true
	default:
		return false
	}
}

func isPermissionNotificationMessage(msg string) bool {
	if msg == "" {
		return false
	}
	return strings.Contains(msg, "permission required") ||
		strings.Contains(msg, "requires permission") ||
		strings.Contains(msg, "needs permission") ||
		strings.Contains(msg, "needs your permission") ||
		strings.Contains(msg, "approval required") ||
		strings.Contains(msg, "needs approval") ||
		strings.Contains(msg, "requires approval") ||
		strings.Contains(msg, "awaiting approval") ||
		strings.Contains(msg, "authorization required") ||
		strings.Contains(msg, "需要您的授权") ||
		strings.Contains(msg, "需要授权") ||
		strings.Contains(msg, "授权许可") ||
		strings.Contains(msg, "等待授权")
}

func isInputRequiredNotificationMessage(msg string) bool {
	if msg == "" {
		return false
	}
	return strings.Contains(msg, "waiting for your input") ||
		strings.Contains(msg, "waiting for input") ||
		strings.Contains(msg, "needs input") ||
		strings.Contains(msg, "等待输入") ||
		strings.Contains(msg, "等待您的输入") ||
		strings.Contains(msg, "等待您的操作")
}

func extractInputHint(msg string) string {
	msg = strings.TrimSpace(msg)
	prefixes := []string{
		"kimi is waiting for your input",
		"waiting for your input",
		"waiting for input",
		"needs input",
	}
	lower := strings.ToLower(msg)
	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			rest := strings.TrimSpace(msg[len(prefix):])
			rest = strings.TrimPrefix(rest, ":")
			rest = strings.TrimSpace(rest)
			if rest != "" {
				return rest
			}
		}
	}
	return common.TruncateRunes(msg, 100)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
