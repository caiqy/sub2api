## Context

- 需求背景见 `proposal.md`，行为契约见 `specs/upstream-release-sync/spec.md`。
- 当前固定基线为 `main@b576f73a2`，运行版本 `0.1.169.3`；目标 tag 为 `v0.1.170@c043c24774228ba891ddf90d783aa6dc7d0855b5` 和 `v0.1.171@f0e7a9c7a23a7d02fb159b62fa809621eb0475a6`，两者形成严格祖先链。
- `v0.1.169..v0.1.170` 包含 62 commits / 242 files，`v0.1.170..v0.1.171` 包含 49 commits / 206 files；总计 392 个上游变更文件，其中 151 个与当前本地演进路径重叠。
- 对当前 HEAD 与 `v0.1.170` 的只读 merge-tree 预测显示 28 个文本冲突，集中在生成代码、settings、gateway、scheduler、usage billing、subscription、account UI 和测试；实际冲突集合以 Build 隔离位置中的 merge 结果为准。
- 本地主线已发布 `192_subscription_cache_invalidation_outbox.sql`；上游 `v0.1.170` 新增 `192_group_profit_control.sql` 和 `193_group_profit_control_auth_cache_invalidation.sql`。迁移执行器以完整文件名作为身份，三个文件必须共存。

## Goals / Non-Goals

**Goals:**

- 建立两个可审计 merge 节点，把首次冲突或回归准确归属到 `v0.1.170` 或 `v0.1.171` 阶段。
- 同时保留上游功能/安全修复和本地能力；每个受影响能力都有自动测试或明确结构审查证据。
- 结果版本为 `0.1.171.1`，tag 祖先、merge 第二父、生成物、依赖和 migration identity 一致。
- Docker-backed integration 未执行时，验证报告准确记录环境原因、未验证契约和残余风险。

**Non-Goals:**

- 不合并 `v0.1.171` 之后的 `upstream/main` 提交，不把后续新 tag 静默加入当前范围。
- 不新增本地产品功能，不借同步进行无关重构。
- 不推送、不打 tag、不触发 GitHub Actions、不发布或构建 Sub2API 镜像，不部署或操作服务器。
- 不承诺未实际执行的 PostgreSQL migration/integration 行为已验证。

## Decisions

1. **单 change、两个正式 tag 阶段。** 每段使用 `git merge --no-ff --no-commit <tag>`，完成冲突融合、能力审查和阶段门禁后再进入下一段。一次合入 `v0.1.171` 虽提交更少，但会失去回归首次引入区间；拆成两个 changes 则会复制严格线性依赖下的矩阵和归档成本。
2. **merge 与兼容修复分离。** Merge commit 只包含目标上游树和完成 merge 所必需的冲突融合，第二父精确指向固定 tag SHA；测试或调用链审查发现的语义回归以独立兼容修复提交处理，便于区分 upstream 与 fork 责任。
3. **实施前固定事实清单。** Build 阶段记录 source base、execution base、两个 tag SHA、严格祖先链、每段 changed-files、预测/实际冲突和 changed-files × 本地能力矩阵。首次 merge 前重新 fetch；若发现更高正式 tag，回到 OpenSpec 更新范围，不在实施中静默扩大。
4. **冲突默认语义融合。** 每个冲突按上游修复、本地定制、接口/配置演进、版本/依赖、生成代码或 migration 分类；可共存时做最小融合，不机械选择 ours/theirs。无法共存的用户可见语义立即阻塞当前阶段并请求用户决定。
5. **重点调用链按 release 分段审查。** `v0.1.170` 聚焦利润门禁、倍率同步、槽位后二次复核、流式用量、内容审核和订阅窗口；`v0.1.171` 聚焦 Codex 身份/版本、过载重试、验证码/auth/settings/CSP、退款与用量持久化、composite reasoning、WebSocket 租约和 prompt audit。两段都交叉审查本地调度、sticky/fallback、请求体 spooling、统一审计、quota reset、alpha-search/composite 路由和前端定制。
6. **每段先聚焦再执行本机 full 门禁。** 先运行 changed-files 命中的聚焦测试，再运行 `make test`、`make build`、两轮 `make -C backend generate` 与生成 diff、`git diff --check`、unmerged index 和真实冲突标记扫描。复杂行为修复保留 RED；纯 merge、文档和生成更新不伪造 RED。
7. **Docker integration 采用本机条件门禁。** 本机 Docker/Testcontainers 可用时运行 migration/repository integration，并确认目标测试出现真实 PASS；不可用或测试被跳过时记录环境证据和残余风险，不使用远程服务器补跑，也不把 package exit 0 当作目标 integration 已通过。
8. **同号 migration 以完整文件名共存。** 保留本地 `191_subscription_quota_advance_receipts.sql`、`192_subscription_cache_invalidation_outbox.sql`，以及上游 `191_passkey_credentials.sql`、`192_group_profit_control.sql`、`193_group_profit_control_auth_cache_invalidation.sql`；不重命名已发布文件。可用时验证空库和已应用本地 192 的升级库。
9. **生成文件从源重新产生。** Ent 以 schema 为源，Wire 以 provider 声明和 `wire.go` 为源，Go/pnpm 锁文件由现有工具维护；不手工拼接生成物。源与对应生成输出归属同一 merge 或兼容修复提交。
10. **最终版本只更新一次。** 中间阶段保持 `0.1.169.3`；两段全部闭合后一次更新为 `0.1.171.1`，不创建 `0.1.170.1` 过程版本。

## Risks / Trade-offs

- [利润控制与本地 layered scheduler、sticky、fallback 或 usage billing 组合后改变候选集/定价时刻] → 建立候选过滤、槽位释放、粘性绑定和请求级定价的聚焦行为测试与调用链审查。
- [Codex 身份归一化、版本同步或过载重试绕过本地 alpha-search、WS、请求体释放和账号切换语义] → 覆盖所有出站入口，复核统一身份来源、重试边界、最终账号和 body 生命周期。
- [验证码服务商合并覆盖本地 settings/UI 定制或自定义 CSP] → 对 auth 入口、provider 互斥、fail-closed、settings 热更新和前端卡片运行聚焦测试与浏览器/构建验证。
- [双方 `192_*` 在真实 PostgreSQL 升级路径中排序、checksum 或幂等失败] → 保留完整文件名并运行空库/本地 192 升级 integration；环境不可用时明确标为 unverified。
- [生成代码冲突掩盖 schema/provider 语义] → 先融合源，再重新生成；两轮生成必须稳定无 diff。
- [工作区用户改动混入 merge 或提交] → Build 采用 Comet 确认的 branch/worktree 隔离，所有暂存使用显式路径，禁止 `git add .`。

## Migration Plan

1. 在确认的隔离位置固定 source/execution base、tag manifest、能力矩阵和本机基线门禁。
2. 合入并封闭 `v0.1.170`：冲突融合、兼容修复、聚焦审查、full 门禁和可用的 integration。
3. 合入并封闭 `v0.1.171`：重复相同阶段门禁，重点覆盖 Codex、验证码、退款/计费和 WS/prompt audit。
4. 更新 `VERSION` 为 `0.1.171.1`，执行最终全门禁、拓扑、migration identity 和能力矩阵终审。
5. 本 change 不部署，因此没有生产迁移或运行态回滚；隔离分支失败时保留证据并修复，或由用户决定后放弃隔离位置，禁止改写共享历史。
