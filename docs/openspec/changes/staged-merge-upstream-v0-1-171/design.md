## Context

- 需求背景见 `proposal.md`，行为契约见 `specs/upstream-release-sync/spec.md`。
- 当前固定基线为 lint remediation 归档后的 `main@16c07d806`，运行版本 `0.1.169.3`；目标 tag 为 `v0.1.170@c043c24774228ba891ddf90d783aa6dc7d0855b5`、`v0.1.171@f0e7a9c7a23a7d02fb159b62fa809621eb0475a6`、`v0.1.172@155c494964c3ea6ecc31f52679525c1034bf0f16` 和 `v0.1.173@29009f0b2ea14edf3b11ae2564fb617ff91a03b4`，四者形成严格祖先链。
- 四段分别包含 62 commits / 242 files、49 commits / 206 files、54 commits / 208 files、120 commits / 352 files；172 有 113 个本地重叠路径，173 在 172 merge 前的初步重叠为 138 个路径，并在 172 阶段闭合后重算精确重叠。GitHub Compare API 只返回前 300 个 files，文件面门禁使用本地固定 commit tree diff。
- 对当前 HEAD 与 `v0.1.170` 的只读 merge-tree 预测显示 28 个文本冲突，集中在生成代码、settings、gateway、scheduler、usage billing、subscription、account UI 和测试；实际冲突集合以 Build 隔离位置中的 merge 结果为准。
- 本地主线已发布 `192_subscription_cache_invalidation_outbox.sql`；上游 `v0.1.170` 新增 `192_group_profit_control.sql` 和 `193_group_profit_control_auth_cache_invalidation.sql`。迁移执行器以完整文件名作为身份，三个文件必须共存。

## Goals / Non-Goals

**Goals:**

- 建立四个可审计 merge 节点，把首次冲突或回归准确归属到对应 release 阶段。
- 同时保留上游功能/安全修复和本地能力；每个受影响能力都有自动测试或明确结构审查证据。
- 结果版本为 `0.1.173.1`，tag 祖先、merge 第二父、生成物、依赖和 migration identity 一致。
- Docker-backed integration 未执行时，验证报告准确记录环境原因、未验证契约和残余风险。

**Non-Goals:**

- 不合并 `v0.1.173` 之后的 `upstream/main` 提交，不把后续新 tag 静默加入当前范围。
- 不新增本地产品功能，不借同步进行无关重构。
- 不推送、不打 tag、不触发 GitHub Actions、不发布或构建 Sub2API 镜像，不部署或操作服务器。
- 不承诺未实际执行的 PostgreSQL migration/integration 行为已验证。

## Decisions

1. **单 change、四个正式 tag 阶段。** 保留已验证但未归档的 170/171 历史证据，使旧 Verify 对新增范围失效，再依次追加 `v0.1.172`、`v0.1.173` 的独立 merge、能力审查和阶段门禁；不拆分新 change，不重写既有历史。
2. **merge 与兼容修复分离。** Merge commit 只包含目标上游树和完成 merge 所必需的冲突融合，第二父精确指向固定 tag SHA；测试或调用链审查发现的语义回归以独立兼容修复提交处理，便于区分 upstream 与 fork 责任。
3. **实施前固定事实清单。** Build 阶段记录 source base、execution base、四个 tag object/SHA、严格祖先链、每段 changed-files、预测/实际冲突和 changed-files × 本地能力矩阵。172 merge 前重新 fetch；若发现高于 173 的正式 tag，回到 OpenSpec 更新范围，不在实施中静默扩大。
4. **冲突默认语义融合。** 每个冲突按上游修复、本地定制、接口/配置演进、版本/依赖、生成代码或 migration 分类；可共存时做最小融合，不机械选择 ours/theirs。无法共存的用户可见语义立即阻塞当前阶段并请求用户决定。
5. **重点调用链按 release 分段审查。** 170/171 保留既有结论；172 按 security/auth/captcha、subscription/billing/persistence、gateway/transport/protocol、schema/migration/frontend 四簇审查；173 按 Grok auth/model mapping、Grok gateway/scheduler、Channel Monitor V2、pricing/schema/frontend 四簇审查，并交叉验证本地调度、sticky/fallback、请求体 spooling、统一审计、quota reset/outbox 和前端定制。
6. **每段先聚焦再执行本机 full 门禁。** 先运行 changed-files 命中的聚焦测试，再运行 `make test`、`make build`、两轮 `make -C backend generate` 与生成 diff、`git diff --check`、unmerged index 和真实冲突标记扫描。复杂行为修复保留 RED；纯 merge、文档和生成更新不伪造 RED。
7. **Docker integration 采用本机条件门禁。** 本机 Docker/Testcontainers 可用时运行 migration/repository integration，并确认目标测试出现真实 PASS；不可用或测试被跳过时记录环境证据和残余风险，不使用远程服务器补跑，也不把 package exit 0 当作目标 integration 已通过。
8. **同号 migration 以完整文件名共存。** 保留本地 `191_subscription_quota_advance_receipts.sql`、`192_subscription_cache_invalidation_outbox.sql`，以及上游 `191_passkey_credentials.sql`、`192_group_profit_control.sql`、`193_group_profit_control_auth_cache_invalidation.sql`；不重命名已发布文件。可用时验证空库和已应用本地 192 的升级库。
9. **生成文件从源重新产生。** Ent 以 schema 为源，Wire 以 provider 声明和 `wire.go` 为源，Go/pnpm 锁文件由现有工具维护；不手工拼接生成物。源与对应生成输出归属同一 merge 或兼容修复提交。
10. **扩展后的最终版本只更新一次。** 170/171 历史中的 `0.1.171.1` 成为中间 fork 版本；172 阶段仍保留该版本，173 全部闭合后一次更新为 `0.1.173.1`。
11. **Grok 默认映射遵循发布契约。** `grok_cross_client_model_map_enabled` 缺失或未配置时为 false；`gpt-*`/`codex-*`/`o*`/`claude-*` 不静默改写。显式开启与账号显式 `model_mapping` 保持可用。该裁决有意修正 173 tag 中 `setting_parse.go` 默认 true 与 release note/常量注释冲突，并以回归测试固定。
12. **认证与监控默认保持保守兼容。** Grok 邮箱密码授权入口隐藏且服务端固定拒绝，兼容配置不生效；Channel Monitor 默认 V1，V2 由管理员显式切换且互斥，普通用户吞吐指标默认隐藏。

## Risks / Trade-offs

- [利润控制与本地 layered scheduler、sticky、fallback 或 usage billing 组合后改变候选集/定价时刻] → 建立候选过滤、槽位释放、粘性绑定和请求级定价的聚焦行为测试与调用链审查。
- [Codex 身份归一化、版本同步或过载重试绕过本地 alpha-search、WS、请求体释放和账号切换语义] → 覆盖所有出站入口，复核统一身份来源、重试边界、最终账号和 body 生命周期。
- [验证码服务商合并覆盖本地 settings/UI 定制或自定义 CSP] → 对 auth 入口、provider 互斥、fail-closed、settings 热更新和前端卡片运行聚焦测试与浏览器/构建验证。
- [172 的 midnight 日额度修复覆盖用户再次确认的实际操作时刻锚点] → merge 前后以 RED 测试锁定新购、用户/管理员手动重置及后续自动窗口，明确拒绝 midnight 滚动语义。
- [响应模型审计或 capacity failover 绕过本地 body/sticky/usage/audit 契约] → 覆盖各协议入口、最终账号/模型、失败用量和单次计费，确保 client-facing rewrite 不污染上游响应模型证据。
- [Grok 跨客户端默认值的 tag/release 不一致导致静默模型改写] → 同时覆盖设置缺失、显式 false/true 和账号显式 mapping，默认关闭且显式开启即时生效。
- [Grok 媒体/Voice/search/free gate 或调度阈值绕过本地 sticky、usage、计费和 request-body 生命周期] → 按入口验证最终模型/账号、单位计费、软门禁恢复、冷却边界与单次资源释放。
- [Channel Monitor V2 被动聚合与本地 V1/用户渠道定制冲突或泄露吞吐] → V1/V2 互斥、默认 V1，验证 admin 完整指标、普通用户脱敏、rollup 幂等与前端模式切换。
- [双方 `192_*` 在真实 PostgreSQL 升级路径中排序、checksum 或幂等失败] → 保留完整文件名并运行空库/本地 192 升级 integration；环境不可用时明确标为 unverified。
- [生成代码冲突掩盖 schema/provider 语义] → 先融合源，再重新生成；两轮生成必须稳定无 diff。
- [工作区用户改动混入 merge 或提交] → Build 采用 Comet 确认的 branch/worktree 隔离，所有暂存使用显式路径，禁止 `git add .`。

## Migration Plan

1. 在确认的隔离位置固定 source/execution base、tag manifest、能力矩阵和本机基线门禁。
2. 合入并封闭 `v0.1.170`：冲突融合、兼容修复、聚焦审查、full 门禁和可用的 integration。
3. 合入并封闭 `v0.1.171`：重复相同阶段门禁，重点覆盖 Codex、验证码、退款/计费和 WS/prompt audit。
4. 保留 170/171 已完成证据，撤销旧 Archive 就绪状态，追加并封闭 `v0.1.172` 的 merge、四能力簇兼容修复和本机门禁。
5. 在 172 阶段门禁通过后追加并封闭 `v0.1.173` 的 merge、四能力簇兼容修复和本机门禁。
6. 更新 `VERSION` 为 `0.1.173.1`，执行最终全门禁、四段拓扑、全部 migration identity 和能力矩阵终审。
7. 本 change 不部署，因此没有生产迁移或运行态回滚；隔离分支失败时保留证据并修复，或由用户决定后放弃隔离位置，禁止改写共享历史。
