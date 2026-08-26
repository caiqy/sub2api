# Outcome

将 upstream 正式 tag `v0.1.183` 安全合并到当前 fork 基线，保留 upstream 行为与既有 fork 定制，并将本地版本闭合为 `0.1.183.1`。

# Scope

- 仅以 `v0.1.183` 的 peeled commit `e8cb019fabf8b55199436229044cbf9aa7a82564` 执行一次 `--no-ff` merge。
- 集成该 tag 的 Responses custom tool-call ID 往返、邮箱换绑别名和并发守卫、OpenAI sticky 容量溢出、OAuth 429 配额暂停、Codex session affinity、channel monitor 聚合、Kimi 并发 403、Antigravity token clamp 等行为。
- 处理冲突、补齐必要的最小回归测试，并完成拓扑、生成稳定性、后端与前端质量门禁和独立只读审计。

# Non-goals

- 不合并 `v0.1.183` 之后的 `upstream/main` 提交或任何其他 tag。
- 不重构与本次合并无关的 fork 代码，不改变既有 scheduler、计费、审计和网关默认行为。
- 不创建或推送 release tag，不触发 release workflow，不部署。

# Acceptance examples

- A1: 集成提交是以当前基线为第一父、`e8cb019fabf8b55199436229044cbf9aa7a82564` 为第二父的 `--no-ff` merge，且结果不含该 tag 后的 upstream 提交。
- A2: Responses custom tool-call 的客户端 ID 与上游 function-call ID 在 HTTP/WS 往返中保持类型合法且可还原。
- A3: upstream 的邮箱别名换绑并发保护、OpenAI sticky 容量处理和 OAuth 429 配额暂停、Codex session affinity、Kimi/Antigravity 兼容与 monitor 聚合修复均保留并覆盖受影响测试。
- A4: fork 的 scheduler、sticky/fallback、DB recheck、请求体重放和清理、审计、每请求计费、运行时设置、网关透传及前端定制语义保持成立。
- A5: `backend/cmd/server/VERSION` 为 `0.1.183.1`，且没有发布动作。
- A6: 受影响测试、后端默认与 unit 测试、lint、前端 ESLint/单测/类型检查、前后端构建以及两次 Ent/Wire 生成稳定性检查通过；Docker 不可用时明确记录未覆盖范围。
- A7: 新的只读 Verifier 对全部验收项给出独立结论。

# Constraints and invariants

- 只接受正式 tag，不使用滚动的 `upstream/main`。
- 无法同时保留 upstream 行为和 fork 核心能力时暂停并请求用户决定。
- 任何生产代码修复必须先有针对该回归的失败测试。

# Decisions

- 使用独立 worktree `comet/merge-upstream-v0-1-183`，目标分支为 `main`。
- 本次范围为单一正式 tag `v0.1.183`，不发布。

# Open questions

- [blocking] CONFIRM: 确认仅合并 `v0.1.183`、将版本更新为 `0.1.183.1`、保留 fork 核心能力、完成质量门禁和独立审计，且不发布或部署。

# Verification expectations

- 在 merge 前后检查冲突标记、merge 父提交、tag 祖先关系和 post-tag upstream 提交排除。
- 优先执行受影响的 apicompat、认证、调度、限流和网关测试；再执行完整质量门禁、两次 Ent/Wire 生成检查和构建。
- 记录 Docker/Testcontainers 与 race detector 等环境限制，不将未运行项目记为通过。
