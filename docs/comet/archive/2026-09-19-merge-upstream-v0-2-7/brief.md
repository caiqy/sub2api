# 目标

将 upstream 正式 tag `v0.2.7` 安全合并到当前已包含 `v0.2.5.2` 的 fork 基线，保留 upstream 行为与既有 fork 定制，并将本地版本闭合为 `0.2.7.1`。

# 范围

- 仅以 `v0.2.7` 的 peeled commit `aea725f2ea644d5592d0bbb1d63b607efa7e200a` 执行一个独立 `--no-ff` merge，不直接合并滚动的 `upstream/main`，也不包含该 tag 之后的 upstream 提交。
- 在 tag 边界完成冲突与无文本冲突语义审查、受影响测试、完整本机质量门禁、构建、生成稳定性检查和独立只读回归审计。
- 覆盖 `v0.2.7` upstream 新增或修改的主要模型、协议、平台、配额、计费、管理端和前端行为，并复核 fork 的 scheduler、sticky/fallback、DB recheck、请求体重放与清理、审计、每请求计费、运行时设置、插件、网关透传、管理员备注和用户 token 排名导出定制。

# 非目标

- 不合并 `v0.2.7` 之后的 upstream 提交或后续 tag。
- 不主动修复在 `v0.2.7` peeled commit 中原样存在、且并非由本次合并、冲突处理或 fork 定制引入的问题。
- 不重构与本次合并无关的代码，不创建或推送 release tag，不触发 release workflow，不部署。

# 验收示例

- A1: Git 历史包含一个独立 `--no-ff` merge，其第一父为合并前 fork HEAD，第二父为 `aea725f2ea644d5592d0bbb1d63b607efa7e200a`；`v0.2.7` 是结果 HEAD 的祖先，且不包含 post-`v0.2.7` upstream 提交。
- A2: 合并后完成冲突标记检查、受影响能力测试、后端默认与 unit 测试、后端 lint、前端 ESLint/i18n/单测/类型检查、前后端构建、migrations 与 Ent/Wire 两轮生成稳定性检查和独立只读回归审计。
- A3: `v0.2.7` upstream 主要能力由 upstream 测试、fork 回归测试和能力级源码审查覆盖，未被 fork 定制破坏。
- A4: fork 的 scheduler、sticky/fallback、DB recheck、请求体生命周期、审计计费、运行时设置、插件、网关透传和前端定制语义保持成立；管理员备注仍只对管理员可见，用户 token 排名导出保持用户名、排序和样式定制。
- A5: 发现由合并、冲突处理或 fork 定制引入的回归时，先以失败测试复现，再以最小修复闭合并重跑受影响及完整门禁；upstream 原样问题只记录证据，不扩大范围。
- A6: `backend/cmd/server/VERSION` 为 `0.2.7.1`，且没有创建或推送 tag、触发 release workflow 或部署。
- A7: Docker/Testcontainers、race detector 或其他环境不支持的检查明确记录未运行范围，不将其记为通过。
- A8: `memory/context/upstream-merge-workflow.md` 仅在本轮产生经验证的新复用经验时增量更新，否则保持不变并在交接中说明。
- A9: 最终由新的独立只读 Verifier 对全部验收项给出独立结论。

# 约束与不变量

- 只接受已从远端获取并核对 peeled commit 的正式 tag，不使用滚动的 `upstream/main`。
- tag 必须形成独立 merge 边界；测试通过不能替代能力级语义 review。
- 无法同时保留 upstream 行为和 fork 核心能力时暂停并请求用户决定。
- 任何生产代码回归修复必须先有针对该回归的失败测试。

# 决策

- 使用当前干净的 `main` 创建隔离分支 `comet/merge-upstream-v0-2-7`，目标分支为 `main`。
- 沿用上一 `merge-upstream-v0-2-5` change 的精确 tag、`--no-ff`、双方语义共存、逐 tag 门禁、专项能力审计和完整质量门禁口径。
- 本次只完成集成与审计，不发布或部署；版本目标为 `0.2.7.1`。

# 待解决问题

- 无。

# 验证预期

- 合并前后检查 merge 父提交、tag 祖先关系、冲突标记和 post-tag 提交排除。
- 审查 upstream 改动与 fork 定制的重叠调用链和边界路径，先运行受影响测试，再运行完整质量门禁、构建、生成稳定性检查和独立审计。
- 对 Docker/Testcontainers、race detector 等环境限制按真实结果记录，不将未运行项目记为通过。
