package kimihooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildHookSettings_RegistersManagedEvents(t *testing.T) {
	got := BuildHookSettings("/tmp/agent-notify")
	for _, hook := range managedHooks {
		if !strings.Contains(got, "event = \""+hook.Event+"\"") {
			t.Fatalf("missing event %s in:\n%s", hook.Event, got)
		}
		if hook.Matcher != "" && !strings.Contains(got, "matcher = \""+hook.Matcher+"\"") {
			t.Fatalf("missing matcher %s in:\n%s", hook.Matcher, got)
		}
	}
	if !strings.Contains(got, "handle-kimi-hook") {
		t.Fatalf("missing handle-kimi-hook in:\n%s", got)
	}
}

func TestInstall_CreatesHooksInNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := Install(path, "/tmp/agent-notify"); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, hook := range managedHooks {
		if !strings.Contains(content, "event = \""+hook.Event+"\"") {
			t.Fatalf("missing event %s in:\n%s", hook.Event, content)
		}
		if hook.Matcher != "" && !strings.Contains(content, "matcher = \""+hook.Matcher+"\"") {
			t.Fatalf("missing matcher %s in:\n%s", hook.Matcher, content)
		}
	}

	installed, err := IsInstalled(path)
	if err != nil {
		t.Fatal(err)
	}
	if !installed {
		t.Fatal("IsInstalled() = false, want true")
	}
}

func TestInstall_PreservesUserHooksAndComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	existing := `# user config
default_model = "kimi-code/k3-256k"

[[hooks]]
event = "PreToolUse"
matcher = "Bash"
command = "echo user-hook"
`
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Install(path, "/tmp/agent-notify"); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "# user config") {
		t.Fatal("lost user comment")
	}
	if !strings.Contains(content, `default_model = "kimi-code/k3-256k"`) {
		t.Fatal("lost user setting")
	}
	if !strings.Contains(content, "echo user-hook") {
		t.Fatal("lost user hook")
	}
	if !strings.Contains(content, "handle-kimi-hook") {
		t.Fatal("missing agent-notify hook")
	}
}

func TestInstall_ReplacesEmptyHooksArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	existing := "default_model = \"k3\"\nhooks = []\n"
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Install(path, "/tmp/agent-notify"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Contains(content, "hooks = []") {
		t.Fatalf("empty hooks array should be removed:\n%s", content)
	}
	if !strings.Contains(content, "[[hooks]]") {
		t.Fatal("expected [[hooks]] tables")
	}
}

func TestInstall_AddsChoiceHooksToExistingInstall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	existing := `default_model = "k3"

[[hooks]]
event = "Stop"
command = "\"/tmp/agent-notify\" handle-kimi-hook"
`
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Install(path, "/tmp/agent-notify"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `matcher = "AskUserQuestion"`) {
		t.Fatalf("missing AskUserQuestion matcher:\n%s", content)
	}
	if !strings.Contains(content, `matcher = "question"`) {
		t.Fatalf("missing question matcher:\n%s", content)
	}
	if strings.Count(content, `event = "Stop"`) != 1 {
		t.Fatalf("Stop hook should stay unique:\n%s", content)
	}
}

func TestInstall_IsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := Install(path, "/tmp/agent-notify"); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Install(path, "/tmp/agent-notify"); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("second install changed file:\n%s\n---\n%s", first, second)
	}
}

func TestUninstall_RemovesOnlyManagedHooks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	existing := `default_model = "k3"

[[hooks]]
event = "PreToolUse"
command = "echo user-hook"
`
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Install(path, "/tmp/agent-notify"); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(path); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Contains(content, "handle-kimi-hook") {
		t.Fatalf("managed hook still present:\n%s", content)
	}
	if !strings.Contains(content, "echo user-hook") {
		t.Fatal("user hook was removed")
	}
	if !strings.Contains(content, `default_model = "k3"`) {
		t.Fatal("user setting was removed")
	}

	installed, err := IsInstalled(path)
	if err != nil {
		t.Fatal(err)
	}
	if installed {
		t.Fatal("IsInstalled() = true after uninstall")
	}
}

func TestUninstall_MissingFile(t *testing.T) {
	if err := Uninstall(filepath.Join(t.TempDir(), "missing.toml")); err != nil {
		t.Fatalf("Uninstall missing file: %v", err)
	}
}
