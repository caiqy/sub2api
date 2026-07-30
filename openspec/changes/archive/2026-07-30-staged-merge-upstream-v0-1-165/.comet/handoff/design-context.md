# Comet Design Handoff

- Change: staged-merge-upstream-v0-1-165
- Phase: design
- Mode: compact
- Context hash: d98e1e10b56d7789525693c8964682f1ecfc9d6a6675803a087da0ddd1eb5e6c

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/staged-merge-upstream-v0-1-165/proposal.md

- Source: openspec/changes/staged-merge-upstream-v0-1-165/proposal.md
- Lines: 1-32
- SHA256: 9249e133558cea7574e94b8fdb15df46c14ffb8aadc01dccf520b77335f1aed8

```md
## Why

截至 2026-07-26，本地主线已包含上游 `v0.1.159`，`backend/cmd/server/VERSION` 为 `0.1.159.6`，并叠加长期本地定制；上游已发布 `v0.1.160` ~ `v0.1.165` 六个后续 release tag，覆盖 security-audit prompt 审计、客户端 IP 请求头体系、composite group routing、OpenAI Live gateway 等大量新能力与修复（每段 24~114 commits、133~257 文件，并新增 12 个 SQL migration 文件）。需要按既有 staged-merge 经验逐版本合入，在不回归本地定制能力的前提下吸收上游演进。

## What Changes

- 从 `main`（`0.1.159.6`）切出临时合并分支，依次使用 `--no-ff` 合入 `v0.1.160`、`v0.1.161`、`v0.1.162`、`v0.1.163`、`v0.1.164`、`v0.1.165` 六个上游 tag。
- 实施前重新 fetch upstream tags 并确认 `v0.1.165` 仍是最新正式 release；若出现更新 tag，暂停并更新 change 范围，不静默跳过。当前 `upstream/main` 仅比 `v0.1.165` 多一个 `VERSION` 同步提交，不作为 release 合并目标。
- 每段合并后运行 **full 门禁**（根目录 `make test` + `make build`、Docker-backed integration、Ent/Wire 两次生成稳定性、migration 集合兼容性、冲突标记检查），并按本地能力矩阵做能力级审查；全部通过后才进入下一段。
- 冲突处理优先"保留上游变更 + 保留本地定制"共存；真实语义冲突无法共存时暂停等待用户决定。
- 对无文本冲突但可能语义覆盖的重点区域（scheduler/sticky、image capability、privacy、runtime 热更新、composite routing 与本地调度定制的交互、OpenAI Live 与本地 OpenAI 定制的交互）做专项 review。
- **BREAKING** 延续既有决定：本地 `openai-first-token-timeout` 契约保持已移除状态，后续 tag 不得恢复旧实现或兼容别名。
- 最终将 `backend/cmd/server/VERSION` 规范为 `0.1.165.1` 并完成 full verify；**不包含**推送、打 tag、发版或部署（验证通过归档后另行发版）。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `upstream-release-sync`: 将分段合并推进到 `v0.1.160`~`v0.1.165` 六个正式 tag；每段从"轻门禁"升级为"full 门禁"（`make test` + `make build` + Docker-backed integration + 生成稳定性/migration 兼容性检查），最终仍需整体 full verify。

## Impact

- Git 历史：新增 6 个按顺序关联正式 tag 的 `--no-ff` merge 节点，以及必要的本地兼容修复提交。
- 后端：security-audit prompt 审计、客户端 IP 请求头解析与可信代理、模型级临时冷却、Grok media/视频代理与 client tool 缓存、composite group routing、ollama Cloud 用量、OpenAI Live gateway、email alias 注册查重、billing 计费口径调整、优雅关停 Cleanup、golang.org/x 依赖升级。
- 前端：客户端 IP 设置界面、step-up 2FA 开关、S3 备份/image storage 配置卡、移动端与 iOS 适配、审计 UI、Alipay deep link、axios/postcss 安全升级。
- 数据库：新增 12 个 migration 文件（172、181-190，其中有两个 `186_*`）；上游 `172_composite_model_routes.sql`、`181_prompt_audit.sql` 分别与本地既有 `172_video_per_second_billing_metadata.sql`、`181_group_duplicate_operation_id.sql` 同号不同名，必须验证按完整文件名记录的迁移执行器在新库和升级库上的行为，不得仅检查编号连续性。
- 本地保护清单：以主 spec 的本地关键能力专项 review 清单为下限，并重点覆盖 advanced/layered scheduler、fallback/WaitPlan 与 DB recheck、Grok/platform sticky、privacy 与 image capability、runtime 热更新、OpenAI prompt cache reuse、网关透传、body replay/request spooling、公开分组屏蔽、用户菜单隐藏、admin 资源控制、前端翻译、subscription quota 原子重置、settings JSON backfill、local test gates。
- 验证：6 段独立 full 门禁记录 + 最终 full verify 报告。

```

## openspec/changes/staged-merge-upstream-v0-1-165/design.md

- Source: openspec/changes/staged-merge-upstream-v0-1-165/design.md
- Lines: 1-55
- SHA256: 58d27f40a669287040daf9e924a36cf7325aca8a74d796236e6a618a68e44914

```md
## Context

- 本地 `main` 已包含上游 `v0.1.159`，`backend/cmd/server/VERSION` 为 `0.1.159.6`，并叠加长期本地定制；不能用 `v0.1.159..HEAD` 的提交数代表本地修复数量，因为其中包含长期分支与归档历史。
- 上游新增 6 个 release tag：`v0.1.160`(24c/133f) → `v0.1.161`(62c/257f) → `v0.1.162`(114c/190f) → `v0.1.163`(69c/171f) → `v0.1.164`(43c/202f) → `v0.1.165`(54c/168f)。
- 六个 tag 构成严格祖先链；当前 `upstream/main` 比 `v0.1.165` 多 1 个 `chore: sync VERSION to 0.1.165` 提交，因此目标锁定正式 tag，而不是 `upstream/main`。
- 既有经验（`2026-07-17-staged-merge-upstream-v0-1-159`）：逐 tag `--no-ff` + 分段门禁 + 能力矩阵审查是可行范式；"测试通过 ≠ 本地能力未被改写"，必须做能力级专项 review。
- 高风险交互区：v0.1.164 的 composite group routing 触及调度/路由核心（与本地 advanced/layered scheduler、Grok sticky、platform sticky 定制交叠）；v0.1.165 的 OpenAI Live gateway 触及 OpenAI 网关路径（与本地 prompt cache reuse、body replay 交叠）；12 个新增 migration 中，上游 172/181 与本地同号不同名，且上游自身存在两个 `186_*`。

## Goals / Non-Goals

**Goals:**

- 6 个上游 tag 全部按顺序合入，Git 祖先关系正确（`v0.1.165` 成为 HEAD 祖先）。
- 每段合并独立通过 full 门禁：根目录 `make test`、`make build`、Docker-backed integration、Ent/Wire 两次生成稳定、migration 新库/升级库兼容、无冲突标记。
- 本地保护清单内的每项能力在最终 HEAD 上仍然成立（能力级验收，不止文件级 diff）。
- `backend/cmd/server/VERSION` 最终规范为 `0.1.165.1`，全量 full verify 通过。

**Non-Goals:**

- 不推送远端、不打 tag、不触发 Release workflow、不部署（归档后另行发版）。
- 不恢复已移除的 `openai-first-token-timeout` 契约。
- 不在本 change 内新增本地功能或重构上游代码。

## Decisions

1. **临时分支承接合并**：全部 6 段在 build 阶段确认的隔离分支完成，final verify 通过后再决定合回方式；分支名与工作方式在 build 决策点联合确认。理由：隔离语义风险，沿用上次成功范式。
2. **逐 tag `--no-ff` 合并而非一次合 `v0.1.165`**：保留每段独立 merge 节点，冲突分摊、回溯粒度小；与用户"逐版本更新"要求一致。
3. **每段 full 门禁（升级自上次的轻门禁）**：每段执行根目录 `make test`（后端默认/unit/lint + 前端 lint/typecheck/Vitest）、`make build`（后端与前端构建），并执行 Ent/Wire 两次生成稳定性检查；失败即停在当段修复，不带病进入下一段。
4. **冲突处理默认"两边共存"**：机械 ours/theirs 禁止；先分类（上游修复/本地定制/接口演进/生成文件），生成文件（Ent/Wire/pnpm-lock）冲突以重新生成为准；无法共存的语义冲突暂停交用户决定。
5. **能力矩阵沿用并扩展**：以主 spec 的专项 review 清单和本地保护能力为行、6 个 tag 为列，每段合并后勾验受影响单元格；composite routing 与 Live gateway 两个新能力列入"与本地定制交互"专项审查。
6. **复杂行为问题走失败测试驱动**：涉及 scheduler、sticky、fallback、runtime config 的回归先补失败测试再最小修复，不直接猜改。
7. **保留同号不同名 migration**：迁移执行器按完整文件名记录并按文件名排序，因此默认同时保留本地和上游 172/181 以及两个上游 186 文件，不擅自重命名已发布 migration；通过新库和已有本地 migration 记录的升级库测试验证依赖顺序，失败时再做最小兼容修复。
8. **保持单 change**：六个 tag 是严格线性依赖，任一阶段失败都必须阻塞后续阶段，且共享同一能力矩阵与最终版本；拆成多个 change 只会复制门禁和上下文，不能独立交付或归档。
9. **Integration 使用 local-serv-ai**：本机没有 Docker；每段把已提交 HEAD 通过 `git archive` 打包，并由 `ssh-skill` 上传到 `local-serv-ai` 临时目录运行 `CI=true make -C backend test-integration`。只拉取 PostgreSQL/Redis Testcontainers 镜像，不构建 Sub2API 镜像、不部署、不触碰服务运行目录。

## Risks / Trade-offs

- [composite group routing 改写调度入口，静默绕过本地 advanced/layered scheduler 定制] → 每段审查调度入口调用链；针对 Grok sticky + advanced scheduler 保留既有本地测试并要求持续绿灯。
- [OpenAI Live gateway 重构 OpenAI 转发路径，破坏本地 prompt cache reuse / body replay] → v0.1.165 段专项 diff 审查 + 定向测试。
- [同号不同名 migration 因排序或依赖关系在升级库失败] → 保留完整文件名，分别验证空库与已应用本地 172/181 的升级库；确认 `190_*_notx.sql` 非事务执行路径。
- [客户端 IP 体系与本地安全/审计定制交叠] → v0.1.162 段核对 settings JSON backfill 与配置热更新路径。
- [每段 full 门禁耗时高] → 接受；失败停段策略避免时间浪费在带病推进上。
- [release tag 内 `VERSION` 比 tag 名低一个三段式版本，可能误降本地版本] → 每段把版本元数据作为独立冲突决策记录，不用它推断 release 内容；最终统一设为 `0.1.165.1`。
- [远程 integration 环境缺少正确 Go 工具链、Docker 或测试被跳过] → 每段远程执行前检查 Go 版本与 `docker info`，设置 `CI=true` 令跳过路径失败；任一前置不满足即阻塞。

## Migration Plan

1. 本 change 只集成源码，不执行生产部署或生产数据库 migration。
2. 每段在隔离分支验证 migration：保留所有已发布文件及校验和，通过 `ssh-skill` 在 `local-serv-ai` 临时目录运行空库和已有本地 migration 记录的升级路径；不得为消除编号重复而重命名历史文件。
3. 任一阶段失败即停在当前 merge 节点修复；尚未合入主线时可直接放弃隔离分支，不需要生产回滚。

## Open Questions

- 文本冲突和语义回归的准确数量只能在逐 tag merge 时确定；无法共存的业务语义仍须暂停交用户决定。
- 172/181 同号 migration 当前由完整文件名隔离，是否还需顺序兼容修复以升级库测试结果为准，不提前改写 migration runner。

```

## openspec/changes/staged-merge-upstream-v0-1-165/tasks.md

- Source: openspec/changes/staged-merge-upstream-v0-1-165/tasks.md
- Lines: 1-52
- SHA256: 4f79e2f43f383f238d5f2454212beeeed0a90676eb3fccbfbf5edc4ee0906a5e

```md
## 1. 阶段 0：基线与能力保护门禁

- [ ] 1.1 记录 base ref `075abc073` 与 `backend/cmd/server/VERSION=0.1.159.6`，重新 fetch upstream tags，确认六个目标 tag 的祖先链且 `v0.1.165` 仍为最新正式 release；若出现更新 tag，暂停更新范围
- [ ] 1.2 按 build 阶段确认的隔离方式创建工作分支/工作区，归属 Comet 规划产物且不夹带 `paseo.json` 等无关改动
- [ ] 1.3 在当前本地 HEAD 上运行本地 full 门禁基线：根目录 `make test` 与 `make build`；失败项记录为阻塞
- [ ] 1.4 通过 `ssh-skill` 将已提交 HEAD 的 `git archive` 上传到 `local-serv-ai` 临时目录，检查 Go 工具链与 Docker，运行 `CI=true make -C backend test-integration`；跳过或前置不满足均阻塞
- [ ] 1.5 运行两次 `make -C backend generate` 并核对 Ent/Wire 第二次无新增 diff，记录生成稳定性基线
- [ ] 1.6 建立能力映射矩阵：以主 spec 专项 review 清单和本地保护能力为行、6 个 tag changed files 为列，标记高风险交叉点
- [ ] 1.7 核对被上游触及且缺少行为断言的本地能力，必要时先补最小失败测试

## 2. 合入 v0.1.160

- [ ] 2.1 `git merge --no-ff v0.1.160`，按"上游修复+本地定制共存"原则处理冲突（重点：security-audit full prompt 与本地 privacy、Grok media 隔离、image_gen 权限、migration 181/182 及本地同号 181）
- [ ] 2.2 运行 full 门禁（`make test`、`make build`、`local-serv-ai` Docker-backed integration、两次 backend generate、migration 新库/升级库、无冲突标记）并修复回归
- [ ] 2.3 按能力矩阵完成 v0.1.160 触及能力的映射审查并记录证据

## 3. 合入 v0.1.161

- [ ] 3.1 `git merge --no-ff v0.1.161`，处理冲突（重点：step-up 2FA 开关化、模型级临时冷却与本地 scheduler、Grok 视频代理、migration 183/184）
- [ ] 3.2 运行 full 门禁并修复回归
- [ ] 3.3 完成 v0.1.161 触及能力的映射审查并记录证据

## 4. 合入 v0.1.162

- [ ] 4.1 `git merge --no-ff v0.1.162`，处理冲突（重点：客户端 IP 请求头与可信代理体系、Grok client tool 缓存与 sticky、S3 备份/image storage）
- [ ] 4.2 运行 full 门禁并修复回归
- [ ] 4.3 完成 v0.1.162 触及能力的映射审查，核对 settings JSON backfill 与配置热更新路径并记录证据

## 5. 合入 v0.1.163

- [ ] 5.1 `git merge --no-ff v0.1.163`，处理冲突（重点：OpenAI reasoning policy、scheduler quota metadata/LastUsedAt、优雅关停 Cleanup、计费修复、axios 安全升级、migration 185）
- [ ] 5.2 运行 full 门禁并修复回归
- [ ] 5.3 完成 v0.1.163 触及能力的映射审查并记录证据

## 6. 合入 v0.1.164

- [ ] 6.1 `git merge --no-ff v0.1.164`，处理冲突（重点：composite group routing、ollama Cloud、Grok 402 冷却、Alipay deep link、migration 172 与本地同号 172、两个上游 186）
- [ ] 6.2 运行 full 门禁并修复回归
- [ ] 6.3 专项审查 composite group routing 入口调用链与本地 advanced/layered scheduler、Grok/platform sticky 的交互，确认本地调度定制未被绕过并记录证据

## 7. 合入 v0.1.165

- [ ] 7.1 `git merge --no-ff v0.1.165`，处理冲突（重点：OpenAI Live gateway、ollama 用量刷新、email alias 注册查重、migration 187-190、postcss 安全升级）
- [ ] 7.2 运行 full 门禁并修复回归
- [ ] 7.3 专项审查 OpenAI Live gateway 与本地 prompt cache reuse、body replay 的交互，确认本地 OpenAI 定制仍生效并记录证据

## 8. 最终验证与收尾

- [ ] 8.1 将 `backend/cmd/server/VERSION` 规范为 `0.1.165.1` 并确认 `openai-first-token-timeout` 未被任何 tag 恢复
- [ ] 8.2 运行最终 full verify：`make test`、`make build`、`local-serv-ai` Docker-backed integration、Ent/Wire 两次生成稳定性检查
- [ ] 8.3 校验 Git 祖先关系（`v0.1.160`~`v0.1.165` 均为 HEAD 祖先）、6 个 `--no-ff` merge 节点、无冲突标记残留；确认 12 个上游 migration 与本地同号 172/181 均保留且新库/升级库验证通过
- [ ] 8.4 按本地保护清单逐项完成能力级专项 review 并输出验证报告（不含推送/发版/部署）

```

## openspec/changes/staged-merge-upstream-v0-1-165/specs/upstream-release-sync/spec.md

- Source: openspec/changes/staged-merge-upstream-v0-1-165/specs/upstream-release-sync/spec.md
- Lines: 1-43
- SHA256: d3c51f18864517b5ce1d169d577f3260f742045a6fab3b62090e3ba18cf6290d

```md
## MODIFIED Requirements

### Requirement: 按正式 release tag 分段集成
维护流程 SHALL 允许将一个最终上游 release 目标拆为具有严格祖先顺序的多个正式 tag 阶段。每个阶段 MUST 完成冲突处理、能力审查和阶段验证后，才能进入下一阶段。

#### Scenario: 顺序合入多个 tag
- **WHEN** 用户选择按 `v0.1.160`、`v0.1.161`、`v0.1.162`、`v0.1.163`、`v0.1.164`、`v0.1.165` 分段集成
- **THEN** 维护流程 MUST 按该顺序建立独立 `--no-ff` merge 节点，且不得跳过尚未完成验证的前置阶段

#### Scenario: 从已验证但未归档的中间 release 继续扩展
- **WHEN** 一个分段合并 change 已通过中间 release 的最终验证但尚未归档，且用户将目标扩展到后续正式 tag
- **THEN** 维护流程 MUST 保留已完成任务和验证报告作为历史证据，使旧验证结果对新增范围失效，并在追加 merge 前重新运行基线与能力映射门禁

#### Scenario: 某阶段首次出现本地能力回归
- **WHEN** 阶段验证发现阶段 0 已保护的本地能力不再成立
- **THEN** 维护流程 MUST 在当前 release 区间内保留失败证据并完成最小修复，不得继续合入下一 tag

### Requirement: 合并后验证本地关键能力
维护流程 SHALL 在每个分段 merge 后运行完整分段门禁（后端默认/unit/lint、前端 lint/typecheck/Vitest、前后端构建、Docker-backed integration、生成代码稳定性、migration 新库与升级库兼容性、冲突标记检查）及该阶段受影响能力的能力级审查，并在最终阶段执行完整自动验证和本地能力专项 review。测试通过 MUST NOT 替代能力级审查结论。

#### Scenario: 分段 full 门禁通过
- **WHEN** 一个目标 tag 的 merge、冲突处理和兼容修复完成
- **THEN** 维护流程 MUST 运行根目录 `make test` 与 `make build`、在 Docker 可用且跳过路径会失败的环境中运行 `make -C backend test-integration`、Ent/Wire 两次生成稳定性检查、migration 新库与已有本地记录升级路径、冲突标记检查，并完成该 tag 触及能力的映射审查，全部通过后才能进入下一阶段

#### Scenario: 最终自动验证通过
- **WHEN** 最终目标 tag 合并完成且无冲突残留
- **THEN** 维护流程 MUST 运行后端默认与 unit 测试、后端 lint、前端 ESLint、前端单测、前端类型检查和构建验证

#### Scenario: 本地关键能力专项 review
- **WHEN** 自动验证完成
- **THEN** 维护流程 MUST 逐项复核 scheduler、各平台 sticky、fallback/WaitPlan、DB recheck、privacy、image capability、异步图片任务与对象存储、图片输入计费、上游计费倍率、会话绑定与 step-up、runtime setting 热更新、网关透传字段、请求体重放与清理、用户资源控制、分组复制、用户批量限额、前端本地功能、版本依赖、生成代码和 migrations，并记录每项证据

#### Scenario: 新增上游能力与本地定制交互专项审查
- **WHEN** 目标 release 区间引入触及调度、路由或网关转发核心路径的新上游能力（如 composite group routing、OpenAI Live gateway）
- **THEN** 维护流程 MUST 审查该能力入口调用链与本地定制（advanced/layered scheduler、Grok/platform sticky、prompt cache reuse、body replay）的交互，并记录不被绕过或改写的证据

#### Scenario: 同号不同名 migration 兼容
- **WHEN** 上游新增 migration 与本地已发布 migration 使用相同数字前缀但文件名和用途不同
- **THEN** 维护流程 MUST 保留双方完整文件名与既有校验和，验证迁移执行器在空库和已应用本地 migration 的升级库上正确执行全部文件，不得仅因数字前缀重复而重命名历史 migration

#### Scenario: Integration 运行环境不可用或测试被跳过
- **WHEN** Docker/Testcontainers 运行环境不可用、工具链前置不满足，或 integration tests 进入跳过路径
- **THEN** 当前阶段 MUST 记录为阻塞且 MUST NOT 记为门禁通过，也不得进入下一 release tag

```
