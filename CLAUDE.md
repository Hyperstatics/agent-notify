# Agent Notify

## 项目概况

Agent Notify 监听 Claude Code、Codex、ZCode、Grok、Kimi、Droid、OpenCode 的生命周期事件，将通知发送到桌面和手机渠道。它以 Go binary 发布，并由 `npx agent-notify` 下载对应平台的 release binary 到 `~/.agent-notify/`。

## 常用命令

```bash
make build
make test
go test ./internal/notify -run TestName
make lint
make fmt
make vet
cd npx && npm test
make release VERSION=v0.x.y
```

`make release` 会打 tag、push、触发 GitHub Actions，并发布 npm；只在用户明确授权发布时执行。release workflow 为 6 个目标构建，并把 `thirdparty/helper/` 的平台 helper 打入归档。

## 架构与关键约束

- TUI（`agent-notify`、`init`、`doctor`、`test`）负责配置向导、写入 `~/.agent-notify/config.yaml` 和安装各 agent 的 hooks。
- Headless hook handler（`handle-*-hook`）从 stdin 读取事件 JSON，转换为 `permission_required`、`input_required`、`run_completed`、`run_failed`、`session_start` 等标准事件，再由 `internal/agenthooks.Dispatch` 解析来源和 workspace、处理 `session_start` 的 focus side effect（不通知）、去重并发送。handler 不得阻塞或 prompt；失败写入 state log 并返回 nil，不能破坏宿主 agent。
- 新增 agent 要有 `event.go`、`settings.go`、`handler.go` 形态的集成包，并接入 `internal/agentintegrations` 与 `internal/agenthooks/dispatch.go:buildSenders`。Kimi 使用 TOML `[[hooks]]`，安装/卸载必须保留注释和其他 key。
- `internal/config.Load` 将 YAML 解入 `Default()`，缺省 key 自动继承默认值；新增字段或 agent 不要再手工补 `Load`。默认值含义改变时，递增 `currentConfigVersion` 并在 `Load` 增加版本迁移。新安装默认关闭所有 agent/channel，由向导启用。
- `internal/state` 保存去重 reservation、按 session ID 的 focus cache 和 append-only log。点击通知回到原终端/IDE；macOS 精度由 `AGENT_NOTIFY_FOCUS_PRECISION`（`app` 默认、`window`）控制，窗口快照在 `session_start` 捕获。
- OpenCode 没有原生 hook，使用 `internal/opencodehooks/plugin_embed.js` 注册到用户 `~/.config/opencode/opencode.json` 或项目 `./opencode.json`，分发到 `~/.agent-notify/opencode-plugin.js`。只依赖公开 Session Events；`session.next.*` 不会发布。`question.asked` 的 choice form 可能让 session 保持 busy，必须保留订阅；`TestPluginSubscriptionMatchesParser` 保证 allowlist 与 parser 同步。
- `internal/notify` 按渠道实现 `Sender`，dispatcher 按 agent/session/event/content-hash/sender 去重；`internal/i18n` 只负责 TUI 文本，通知格式在 `internal/notify/format.go`。`npx/` 是除 `tar` 外无运行时依赖的 CommonJS launcher，`npm run prepack` 同步仓库 README。

## 修改时注意

- ZCode hook schema 严格：未知 event name 会静默丢弃整个 hooks config，修改 `internal/zcodehooks/settings.go` 要特别谨慎。
- 代码注释以中文为主，跟随周边文件风格。`docs/` 已 gitignore，是本地设计归档，不要在提交代码或文档中引用它。
