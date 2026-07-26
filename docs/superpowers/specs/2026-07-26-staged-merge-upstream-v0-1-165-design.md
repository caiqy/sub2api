---
comet_change: staged-merge-upstream-v0-1-165
role: technical-design
canonical_spec: openspec
---

# 分段合并上游 v0.1.165 技术设计

## 1. 背景与边界

本地基线为 `main@075abc073`，已包含上游 `v0.1.159`，运行时版本文件 `backend/cmd/server/VERSION` 为 `0.1.159.6`。本地主线在长期演进中叠加了 scheduler、Sticky、请求体生命周期、privacy、用户资源控制和前端本地功能等定制，不能用 `v0.1.159..HEAD` 的提交数简单表示本地修复数量。

截至 2026-07-26，目标 release 链如下：

| 阶段 | 上游区间 | commits | files | peeled SHA |
|---|---|---:|---:|---|
| 1 | `v0.1.159..v0.1.160` | 24 | 133 | `8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62` |
| 2 | `v0.1.160..v0.1.161` | 62 | 257 | `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c` |
| 3 | `v0.1.161..v0.1.162` | 114 | 190 | `27f094e0960ebd8e52de7ff7e763c6fec2ff4057` |
| 4 | `v0.1.162..v0.1.163` | 69 | 171 | `d0bdd7e771636a8d315f542cafd39484f39bd60c` |
| 5 | `v0.1.163..v0.1.164` | 43 | 202 | `cd8bb98c44303b2c8f04c0da340447c992f0cb7d` |
| 6 | `v0.1.164..v0.1.165` | 54 | 168 | `e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29` |

六个 tag 构成严格祖先链。当前 `upstream/main@2730c1c43` 比 `v0.1.165` 多一个 `chore: sync VERSION to 0.1.165` 提交；它不是正式 release 目标。实施前必须重新 fetch：若出现新的正式 tag，暂停并更新 OpenSpec 范围，不静默扩展。

本 change 不推送、不打 tag、不触发 Release workflow、不部署，也不在任何服务器构建 Sub2API 镜像。`local-serv-ai` 只承担 Testcontainers integration，不接触服务运行目录。

## 2. 方案选择

### 2.1 采用：单 change 六段合并

在一个隔离分支中按 tag 顺序建立六个 `--no-ff` merge 节点。每段拥有独立冲突台账、能力矩阵结论和 full 门禁；失败停在首次出现问题的 release 区间。

该方案保留了上一轮 staged merge 的有效属性，同时把证据收敛为一份持续更新的 build ledger 和一份最终验证报告，避免为每个小任务创建碎片文档。

### 2.2 不采用的方案

- 一次合入 `v0.1.165`：提交更少，但无法定位首次回归版本，也不满足逐版本门禁。
- 每个 tag 独立 change：隔离更强，但六段严格线性依赖，拆分会重复基线、规格、归档和能力矩阵，不能独立交付。

## 3. Git 拓扑与阶段状态机

实际 merge 使用：

```text
git merge --no-ff --no-commit <tag>
```

`--no-commit` 确保无文本冲突时也先停下，允许在形成 merge commit 前核对 `VERSION`、生成源、冲突分类和高风险入口。状态机为：

```text
确认上一阶段已封闭
  -> merge --no-ff --no-commit <tag>
  -> 文本冲突融合与冲突台账
  -> 创建 merge commit
  -> changed-files × 能力矩阵审查
  -> 保留失败测试并做最小兼容修复
  -> 提交修复与生成结果
  -> 本地 full 门禁
  -> local-serv-ai integration 门禁
  -> 记录证据并封闭阶段
  -> 允许下一 tag
```

### 3.1 提交边界

- merge commit：只承载上游树和为完成 merge 必需的冲突融合；第二父必须等于目标 peeled SHA。
- 兼容修复提交：承载 merge 后由测试或语义审查发现的回归，按单一行为聚焦。
- 阶段证据：写入同一 build ledger；每段封闭前持久化，不创建逐任务报告文件。
- 最终元数据提交：六段通过后将 `backend/cmd/server/VERSION` 从 `0.1.159.6` 一次改为 `0.1.165.1`。

中间阶段不采用 tag 内落后一版的三段式 `VERSION`，也不创建 `0.1.160.1` 等过程版本。阶段身份由 peeled SHA、merge 第二父和台账证明。

## 4. 分段风险面

| Tag | 重点风险 | 新增 migration |
|---|---|---|
| `v0.1.160` | security-audit full prompt 与本地 privacy；Grok media 隔离；image capability | `181_prompt_audit.sql`、`182_prompt_audit_full_prompt.sql` |
| `v0.1.161` | step-up 2FA；模型级临时冷却与本地 scheduler；Grok 视频代理 | `183_ops_ingress_reject_aggregates.sql`、`184_auth_cache_invalidation_outbox.sql` |
| `v0.1.162` | 客户端 IP/可信代理；settings backfill 与热更新；Grok client tool cache/Sticky；image storage | 无 |
| `v0.1.163` | reasoning policy；scheduler quota metadata/LastUsedAt；Cleanup；计费与 axios | `185_group_reasoning_effort_policy.sql` |
| `v0.1.164` | composite group routing 与 advanced/layered scheduler、Grok/platform Sticky；Ollama Cloud；Alipay | `172_composite_model_routes.sql`、两个 `186_*` |
| `v0.1.165` | OpenAI Live 与 prompt cache/body replay；Ollama 用量；email alias；postcss | `187_*`、`188_*`、`189_*`、`190_*_notx.sql` |

风险表只是审查起点。每段仍以实际 changed-files 与本地能力矩阵的交集为准，不因表中未列出就跳过受影响能力。

## 5. 能力矩阵与语义审查

能力矩阵来源为：

1. `openspec/specs/` 的本地主规格；
2. `memory/context/upstream-merge-workflow.md` 与知识库；
3. `v0.1.159` staged merge 的能力矩阵和验证报告；
4. `main@075abc073` 上仍生效的本地行为测试。

每行记录行为契约、入口/调用链、关键文件、受影响 tag、自动测试、人工审查点、阶段结果和证据。状态只使用：

- `protected`：直接行为测试通过；
- `gap`：上游触及但缺少断言，必须先补最小测试；
- `manual`：生成物、migration、依赖或跨层契约需要结构证据；
- `approved-removal`：仅允许已确认移除的本地 `openai-first-token-timeout`。

最终不得残留 `gap`。重点能力至少覆盖：advanced/layered scheduler、fallback/WaitPlan、DB recheck、Grok/platform Sticky、privacy、image capability、异步图片与对象存储、图片/视频计费、上游倍率、session/step-up、runtime 热更新、网关透传、prompt cache、body replay/spooling、失败 usage、用户资源控制、公开分组屏蔽、菜单隐藏、前端翻译、quota 原子重置、settings backfill 和 local test gates。

### 5.1 冲突台账

每个冲突文件至少记录：类别、ours 行为、theirs 行为、融合结果、验证证据。类别限制为上游修复、本地定制、接口/配置演进、版本/依赖、生成代码和 migration。

可共存时做最小融合；无法共存且不属于已批准移除项时，停止当前阶段并等待用户决定。已验证过的历史决策可以复用其业务结论，但不能机械复用 ours/theirs。

### 5.2 无文本冲突

Git 没有报告冲突时，仍检查 changed-files 是否触及入口、条件、DTO、配置解析、运行时缓存、scheduler factory、route registry、provider、schema 或生成结果。结构审查确定调用链影响，行为测试验证最终语义。

## 6. 生成文件与依赖

- Ent：以 `backend/ent/schema/` 和生成入口为源，不手改生成结果。
- Wire：以 provider 声明和 `wire.go` 为源，不手改 `wire_gen.go`。
- 前端 lockfile：先融合 `frontend/package.json` 等 manifest，再使用仓库现有 pnpm 版本重算。
- Go 依赖：保留上游安全升级与本地实际依赖，使用 Go module 工具更新 `go.mod`/`go.sum`，不手工拼接 checksum。

每段修复提交后运行两次 `make -C backend generate`；第一次不得产生未解释 diff，第二次必须完全稳定。

## 7. Migration 兼容设计

上游区间新增 12 个文件：

```text
172_composite_model_routes.sql
181_prompt_audit.sql
182_prompt_audit_full_prompt.sql
183_ops_ingress_reject_aggregates.sql
184_auth_cache_invalidation_outbox.sql
185_group_reasoning_effort_policy.sql
186_alipay_mobile_precreate_deep_link.sql
186_group_auth_cache_image_generation.sql
187_add_usage_log_session_id.sql
188_allow_live_usage_request_type.sql
189_add_group_allow_live.sql
190_add_users_email_alias_dedup_index_notx.sql
```

本地已有 `172_video_per_second_billing_metadata.sql` 和 `181_group_duplicate_operation_id.sql`。迁移执行器按完整 filename 作为主键、按完整文件名排序并校验 checksum，因此同数字前缀不等于数据库记录冲突。设计默认保留全部文件，不重命名已发布 migration。

新增一条聚焦 integration 回归：

1. 从 embedded migration FS 构造过滤视图，排除本次 12 个上游文件；
2. 在隔离 PostgreSQL 数据库应用该基线，模拟已运行 `0.1.159.6` 的本地实例；
3. 断言本地 172/181 已记录；
4. 使用完整 embedded FS 再执行 migration；
5. 断言当前阶段已存在的上游新增文件均被记录，最终阶段断言 12/12；
6. 重复执行确认幂等和 checksum 稳定；
7. 对 `190_*_notx.sql` 保留并验证非事务执行规则。

该测试直接使用现有 `applyMigrationsFS` 和 integration harness，不新增 migration runner 抽象。

## 8. 验证设计

### 8.1 本地门禁

阶段 0 和每个 tag 均执行：

```text
make test
make build
make -C backend generate
make -C backend generate
git diff --check
```

并运行能力矩阵命中的聚焦测试、冲突标记扫描和生成 diff 检查。`make test` 覆盖后端默认/unit/lint 与前端 lint/typecheck/Vitest；`make build` 覆盖后端和前端构建。

### 8.2 local-serv-ai Integration 门禁

当前工作站没有 Docker。每段在 merge 和兼容修复均已提交、本地门禁通过后：

1. 用 `git archive HEAD` 生成仅包含已提交源码的临时归档；
2. 通过 `ssh-skill` 上传到 `local-serv-ai` 的唯一临时目录；
3. 预检远程 Go 版本满足当前 `backend/go.mod`、`make` 可用、`docker info` 成功；
4. 使用 `CI=true make -C backend test-integration`，使 Docker 不可用时的跳过路径转为失败；
5. 保存退出码和 verbose 日志；阶段 0 确认既有 migration schema integration test PASS，v0.1.160 起确认渐进式 upgrade test PASS；无关环境型 skip 记录但不自动判定失败；
6. 通过 `ssh-skill` 清理临时源码、归档和残留测试容器。

Testcontainers 只拉取仓库既有的 PostgreSQL/Redis 测试镜像。禁止构建 Sub2API 镜像，禁止复用生产数据库/Redis，禁止写入 Sub2API 服务目录。任何远程前置失败、全套 integration 非零、目标 migration/repository test 未 PASS 或清理异常都记录为阶段阻塞；其他 skip 必须记入 ledger，并在命中本次能力时按普通失败处理。

### 8.3 最终验证

最终重复全部本地与远程门禁，并检查：

- `v0.1.160`~`v0.1.165` 均为结果 HEAD 祖先；
- 六个 merge 的第二父分别等于固定 peeled SHA；
- `backend/cmd/server/VERSION` 精确为 `0.1.165.1`；
- 冲突标记和 `git diff --check` 无问题；
- 12 个上游 migration 与本地同号 172/181 均保留，新库/升级库通过；
- 旧 `openai-first-token-timeout` 配置、错误和 watchdog 无业务代码残留；
- 能力矩阵无 `gap`，每项有自动或人工证据；
- OpenSpec 校验通过；
- 前端启动本地 dev server，通过 Chrome DevTools 烟测本次变更触及的关键后台页面，无控制台或关键网络错误。

## 9. 错误处理与回退

- 基线失败：停在首个 merge 前，先区分既有失败与环境阻塞。
- 文本冲突可融合：完成最小融合并记录直接证据。
- 业务语义不可共存：不创建下一 merge 节点，等待用户决定。
- 阶段门禁失败：保留可复现失败，最小修复后从该阶段完整重跑。
- Docker/远程工具链不可用、全套 integration 失败或目标 migration/repository test 未 PASS：阶段阻塞，不降级为 manual pass。
- 生成不稳定：回到 schema/provider/manifest 源修复，不维护不可复现生成物。
- 最终验证失败：不得进入 archive 或分支收尾。

合并仍在隔离分支时，回退方式是放弃该分支/工作区。若后续经用户选择合入主线，则按 merge 节点和对应兼容修复 revert，不改写已推送历史。

## 10. 证据与完成条件

实施期只新增两份主要证据：

- `docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`：六段冲突台账、能力矩阵、命令与结果；
- `docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-verify.md`：最终验证结论、残余风险和未执行项。

完成要求：六个 tag 顺序成为 HEAD 祖先；29 项 OpenSpec tasks 完成；每段本地与远程 full 门禁真实执行并通过；同号 migration、新上游能力与本地定制均有证据；`VERSION=0.1.165.1`；唯一批准移除项保持移除；不存在未解释回归、未处理 `gap` 或被当作通过的跳过项。
