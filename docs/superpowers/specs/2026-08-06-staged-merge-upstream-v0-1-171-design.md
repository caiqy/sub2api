---
comet_change: staged-merge-upstream-v0-1-171
role: technical-design
canonical_spec: openspec
---

# 分段合并上游 v0.1.171 技术设计

## 1. 背景与事实源

本设计细化 `docs/openspec/changes/staged-merge-upstream-v0-1-171/` 中已确认的 proposal、delta spec、design 和 tasks。OpenSpec delta spec 是验收事实源，本文只定义实施结构、冲突策略、风险控制和测试方法。

不可变 source base 为 `main@b576f73a2`，运行版本为 `0.1.169.3`。该基线已包含上游 `v0.1.169`、此前分段合并的兼容修复，以及其后的本地请求体内存驻留治理、alpha-search/composite 路由、可选 pprof 和相关回归加固。Build 期间可能先提交当前 change 的规划产物，因此实际 execution base MAY 是 source base 的 planning-only 后代；`b576f73a2` 必须是其祖先，两者之间只允许当前 change 规划产物，禁止应用源码差异。

目标 tag 形成严格祖先链：

| 阶段 | Tag peeled SHA | 上游区间 | Commits | Files |
|---|---|---|---:|---:|
| 1 | `c043c24774228ba891ddf90d783aa6dc7d0855b5` | `v0.1.169..v0.1.170` | 62 | 242 |
| 2 | `f0e7a9c7a23a7d02fb159b62fa809621eb0475a6` | `v0.1.170..v0.1.171` | 49 | 206 |

两段合计修改 392 个文件；从 `v0.1.169` 分叉后，本地演进和目标上游区间有 151 个重叠路径，主要集中在 `backend/internal/service`、handler、repository、settings/auth、前端账号/设置和生成代码。设计阶段只读 `git merge-tree --write-tree HEAD v0.1.170^{commit}` 预测首段 28 个文本冲突；实际冲突集合以 Build 隔离位置为准。

Migration identity 已形成新的同号风险：本地已有 `192_subscription_cache_invalidation_outbox.sql`，上游新增 `192_group_profit_control.sql` 和 `193_group_profit_control_auth_cache_invalidation.sql`。执行器沿用完整文件名身份，双方 192 必须与既有双方 191 一并共存。

## 2. 目标与边界

### 2.1 目标

- `v0.1.170`、`v0.1.171` 按顺序成为结果 HEAD 的祖先，并各自保留可审计 merge 节点。
- 上游功能/修复与本地定制共存；回归归属到首次出现的 release 区间。
- Merge commit 保持纯净；语义兼容修复按用户确认的能力簇拆分。
- 双方 `191_*`、双方 `192_*` 与上游 `193_*` migration 保持完整文件名和 identity。
- 最终版本精确为 `0.1.171.1`，本机自动门禁、能力级 review、生成稳定性和拓扑验证完成。
- Docker-backed integration 未执行时，报告准确列出环境原因和未验证契约。

### 2.2 非目标

- 不合入 `v0.1.171` 之后的 `upstream/main` 提交，不静默吸收实施前出现的更高 tag。
- 不新增本地产品功能，不做无关重构。
- 不推送、不打 tag、不触发 GitHub Actions、不发布或构建 Sub2API 镜像。
- 不部署，不操作服务器、数据库、Redis 或 Nginx。
- 不把静态检查、编译成功或旧 integration 证据写成当前 PostgreSQL 升级路径已通过。

## 3. 采用方案

采用“单 change、两段受审 merge、按能力簇修复、逐段封闭”。拒绝以下替代方案：

- 一次合入 `v0.1.171`：历史更短，但无法区分回归首次来自 170 还是 171。
- 两段 merge、每段一个汇总修复提交：提交更少，但调度、网关、认证、订阅和前端回归会耦合，审查与回退成本更高。
- 两个独立 changes：阶段严格线性依赖，共享基线、能力矩阵和最终版本，拆分只会复制上下文与归档工作。

实施状态机：

```text
阶段 0：固定 refs / 基线门禁 / 能力矩阵 / Docker 预检
  -> merge --no-ff --no-commit v0.1.170
  -> 冲突分类、语义融合、源驱动生成
  -> 创建纯 merge commit
  -> 聚焦 RED -> 按能力簇最小兼容提交
  -> v0.1.170 聚焦 + full + 可用的 integration 门禁
  -> merge --no-ff --no-commit v0.1.171
  -> 重复冲突、生成、能力簇修复和阶段门禁
  -> VERSION=0.1.171.1
  -> 最终全门禁 / 拓扑 / migration identity / 能力终审
```

工作区隔离、执行方式、TDD 模式和审查模式由 Build 阶段联合决策点确认，本文不替用户提前选择。

## 4. Git 与提交边界

每段唯一允许的 merge 起点为：

```text
git merge --no-ff --no-commit v0.1.170
git merge --no-ff --no-commit v0.1.171
```

提交边界：

- **merge commit**：只承载目标上游树和完成 merge 必需的冲突融合；第二父必须等于固定 peeled SHA。
- **scheduler/usage 修复提交**：利润门禁、候选过滤、sticky/fallback、槽位复核、倍率和 usage billing。
- **gateway/body 修复提交**：HTTP/WS、Codex identity、alpha-search、请求体 spooling/释放、错误与 failover。
- **audit/auth 修复提交**：prompt/security audit、验证码、认证入口、settings/CSP。
- **subscription/migration 修复提交**：quota reset、订阅续期、退款/余额、outbox、migration 与生成源。
- **frontend 修复提交**：账号、分组、认证设置、退款/用量及既有本地 UI 定制。
- **证据提交**：只更新 build ledger、任务状态和阶段结论。
- **最终版本提交**：两段封闭后一次更新 `backend/cmd/server/VERSION` 为 `0.1.171.1`。

不要求每个能力簇都产生提交；没有真实修复时不创建空提交。一次修复只属于一个主要能力簇，必要的直接测试和生成输出与源修复同提交。所有暂存使用显式路径，禁止 `git add .`；`.comet/current-change.json` 不得混入业务提交。

## 5. 阶段 0 与能力矩阵

Build ledger 固定路径：

`docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-build.md`

Ledger 记录 source/execution base、branch/worktree、tag manifest、dirty path 归因、每段 changed-files、预测/实际冲突、能力矩阵、命令和退出码、失败/修复、阶段结论、Docker 预检和残余风险。

能力矩阵每行包含：行为契约、入口/调用链、关键文件、受影响 tag、聚焦测试、人工审查点、状态和证据。状态：

- `protected`：直接行为测试通过。
- `manual`：生成物、依赖或静态调用链证据已审查。
- `unverified`：仅用于有环境证据的 integration 缺口。
- `gap`：上游触及但缺少充分证据；阶段结束前必须归零。

矩阵至少覆盖：

- advanced/layered scheduler、sticky、fallback/WaitPlan、DB recheck、pool mode 和同账号重试；
- Grok/platform/session/previous-response sticky、privacy、image capability；
- OpenAI HTTP/WS、turn ownership、最终 outbound model、failed usage、prompt cache reuse、proxy circuit 和错误帧；
- alpha-search 原生端点与 Responses fallback、PAT 401 副作用、WebSearchCalls 计费、请求体 handle 生命周期；
- 请求体 replay/spooling/cleanup、异步图片任务与对象存储、图片输入计费；
- 统一 prompt/security audit、Images 单入口、legacy moderation 单次执行、payload 按需冻结与内容释放顺序；
- settings 热更新、repository scoped update、API Key auth cache、session binding 和 step-up；
- subscription quota cycle reset 的单窗口资格、事务锁、receipt、版本化 tombstone 和 outbox；
- 用户资源控制、分组复制/批量限额、账号 shadow、前端菜单/设置/用量/订阅/渠道/移动端定制；
- Ent/Wire、Go/pnpm 依赖、CSP/deploy 配置和 migrations。

## 6. 冲突处理

每个冲突仅使用六类之一：上游修复、本地定制、接口/配置演进、版本/依赖、生成代码、migration。台账记录 ours 行为、theirs 行为、融合结果和验证证据。可共存时做最小融合；无法共存且用户未批准移除的行为立即阻塞当前阶段。

首段预测的 28 个冲突按风险聚类：

- **版本/生成代码**：`VERSION`、`wire_gen.go`、Ent client/group/mutation；
- **settings/auth cache**：setting handler/update、admin account、API Key auth cache；
- **gateway handler**：gateway/chat、Gemini、alpha-search、chat/embeddings/count_tokens；
- **scheduler/usage**：gateway scheduling、OpenAI account/gateway scheduling、usage billing；
- **audit/subscription**：content moderation、subscription service、user subscription；
- **frontend**：Create/Edit account modal 及测试。

审查时点：

1. **merge commit 前**：检查全部文本冲突、编译接口、route/DTO/config/provider/schema 和高风险入口；已知不可共存语义存在时不创建 merge commit。
2. **merge commit 后**：运行 changed-files × 能力矩阵、调用链和行为测试；可修复回归保留 RED 后按能力簇提交，需要用户取舍时停在当前阶段。

特殊文件策略：

- `VERSION`：两段冲突均保留 `0.1.169.3`，最终再更新为 `0.1.171.1`。
- Go 依赖：先融合 `go.mod`，再由 Go module 工具生成 `go.sum`。
- Ent/Wire：先融合 schema/provider 源，再重新生成；不手工编辑生成语义。
- 前端 lockfile：先融合 manifest，再用仓库现有 pnpm 版本生成。
- Migration：保留所有已发布完整文件名和内容，不因数字前缀重复而重命名。

## 7. 分段风险与审查点

### 7.1 v0.1.170

利润控制把账号成本倍率引入候选过滤、槽位后二次复核和 sticky 绑定。必须验证：默认关闭时行为不变；不合格账号不参与排序且不提前绑定；槽位获取后倍率变化会释放槽位并重新选号；同一请求在等待/重试/切号中保持固定定价时刻；组合分组不被错误直接门禁；本地 layered scheduler、WaitPlan 和 DB recheck 不被短路。

账号倍率自动同步扩大到 API Key 平台，并引入系统托管写回、值域校验和 cache invalidation。需与本地 account repository scoped update、API Key auth cache 和 subscription cache outbox 交叉审查，防止部分更新清空本地字段或漏发缓存失效。

其他重点：Anthropic 流式中断用量、OpenAI WS 外部取消错误帧、流内 429/pool 重试、Responses 工具图片、内容审核代理/最新输入、订阅窗口对齐、settings 支付方式保持、图片 data URL。它们分别命中本地 usage、WS lease、request-body handle、统一审计、quota reset 和前端设置保护面。

### 7.2 v0.1.171

Codex identity 将 User-Agent/originator/version 统一到动态版本来源。必须追踪 HTTP、透传、WebSocket 握手、探针、模型列表、账号测试和 alpha-search；账号级自定义 UA 只能贡献允许的客户端/环境信息，不得恢复陈旧版本。alpha-search 的 PAT fallback、request-body handle、失败副作用和按次计费必须保持。

`server_is_overloaded` / `slow_down` 改为同账号有界重试后切号，不再据此冷却账号。必须与 pool mode、UpstreamFailoverError、sticky、账号状态副作用、最终 outbound account/model 和请求体重放交叉测试，确保重试不复用已关闭 body、不重复计费、不错误禁用账号。

验证码新增腾讯天御和阿里云 2.0，并合并为互斥 provider 设置。认证矩阵覆盖注册、登录、找回密码、OAuth 启动和 passkey；校验失败按 fail-closed，既有 Turnstile 配置保持，CSP 仅增加必要域名，本地 settings 热更新、自定义菜单和前端认证流程不丢失。

计费/数据重点包括：计费失败仍落 usage 且实收为 0、余额不足退款显式 force、Stripe 幂等、并发订阅续期、重置额度缓存。需验证本地 quota reset/outbox、usage hash/倍率、事务锁和管理端状态流转。

其余重点包括 composite reasoning effort、Messages 临时不可调度切号、入站 WS 租约终止事件、请求取消后的 snapshot 短路和 Responses `output_text` prompt audit。

## 8. Migration 与生成稳定性

最终 migration 集至少保留：

- `191_passkey_credentials.sql`
- `191_subscription_quota_advance_receipts.sql`
- `192_subscription_cache_invalidation_outbox.sql`
- `192_group_profit_control.sql`
- `193_group_profit_control_auth_cache_invalidation.sql`

验证分两层：

1. 始终执行非 Docker 的 migration 嵌入、排序/checksum、文件存在性和编译测试。
2. 本机 Docker 可用时，执行 PostgreSQL 空库和从 `main@b576f73a2` migration 集升级的 integration。升级基线 FS 必须包含本地 191/192 与上游 passkey 191，但排除上游 group profit 192/193；随后应用完整 FS，验证新增 192/193、幂等和 checksum。

Implementation plan 应固定升级测试名称及 `--- PASS:` 匹配规则；命令 exit 0、package `ok`、`no tests to run` 或 `--- SKIP:` 不能单独作为目标 integration 通过证据。

生成物归属必须闭合：merge 冲突涉及生成文件时，源完成融合后重新生成，并把必要输出放入 merge commit；兼容修复改动生成源时，对应输出放入同一能力簇提交。阶段结束前连续运行两次 backend generate，两轮均不得产生 diff。

## 9. 本机验证策略

每段先运行能力矩阵命中的聚焦测试，再执行：

```text
make test
make build
make -C backend generate
生成 diff 检查
make -C backend generate
生成 diff 检查
git diff --check
unmerged index 检查
tracked conflict marker 扫描
Docker 可用性预检；可用时运行本阶段 integration
```

`make test` 覆盖后端默认/unit/lint 与前端 lint/typecheck/Vitest，`make build` 覆盖前后端构建。Windows 命令形态沿用仓库已验证方式，不通过修改产品代码绕过环境问题。

Docker/Testcontainers 不可用时，ledger 和最终报告把目标 migration/repository 契约标为 `unverified`；不得连接远程服务器补验。Docker 可用时必须在日志中确认目标 top-level test 出现真实 `--- PASS:`。

最终验证报告路径：

`docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-verify.md`

## 10. 失败处理与回退

- 基线门禁失败：首个 merge 前停止，区分既有失败、环境阻塞和保护缺口。
- merge 前冲突可融合：完成最小语义融合并记录证据；不可共存语义等待用户决定。
- merge 后聚焦/full 门禁失败：保留 RED，按能力簇最小修复，从本段聚焦测试重跑。
- 生成不稳定：回到 schema/provider/manifest 源修复，不提交不可复现输出。
- 目标 integration 真实失败：阻塞当前阶段；环境不可用或明确 skip 才能按 spec 记为 unverified。
- 实施前出现更高正式 tag：不 merge，回到 OpenSpec 更新 proposal/spec/design/tasks 并重新确认。
- 异常 dirty path：按 Comet dirty-worktree 协议归因，未归因前不暂存、不覆盖、不推进。

工作在隔离分支/worktree 时，彻底回退可由用户决定放弃该隔离位置；不得使用破坏性命令覆盖 main 或用户改动。若历史已共享，回退使用 merge 节点与对应能力簇提交的 revert，不改写远端历史。

## 11. 完成条件

- 两个正式 tag 都是结果 HEAD 祖先，两个 merge 第二父与固定 SHA 一致。
- `backend/cmd/server/VERSION` 精确为 `0.1.171.1`。
- 能力矩阵无 `gap`；`unverified` 仅来自已记录的本机 integration 环境边界。
- 利润控制/倍率同步、Codex identity/过载重试、验证码/auth/settings、退款/usage/subscription、WS/prompt audit 与本地保护面均有直接行为或结构证据。
- alpha-search、request-body 生命周期、layered scheduler、统一 audit 和 quota reset/outbox 保持本地契约。
- 双方 191/192 与上游 193 migration 保留；真实 PostgreSQL 升级仅在实际执行成功时记为通过。
- 聚焦测试、`make test`、`make build`、两轮 generate 和静态检查在最终 source HEAD 上通过。
- 每段结束时 worktree/index 除 Comet runtime selection 外干净，生成输出归属于对应 merge 或能力簇提交。
- OpenSpec strict validate 和 Comet Verify 通过或按流程记录明确允许的环境边界。
- 没有推送、发版、部署或服务器操作。
