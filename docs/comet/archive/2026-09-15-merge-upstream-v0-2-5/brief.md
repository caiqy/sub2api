# 目标

将 upstream 最新正式 tag `v0.2.5` 安全合并到当前已包含 `v0.2.4` 的 fork 基线，保留 upstream 行为与既有 fork 定制，并将本地版本闭合为 `0.2.5.1`。

# 范围

- 只以 `v0.2.5` 的 peeled commit `86f93c28ee34cc74b629dafb748bd5ac5ca8c5ea` 执行独立 `--no-ff` merge，不直接合并滚动的 `upstream/main`。
- 在该 tag 边界完成冲突与无文本冲突语义审查、受影响测试、完整本机质量门禁、构建和独立只读回归审计。
- 覆盖 OpenAI/Codex/WS/Images、OpenCode Go、Antigravity/Gemini、DeepSeek/Ollama/Grok、协议转换与计费调度，以及订阅/API Key 批量管理、认证、站点模式、监控、代理、支付和相关前端行为。
- 最终复核 fork 的 scheduler、sticky/fallback、DB recheck、请求体生命周期、审计与计费边界、运行时设置、插件和网关透传，以及管理员用量用户备注、用户 token 排名导出等近期定制。

# 非目标

- 不合并 `v0.2.5` 之后的 `upstream/main` 提交或后续 tag。
- 不主动修复 `v0.2.5` upstream 原样存在、且并非由本次合并、冲突处理或 fork 定制引入的问题。
- 不重构与本次合并无关的代码，不创建或推送 release tag，不触发 release workflow，不部署。

# 验收示例

- A1: Git 历史包含一个独立 `--no-ff` merge，其第二父为 `86f93c28ee34cc74b629dafb748bd5ac5ca8c5ea`；最终 HEAD 包含 `v0.2.5` 且不包含 post-`v0.2.5` upstream 提交。
- A2: 合并后完成冲突标记检查、受影响能力测试、后端默认与 unit 测试、后端 lint、前端 ESLint/i18n/单测/类型检查、前后端构建和独立只读回归审计；发现的 fork 合并回归先用失败测试和最小修复闭合。
- A3: `v0.2.5` 的 OpenAI/Codex WebSocket 执行作用域、池生命周期、抢占与心跳，原生 Images、Responses Lite namespace、模型映射、配额窗口和 OpenCode Go 能力成立。
- A4: `v0.2.5` 的 Antigravity/Gemini 模型与流式兼容、DeepSeek 模型校验与定价、Ollama Cloud 限流窗口、Grok media/Responses 及协议转换行为成立。
- A5: `v0.2.5` 的订阅与 API Key 批量操作、注册密码确认、站点类型开关、监控与 Ops、代理凭据、支付及相关前端行为成立。
- A6: fork 的 scheduler、sticky/fallback、DB recheck、请求体重放与清理、审计、每请求计费、运行时设置、插件边界、网关透传和前端定制语义保持成立。
- A7: 管理员用量记录继续展示仅管理员可见的用户备注，普通用户接口不泄露备注；用户 token 排名 Excel 导出的用户名、排序与样式定制保持成立。
- A8: `backend/cmd/server/VERSION` 为 `0.2.5.1`，且没有创建或推送 tag、触发 release workflow 或部署。
- A9: migrations、Ent/Wire 两次生成结果稳定；环境不支持的检查明确记录未覆盖范围；最终由新的只读 Verifier 对全部验收项给出独立结论。
- A10: `memory/context/upstream-merge-workflow.md` 仅在本轮产生经验证的新复用经验时增量更新，否则保持不变。

# 约束与不变量

- 只接受已从远端获取并核对 peeled commit 的正式 tag，不使用滚动的 `upstream/main`。
- tag 必须形成独立 merge 边界；测试通过不能替代能力级语义 review。
- 无法同时保留 upstream 行为和 fork 核心能力时暂停并请求用户决定。
- 任何生产代码回归修复必须先有针对该回归的失败测试。

# 决策

- 用户先将原工作区改动提交并推送到 `origin/main`，再使用隔离分支 `comet/merge-upstream-v0-2-5` 执行本 change，目标分支为 `main`。
- 远端核对结果显示 `v0.2.4` 之后只有一个新的正式三段式 tag `v0.2.5`，因此本轮单独合并该 tag，最终版本为 `0.2.5.1`。
- 沿用此前 change 的精确 tag、`--no-ff`、双方语义共存、逐 tag 门禁、专项能力审计和完整质量门禁口径。
- 本次只完成集成与审计，不发布或部署；只修复由 fork 合并引入的回归，不扩大为 upstream 既有问题修复。

# 待解决问题

- 无。

# 验证预期

- 合并前后检查 merge 父提交、tag 祖先关系、冲突标记和 post-tag 提交排除。
- 审查 upstream 改动与 fork 定制的重叠调用链和边界路径，先运行受影响测试，再运行完整质量门禁、构建和独立审计。
- 对 migrations、Ent 和 Wire 执行生成与稳定性检查；Docker/Testcontainers、race detector 等受环境限制的项目按真实结果记录。
