package kimihooks

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hellolib/agent-notify/internal/notify"
)

func parseMessageBytes(data []byte) (notify.Message, error) {
	return ParseMessage(bytes.NewReader(data))
}

func TestParseSessionStart(t *testing.T) {
	data := readTestdata(t, "session_start.json")
	msg, err := parseMessageBytes(data)
	if err != nil {
		t.Fatalf("ParseMessage() error = %v", err)
	}
	if msg.Agent != "kimi" {
		t.Fatalf("Agent = %q, want kimi", msg.Agent)
	}
	if msg.Event != "session_start" {
		t.Fatalf("Event = %q, want session_start", msg.Event)
	}
	if msg.SessionID != "sess-kimi-0" {
		t.Fatalf("SessionID = %q, want sess-kimi-0", msg.SessionID)
	}
	if msg.Workspace != "/Users/demo/project" {
		t.Fatalf("Workspace = %q, want /Users/demo/project", msg.Workspace)
	}
}

func TestParsePermissionRequest(t *testing.T) {
	data := readTestdata(t, "permission_request.json")
	msg, err := parseMessageBytes(data)
	if err != nil {
		t.Fatalf("ParseMessage() error = %v", err)
	}
	if msg.Event != "permission_required" {
		t.Fatalf("Event = %q, want permission_required", msg.Event)
	}
	if !strings.Contains(msg.Body, "Bash") {
		t.Fatalf("Body = %q, want tool name Bash", msg.Body)
	}
}

func TestParseNotificationPermission(t *testing.T) {
	data := readTestdata(t, "notification_permission.json")
	msg, err := parseMessageBytes(data)
	if err != nil {
		t.Fatalf("ParseMessage() error = %v", err)
	}
	if msg.Event != "permission_required" {
		t.Fatalf("Event = %q, want permission_required", msg.Event)
	}
	if !strings.Contains(msg.Body, "Permission required") {
		t.Fatalf("Body = %q, want permission text", msg.Body)
	}
}

func TestParseNotificationWaitingInput(t *testing.T) {
	data := readTestdata(t, "notification_waiting_input.json")
	msg, err := parseMessageBytes(data)
	if err != nil {
		t.Fatalf("ParseMessage() error = %v", err)
	}
	if msg.Event != "input_required" {
		t.Fatalf("Event = %q, want input_required", msg.Event)
	}
	if !strings.Contains(msg.Body, "提示:") {
		t.Fatalf("Body = %q, want 提示 prefix", msg.Body)
	}
}

func TestParseNotificationTaskCompleted(t *testing.T) {
	data := readTestdata(t, "notification_task_completed.json")
	msg, err := parseMessageBytes(data)
	if err != nil {
		t.Fatalf("ParseMessage() error = %v", err)
	}
	if msg.Event != "run_completed" {
		t.Fatalf("Event = %q, want run_completed", msg.Event)
	}
	if !strings.Contains(msg.Body, "summarize docs") {
		t.Fatalf("Body = %q, want task body", msg.Body)
	}
}

func TestParseStop(t *testing.T) {
	data := readTestdata(t, "stop.json")
	msg, err := parseMessageBytes(data)
	if err != nil {
		t.Fatalf("ParseMessage() error = %v", err)
	}
	if msg.Event != "run_completed" {
		t.Fatalf("Event = %q, want run_completed", msg.Event)
	}
}

func TestParseStopFailure(t *testing.T) {
	data := readTestdata(t, "stop_failure.json")
	msg, err := parseMessageBytes(data)
	if err != nil {
		t.Fatalf("ParseMessage() error = %v", err)
	}
	if msg.Event != "run_failed" {
		t.Fatalf("Event = %q, want run_failed", msg.Event)
	}
	if !strings.Contains(msg.Body, "API timeout") {
		t.Fatalf("Body = %q, want error message", msg.Body)
	}
}

func TestParsePostToolUseFailure(t *testing.T) {
	data := readTestdata(t, "post_tool_use_failure.json")
	msg, err := parseMessageBytes(data)
	if err != nil {
		t.Fatalf("ParseMessage() error = %v", err)
	}
	if msg.Event != "run_failed" {
		t.Fatalf("Event = %q, want run_failed", msg.Event)
	}
	if !strings.Contains(msg.Body, "Bash") {
		t.Fatalf("Body = %q, want tool name", msg.Body)
	}
}

func TestParseUnsupportedEvent(t *testing.T) {
	_, err := parseMessageBytes([]byte(`{"hook_event_name":"PreToolUse"}`))
	if err == nil {
		t.Fatal("expected error for unsupported event")
	}
}

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "kimi-hooks", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
