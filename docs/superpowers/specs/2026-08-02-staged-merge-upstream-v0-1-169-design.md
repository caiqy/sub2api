---
comet_change: staged-merge-upstream-v0-1-169
role: technical-design
canonical_spec: openspec
archived-with: 2026-08-03-staged-merge-upstream-v0-1-169
status: final
---

# 分段合并上游 v0.1.169 技术设计

## 1. 背景与事实源

本设计细化 `docs/openspec/changes/staged-merge-upstream-v0-1-169/` 中已确认的 proposal、delta spec、design 和 tasks。OpenSpec delta spec 是验收事实源，本文只定义实施结构、冲突策略、风险控制和测试方法。

不可变 source base 为 `main@e9a0e4aa5`，运行版本为 `0.1.165.4`。该基线已包含上游 `v0.1.165`、此前分段合并的兼容修复，以及本地 `v0.1.165.2`~`v0.1.165.4` subscription quota cycle reset。Comet 在 Build setup 中可能先提交当前 change 的 OpenSpec、Design Doc 和 Plan，因此实际 execution base MAY 是 source base 的 planning-only 后代：`e9a0e4aa5` 必须是其祖先，两者之间只允许当前 change 的规划产物，禁止应用源码差异。上游范围、migration 历史 identity 和最终 merge 拓扑继续以 immutable source base 为准。

目标 tag 形成严格祖先链：

| 阶段 | Tag peeled SHA | 上游区间 | Commits | Files |
|---|---|---|---:|---:|
| 1 | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | `v0.1.165..v0.1.166` | 62 | 142 |
| 2 | `99c8e4bf7564823bafbab369acab6539e734c1bb` | `v0.1.166..v0.1.168` | 36 | 170 |
| 3 | `26d894ef4f50645a4bf1030e378ac892f17d0223` | `v0.1.168..v0.1.169` | 38 | 72 |

`v0.1.167` 没有正式 tag，不建立阶段。`upstream/main` 在 `v0.1.169` 后的提交不属于目标。实施前重新 fetch；若发现更高正式 tag，停止并回到 OpenSpec 更新范围。

## 2. 目标与边界

### 2.1 目标

- 三个目标 tag 按顺序成为结果 HEAD 的祖先，并各自保留可审计的 merge 节点。
- 上游功能和安全修复与本地定制共存；任何本地能力回归都归属到首次出现的 release 区间。
- 上游 `191_passkey_credentials.sql`、本地 `191_subscription_quota_advance_receipts.sql` 和 `192_subscription_cache_invalidation_outbox.sql` 原名保留。
- 最终版本精确为 `0.1.169.1`，本机自动门禁和能力级审查完成。
- Docker-backed integration 未执行时，验证报告准确列出环境原因和未验证契约。

### 2.2 非目标

- 不合入 tag 后的 `upstream/main` 提交，不创建虚构 `v0.1.167` 阶段。
- 不新增本地产品功能，不做无关重构。
- 不推送、不打 tag、不触发 GitHub Actions、不发布或构建 Sub2API 镜像。
- 不部署，不操作任何服务器、数据库、Redis 或 Nginx；生产临时盾继续保留。
- 不把静态检查、编译成功或旧 integration 证据写成当前 PostgreSQL 升级路径已通过。

## 3. 采用方案

采用“单 change、三段受审 merge、逐段封闭”。拒绝以下替代方案：

- 三个 merge 后统一验证：回归会叠加，无法可靠归属首次引入阶段。
- 先 cherry-pick GHSA 修复：本 change 不部署，没有提前获得运行时修复的收益，却会增加重复补丁和 ancestry 复杂度。
- 三个独立 changes：阶段严格线性依赖，共享能力矩阵和最终版本，拆分只会复制上下文与归档工作。

状态机为：

```text
固定阶段 0
  -> merge --no-ff --no-commit <tag>
  -> 冲突分类与语义融合
  -> 提交前高风险入口/接口审查
  -> 创建 merge commit
  -> 提交后 changed-files × 能力矩阵与行为审查
  -> 保留 RED，做最小兼容修复并独立提交
  -> 生成稳定且工作区/索引干净
  -> 本机阶段门禁
  -> 记录证据并封闭阶段
  -> 下一 tag
  -> VERSION=0.1.169.1
  -> 最终全门禁、拓扑和能力终审
```

实施隔离方式、执行模式、TDD 模式和审查模式由 Build 阶段联合决策点确认，本文不提前替用户选择。

## 4. Git 与提交边界

每段唯一允许的 merge 起点是：

```text
git merge --no-ff --no-commit <tag>
```

`--no-commit` 让无冲突 merge 也停在受审状态。提交边界如下：

- **merge commit**：只承载目标上游树和完成 merge 必需的冲突融合；第二父必须等于固定 peeled SHA。
- **兼容修复提交**：承载测试或调用链审查证明的语义回归、必要补测和源文件修复。
- **证据提交**：只更新 build ledger、任务状态和阶段结论。
- **最终版本提交**：三段封闭后一次更新 `backend/cmd/server/VERSION` 为 `0.1.169.1`。

中间阶段保持 `VERSION=0.1.165.4`，不创建 `0.1.166.1` 或 `0.1.168.1`。所有暂存使用显式路径，禁止 `git add .`；`.comet/current-change.json` 等用户/运行时文件不得混入业务提交。

## 5. 阶段 0 与能力矩阵

首次 merge 前建立 build ledger：

`docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

ledger 固定记录：source/execution base、branch/worktree、tag manifest、实际存在的 `.comet/current-change.json` runtime selection、每段 changed-files、冲突台账、能力矩阵、命令/退出码、失败与修复、阶段结论、Docker 预检和残余风险；不建立任意用户路径排除清单。

能力矩阵每行包含：行为契约、入口/调用链、关键文件、受影响 tag、聚焦测试、人工审查点、状态和证据。状态含义：

- `protected`：直接行为测试通过。
- `manual`：生成物、依赖或静态调用链证据已审查。
- `unverified`：仅用于环境原因无法执行的 integration 契约，必须带原因和影响。
- `gap`：上游触及但缺少充分证据；阶段结束前必须归零。

矩阵至少覆盖：

- advanced/layered scheduler、sticky、fallback/WaitPlan、DB recheck；
- Grok sticky、platform sticky、session/previous-response sticky、privacy 和 image capability；
- OpenAI HTTP/WS、Live、turn ownership、最终 outbound model、failed usage、prompt cache reuse、网关透传字段和 proxy circuit；
- prompt/security audit，以及 Images 只走统一审核入口、legacy moderation 每请求最多一次、线程安全 payload 最多冻结一次、关闭态不构造大 payload、运行态/范围判定后才求值、文本释放前保持可用的完整契约；
- request-body replay/spooling/cleanup、异步图片任务与对象存储、图片输入计费和上游计费倍率；
- settings 热更新、repository scoped updates、用户/API Key 更新、会话绑定与 step-up；
- subscription quota cycle reset 的单窗口资格、事务锁、receipt、版本化 tombstone 和 outbox；
- 用户资源控制、分组复制、用户批量限额、Passkey、模型广场和全部前端本地功能；后者至少包含菜单、设置、用量、订阅、渠道展示、移动端适配及上游触及的其他本地页面；
- pricing、count_tokens、release fallback 资源、部署安全配置；
- Ent/Wire、Go/pnpm 依赖和 migrations。

## 6. 冲突处理

每个冲突文件只使用六类之一：上游修复、本地定制、接口/配置演进、版本/依赖、生成代码、migration。台账记录 ours 行为、theirs 行为、融合结果和验证证据。可共存时做最小融合；无法共存且用户未批准移除的行为立即阻塞当前阶段。

审查分为两个明确时点：

1. **merge commit 前**：在未提交 merge 状态中检查全部文本冲突、构建接口、路由注册、DTO/config、provider/schema 和高风险入口，确认不存在已知不可共存语义。发现此类问题时不创建 merge commit。
2. **merge commit 后**：运行完整 changed-files × 能力矩阵、调用链和行为测试。此时发现的可修复回归保留 RED 并用独立兼容提交修复；发现需要用户取舍的语义则停在当前阶段，不开始下一 tag。

设计阶段执行的只读 `git merge-tree --write-tree HEAD v0.1.166^{}` 预测首段 17 个文本冲突，重点包括：

- `backend/cmd/server/VERSION`、`backend/go.sum`；
- settings config test 与 update handler；
- gateway Responses、OpenAI gateway、router；
- Responses 转发、usage、Gemini compat、Ollama usage；
- OpenAI WS forwarder、v2 relay/adapter；
- `frontend/src/views/admin/__tests__/UsageView.spec.ts`。

这只是首段规划输入，实际冲突集合以 Build 隔离位置中的 merge 结果为准。

特殊文件策略：

- `VERSION`：每段冲突均保留本地 `0.1.165.4`，最终再改为 `0.1.169.1`。
- Go 依赖：先融合 `go.mod` 的真实依赖需求，再使用 Go module 工具生成 `go.sum`，不手拼 checksum。
- Ent/Wire：修改 schema/provider 源后重新生成，不直接维护生成结果。
- 前端 lockfile：先融合 manifest，再用仓库现有 pnpm 版本生成。
- Migration：保留所有已发布完整文件名和内容，不因数字前缀重复而重命名。

## 7. 分段风险与审查点

### 7.1 v0.1.166

重点是 panel API rate limit、settings 部分更新、OpenAI WS 每轮模型计费、最终上游模型、composite routing 和依赖安全更新。首段冲突与本地 OpenAI WS/usage 定制高度重叠，必须验证 turn 级模型、terminal ownership、failed usage、account failover 和 effective composite route 未退化。

上游对 user/API Key/settings 更新语义的调整还必须与本地 quota reset 的事务、receipt 和缓存失效路径做调用链交叉审查；即使没有文本冲突，也不能仅凭全量测试通过放行。

### 7.2 v0.1.168

重点是 Passkey、模型广场、Kimi K3、repository scoped column updates、prompt audit 配置恢复、OpenAI Live store 容错和相关前端页面。Migration 必须同时保留上游 `191_passkey_credentials.sql` 与本地 191/192。

Passkey 会扩展 auth、user repository、settings、Wire 和前端路由；审查必须确认本地用户资源控制、隐藏菜单、settings backfill 和 quota reset 不因 scoped update 或 DTO 演进丢字段。

### 7.3 v0.1.169

重点是 GHSA-vrxq-qm4h-6hgg、no-new-privileges、release pricing fallback 资源、proxy stream circuit fail-open、Qwen3Guard 辅助字段、count_tokens、pricing 和 token refresh。

路径安全审查覆盖所有客户端可控片段进入上游 URL 拼接的入口，包括 Responses 子路径和 Gemini 模型名。闭集规则固定为：每段最多 128 bytes、后缀最多 8 段、字符仅允许 ASCII `[A-Za-z0-9_.-]`，且纯点片段拒绝。`/responses` 与 `/responses/` 等根路径的空 suffix 合法；非空 suffix 内的空 segment（如 `//double` 或 `/compact//detail`）拒绝。

Responses 负向矩阵在 `/v1/responses`、`/responses` 和 `/backend-api/codex/responses` 三类入口覆盖 raw URL 中的 `..`、`%2e`/`%2f` 编码 traversal、反斜杠、query/fragment 编码注入、双斜杠、超过 128 bytes 的单段、超过 8 段的后缀、控制字符、非 ASCII 与闭集外标点；解析后的不合规片段必须在边缘返回 `404` 和 `Unsupported responses subpath`，不得参与上游 suffix 拼接或被误判为 compact。Gemini native/compat 的不合规模型名必须在构造上游 URL 前返回错误且不发出上游请求。合法根路径、尾斜杠、`compact`、response id、合规模型名和既定 action 保持原样。必须运行上游新增 guard/route/Gemini 测试，并追踪本地 gateway route、compat bridge 和 prompt audit 是否存在绕过。该代码验证不等于生产已升级，Nginx 临时盾仍保留。

## 8. Migration 与生成稳定性

迁移执行器按完整 filename 记录、排序并校验 checksum，因此双方 `191_*` 默认可以共存。验证分两层：

1. 始终执行非 Docker 的 migration 编译、嵌入文件、排序/checksum 和文件存在性测试。
2. 本机 Docker 可用时，执行 PostgreSQL 新库与从本地 `0.1.165.4` migration 集合升级的 integration，确认双方 191、本地 192、幂等和 checksum 稳定。升级测试必须先应用包含本地 191/192、但排除上游 `191_passkey_credentials.sql` 的基线 FS，再应用完整 FS；不能只测试从空库开始。

若 Docker 不可用，第一层不能替代第二层；ledger 和最终报告将第二层标记为 `unverified`。

生成物的提交归属必须闭合：merge 期间若生成文件冲突，在源文件完成融合后重新生成，并把该输出作为完成 merge 必需的结果放入 merge commit；兼容修复改动生成源时，把对应生成输出放入同一兼容修复提交。所有源码/生成提交完成后再运行两次 `make -C backend generate`，两轮都必须无 diff，且阶段结束时 worktree 与 index 过滤实际存在的 `?? .comet/current-change.json` 后保持干净。生成不稳定时回到 schema/provider/manifest 源修复。

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

`make test` 应覆盖后端默认/unit/lint 与前端 lint/typecheck/Vitest，`make build` 覆盖前后端构建。若仓库当前 Makefile 在 Windows 需要显式 shell 或版本参数，implementation plan 使用仓库已验证的命令形态，不通过修改产品代码绕过环境问题。

每个阶段都先用轻量命令检查 `docker` 是否出现。设计阶段 `docker info` 已因找不到可执行文件失败；若阶段 0 仍缺失，中间阶段不重复完整 `docker info` 诊断，只记录轻量检查结果并引用同一 `unverified` 环境根因。任一阶段检测到 Docker 可用时，该阶段必须运行 verbose integration，并在日志中确认本阶段要求的 migration/repository top-level 测试出现真实 `--- PASS:`；命令 exit `0`、package `ok`、`no tests to run` 或 `--- SKIP:` 均不能单独作为通过证据。测试真实失败时阻塞；Docker/Testcontainers 实际不可用导致目标跳过时按 canonical spec 记为 `unverified`，不得写成 PASS。最终阶段再次检查，并在可用时补跑最终完整 integration。不得连接 `local-serv-ai` 或其他服务器补验。

Migration 的最低目标证据包括现有新库幂等/schema 测试，以及一条覆盖本地 `0.1.165.4` migration 集合到上游 `v0.1.169` 的升级测试；implementation plan 固定其最终测试名和 `--- PASS:` 匹配规则。任何目标名变更都必须同步更新 ledger，禁止用宽泛 package 成功替代。

最终报告路径为：

`docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-verify.md`

## 10. 失败处理与回退

- 基线门禁失败：在首个 merge 前停止，区分既有失败、环境阻塞和本 change 缺口。
- 文本冲突可融合：完成最小语义融合并记录直接证据。
- merge 提交前发现业务语义不可共存：不创建当前 merge commit，等待用户决定。
- merge 提交后才发现需要用户取舍的语义：保留当前 merge 与失败证据，不创建下一阶段节点，等待用户决定。
- 聚焦或 full 门禁失败：保留原始 RED，最小修复后从本段聚焦测试开始重跑。
- 生成不稳定：回到源文件修复，不提交不可复现生成物。
- Docker 不可用：只按已确认 spec 记录 `unverified`，不得写成 PASS，也不得远程补跑。
- 最终验证失败：不得进入 Verify/Archive 完成状态。

工作仍在隔离分支时，彻底回退可放弃该分支/worktree；不得使用破坏性命令覆盖主线或用户未提交改动。执行位置除 Comet runtime selection 外必须干净；若发现其他用户改动，停止并由 Comet 重建隔离位置。若未来经用户选择合入并共享历史，回退按 merge 节点与对应兼容修复提交执行 revert，不改写远端历史。

## 11. 完成条件

- 三个正式 tag 均为结果 HEAD 祖先，三个 merge 第二父与固定 SHA 一致。
- `backend/cmd/server/VERSION` 精确为 `0.1.169.1`。
- 能力矩阵无 `gap`；所有 `unverified` 都仅来自已记录的本机 integration 环境边界。
- GHSA 路径 guard 及三个阶段命中的本地关键行为有自动或结构证据。
- Images 统一入口、legacy 单次执行、线程安全单次冻结、关闭态零构造和文本释放前生命周期均有直接行为证据。
- 双方 191 与本地 192 文件保留；真实 PostgreSQL 升级验证只在实际执行成功时记为通过。
- 聚焦测试、本机 `make test`、`make build`、两轮 generate 和静态检查在最终 source HEAD 上通过。
- 每段结束时 worktree/index 过滤实际存在的 `?? .comet/current-change.json` 后干净，生成输出均归属于对应 merge 或兼容修复提交。
- OpenSpec 严格校验和 Comet Verify 通过或以流程允许的明确偏差结论结束。
- 没有推送、发版、部署、服务器操作或 Nginx 修改。

## 12. Implementation Divergence

本节记录 delta spec 与 design doc 之间的偏差及实际执行结果，供归档时审计。

- **偏差场景**：delta spec「从未合入默认分支的已验证分支发布测试版本」（`specs/upstream-release-sync/spec.md`）描述了用户明确授权后从已验证分支发布版本并验收的流程要求；本 design doc 第 2.2 节将「不推送、不打 tag、不触发 GitHub Actions、不发布或构建镜像」列为非目标。两者并非同一约束：非目标限定的是 change 内部 build/verify 阶段的默认执行边界，spec 场景描述的是用户另行授权发布时的流程语义。
- **实际执行**：2026-08-03 用户明确授权「提交并发布下一个版本」后，`main` 快进至最终验证 HEAD（`2e3264905`），创建并推送 annotated tag `v0.1.169.1`；Release workflow（run `30776732045`）成功产出 `sub2api_0.1.169.1_linux_amd64.tar.gz` 与 `checksums.txt`，GHCR `0.1.169.1` 与 `latest` 指向同一 digest，`sync-version-file` 验证 tag commit 为 default branch HEAD 祖先后完成 VERSION 同步。
- **结论**：该场景已按 delta spec 语义实际执行并通过，无需修改第 2.2 节表述；此节仅作为验证阶段追加的漂移记录（verify 允许产物），不改变任何实现。
