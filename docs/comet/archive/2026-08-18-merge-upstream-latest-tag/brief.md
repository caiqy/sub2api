# Outcome

将当前项目的 `main` 与 upstream 正式 release tag `v0.1.177` 合并，保留上游修复和本地定制能力，并形成可验证的集成结果。

# Scope

- 从干净的当前 `main` 基线在临时集成分支执行合并，不直接改写 `main`。
- 仅合入 `upstream` 的正式 tag `v0.1.177`（peeled SHA `073e92d17178a1ccdb0a27017f572f10c9c7ab62`），不合入该 tag 之后的 `upstream/main`。
- 处理冲突和无文本冲突的语义兼容，保留本地能力；必要时补充最小回归测试和兼容修复。
- 将服务端 `VERSION` 由当前本地 `0.1.176.4` 对齐为下一个本地派生版本 `0.1.177.1`。

# Non-goals

- 不创建或推送 Git tag，不触发 release workflow，不部署。
- 不合入 `v0.1.177` 之后的任何 upstream 提交。
- 不在未获得明确授权时移除无法与上游共存的本地能力。

# Acceptance examples

- A1：结果历史包含一个 `--no-ff` 合并节点，其第一父为合并前的集成分支 HEAD，第二父为 `v0.1.177` 的 peeled SHA；`v0.1.177` 是结果 HEAD 的祖先。
- A2：合并结果包含 `v0.1.177` 的 17 个提交和 76 个文件更新，不包含该 tag 之后的 `upstream/main` 提交。
- A3：合并前本地质量门禁通过；被上游触及且缺少断言的高风险本地能力有最小回归测试保护。
- A4：冲突和语义审查同时保留上游的 Codex turn-state、fingerprint/compaction、分组用量日汇总与 migration 更新，以及本地 scheduler、sticky/fallback、请求体重放、审计、计费和前端定制语义。
- A5：每个合并阶段通过冲突标记检查、受影响能力测试、Ent/Wire 两次生成稳定性检查、`make test` 和 `make build`。
- A6：最终通过后端默认与 unit 测试、后端 lint、前端 ESLint、前端单测、前端类型检查和构建验证，并完成受影响能力专项审查。
- A7：`backend/cmd/server/VERSION` 为 `0.1.177.1`；不创建或推送 release tag。
- A8：若 Docker-backed integration 未执行，其命令、原因和受影响的 migration/repository 契约在验收报告中明确列为残余风险，不得记为通过。

# Constraints and invariants

- 当前 `main` 相对 `v0.1.177` 本地领先 1689 个提交、上游领先 17 个提交；合并必须隔离在临时分支中进行。
- 上游 `v0.1.177` 以 `v0.1.176`（`e803e3851c0a7e222cfadeafad7b8636ab959d11`）为祖先，包含 OpenAI compaction/turn-state/fingerprint 与分组用量日汇总、时区 migration 等更新。
- 出现无法同时保留的语义时暂停并请求用户选择；不得静默采用上游或删除本地能力。
- Docker 当前未安装；沿用既有仅本机验证惯例，将 Docker-backed integration 作为残余风险记录，绝不记为通过。

# Decisions

- 目标选择 `upstream` 的最新正式 release tag `v0.1.177`，而非尚未发布新 tag 的 `upstream/main`。
- 沿用既有上游合并惯例：隔离分支、精确 `--no-ff` merge、保留双方语义、完整门禁和本地派生 VERSION。
- 沿用既有 Docker 不可用时的仅本机验证配置；未执行的 migration/repository integration 将在验收报告中保留为残余风险。
- 该 change 仅处理集成与验证；发布由独立的用户请求和 release 流程处理。

# Open questions

- 无。

# Verification expectations

- 合并前后执行既有后端、前端和生成代码门禁，审查新增 migration 的新库与已有本地记录升级路径。
- 重点复核 OpenAI compaction、turn-state、fingerprint、请求体生命周期、计费、scheduler/sticky/fallback、分组用量日汇总、时区、管理端分组与账户视图的交互。
