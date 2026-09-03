package codexhooks

import (
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// readCodexApprovalsReviewer 读取 ~/.codex/config.toml 的 approvals_reviewer。
// 测试可覆盖。读失败时返回空串，表示未知、保持通知。
var readCodexApprovalsReviewer = func() string {
	path := codexConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg struct {
		ApprovalsReviewer string `toml:"approvals_reviewer"`
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	return strings.TrimSpace(cfg.ApprovalsReviewer)
}

func codexConfigPath() string {
	if home := strings.TrimSpace(os.Getenv("CODEX_HOME")); home != "" {
		return filepath.Join(home, "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex", "config.toml")
}

func shouldSkipPermissionNotify(p payload) (bool, string) {
	if p.RequiresUserInput != nil && !*p.RequiresUserInput {
		return true, "skip permission: requires_user_input=false"
	}
	reviewer := effectiveApprovalsReviewer(p)
	if isAutomatedReviewer(reviewer) {
		return true, "skip permission: approvals_reviewer=" + reviewer
	}
	return false, ""
}

func effectiveApprovalsReviewer(p payload) string {
	for _, v := range []string{p.ApprovalsReviewer, p.ApprovalRoute, p.ApprovalContext.ApprovalsReviewer} {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return readCodexApprovalsReviewer()
}

func isAutomatedReviewer(reviewer string) bool {
	switch strings.ToLower(strings.TrimSpace(reviewer)) {
	case "auto_review", "guardian_subagent", "guardian":
		return true
	default:
		return false
	}
}
