# Brainstorm Summary

- Change: staged-merge-upstream-v0-1-169
- Date: 2026-08-02

## 确认的技术方案

- 采用单 change、三个正式 tag 逐段封闭：阶段 0 后依次执行 `v0.1.166`、`v0.1.168`、`v0.1.169`，最后一次更新 `VERSION=0.1.169.1`。
- 每段使用 `git merge --no-ff --no-commit <tag>`；merge commit 只包含上游树与完成 merge 必需的冲突融合，第二父固定为目标 peeled SHA。测试或语义审查发现的兼容修复使用后续独立提交。
- 以固定 tag manifest、持续更新的 build ledger 和 changed-files × 本地能力矩阵驱动放行。每段只有在聚焦测试、本机 full 门禁和能力矩阵无未解释 gap 后才能进入下一 tag。
- 冲突按上游修复、本地定制、接口/配置演进、版本/依赖、生成代码、migration 六类处理；逐文件记录 ours、theirs、融合结论和证据，不机械选择 ours/theirs。
- 上游 `191_passkey_credentials.sql` 与本地 `191_subscription_quota_advance_receipts.sql`、`192_subscription_cache_invalidation_outbox.sql` 保持原名共存；生成文件和依赖从各自源重新生成，不手工拼接。
- 范围止于代码同步与本机验证；不推送、发版、部署、操作服务器或移除生产 Nginx 临时盾。实施隔离方式留给 Build 阶段联合决策点；不可变 source base 与 planning-only execution base 分开记录，后者不得包含应用源码差异。

## 关键取舍与风险

- 逐段 full 门禁耗时更多，但能准确归属首次回归；已拒绝“仅最终统一验证”。
- 已拒绝先 cherry-pick GHSA 修复：当前 change 不部署，安全修复先行没有运行时收益，却会制造重复补丁和 ancestry 复杂度。
- 本机当前找不到 `docker`。每段执行轻量可用性检查，阶段 0 与最终阶段保留完整诊断；持续不可用时，中间阶段继承“integration 与真实 PostgreSQL 升级路径未验证”风险，不重复完整失败命令，也不使用远程服务器补跑。任一阶段检测到 Docker 可用时，该阶段 integration 变为强制门禁。
- 只读 merge-tree 预测 `v0.1.166` 首段有 17 个文本冲突，集中于 settings、OpenAI gateway/Responses/WS、usage、Gemini compat、Go 依赖和前端 UsageView 测试，必须按行为融合。
- 若上游语义与未批准移除的本地能力无法共存，立即停在当前阶段等待用户决定。
- Merge commit 前先完成冲突与高风险入口的阻塞性审查；commit 后再运行完整能力矩阵和行为测试。生成输出必须归入对应 merge/修复提交，每段结束工作区与索引保持干净。

## 测试策略

- 阶段 0：固定 tag SHA、changed-files、能力矩阵和当前本地保护测试。
- 每段：受影响聚焦测试 → `make test` → `make build` → 两轮 backend generate → 生成 diff、unmerged 文件、冲突标记、whitespace 和工作区/index clean 检查；Docker 可用时追加本阶段 integration。
- `v0.1.166` 聚焦 settings 部分更新、panel rate limit、OpenAI WS 每轮模型/usage、composite route；`v0.1.168` 聚焦 Passkey/双方 191、repository scoped updates、prompt audit 配置恢复和 OpenAI Live store；`v0.1.169` 聚焦 GHSA 路径闭集、Gemini/Responses URL 拼接、代理断流熔断、count_tokens、pricing 和 release 资源。能力矩阵完整承接 canonical spec 的 privacy、image capability、异步图片/对象存储、计费倍率、session/step-up、透传、资源控制、分组复制、批量限额、全部前端本地功能与 Images 精确惰性审核契约。
- GHSA 矩阵区分合法根路径空 suffix 和非空 suffix 中的空 segment，固定 128-byte 单段、8 段上限、ASCII `[A-Za-z0-9_.-]` 且拒绝纯点片段，并要求三类 Responses 入口明确 `404`、Gemini 构造前报错。
- Docker 可用时运行 verbose integration 与 migration 新库/升级库验证，并要求目标 top-level 测试日志出现真实 `--- PASS:`；exit `0`、package `ok`、零测试或 skip 都不能单独作为通过。不可用时记录命令、环境证据、未验证契约和残余风险。
- 最终重跑全部本机门禁，校验 tag ancestry、三个 merge 第二父、`VERSION=0.1.169.1`、双方 191/本地 192 和能力矩阵终态。

## Spec Patch

无。现有 delta spec 已覆盖本机 integration 不可用时的处理和双方 191 migration 契约。
