# Outcome

将当前 fork 从已合入 upstream `v0.1.179` 的 `main` 基线，依次合并 upstream 正式 release tag `v0.1.180`、`v0.1.181`、`v0.1.182`，每版独立完成回归审计与验证，同时保留本地定制能力；在 `v0.1.182` 审计闭合后停止并等待用户再次授权。

# Scope

- 严格按 `v0.1.180 -> v0.1.181 -> v0.1.182` 顺序合并，每个 tag 形成一个 `--no-ff` merge commit。
- 每版处理文本冲突和无文本冲突的语义兼容，保留 upstream 新行为与本地 scheduler、sticky/fallback、请求体生命周期、审计、计费、运行时设置及前端定制。
- 每版在进入下一 tag 前执行受影响能力测试、完整本机质量门禁、构建和独立只读回归审计；审计问题先修复并复验。
- 最终将 `backend/cmd/server/VERSION` 对齐为本地派生版本 `0.1.182.1`。

# Non-goals

- 不合入 `v0.1.182` 之后的 `upstream/main` 提交或未获授权的后续 tag。
- 不创建或推送 Git tag，不推送分支，不触发 release workflow，不部署。
- 不在未获用户明确授权时移除无法与 upstream 共存的本地能力。

# Acceptance examples

- A1：结果历史依次包含 `v0.1.180`、`v0.1.181`、`v0.1.182` 三个独立 `--no-ff` merge 节点；各节点第二父分别为 `c40edb4070a9274e8c23f161b4ed552051b14698`、`3af5443b224823ae507a50c7b113aa50604409c8`、`5a7d469622911a6b1291a692376df5fa03f9ac2e`，且不包含 `v0.1.182` 之后的 upstream 提交。
- A2：每版合并后、下一版合并前，均完成冲突标记检查、受影响能力测试、后端默认与 unit 测试、后端 lint、前端 ESLint、前端单测、类型检查、前后端构建，并留下独立回归审计结论。
- A3：`v0.1.180` 的插件系统、Fast mode 计费、模型列表限制、Grok/OpenAI 兼容和模型广场更新可用，且不破坏 fork 的网关、调度、审计、计费和前端语义。
- A4：`v0.1.181` 的 Gemini tool schema 清理、Grok CLI User-Agent、Responses Lite `parallel_tool_calls` 和 rejected input status 清理可用，且相关 fork 定制仍受测试或源码审查保护。
- A5：`v0.1.182` 的 Responses Lite/WS tool-call 规范化与数值精度、OAuth 图片提示、Kimi K3 路由、Antigravity Sonnet 路由、Anthropic cache TTL 计费、OpenCode reset、monitor 平台归因和支付余额刷新可用，且相关 fork 定制仍受测试或源码审查保护。
- A6：Ent/Wire 生成结果稳定，`git diff --check` 通过，最终工作树只包含本 change 的正式产物和已提交集成结果。
- A7：最终 `backend/cmd/server/VERSION` 为 `0.1.182.1`，且未创建/推送 release tag、未触发发布或部署。
- A8：Docker/Testcontainers 未执行时，明确记录原因和 migration/repository 契约风险，不得将相关 integration 记为通过。
- A9：流程在 `v0.1.182` 审计闭合后停止；未合入 `v0.1.182` 之后的 upstream 提交或未获授权 tag。

# Constraints and invariants

- 起始 `HEAD` 为 `d9048b5bb7`，已包含 upstream `v0.1.179`，工作树在创建 change 前干净且与 `origin/main` 一致。
- 三枚获授权目标 tag 已确认线性递进；`v0.1.182` 由用户在前两版审计闭合后追加授权。
- 本次 Comet change 绑定当前 `main` 工作区；所有 merge 和修复必须保持可追溯提交，不得改写已有历史。
- 出现双方语义无法同时保留的真实冲突时暂停并请求用户决定，不得静默采用 ours/theirs。
- 本机无 Docker；其余可执行本机门禁必须完成。

# Decisions

- 沿用此前 upstream 合并任务的精确 tag、`--no-ff`、双方语义共存、专项能力审计和完整质量门禁口径。
- 将三个获授权 tag 作为同一 Comet change 中的顺序阶段处理；前两版已逐版闭合，`v0.1.182` 继续保持相同门禁与审计口径。
- 本次只完成集成与审计，不执行发布动作。

# Open questions

- 无。

# Verification expectations

- 每版根据实际变更运行定向 Go/Vitest 测试，并执行 `make test`、`make build`；审计覆盖该 tag 的 upstream 行为和与 fork 定制重叠的调用链。
- 最终复核 merge parent/祖先关系、排除 post-tag 提交、版本文件、Ent/Wire 生成稳定性和发布非目标。
- Verifier 必须逐项给出验收结论；Docker-backed integration 保留为明确残余风险。
