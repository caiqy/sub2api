---
comet_change: restore-backend-lint-gate
role: technical-design
canonical_spec: openspec
archived-with: 2026-08-07-restore-backend-lint-gate
status: final
---

# Restore Backend Lint Gate 深度技术设计

## 1. 事实源与问题边界

实现基线固定为 `b576f73a22c4bf23d61727fc93950766a7e33929`。在 `backend` 目录使用 Go 1.26.5、golangci-lint 2.9.0 与仓库现有 `.golangci.yml` 执行：

```powershell
golangci-lint run ./... --max-issues-per-linter 0 --max-same-issues 0
```

基线结果为 exit 1、144 issues、39 files：

| 类别 | 数量 | 性质 |
| --- | ---: | --- |
| `ineffassign` | 140 | 主要是请求体内存优化后对已死局部 `[]byte` 的 `nil` 赋值 |
| `staticcheck` | 3 | QF1003 tagged switch 建议 |
| `unused` | 1 | 无调用方的私有 `sendCCUpstreamRequest` 方法 |

upstream 原始 `v0.1.169` 与 `v0.1.171` 在相同 CI lint 策略下均为 0 issues，因此本 change 只处理 fork-local 债务。OpenSpec proposal、design 与 tasks 是范围事实源；现有 `local-test-gates` 与 `request-body-retention-control` 是行为合同。本 change 使用 `skip_specs: true`，不修改需求。

逐文件、逐位置 manifest 与受保护 blob 固定在 `docs/superpowers/reports/2026-08-06-restore-backend-lint-gate-baseline.md`。`request-body-retention-control` 的既有 Purpose 仍含历史 `TBD`，但其 Requirements 是本 change 的规范事实源；async image 与 active-reader cleanup 由 `local-test-gates` 及已具名测试补充。修正文档 Purpose 不属于本 lint change。

## 2. 目标与不变量

### 2.1 目标

1. unchanged lint gate 的 uncapped 输出为 `0 issues`。
2. 根级 `make test` exit 0。
3. 请求体 ownership、spool、retry、failover、审计、计费和上游 wire body 语义不变。
4. 修复按风险域可独立审查、验证和回滚。

### 2.2 不变量

- 不修改 `backend/.golangci.yml`、`.github/workflows/backend-ci.yml`、Go 版本、Make target 或 issue cap。
- 不添加 `//nolint`、无意义读取、反射或仅用于躲避分析器的清零 helper。
- 不改变 public API、配置、数据库 schema、HTTP status、错误文本、header、payload 或依赖。
- 不手工编辑生成文件，不运行与本 change 无关的生成器。
- 不处理 upstream 合并、版本、前端功能、部署或远程环境。
- 不因同一 package 中尚有其他批次问题而把当前批次宣称为局部全绿；每次都核对全量 uncapped 集合。

## 3. 基线清单分区

### 3.1 批次 A：handler、routes 与 QF1003

该批共 36 issues，其中 33 个 `ineffassign`、3 个 `staticcheck`：

```text
internal/handler/gateway_handler.go
internal/handler/gateway_handler_chat_completions.go
internal/handler/gateway_handler_responses.go
internal/handler/gemini_v1beta_handler.go
internal/handler/grok_media.go
internal/handler/image_task_handler.go
internal/handler/openai_alpha_search.go
internal/handler/openai_chat_completions.go
internal/handler/openai_gateway_count_tokens.go
internal/handler/openai_gateway_handler.go
internal/handler/openai_live.go
internal/handler/request_body_memory_retention_test.go
internal/server/routes/gateway.go
```

### 3.2 批次 B：通用 Gateway、Anthropic、Bedrock 与 Antigravity

该批共 63 个 `ineffassign`：

```text
internal/service/antigravity_gateway_claude.go
internal/service/antigravity_gateway_compat.go
internal/service/antigravity_gateway_gemini.go
internal/service/antigravity_gateway_service_test.go
internal/service/antigravity_gateway_upstream.go
internal/service/gateway_anthropic_apikey_passthrough_test.go
internal/service/gateway_anthropic_passthrough.go
internal/service/gateway_bedrock.go
internal/service/gateway_count_tokens.go
internal/service/gateway_forward.go
internal/service/gateway_forward_as_chat_completions.go
internal/service/gateway_upstream_request.go
internal/service/gemini_chat_completions_compat_service.go
```

### 3.3 批次 C：OpenAI、Gemini、Grok 与 unused

该批共 45 issues，其中 44 个 `ineffassign`、1 个 `unused`：

```text
internal/service/openai_alpha_search.go
internal/service/openai_gateway_cc_pipeline.go
internal/service/openai_gateway_chat_completions.go
internal/service/openai_gateway_chat_completions_raw.go
internal/service/openai_gateway_count_tokens.go
internal/service/openai_gateway_forward.go
internal/service/openai_gateway_grok.go
internal/service/openai_gateway_grok_chat_bridge.go
internal/service/openai_gateway_messages.go
internal/service/openai_gateway_messages_chat_fallback.go
internal/service/openai_gateway_passthrough.go
internal/service/openai_gateway_responses_chat_fallback.go
internal/service/openai_live.go
```

三个批次合计 `36 + 63 + 45 = 144`。实施前把 lint 原始输出和分区结果写入 build report，后续每批以“目标集合消失、非目标集合未增加”为关闭条件。

## 4. 逐项修复算法

### 4.1 纯死局部赋值

形态：局部或参数 body 在最后一次读取后被赋为 `nil`，之后没有可观察读取。

```go
req, err := build(body)
body = nil
return wait(req, err)
```

删除赋值，不引入替代语句：

```go
req, err := build(body)
return wait(req, err)
```

理由：静态分析确认赋入的 `nil` 没有后续值读取，但这不等同于证明编译器 liveness、heap retention 或资源 ownership。Go 编译器通常按最后使用点缩短局部变量存活期；现有 heap retention 测试负责验证当前工具链和平台下没有运行时保留回退。

### 4.2 混合赋值

形态：同一多重赋值同时清理 owner 字段和已死局部：

```go
input.SourceBody, input.Body, sourceBody, body = nil, nil, nil, nil
```

拆分后保留可观察 owner 清理，只删除 linter 指向的死局部：

```go
input.SourceBody = nil
input.Body = nil
```

不能机械删除整行。对 request handle、input struct、context snapshot 或跨 closure 捕获字段，必须先确认调用方和 cleanup ownership，再决定是否保留。

### 4.3 retry/failover 局部

attempt、retry、canonical、wire 等 body 只在当前迭代使用时，删除迭代尾部无效清零。若变量被 closure、defer、goroutine 或返回对象捕获，则它不属于纯死局部，应保留真实 owner 清理或缩小捕获对象。

不得为所有循环统一提取 helper。只有现有内存矩阵明确指出某个路径跨上游等待保留大 slice 时，才把该路径的“物化、改写、构造 handle”收进窄函数，使 slice 在函数返回时离开作用域。

### 4.4 测试中的 GC 辅助清零

测试文件里的 `body = nil`、`materialized = nil` 等语句先按死局部删除，再运行原测试。若 retained heap 阈值回退，则用内层函数返回 handle、hash 或断言所需小值，使大 slice 不在调用 `runtime.GC` 的 frame 中；不通过清零 helper 欺骗 linter。

### 4.5 QF1003

将等值 `if/else if` 链改为 tagged switch，保持：

- case 顺序；
- default 行为；
- 提前 return；
- 原错误映射与测试 branch 选择。

不合并分支、不调整 status/path/branch 的业务含义。

### 4.6 unused 方法

在 CodeGraph 与编译器均确认 `sendCCUpstreamRequest` 没有静态、接口、callback 或测试调用方后，删除整个私有方法。若发现调用方，则本清单事实失效，停止并重新设计，不能用注释或伪调用保活。

## 5. TDD 与批次协议

问题本身是质量 gate RED，而不是运行时行为 RED。因此每批采用以下协议：

1. **RED**：运行全量 uncapped lint，确认该批目标集合存在且数量与 manifest 一致。
2. **行为基线**：在修改前运行该批对应的现有 package 与请求体测试，必须为 GREEN。若既有行为测试失败，先停止并判断 baseline，不把它归因给 lint 修复。
3. **最小修改**：只应用第 4 节规则，不增加预防性抽象。
4. **GREEN**：重新运行全量 uncapped lint；目标集合必须清零，总集合只允许减少。运行对应行为测试。
5. **审查**：检查 diff 中无配置改动、suppression、owner 清理误删和无关整理，再提交该批。

不为“行为本来正确”的 lint 问题人为制造 source-shape 测试。默认 application/test allowlist 仅含 baseline manifest 的 39 个文件。只有发现现有测试未覆盖真实行为边界时，才先更新本 Design Doc、baseline manifest 与 OpenSpec tasks，经范围审查后添加能复现该边界的失败测试；未完成该流程前不得修改额外 backend 文件。

## 6. 测试策略

### 6.1 基线身份

记录以下证据：

```powershell
git rev-parse HEAD
git diff --exit-code b576f73a22c4bf23d61727fc93950766a7e33929 -- backend/.golangci.yml backend/go.mod backend/go.sum .github/workflows/backend-ci.yml Makefile backend/Makefile
go version
golangci-lint version
```

HEAD 允许包含本 change 的文档/代码提交，但上述配置路径相对 base 必须无差异。

### 6.2 聚焦验证

从 `backend` 目录运行：

```powershell
go test -count=1 ./internal/handler ./internal/server/routes
go test -count=1 ./internal/service
```

retained heap 集合：

```powershell
$pattern = '^(TestAsyncImageRequestBodyMemoryRetentionWhileWorkersBlocked|TestRequestBodyMemoryRetentionWhileUpstreamBlocked)$'
go test ./internal/handler -list $pattern  # 预期 2 个 top-level tests
go test -count=1 ./internal/handler -run $pattern
```

handler spool/replay 集合：

```powershell
$pattern = '^(TestGatewayHandler_MessagesAndResponsesReplayLargeBodiesAcrossFailover|TestOpenAIGatewayHandler_ChatAndEmbeddingsReplayMappedSpoolAcrossFailover|TestOpenAIGatewayHandler_ChatReplayRawSpoolAcrossFailoverWhenResponsesUnsupported|TestAsyncImageHandlerSpoolsReplaysAndCleansOwnedBody|TestAsyncImageHandlerSpoolsReplaysAndCleansOwnedMultipartBody|TestGrokMedia_GenerateEditVideoRejectUpstreamFailoverPreserveRequestSemantics|TestGrokMedia_MultipartSpoolPreservesFilesAndOmitsSnapshots)$'
go test ./internal/handler -list $pattern  # 预期 7 个 top-level tests
go test -count=1 ./internal/handler -run $pattern
```

service spool/retry/failover 集合：

```powershell
$pattern = '^(TestAntigravityGatewayService_ClaudeForwardHandleSignatureRetryReparsesFileBackedCanonical|TestAntigravityRetryLoopReopensGeminiPayloadHandleForRetry|TestGatewayService_AnthropicPassthroughRetryRereadsHandleAfterForwardFirstAttemptBytes|TestForwardAsResponsesHandle_SpoolTransportErrorPreservesSentinel|TestGeminiMessagesCompatSignatureRetryBuildsHandleFromCanonical|TestOpenAIGatewayService_RejectedFieldRetryReturnsSpoolError|TestOpenAIForwardReusesBoundRequestBodyHandle|TestOpenAIForwardPreservesBoundRequestBodyHandleWhenHTTPDoErrors|TestOpenAILiveCreateUpstreamHandleTransportSpoolErrorClosesBodies)$'
go test ./internal/service -list $pattern  # 预期 9 个 top-level tests
go test -count=1 ./internal/service -run $pattern
```

`TestRequestBodyMemoryRetentionWhileUpstreamBlocked` 覆盖 25 条 handler/upstream 阻塞分支，每条比较 2MB 与 8.9MB 请求并断言 retained growth 小于 3MB。`TestAsyncImageRequestBodyMemoryRetentionWhileWorkersBlocked` 是独立测试，内部重复 3 轮并验证四个 worker 不保留四份完整 body。单次命令失败即视为失败，不靠重复执行挑选绿色结果。

### 6.3 最终 gate

从 `backend` 目录运行：

```powershell
golangci-lint run ./... --max-issues-per-linter 0 --max-same-issues 0
```

从仓库根目录运行：

```powershell
make test
git diff --check b576f73a22c4bf23d61727fc93950766a7e33929..HEAD
git diff --check
git diff --exit-code b576f73a22c4bf23d61727fc93950766a7e33929 -- backend/.golangci.yml backend/go.mod backend/go.sum .github/workflows/backend-ci.yml Makefile backend/Makefile
git diff --name-only b576f73a22c4bf23d61727fc93950766a7e33929..HEAD
git diff --name-only
git diff --cached --name-only
git ls-files --others --exclude-standard
git status --short --untracked-files=all
```

最终证据必须同时包含 uncapped `0 issues` 与 `make test` exit 0。默认 lint 输出不能替代 uncapped 证明。application/test changed files 必须属于 baseline manifest 的 39 文件，或属于按第 5 节先行更新并确认的 manifest 扩展；其余 committed 允许路径仅限本 change 的 OpenSpec/Comet artifacts、Design Doc、plan 和验证报告。最终 staged/unstaged tracked diff 必须为空；untracked 只允许 `.comet/current-change.json` 这一 runtime selection 文件。

## 7. 失败处理

| 失败 | 处理 |
| --- | --- |
| 删除局部清零后内存矩阵失败 | 锁定失败 branch，只对对应物化路径收窄作用域；修复前保留 RED 输出 |
| spool/retry/failover 测试失败 | 检查是否误删 owner/struct/handle 清理；恢复真实清理，不恢复死局部赋值 |
| 当前批次产生新 lint | 在同批最小修复；若需要改变架构，暂停并更新 design/plan |
| 非目标集合变化或 base/config 漂移 | 停止，重新核对 base 与 manifest；不得静默吸收 |
| package 测试原本即失败 | 记录 baseline 并请求范围决策；不把现有失败标成修复成功 |
| 根级 `make test` 出现非 lint 新失败 | 判断是否由本 change 引入；引入则修复，既有且超范围则停止，不建立例外 |

## 8. 审查与提交边界

计划使用三个实现提交和一个最终证据提交：

1. handler/routes/QF1003；
2. Gateway/Anthropic/Bedrock/Antigravity；
3. OpenAI/Gemini/Grok/unused；
4. 仅在确有 tracked 报告更新时提交最终验证证据。

每批 reviewer 必查：

- manifest 中该批命中是否全部消失；
- 是否保留所有 owner、input field、handle 与 cleanup 调用；
- 是否新增 suppression、假读取或通用清零 helper；
- 请求构造、retry/failover 与错误分支是否改变；
- 测试命令是否实际匹配测试并 exit 0。

## 9. 集成与恢复后续 change

本 change 完成并集成到 `main` 后，`staged-merge-upstream-v0-1-171` 不能直接沿用旧 immutable source base。其绑定分支是 `feature/20260806/staged-merge-upstream-v0-1-171`，当前 durable blocked checkpoint 是 `5cbce94f5`。恢复时必须：

1. 记录 lint remediation 集成后的新 `main` commit，并由用户确认它替代 `b576f73a` 成为 source base。
2. 切换到上述绑定分支，验证 `5cbce94f5` 仍是其祖先且工作区无非 Comet 临时改动。
3. 将新 `main` 合并进绑定分支，保留 `5cbce94f5` 及其 Task 1-4 历史，不使用 rebase 改写 checkpoint。
4. 更新该 change 的 OpenSpec、Design Doc、plan、`.comet.yaml base_ref` 与 manifest，使新 base identity 可审计。
5. 从 Task 4 的完整 gate 重新开始，不复用旧的部分绿色证据；只有完整 gate 通过才解除 blocked。

以上步骤属于后续 change 的恢复任务，不在本 lint remediation change 中执行。

## 10. 验收矩阵

| 验收项 | 成功条件 |
| --- | --- |
| Baseline identity | source base 与受保护配置 hash 可复现 |
| Issue closure | 144 项分区总数闭合，uncapped lint 为 0 issues |
| Runtime semantics | handler/service package 测试 exit 0 |
| Memory retention | 25 分支 handler/upstream 矩阵与独立 async image retention 测试 exit 0 |
| Replay ownership | spool、retry/failover 聚焦测试匹配非空且 exit 0 |
| Repository gate | 根级 `make test` exit 0 |
| Scope integrity | 无 lint/CI/Go/Make 配置改动，无 suppression，无无关功能变更 |

## 11. Spec Patch 与开放问题

Spec Patch：无。`local-test-gates` 与 `request-body-retention-control` 已覆盖本 change 的行为合同。

开放问题：无。最小删除策略、lint-as-RED、批次边界、失败处理与测试策略均已确认。
