package kimihooks

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/hellolib/agent-notify/internal/common"
)

// hookCommandMarker 用于识别本插件写入的 Kimi hook。
const hookCommandMarker = "handle-kimi-hook"

// managedEvents 是本插件托管的 Kimi Code 事件列表。
// Kimi Code 把 hook 写在 ~/.kimi-code/config.toml 的 [[hooks]] 数组里，
// 事件名为 PascalCase；stdin 的 hook_event_name 与之对应。
// PermissionRequest 是独立事件；Notification 覆盖授权/等待输入/后台任务。
// StopFailure / PostToolUseFailure 映射为 run_failed。
var managedEvents = []string{
	"SessionStart",
	"PermissionRequest",
	"Notification",
	"Stop",
	"StopFailure",
	"PostToolUseFailure",
}

type hookEntry struct {
	Event   string `toml:"event"`
	Matcher string `toml:"matcher,omitempty"`
	Command string `toml:"command"`
	Timeout int    `toml:"timeout,omitempty"`
}

type kimiTOML struct {
	Hooks []hookEntry `toml:"hooks"`
}

// BuildHookSettings 生成写入 Kimi config.toml 的 [[hooks]] 文本，供 print-hooks 使用。
func BuildHookSettings(binaryPath string) string {
	command := buildHookCommand(binaryPath)
	var b strings.Builder
	for i, event := range managedEvents {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(formatHookBlock(event, command))
	}
	return b.String()
}

func buildHookCommand(binaryPath string) string {
	binaryPath = common.ResolveBinaryPath(binaryPath)
	return common.QuotePathForShell(binaryPath) + " " + hookCommandMarker
}

func formatHookBlock(event, command string) string {
	return fmt.Sprintf("[[hooks]]\nevent = %s\ncommand = %s\n", strconv.Quote(event), strconv.Quote(command))
}

// Install 以增量方式把 agent-notify 的 [[hooks]] 追加到 Kimi config.toml。
// 已存在 handle-kimi-hook 的事件跳过；用户自己的 hook 原样保留。
// 不做整文件 parse→marshal 往返，以免丢掉注释与排版。
func Install(path string, binaryPath string) error {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	command := buildHookCommand(binaryPath)
	have := managedEventsPresent(data)

	var missing []string
	for _, event := range managedEvents {
		if !have[event] {
			missing = append(missing, event)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	text := stripEmptyHooksArray(string(data))
	if len(text) > 0 && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if len(strings.TrimSpace(text)) > 0 {
		text += "\n"
	}
	for _, event := range missing {
		text += formatHookBlock(event, command)
		text += "\n"
	}

	perm := os.FileMode(0o600)
	if st, statErr := os.Stat(path); statErr == nil {
		perm = st.Mode().Perm()
	}
	return common.WriteFileAtomicWithBackup(path, []byte(text), perm)
}

func managedEventsPresent(data []byte) map[string]bool {
	have := map[string]bool{}
	if len(data) == 0 {
		return have
	}
	var parsed kimiTOML
	if err := toml.Unmarshal(data, &parsed); err != nil {
		return have
	}
	for _, h := range parsed.Hooks {
		if strings.Contains(h.Command, hookCommandMarker) {
			have[h.Event] = true
		}
	}
	return have
}

var emptyHooksArrayRe = regexp.MustCompile(`(?m)^\s*hooks\s*=\s*\[\s*\]\s*(#.*)?\n?`)

// stripEmptyHooksArray 去掉 `hooks = []`。该空数组与随后追加的 [[hooks]]
// 共用 hooks 键，会让 Kimi 的 TOML 加载失败。
func stripEmptyHooksArray(text string) string {
	return emptyHooksArrayRe.ReplaceAllString(text, "")
}

var hooksTableHeaderRe = regexp.MustCompile(`^\s*\[\[hooks\]\]\s*(#.*)?$`)

// IsInstalled 检查 config.toml 中是否已挂载 agent-notify 的 Kimi hook。
func IsInstalled(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	have := managedEventsPresent(data)
	for _, event := range managedEvents {
		if have[event] {
			return true, nil
		}
	}
	return false, nil
}

// Uninstall 仅移除 command 含 handle-kimi-hook 的 [[hooks]] 块。
// 用户自己的 hook 与其余配置原样保留。文件不存在时是 no-op。
func Uninstall(path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	next := removeManagedHookBlocks(string(data))
	if next == string(data) {
		return nil
	}
	perm := os.FileMode(0o600)
	if st, statErr := os.Stat(path); statErr == nil {
		perm = st.Mode().Perm()
	}
	return common.WriteFileAtomicWithBackup(path, []byte(next), perm)
}

func removeManagedHookBlocks(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); {
		if hooksTableHeaderRe.MatchString(lines[i]) {
			block, next := collectTableBlock(lines, i)
			if blockContainsMarker(block, hookCommandMarker) {
				i = next
				continue
			}
			out = append(out, block...)
			i = next
			continue
		}
		out = append(out, lines[i])
		i++
	}
	return strings.Join(out, "\n")
}

func collectTableBlock(lines []string, start int) ([]string, int) {
	next := len(lines)
	for i := start + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") {
			next = i
			break
		}
	}
	return lines[start:next], next
}

func blockContainsMarker(block []string, marker string) bool {
	for _, line := range block {
		if strings.Contains(line, marker) {
			return true
		}
	}
	return false
}
