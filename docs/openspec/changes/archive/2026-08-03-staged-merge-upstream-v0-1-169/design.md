## Context

- 当前主线版本为 `0.1.165.4`，包含上游 `v0.1.165`、此前分段合并后的本地兼容修复，以及 `v0.1.165.2`~`v0.1.165.4` 的 subscription quota cycle reset 本地能力。
- 目标正式 tag 形成严格祖先链：`v0.1.166`（62 commits / 142 files）→ `v0.1.168`（36 commits / 170 files）→ `v0.1.169`（38 commits / 72 files）。`v0.1.167` 只有版本同步历史而无正式 tag，`upstream/main` 在 `v0.1.169` 之后的提交不属于目标。
- `v0.1.169` 修复 GHSA-vrxq-qm4h-6hgg；生产 Nginx 临时盾只有在未来实际部署修复版本后才能移除，本 change 不操作生产环境。
- 上游 `v0.1.168` 新增 `191_passkey_credentials.sql`，本地已有 `191_subscription_quota_advance_receipts.sql` 与 `192_subscription_cache_invalidation_outbox.sql`。迁移执行器沿用完整文件名身份，双方 191 必须共存。
- Build 编排可能先提交当前 change 的 OpenSpec、Design Doc 和 Plan 再完成隔离绑定，因此必须区分不可变 source base 与 planning-only execution base。source base 必须是 execution base 的祖先，两者之间只允许当前 change 的规划产物；应用源码差异或除 Comet runtime selection 外的 dirty path 均阻塞实施。

## Goals / Non-Goals

**Goals:**

- 建立三个可审计的 merge 节点，并把首次冲突或回归准确归属到 `166`、`168` 或 `169` 阶段。
- 同时保留上游功能/安全修复和当前本地能力；每个受影响能力都有自动测试或明确的结构审查证据。
- 结果版本为 `0.1.169.1`，tag 祖先、生成物、依赖、migration 和本机质量门禁一致。
- 对未运行的 Docker-backed integration 保留真实风险，不用远程执行或文字性结论替代。

**Non-Goals:**

- 不合并 `upstream/main` 的 tag 后提交，不为不存在的 `v0.1.167` tag构造阶段。
- 不新增本地产品功能，不借同步进行无关重构。
- 不推送、不打 tag、不触发 GitHub Actions、不发布镜像、不部署、不操作服务器或 Nginx。
- 不承诺未实际执行的 Docker-backed migration/integration 行为已验证。

## Decisions

1. **单 change、三个正式 tag 阶段。** 每段使用 `git merge --no-ff --no-commit <tag>`，先检查冲突和高风险路径，再创建 merge commit。替代方案“一次合入 `v0.1.169`”提交更少，但会丢失首次回归区间；三个独立 changes 又会复制线性依赖、能力矩阵和归档成本。
2. **merge 与兼容修复分离。** Merge commit 只包含上游树和完成 merge 所必需的冲突融合，第二父精确指向目标 peeled SHA；语义审查或测试发现的本地回归放入后续聚焦修复提交。这样可以审计哪些变化来自上游、哪些是本地兼容层。
3. **阶段 0 固定基线和能力矩阵。** 在首次 merge 前记录实施 base ref、三个 tag SHA、各段 changed-files，以及本地能力到入口、调用链和测试的映射。矩阵至少覆盖 scheduler/sticky/fallback/DB recheck、OpenAI HTTP/WS 与 usage、prompt/security audit、request-body 生命周期、settings/repository 更新、subscription quota cycle reset、前端本地功能、生成物和 migrations。
4. **冲突以语义融合为默认。** 不机械选择 ours/theirs；按上游修复、本地定制、接口/配置演进、版本/依赖、生成代码、migration 分类。可共存时做最小融合，无法共存时停止当前阶段并请求用户决定。
5. **每段先聚焦、再本机 full 门禁。** 先运行 changed-files 命中的聚焦测试，再运行 `make test`、`make build`、两轮 `make -C backend generate`、生成 diff、`git diff --check`、未合并文件和真实冲突标记扫描。复杂行为回归遵循失败测试优先；纯 merge、文档与生成更新不伪造 RED。
6. **Docker integration 采用显式本机配置。** 若本机 Docker/Testcontainers 可用，则运行 integration 和 migration 新库/升级库测试；若不可用，记录环境证据、未验证契约和残余风险后继续，不连接 `local-serv-ai` 或其他服务器。该选择缩短验证链，但验收结论必须标明边界。
7. **双方 191 migration 原名保留。** 先融合 schema 与 runner 输入，再由现有 migration 测试验证完整文件名、排序、checksum、幂等和升级路径；不重命名已发布 migration，也不新增 runner 抽象。若本机 integration 不可用，静态与非 Docker 测试不能替代数据库升级证明。
8. **生成文件从源重新产生。** Ent 以 `backend/ent/schema/` 为源，Wire 以 provider 声明和 `wire.go` 为源，前端 lockfile 由现有 pnpm 版本生成，Go checksum 由 Go module 工具维护；不手工拼接生成物。
9. **安全修复按入口和调用链验收。** `v0.1.169` 的路径片段闭集校验必须覆盖 Responses 子路径、Gemini 模型名和非法编码/分隔符拒绝，同时确认本地网关路由、prompt audit 和 request-body 行为不绕过校验。Nginx 盾保留到未来独立的生产部署确认。
10. **版本只在三段闭合后更新。** 中间阶段保持当前本地四段式版本，全部阶段通过后一次更新为 `0.1.169.1`；不创建 `0.1.166.1` 或 `0.1.168.1` 过程版本。

## Risks / Trade-offs

- [上游 repository scoped update、Passkey 或 settings 变更覆盖本地 subscription quota reset 的事务与缓存失效语义] → 把 receipt、单窗口资格、事务锁、版本化 tombstone/outbox 和前端资格判断列为 `v0.1.166/168` 聚焦能力。
- [GHSA 修复在本地网关分支或兼容入口中被绕过] → 追踪所有客户端可控路径片段到上游 URL 拼接点，并运行上游安全测试及本地路由回归。
- [OpenAI WS、Live、代理断流熔断与本地 usage/sticky/failover 定制互相覆盖] → 按阶段审查最终 outbound model、turn ownership、failed usage、fallback/WaitPlan 与代理隔离降级。
- [双方 191 在真实 PostgreSQL 升级路径中失败] → 本机 Docker 可用时执行新库/升级库 integration；不可用时将该项保留为明确未验证风险，禁止宣称通过。
- [本机 full 门禁耗时较高] → 仍按阶段执行；先跑聚焦测试以尽早失败，不新增并行测试框架。
- [工作区现有用户改动混入 merge 或提交] → build 前选择隔离分支/worktree并记录 base ref，所有暂存使用显式路径，禁止 `git add .`。
- [只完成代码同步但生产仍受旧版本影响] → 本 change 不声称生产漏洞已消除；Nginx 盾继续保留，发布部署另行授权和验证。

## Migration Plan

1. Build 阶段在确认的隔离位置记录 source base、planning-only execution base、当前版本和唯一允许的 Comet runtime selection；其他 dirty path 阻塞。
2. 依次完成 `v0.1.166`、`v0.1.168`、`v0.1.169` 的 merge、冲突融合、聚焦修复、能力审查与本机门禁；任一阶段失败时不开始下一段。
3. 三段闭合后更新 `VERSION` 为 `0.1.169.1`，重跑最终本机 full verify、tag 拓扑和能力矩阵终审。
4. 本 change 不部署，因此没有生产 migration 或运行态回滚。隔离分支上的失败通过保留当前失败证据并修复，或在用户决定后放弃隔离分支处理；不得改写已共享历史。
