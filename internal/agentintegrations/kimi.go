package agentintegrations

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/kimihooks"
)

// KimiIntegration implements Integration for Moonshot Kimi Code CLI.
type KimiIntegration struct{}

// NewKimiIntegration creates a new Kimi integration.
func NewKimiIntegration() *KimiIntegration {
	return &KimiIntegration{}
}

// Name returns the display name for Kimi.
func (k *KimiIntegration) Name() string {
	return "Kimi"
}

// DetectInstalled checks if the kimi CLI is installed, or if ~/.kimi-code exists.
func (k *KimiIntegration) DetectInstalled() bool {
	if _, err := exec.LookPath("kimi"); err == nil {
		return true
	}
	home, err := kimiHomeDir()
	if err != nil {
		return false
	}
	info, err := os.Stat(home)
	return err == nil && info.IsDir()
}

// SettingsPath returns the path to Kimi Code's user config.toml.
// Kimi Code currently has no project-level config file.
func (k *KimiIntegration) SettingsPath(scope string) (string, error) {
	switch scope {
	case "user":
		home, err := kimiHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "config.toml"), nil
	case "project":
		// Kimi Code currently only reads the user config. Return a repo-local
		// path so clean/uninstall can still scan it (no-op if the file is absent).
		return filepath.Join(".kimi-code", "config.toml"), nil
	default:
		return "", fmt.Errorf("unsupported scope: %s", scope)
	}
}

// Install writes [[hooks]] entries for SessionStart / PermissionRequest /
// Notification / Stop / StopFailure / PostToolUseFailure into config.toml.
func (k *KimiIntegration) Install(settingsPath, binaryPath string) error {
	return kimihooks.Install(settingsPath, common.ResolveBinaryPath(binaryPath))
}

// Uninstall removes only agent-notify hook entries from the Kimi config file.
func (k *KimiIntegration) Uninstall(settingsPath string) error {
	return kimihooks.Uninstall(settingsPath)
}

// IsHookInstalled checks whether agent-notify hooks are registered for Kimi.
func (k *KimiIntegration) IsHookInstalled(settingsPath string) (bool, error) {
	return kimihooks.IsInstalled(settingsPath)
}

func kimiHomeDir() (string, error) {
	if v := strings.TrimSpace(os.Getenv("KIMI_CODE_HOME")); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kimi-code"), nil
}
