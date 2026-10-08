package pihooks

import (
	"fmt"
	"io"
	"strings"

	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/notify"
)

type payload struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`
	Body      string `json:"body"`
}

func ParseMessage(stdin io.Reader) (notify.Message, error) {
	var p payload
	if err := common.DecodeHookPayload(stdin, &p); err != nil {
		return notify.Message{}, err
	}

	body := strings.TrimSpace(p.Body)
	switch p.Type {
	case "session_start":
		return notify.Message{
			Agent: "pi", Event: "session_start", SessionID: p.SessionID,
			Workspace: p.CWD, Title: notify.FormatTitle("pi", "session_start"),
			Body: notify.DefaultBody("session_start"),
		}, nil
	case "permission_required", "input_required", "run_completed", "run_failed":
		if body == "" {
			body = notify.DefaultBody(p.Type)
		}
		if p.Type == "permission_required" && body == "" {
			body = "操作需要您的确认"
		}
		if p.Type == "input_required" && body == "" {
			body = "等待您的输入"
		}
		if p.Type == "run_failed" && body == "" {
			body = "运行出错"
		}
		return notify.Message{
			Agent: "pi", Event: p.Type, SessionID: p.SessionID,
			Workspace: p.CWD, Title: notify.FormatTitle("pi", p.Type), Body: body,
		}, nil
	default:
		return notify.Message{}, fmt.Errorf("unsupported Pi event: %s", p.Type)
	}
}
