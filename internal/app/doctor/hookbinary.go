package doctor

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/hellolib/agent-notify/internal/common"
)

// hookBinaryPath 从 agent 的配置文件里取出已注册 hook 命令指向的二进制路径。
// JSON hooks（Claude/Codex/Grok/Droid/ZCode/OpenCode）递归遍历 command 字段；
// TOML hooks（Kimi [[hooks]]）在 JSON 解析失败后按行提取 command = "..."。
// 找不到返回空串。
func hookBinaryPath(settingsPath, marker string) string {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return ""
	}
	var root any
	if err := json.Unmarshal(data, &root); err == nil {
		if cmd := findCommandWithMarker(root, marker); cmd != "" {
			return common.BinaryPathFromHookCommand(cmd, marker)
		}
	}
	text := string(data)
	if strings.Contains(text, "agent-notify:pi-extension") {
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "const BINARY =") {
				continue
			}
			raw := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "const BINARY ="), ";"))
			if binaryPath, err := strconv.Unquote(raw); err == nil {
				return binaryPath
			}
		}
	}
	return hookBinaryPathFromText(text, marker)
}

func hookBinaryPathFromText(text, marker string) string {
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, marker) {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if key != "command" {
			continue
		}
		raw := strings.TrimSpace(parts[1])
		val, err := strconv.Unquote(raw)
		if err != nil {
			continue
		}
		if strings.Contains(val, marker) {
			return common.BinaryPathFromHookCommand(val, marker)
		}
	}
	return ""
}

func findCommandWithMarker(node any, marker string) string {
	switch v := node.(type) {
	case map[string]any:
		if cmd, ok := v["command"].(string); ok && strings.Contains(cmd, marker) {
			return cmd
		}
		for _, child := range v {
			if found := findCommandWithMarker(child, marker); found != "" {
				return found
			}
		}
	case []any:
		for _, child := range v {
			if found := findCommandWithMarker(child, marker); found != "" {
				return found
			}
		}
	}
	return ""
}

// hookBinaryMissing 判断已注册的 hook 命令是否指向一个不存在的二进制。
// 典型场景:用户先用本地构建安装,后改用 npx,旧路径已被删除(issue #34)。
// 无法判定时(读不到配置、解析不出路径、裸命令名依赖 PATH)返回 false,不误报。
func hookBinaryMissing(settingsPath, marker string) bool {
	binPath := hookBinaryPath(settingsPath, marker)
	if binPath == "" {
		return false
	}
	if !strings.ContainsAny(binPath, `/\`) {
		return false
	}
	_, err := os.Stat(binPath)
	return os.IsNotExist(err)
}
