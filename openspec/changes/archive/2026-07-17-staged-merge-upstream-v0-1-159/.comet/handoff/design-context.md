# Comet Design Handoff

- Change: staged-merge-upstream-v0-1-156
- Phase: design
- Mode: compact
- Context hash: 50e6e6fbd6c4e02c2cf98d751d54004003f874b16db7d8ac48de739c021a7aa3

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/staged-merge-upstream-v0-1-156/proposal.md

- Source: openspec/changes/staged-merge-upstream-v0-1-156/proposal.md
- Lines: 1-30
- SHA256: bdea95d4f7d2cf7d29858d2763008ef30edec8e6d00ab8d2f211535355b9dae5

```md
## Why

本地分支与上游自 `v0.1.151` 后已分别积累大量提交，并共同修改网关、调度、运行时设置和生成代码等高风险区域。一次性合入 `v0.1.156` 难以定位无文本冲突的语义回归，因此需要先补齐本地能力保护门禁，再按正式 release tag 分段集成并逐段验证。

## What Changes

- 在首次 merge 前建立强制阶段 0：验证当前本地测试基线，将本地能力映射到现有测试，并只补充上游改动触及且缺少行为断言的高风险测试。
- 在隔离分支依次使用 `--no-ff` 合入 `v0.1.152`、`v0.1.153`、`v0.1.155` 和 `v0.1.156`，每段记录冲突决策并运行受影响能力及全部本地保护测试。
- 对无文本冲突的调用链、配置、生成代码和 migration 进行能力级审查；除明确例外外，本地能力不得回归，无法共存时暂停等待用户决定。
- **BREAKING** 在 `v0.1.156` 阶段完整移除本地 `openai-first-token-timeout` 实现、运行时配置、管理端 UI、测试和相关文档契约，完全采用上游的 HTTP 首输出超时与客户端 WebSocket 首消息超时语义。
- 最终执行后端与前端全量门禁、生成代码一致性、migration、冲突标记、Git 祖先关系及能力矩阵验证；不包含推送、发布或部署。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `upstream-release-sync`: 将单次 release 合并扩展为可按多个正式 tag 分段集成，并增加合并前测试保护门禁、逐段能力验证和明确例外管理。
- `openai-first-token-timeout`: 移除本地首 Token 超时契约，改为完全跟随上游 `v0.1.156` 提供的超时行为。

## Impact

- Git 历史：新增四个按顺序关联正式 tag 的 merge 节点，以及必要的本地兼容修复提交。
- 后端：重点影响 OpenAI 网关、WebSocket、调度与 Sticky、运行时设置、请求体生命周期、内容审计、图片能力、Wire/Ent 和 migrations。
- 前端：重点影响管理设置、账号与资源控制、用量与图片功能、i18n 及相关测试。
- 配置与 API：删除本地 `openai_text_first_token_timeout`、`openai_image_first_token_timeout` 及其运行时设置/API/UI；采用上游配置字段和默认行为。
- 验证：沿用无 Docker 的本地质量门禁，并增加本次上游同步专用的能力映射、逐段验证和最终全量审查记录。

```

## openspec/changes/staged-merge-upstream-v0-1-156/design.md

- Source: openspec/changes/staged-merge-upstream-v0-1-156/design.md
- Lines: 1-83
- SHA256: 281d4b968aa4df3f7b859626dd8a13c1e2f1cfcecdcf1db235bc6c861e5958c7

[TRUNCATED]

```md
## Context

本地 `main@d1cc02502` 与 `origin/main` 同步且工作区干净。上游最新正式 release `v0.1.156` 指向 `12f991dde8a58e183d4bd16a87ef6fd0df714757`；相对共同基线，本地和目标分别有 541 与 300 个独有提交，最终 release 区间涉及 503 个文件。上游 `main` 另有 release 后提交，不属于本次范围。

过去的上游合并证明，文本冲突和全量测试都不能单独发现调用链绕过、设置字段漏迁、生成 provider 丢失等语义覆盖。本次必须先建立本地能力保护基线，再缩小每次集成和回归定位范围。

## Goals / Non-Goals

**Goals:**

- 在首次 merge 前证明当前本地门禁可运行，并为上游触及的本地独有能力补齐最小行为保护。
- 按 `v0.1.152`、`v0.1.153`、`v0.1.155`、`v0.1.156` 四个不可变 release tag 分段集成。
- 每段独立记录冲突决策、受影响能力、测试证据和残余风险，使首次回归可定位到单个 release 区间。
- 除用户明确授权的本地首 Token 超时外，保持全部本地能力及边界语义。
- 最终证明 `v0.1.156` 为结果祖先，且全量代码、生成物、migration 和前端门禁通过。

**Non-Goals:**

- 不合入 `v0.1.156` 后的 `upstream/main` 提交。
- 不发布、不部署、不推送，也不预先决定是否合回 `main`。
- 不重写本地历史，不把 541 个本地提交 rebase 或 cherry-pick 到上游。
- 不建立新的测试框架、通用 merge 工具或无关抽象。

## Decisions

### 1. 在同一 change 内设置阶段 0 强制门禁

首次 merge 前运行当前仓库定义的后端默认测试、后端 unit 测试、后端 lint、前端 ESLint、typecheck、Vitest 和 build。随后把本地能力清单映射到现有测试，仅对同时满足“本地独有、目标 release 触及、缺少行为级断言”的高风险路径补测试。

选择同一 change 而非单独测试 change，因为这些测试直接服务本次 merge，且必须基于尚未集成上游的当前本地语义。阶段 0 未通过时不得开始 `v0.1.152` merge。

### 2. 按四个正式 tag 顺序执行 `--no-ff` merge

在独立 feature 分支依次合入：

1. `v0.1.152`：56 个上游提交，128 个文件变化。
2. `v0.1.153`：41 个上游提交，97 个文件变化。
3. `v0.1.155`：71 个上游提交，238 个文件变化。
4. `v0.1.156`：132 个上游提交，253 个文件变化。

每段完成冲突处理和该段验证后才进入下一段。选择分段 merge 而非一次合入 `v0.1.156`，是为了缩小语义回归归因范围；代价是相邻版本可能重复触发冲突。选择 tag merge 而非 cherry-pick，是为了保留完整上游祖先关系和 release 依赖。

### 3. 使用能力矩阵和冲突台账，而非仅检查冲突文件

每段台账记录冲突文件、两侧行为、最终融合、验证命令和结果。能力矩阵至少覆盖 scheduler、各平台 Sticky、fallback/WaitPlan、DB recheck、协议转换与透传、privacy/内容审计、image capability、运行时设置热更新、请求体重放与清理、用户资源控制、前端本地功能、测试门禁，以及版本、依赖、Wire/Ent 和 migrations。

无文本冲突但上游修改了相关入口、条件、DTO、缓存或 provider 时，同样必须给出能力结论。发现回归时先保留或补充失败测试，再做最小兼容修复；真实业务语义无法共存时暂停等待用户选择。

### 4. 在 `v0.1.156` 阶段完全采用上游超时语义

本地 `openai-first-token-timeout` 是唯一预先批准的移除例外。最终删除本地 HTTP SSE 与 WebSocket 上游首输出 watchdog、文本/生图分档、运行时设置、管理端字段、专用错误与测试，不保留兼容别名。

合并结果仅采用上游 `v0.1.156` 的 native HTTP Responses 首输出超时和客户端 WebSocket 首消息超时，包括其配置名称、默认关闭行为、高 reasoning effort 覆盖、failover 与账号超时处理语义。选择完整替换而非融合，是用户在获知覆盖差异后作出的明确决定。

### 5. 每段定向验证，最终全量验证

每段运行阶段 0 建立的本地保护测试，并按该段 changed files 增补受影响包或前端测试。最终运行完整本地质量门禁、`git diff --check`、冲突标记扫描、生成代码可复现检查、migration 文件与执行顺序复核，并验证四个目标 tag 均为结果祖先。

## Risks / Trade-offs

- [相邻 tag 重复冲突] → 每段保留冲突台账和小步兼容提交，优先复用已确认决策，但不机械采用 ours/theirs。
- [现有测试无法覆盖所有语义] → 阶段 0 做能力到测试的映射，合并后仍执行人工能力审查，不把测试通过等同于能力保持。
- [阶段 0 全量门禁存在既有失败] → 在任何 merge 前先收敛或明确记录阻塞，不把既有失败误归因于上游。
- [删除本地首 Token 超时留下配置或调用残骸] → 使用符号与配置键扫描、编译、前后端契约测试和 delta spec 验证确认完整退场。
- [生成代码或 migration 静默覆盖本地模型] → 以 schema/provider 为源重新生成并检查稳定 diff，逐个复核同号 migration 与运行顺序。
- [四段 merge 增加历史节点] → 节点与正式 release 一一对应，可审计性和故障定位收益高于历史简洁性。

## Migration Plan

1. 从已同步且干净的 `main` 创建隔离 feature 分支并固定 base、tag peel SHA 和 release 范围。
2. 完成阶段 0 测试基线、能力映射和必要缺口补测。
3. 顺序合入四个 tag；每段完成冲突台账、最小兼容修复和阶段验证后再继续。
4. 在 `v0.1.156` 阶段移除本地首 Token 超时并验证上游替代行为。
5. 完成最终全量门禁、能力矩阵、生成物和 Git 祖先验证。
6. 由用户在 finishing-branch 决策点选择合回 `main`、保留分支或其他处置；发布和部署另行执行。

合并进入 `main` 前，回退方式是删除或放弃隔离分支。进入 `main` 后优先 revert 对应 tag merge 或后续兼容修复提交，不改写已推送历史。

## Open Questions


```

Full source: openspec/changes/staged-merge-upstream-v0-1-156/design.md

## openspec/changes/staged-merge-upstream-v0-1-156/tasks.md

- Source: openspec/changes/staged-merge-upstream-v0-1-156/tasks.md
- Lines: 1-40
- SHA256: 0e4deb8776037d3de274b6da5647553ceaf1e9b40483bb13458a2acd232e3df0

```md
## 1. 固定基线与建立合并前门禁

- [ ] 1.1 在用户确认的隔离工作区固定本地 base、四个 tag peel SHA、`upstream/main` release 后范围和干净工作树证据
- [ ] 1.2 在任何 merge 前运行当前 `HEAD` 的 `make test`、前端 build 及既定生成代码检查，记录稳定基线或阻塞失败
- [ ] 1.3 根据本地独有提交、目标 tag changed files 和既有规格建立本地能力到行为测试的映射矩阵
- [ ] 1.4 为上游会触及且缺少行为断言的高风险本地能力添加最小失败测试，并重跑阶段 0 门禁至通过

## 2. 分段合入 v0.1.152

- [ ] 2.1 使用 `--no-ff` 合入 `v0.1.152`，逐文件记录冲突类别、两侧行为、融合结论和验证方式
- [ ] 2.2 审查 `v0.1.151..v0.1.152` 触及的本地能力，对回归先保留失败测试再做最小兼容修复
- [ ] 2.3 运行全部本地保护测试和本阶段受影响能力测试，记录通过证据后再进入下一 tag

## 3. 分段合入 v0.1.153

- [ ] 3.1 使用 `--no-ff` 合入 `v0.1.153`，更新冲突台账并复核上一阶段已确认的融合决策
- [ ] 3.2 审查 `v0.1.152..v0.1.153` 触及的本地能力，对回归先保留失败测试再做最小兼容修复
- [ ] 3.3 运行全部本地保护测试和本阶段受影响能力测试，记录通过证据后再进入下一 tag

## 4. 分段合入 v0.1.155

- [ ] 4.1 使用 `--no-ff` 合入 `v0.1.155`，更新冲突台账并单独复核网关、调度、设置、前端和生成代码变化
- [ ] 4.2 审查 `v0.1.153..v0.1.155` 触及的本地能力，对回归先保留失败测试再做最小兼容修复
- [ ] 4.3 运行全部本地保护测试和本阶段受影响能力测试，记录通过证据后再进入下一 tag

## 5. 分段合入 v0.1.156 并替换超时能力

- [ ] 5.1 使用 `--no-ff` 合入 `v0.1.156`，更新冲突台账并复核该阶段新增的 OpenAI 首输出和 WebSocket 首消息超时
- [ ] 5.2 完整删除本地首 Token 超时的后端逻辑、配置、运行时 API、管理端 UI、专用测试和兼容文档，保留且验证上游原生语义
- [ ] 5.3 扫描本地旧配置键、错误类型、结构化日志和 watchdog 符号，修复删除后的编译或契约依赖且不保留兼容别名
- [ ] 5.4 审查 `v0.1.155..v0.1.156` 触及的其余本地能力，对回归先保留失败测试再做最小兼容修复
- [ ] 5.5 运行全部本地保护测试、上游超时测试和本阶段受影响能力测试，记录通过证据

## 6. 生成物、元数据与完整能力审查

- [ ] 6.1 复核 `VERSION`、Go 与前端依赖、配置默认值、Wire/Ent 生成结果和 migrations，确认生成稳定且无本地 schema/provider 丢失
- [ ] 6.2 逐项完成 scheduler、Sticky、fallback/WaitPlan、DB recheck、网关转换与透传、privacy、image capability、运行时热更新、请求体生命周期、用户资源控制和前端本地功能能力矩阵
- [ ] 6.3 运行 `make test`、前端 build、必要的生成代码复验、`git diff --check` 和冲突标记扫描
- [ ] 6.4 验证四个目标 tag 均为结果祖先、merge 节点顺序正确、工作树仅含预期变更，并完成 thorough review
- [ ] 6.5 更新验证报告，记录每阶段冲突、修复、测试结果、明确移除项、残余风险和未执行的发布部署事项

```

## openspec/changes/staged-merge-upstream-v0-1-156/specs/openai-first-token-timeout/spec.md

- Source: openspec/changes/staged-merge-upstream-v0-1-156/specs/openai-first-token-timeout/spec.md
- Lines: 1-29
- SHA256: 1286b822d2d3dc507d534d0cf26a424c89914378db7e87bbfddd71def52f9787

```md
## REMOVED Requirements

### Requirement: 流式请求按明确生图意图选择首 Token 超时
**Reason**: 用户在了解上游仅部分覆盖及行为差异后，明确选择完全采用上游 `v0.1.156` 超时实现，不再维护本地文本/生图分档。
**Migration**: 删除本地分类与分档逻辑；部署方改用上游 `openai_first_output_timeout_seconds` 和 `openai_high_effort_first_output_timeout_seconds`。

### Requirement: 首 Token 等待采用业务事件边界
**Reason**: 本地业务事件分类与 watchdog 生命周期由上游 native HTTP 首输出判定替代。
**Migration**: 删除本地首 Token 事件分类和与通用流间隔超时的专用协调逻辑，采用上游 `v0.1.156` 响应暂存及首语义输出边界。

### Requirement: 首 Token 超时可运行时配置
**Reason**: 本地运行时配置、API 和管理端 UI 不属于上游实现，用户已批准完整移除。
**Migration**: 删除 `openai_text_first_token_timeout` 与 `openai_image_first_token_timeout` 的配置、持久化、API、UI 和兼容逻辑；按部署配置使用上游字段，上游默认值为关闭。

### Requirement: HTTP SSE 超时直接返回不可重试错误
**Reason**: 用户选择采用上游首输出超时的错误与 failover 语义。
**Migration**: 删除本地 `first_token_timeout` 不可重试错误，采用上游 `first_output_timeout`、failover 和账号流超时处理。

### Requirement: WebSocket 超时取消并清理当前 response
**Reason**: 上游 `v0.1.156` 未提供等价的上游 WebSocket 首输出 watchdog，用户仍明确批准移除本地实现。
**Migration**: 删除 pooled WebSocket 和 V2 passthrough relay 的本地首输出计时、cancel/drain 与连接复用保护；仅保留上游客户端首消息超时及既有 WebSocket 读写超时。

### Requirement: 首 Token 超时不影响账号调度状态
**Reason**: 用户选择采用上游 HTTP 首输出超时的账号处理与 failover 行为。
**Migration**: 删除本地请求级不可 failover 特判，采用上游 `HandleStreamTimeout` 和 `UpstreamFailoverError` 语义。

### Requirement: 超时诊断信息可观测
**Reason**: 本地专用失败 usage、Ops 字段和结构化日志随本地首 Token 能力移除。
**Migration**: 采用上游 `first_output_timeout` 的日志与 Ops 事件；不保留 `gateway.openai_first_token_timeout` 兼容事件。

```

## openspec/changes/staged-merge-upstream-v0-1-156/specs/upstream-release-sync/spec.md

- Source: openspec/changes/staged-merge-upstream-v0-1-156/specs/upstream-release-sync/spec.md
- Lines: 1-55
- SHA256: fe799d67880a01cc48e08e3eed4d3109b4ac8be89b467217ad64840713631312

```md
## ADDED Requirements

### Requirement: 合并前建立本地能力保护门禁
维护流程 MUST 在首次上游 merge 前验证当前本地质量门禁，将本地能力映射到现有行为测试，并为上游目标触及且缺少保护的高风险本地能力补充最小回归测试。该门禁未通过时 MUST NOT 开始上游 merge。

#### Scenario: 当前本地基线稳定
- **WHEN** 维护流程尚未合入首个目标 tag
- **THEN** 后端与前端既定本地质量门禁 MUST 在当前本地 `HEAD` 上通过，或将既有失败明确标记为阻塞

#### Scenario: 高风险本地能力缺少行为断言
- **WHEN** 本地独有能力所在路径被目标 release 修改，且现有测试不能断言该能力的关键行为
- **THEN** 维护流程 MUST 在首次 merge 前添加可复现的最小回归测试

### Requirement: 按正式 release tag 分段集成
维护流程 SHALL 允许将一个最终上游 release 目标拆为具有严格祖先顺序的多个正式 tag 阶段。每个阶段 MUST 完成冲突处理、能力审查和阶段验证后，才能进入下一阶段。

#### Scenario: 顺序合入多个 tag
- **WHEN** 用户选择按 `v0.1.152`、`v0.1.153`、`v0.1.155`、`v0.1.156` 分段集成
- **THEN** 维护流程 MUST 按该顺序建立独立 merge 节点，且不得跳过尚未完成验证的前置阶段

#### Scenario: 某阶段首次出现本地能力回归
- **WHEN** 阶段验证发现阶段 0 已保护的本地能力不再成立
- **THEN** 维护流程 MUST 在当前 release 区间内保留失败证据并完成最小修复，不得继续合入下一 tag

## MODIFIED Requirements

### Requirement: 保留上游更新和本地定制
维护流程 MUST 在冲突处理和无文本冲突的语义审查中优先保留上游修复和本地定制能力。仅当用户在了解行为差异后明确批准某项本地能力移除时，维护流程 MAY 将其登记为例外；其他无法共存的语义 MUST 暂停等待用户确认。

#### Scenario: 冲突能力可以共存
- **WHEN** upstream 更新和本地定制修改同一文件或调用链但行为可以同时成立
- **THEN** 合并结果 MUST 同时保留上游更新和本地定制语义

#### Scenario: 用户明确批准能力移除
- **WHEN** upstream 仅部分覆盖本地能力，且用户在获知缺失范围和行为差异后仍明确选择完全采用上游
- **THEN** 维护流程 MAY 删除该本地能力，但 MUST 在 proposal、delta spec、任务和验证报告中记录例外范围

#### Scenario: 未批准的能力不能共存
- **WHEN** upstream 更新和本地定制存在不可共存语义，且该能力不在已批准例外中
- **THEN** 维护流程 MUST 停止自动处理并请求用户选择保留策略

### Requirement: 合并后验证本地关键能力
维护流程 SHALL 在每个分段 merge 后运行该阶段受影响能力与全部本地保护测试，并在最终阶段执行完整自动验证和本地能力专项 review。测试通过 MUST NOT 替代能力级审查结论。

#### Scenario: 分段自动验证通过
- **WHEN** 一个目标 tag 的 merge、冲突处理和兼容修复完成
- **THEN** 维护流程 MUST 运行阶段 0 建立的保护测试和该 tag 触及能力的定向验证，再决定是否进入下一阶段

#### Scenario: 最终自动验证通过
- **WHEN** 最终目标 tag 合并完成且无冲突残留
- **THEN** 维护流程 MUST 运行后端默认与 unit 测试、后端 lint、前端 ESLint、前端单测、前端类型检查和构建验证

#### Scenario: 本地关键能力专项 review
- **WHEN** 最终自动验证完成
- **THEN** 维护流程 MUST 逐项复核 scheduler、各平台 sticky、fallback/WaitPlan、DB recheck、privacy、image capability、runtime setting 热更新、网关透传字段、请求体重放与清理、用户资源控制、前端本地功能、版本依赖、生成代码和 migrations，并记录每项证据

```
