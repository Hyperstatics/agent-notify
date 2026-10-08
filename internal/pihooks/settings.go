package pihooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/hellolib/agent-notify/internal/common"
)

const extensionMarker = "agent-notify:pi-extension"

var binaryConstRe = regexp.MustCompile(`(?m)^const BINARY = (".*");\r?$`)

// Install writes the bundled extension to Pi's auto-discovered extensions directory.
func Install(extensionPath, binaryPath string) error {
	if data, err := os.ReadFile(extensionPath); err == nil && !bytes.Contains(data, []byte(extensionMarker)) {
		return fmt.Errorf("refusing to overwrite a Pi extension without the %s marker: %s", extensionMarker, extensionPath)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return WriteExtensionFile(extensionPath, binaryPath)
}

// IsInstalled reports whether extensionPath contains the Agent Notify extension.
func IsInstalled(extensionPath string) (bool, error) {
	data, err := os.ReadFile(extensionPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return bytes.Contains(data, []byte(extensionMarker)), nil
}

// Uninstall removes the Agent Notify extension and leaves unrelated extensions untouched.
func Uninstall(extensionPath string) error {
	installed, err := IsInstalled(extensionPath)
	if err != nil || !installed {
		return err
	}
	return os.Remove(extensionPath)
}

func renderPlugin(binaryPath string) []byte {
	encodedBinary, err := json.Marshal(binaryPath)
	if err != nil {
		return nil
	}
	return []byte(strings.ReplaceAll(PluginJS, "__AGENT_NOTIFY_BINARY__", string(encodedBinary)))
}

// WriteExtensionFile writes the bundled Pi extension with its binary path embedded.
func WriteExtensionFile(extensionPath, binaryPath string) error {
	if err := os.MkdirAll(filepath.Dir(extensionPath), 0o755); err != nil {
		return err
	}
	return common.WriteFileAtomic(extensionPath, renderPlugin(binaryPath), 0o644)
}

// RefreshExtensionIfStale updates an installed extension while preserving its binary path.
func RefreshExtensionIfStale(extensionPath string) (bool, error) {
	actual, err := os.ReadFile(extensionPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if !bytes.Contains(actual, []byte(extensionMarker)) {
		return false, nil
	}
	match := binaryConstRe.FindSubmatch(actual)
	if match == nil {
		return false, nil
	}
	binaryPath, err := strconv.Unquote(string(match[1]))
	if err != nil {
		return false, nil
	}
	if bytes.Equal(actual, renderPlugin(binaryPath)) {
		return false, nil
	}
	return true, WriteExtensionFile(extensionPath, binaryPath)
}
