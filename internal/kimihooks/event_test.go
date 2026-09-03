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

func TestParseNotificationTaskCompletedIsSkipped(t *testing.T) {
	data := readTestdata(t, "notification_task_completed.json")
	_, err := parseMessageBytes(data)
	if err == nil {
		t.Fatal("background task.completed should be skipped")
	}
}

func TestParseSubagentStartAndStop(t *testing.T) {
	msg, err := parseMessageBytes([]byte(`{"hook_event_name":"SubagentStart","session_id":"sess-kimi-sa","cwd":"/tmp","agent_name":"explore"}`))
	if err != nil {
		t.Fatalf("SubagentStart error = %v", err)
	}
	if msg.Event != "subagent_start" {
		t.Fatalf("Event = %q, want subagent_start", msg.Event)
	}
	msg, err = parseMessageBytes([]byte(`{"hook_event_name":"SubagentStop","session_id":"sess-kimi-sa","cwd":"/tmp","agent_name":"explore"}`))
	if err != nil {
		t.Fatalf("SubagentStop error = %v", err)
	}
	if msg.Event != "subagent_stop" {
		t.Fatalf("Event = %q, want subagent_stop", msg.Event)
	}
}

func TestParseStopWithSubagentTypeIsSkipped(t *testing.T) {
	_, err := parseMessageBytes([]byte(`{"hook_event_name":"Stop","session_id":"sess-kimi-sa","cwd":"/tmp","subagent_type":"explore"}`))
	if err == nil {
		t.Fatal("Stop with subagent_type should be skipped")
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

func TestParseAskUserQuestion(t *testing.T) {
	data := readTestdata(t, "ask_user_question.json")
	msg, err := parseMessageBytes(data)
	if err != nil {
		t.Fatalf("ParseMessage() error = %v", err)
	}
	if msg.Event != "input_required" {
		t.Fatalf("Event = %q, want input_required", msg.Event)
	}
	if !strings.Contains(msg.Body, "Which approach should we take?") {
		t.Fatalf("Body = %q, want question text", msg.Body)
	}
}

func TestParseTaskStartedQuestion(t *testing.T) {
	data := readTestdata(t, "task_started_question.json")
	msg, err := parseMessageBytes(data)
	if err != nil {
		t.Fatalf("ParseMessage() error = %v", err)
	}
	if msg.Event != "input_required" {
		t.Fatalf("Event = %q, want input_required", msg.Event)
	}
	if !strings.Contains(msg.Body, "Choose a color scheme") {
		t.Fatalf("Body = %q, want task description", msg.Body)
	}
}

func TestParseUnsupportedEvent(t *testing.T) {
	_, err := parseMessageBytes([]byte(`{"hook_event_name":"PreToolUse","tool_name":"Bash"}`))
	if err == nil {
		t.Fatal("expected error for unsupported PreToolUse tool")
	}
	_, err = parseMessageBytes([]byte(`{"hook_event_name":"TaskStarted","kind":"agent"}`))
	if err == nil {
		t.Fatal("expected error for non-question TaskStarted")
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
