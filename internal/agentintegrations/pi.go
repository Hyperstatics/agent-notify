package agentintegrations

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hellolib/agent-notify/internal/common"
	"github.com/hellolib/agent-notify/internal/pihooks"
)

// PiIntegration implements Integration for Pi.
type PiIntegration struct{}

func NewPiIntegration() *PiIntegration { return &PiIntegration{} }

func (p *PiIntegration) Name() string { return "Pi" }

func (p *PiIntegration) DetectInstalled() bool {
	if _, err := exec.LookPath("pi"); err == nil {
		return true
	}
	agentDir, err := piAgentDir()
	if err != nil {
		return false
	}
	info, err := os.Stat(agentDir)
	return err == nil && info.IsDir()
}

// SettingsPath returns Pi's auto-discovered extension file for the selected scope.
func (p *PiIntegration) SettingsPath(scope string) (string, error) {
	switch scope {
	case "user":
		agentDir, err := piAgentDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(agentDir, "extensions", "agent-notify.js"), nil
	case "project":
		return filepath.Join(".pi", "extensions", "agent-notify.js"), nil
	default:
		return "", fmt.Errorf("unsupported scope: %s", scope)
	}
}

func (p *PiIntegration) Install(settingsPath, binaryPath string) error {
	return pihooks.Install(settingsPath, common.ResolveBinaryPath(binaryPath))
}

func (p *PiIntegration) Uninstall(settingsPath string) error {
	return pihooks.Uninstall(settingsPath)
}

func (p *PiIntegration) IsHookInstalled(settingsPath string) (bool, error) {
	return pihooks.IsInstalled(settingsPath)
}

func piAgentDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("PI_CODING_AGENT_DIR")); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pi", "agent"), nil
}
