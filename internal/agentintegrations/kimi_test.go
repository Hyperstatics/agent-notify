package agentintegrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKimiIntegration_Name(t *testing.T) {
	k := NewKimiIntegration()
	if k.Name() != "Kimi" {
		t.Errorf("Name() = %q, want Kimi", k.Name())
	}
}

func TestKimiIntegration_SettingsPath(t *testing.T) {
	k := NewKimiIntegration()

	t.Run("user", func(t *testing.T) {
		t.Setenv("KIMI_CODE_HOME", "")
		path, err := k.SettingsPath("user")
		if err != nil {
			t.Fatal(err)
		}
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		expected := filepath.Join(home, ".kimi-code", "config.toml")
		if path != expected {
			t.Errorf("SettingsPath(user) = %q, want %q", path, expected)
		}
	})

	t.Run("KIMI_CODE_HOME", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("KIMI_CODE_HOME", dir)
		path, err := k.SettingsPath("user")
		if err != nil {
			t.Fatal(err)
		}
		expected := filepath.Join(dir, "config.toml")
		if path != expected {
			t.Errorf("SettingsPath(user) = %q, want %q", path, expected)
		}
	})

	t.Run("project", func(t *testing.T) {
		path, err := k.SettingsPath("project")
		if err != nil {
			t.Fatal(err)
		}
		expected := filepath.Join(".kimi-code", "config.toml")
		if path != expected {
			t.Errorf("SettingsPath(project) = %q, want %q", path, expected)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if _, err := k.SettingsPath("other"); err == nil {
			t.Fatal("expected error for invalid scope")
		}
	})
}

func TestKimiIntegration_Install(t *testing.T) {
	k := NewKimiIntegration()
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "config.toml")

	if err := k.Install(settingsPath, "/tmp/agent-notify"); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, event := range []string{"SessionStart", "PermissionRequest", "Notification", "Stop", "StopFailure", "PostToolUseFailure"} {
		if !strings.Contains(content, `event = "`+event+`"`) {
			t.Errorf("config should register %s, got:\n%s", event, content)
		}
	}
	if !strings.Contains(content, "handle-kimi-hook") {
		t.Errorf("config should contain handle-kimi-hook, got:\n%s", content)
	}

	installed, err := k.IsHookInstalled(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !installed {
		t.Fatal("IsHookInstalled() = false, want true")
	}
}

func TestKimiIntegration_Uninstall(t *testing.T) {
	k := NewKimiIntegration()
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "config.toml")

	if err := k.Install(settingsPath, "/tmp/agent-notify"); err != nil {
		t.Fatal(err)
	}
	if err := k.Uninstall(settingsPath); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	installed, err := k.IsHookInstalled(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if installed {
		t.Fatal("IsHookInstalled() = true after uninstall, want false")
	}
}

func TestKimiIntegration_DetectInstalled(t *testing.T) {
	k := NewKimiIntegration()
	_ = k.DetectInstalled()
}
