# 目标

将 upstream 最新正式 tag `v0.2.8` 安全集成到基于 `v0.2.7` 的 fork，在保留双方行为的前提下将本地版本闭合为 `0.2.8.1`。

# 范围

- 以远端核对过的 `v0.2.8` peeled commit `fd80b08c90b55edcad5b00171b53f08721d30da1` 为唯一上游合并父，在独立分支执行一个 `--no-ff` merge；不合并滚动的 `upstream/main`。
- 处理文本冲突及无文本冲突的语义重叠，保留上游新增功能、修复与 fork 现有定制。
- 运行受影响测试、本机可执行的完整质量门禁与构建，完成独立只读回归审计。

# 非目标

- 不合并 `v0.2.8` 之后的 upstream 提交或后续 tag。
- 不主动修复 tag 中原样存在、且未受本次合并或 fork 定制影响的上游问题；不做无关重构。
- 不创建或推送 release tag，不触发发布工作流，不部署；是否将 change 本地合回 `main` 在归档阶段另行选择。

# 验收示例

- A1: Git 历史存在独立 `--no-ff` merge，其第一父为合并前 fork HEAD `0c03d2ef2f9f63d164d59ec459f2d5387196ec57`，第二父精确为 `fd80b08c90b55edcad5b00171b53f08721d30da1`；结果包含 `v0.2.8`，不包含该 tag 之后的上游提交。
- A2: 冲突标记、无文本冲突的重叠调用链及双方新增行为经过能力级审查；`v0.2.8` 的模型、OpenCode Go 用量、推理计费、备份、内容审计、插件和管理端/前端等主要变化有对应 upstream 测试或专项验证。
- A3: fork 的 scheduler、sticky/fallback、DB recheck、请求体重放与清理、审计与逐请求计费、运行时设置、插件、网关透传和前端定制保持可用；管理员备注仅管理员可见，用户 token 排名导出保留用户名、排序及样式，Codex 打票及 OpenCode 会话定制不丢失。
- A4: 由合并、冲突处理或 fork 定制引入的回归先用失败测试复现，再做最小修复，并重跑相关测试及完整门禁；tag 原样问题记录证据，不扩大修复范围。
- A5: 后端默认与 unit 测试、lint、构建，前端 ESLint/i18n/单测/类型检查与构建，以及 migrations、Ent/Wire 两轮生成稳定性检查在本机可执行范围内完成；独立只读 Verifier 对全部验收项给出结论。
- A6: `backend/cmd/server/VERSION` 为 `0.2.8.1`，且未创建或推送 tag、未触发 release workflow、未部署。
- A7: Docker/Testcontainers、race detector 或其他环境不支持的检查明确记录为未运行，不视为通过。
- A8: `memory/context/upstream-merge-workflow.md` 仅在本轮验证出新的可复用经验时增量更新，否则保持不变并说明。

# 约束与不变量

- 正式 tag 的远端引用与本地 peeled commit 必须一致；单一 merge 边界可追溯。
- 合并时保留 upstream 与 fork 的有效行为；无法同时成立的真实业务语义冲突须暂停并请求用户决定。
- 测试通过不能替代对 fork 关键行为的专项语义审查；生产代码回归修复先有失败测试。

# 决策

- 从干净的 `main` 基线创建隔离分支 `comet/merge-upstream-v0-2-8`，目标分支为 `main`。
- 远端最新三段式正式 tag 是 `v0.2.8`，已核对其 peeled commit；当前 fork 版本 `0.2.7.4`，本次合并版本目标 `0.2.8.1`。
- 沿用前次 Comet 合并的精确 tag、`--no-ff`、双方语义共存、完整门禁和独立回归审查惯例；本轮只集成与审计，不发布。
- 用户确认内容审计延续 fork 的有界多轮用户/助手文本扫描，排除 system/tool 与工具结果；不采用上游仅扫描最新用户轮次的收窄边界。关键词扫描仍应检查客户端提供的 reminder 文本，而语义 API 扫描过滤 reminder。
- OpenCode Go 用量的同 key 共享快照和刷新节流按 upstream `v0.2.8` 原样保留；fork 不另外规定单账号或整组切换代理时的快照失效行为，也不审查上游 tag 自身存在的问题。

# 待解决问题

- 无。

# 验证预期

- 检查 merge 父提交、tag 祖先关系、post-tag 提交排除、未解决冲突与 `git diff --check`。
- 针对 upstream 与 fork 的重叠调用链先运行受影响测试，再完成全量门禁、生成稳定性及独立只读审计。
- 不具备所需环境的检查按真实情况记录，不能冒充通过。
