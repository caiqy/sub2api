---
generated_from_state_version: 15
---

# 验证

## 当前结果

- 结果: **已归档**
- 验证情况: **已完成检查，验证结果已确认**
- 目标周期: 1
- 迭代: 2
- 验证器尝试次数: 2
- 完成时间: 2026-09-15T23:34:55.371Z
- 摘要: 最终验证通过。478371caa 修复 OAuth Images 在最终账号模型映射前过早固定协议的问题；正式回归、独立只读审查、Runtime 八项门禁、双父归因、版本与不发布边界均成立。

## 验收

| 编号 | 结果 | 来源 | 验收项 | 原因 |
| --- | --- | --- | --- | --- |
| A1 | passed | brief.md | A1: Git 历史包含一个独立 `--no-ff` merge，其第二父为 `86f93c28ee34cc74b629dafb748bd5ac5ca8c5ea`；最终 HEAD 包含 `v0.2.5` 且不包含 post-`v0.2.5` upstream 提交。 | HEAD 478371caa 为独立修复提交；原 merge 3fb0a6123 的第二父为 86f93c28e，v0.2.5 可达且未发现 post-tag upstream 提交。 |
| A2 | passed | brief.md | A2: 合并后完成冲突标记检查、受影响能力测试、后端默认与 unit 测试、后端 lint、前端 ESLint/i18n/单测/类型检查、前后端构建和独立只读回归审计；发现的 fork 合并回归先用失败测试和最小修复闭合。 | Runtime dispatch-r2 八项完整门禁及独立只读审查全部通过。 |
| A3 | passed | brief.md | A3: `v0.2.5` 的 OpenAI/Codex WebSocket 执行作用域、池生命周期、抢占与心跳，原生 Images、Responses Lite namespace、模型映射、配额窗口和 OpenCode Go 能力成立。 | upstream 能力与 fork 回归证据保留，未发现本轮修复引入的能力回归。 |
| A4 | passed | brief.md | A4: `v0.2.5` 的 Antigravity/Gemini 模型与流式兼容、DeepSeek 模型校验与定价、Ollama Cloud 限流窗口、Grok media/Responses 及协议转换行为成立。 | 指定模型、流式、DeepSeek、Ollama Cloud、Grok 与协议转换能力由测试和源码审查覆盖。 |
| A5 | passed | brief.md | A5: `v0.2.5` 的订阅与 API Key 批量操作、注册密码确认、站点类型开关、监控与 Ops、代理凭据、支付及相关前端行为成立。 | OAuth Images 生成、编辑、stream、multipart、映射和回退行为已通过正式回归。 |
| A6 | passed | brief.md | A6: fork 的 scheduler、sticky/fallback、DB recheck、请求体重放与清理、审计、每请求计费、运行时设置、插件边界、网关透传和前端定制语义保持成立。 | scheduler、sticky/fallback、DB recheck、请求体生命周期、审计计费、运行时设置、插件、网关和前端定制未发现回归。 |
| A7 | passed | brief.md | A7: 管理员用量记录继续展示仅管理员可见的用户备注，普通用户接口不泄露备注；用户 token 排名 Excel 导出的用户名、排序与样式定制保持成立。 | 原始 overlay、OAuth 与 SetupToken 路径均通过。 |
| A8 | passed | brief.md | A8: `backend/cmd/server/VERSION` 为 `0.2.5.1`，且没有创建或推送 tag、触发 release workflow 或部署。 | VERSION 为 0.2.5.1，未创建或推送 tag，未触发 release workflow，未部署。 |
| A9 | passed | brief.md | A9: migrations、Ent/Wire 两次生成结果稳定；环境不支持的检查明确记录未覆盖范围；最终由新的只读 Verifier 对全部验收项给出独立结论。 | generation、build、merge-boundary 及完整 Runtime 验证通过，结果保持在 v0.2.5 边界。 |
| A10 | passed | brief.md | A10: `memory/context/upstream-merge-workflow.md` 仅在本轮产生经验证的新复用经验时增量更新，否则保持不变。 | memory/context/upstream-merge-workflow.md 增量记录了本轮真实验证出的 Images 请求体缓存经验。 |
| A11 | passed | specs/upstream-release-sync/spec.md | 单一可追溯 merge 节点 - **WHEN** 当前 fork 基线已包含 `v0.2.4` - **THEN** 结果 merge 节点的第二父为 `86f93c28ee34cc74b629dafb748bd5ac5ca8c5ea`，`v0.2.5` 是结果 HEAD 的祖先，且结果不包含 post-`v0.2.5` upstream 提交 | v0.2.5 merge 节点及 candidate 修复提交的历史归因可审计。 |
| A12 | passed | specs/upstream-release-sync/spec.md | 合并审计发现 fork 回归 - **WHEN** 测试、构建或能力级源码审查发现合并结果破坏 fork 既有行为 - **THEN** 流程必须停留在 `v0.2.5` 边界，以失败测试和最小修复闭合问题后重新执行受影响及完整门禁 | 发现 Images fork 回归后先以失败测试复现，再以最小修复闭合并完成完整重跑。 |
| A13 | passed | specs/upstream-release-sync/spec.md | 发现 upstream 原样问题 - **WHEN** 审查发现问题在 `v0.2.5` peeled commit 中已原样存在，且本次合并与 fork 定制没有扩大或改变该问题 - **THEN** 记录该问题及证据但不在本 change 主动修复，不得以修复 upstream 既有问题为由扩大范围 | 逐版能力、fork 回归和独立只读审查证据均已提供。 |
| A14 | passed | specs/upstream-release-sync/spec.md | v0.2.5 能力验证 - **WHEN** `v0.2.5` merge 完成并准备交付 - **THEN** 该 tag 的主要能力由对应 upstream 测试、fork 回归测试和能力级源码审查覆盖 | 重叠调用链由正式 Images 回归测试及独立 reviewer 复核证明成立。 |
| A15 | passed | specs/upstream-release-sync/spec.md | upstream 修改重叠调用链 - **WHEN** upstream 更新触及上述 fork 能力所在文件或调用路径 - **THEN** 合并结果必须由现有或新增回归测试、或明确源码审查证明双方行为仍成立 | Docker/Testcontainers 未运行范围已明确记录，其余可执行门禁全部通过。 |
| A16 | passed | specs/upstream-release-sync/spec.md | 管理员备注和导出定制保持 - **WHEN** 合并后的管理员用量接口、普通用户用量接口和用户 token 排名导出被验证 - **THEN** 管理员仍可查看用户备注、普通用户响应不包含备注，且导出的用户名、排序和样式符合 fork 当前行为 | 管理员备注、普通用户隐私和导出定制未被本轮变更破坏，既有证据成立。 |
| A17 | passed | specs/upstream-release-sync/spec.md | 环境不支持部分检查 - **WHEN** 本机缺少 Docker/Testcontainers 或其他必要运行条件 - **THEN** 必须执行其余门禁并明确记录未运行范围，不得将其记为通过 | 未发生额外发布、部署或越过授权 upstream 边界的动作。 |
| A18 | passed | specs/upstream-release-sync/spec.md | 集成完成但不发布 - **WHEN** `v0.2.5` 合并及全部可执行验证通过 - **THEN** 项目保留可审计的集成提交和 `0.2.5.1` 版本文件，不创建或推送 Git tag、不触发 release workflow、不部署，并在任何后续 upstream tag 或 post-`v0.2.5` 提交前停止等待用户授权 | 项目保留可审计 merge 与 0.2.5.1 版本文件，并停止在 v0.2.5 边界。 |
| A19 | passed | specs/upstream-release-sync/spec.md | 本轮没有新的可复用经验 - **WHEN** 本轮实际过程没有超出现有项目记忆的有效原则 - **THEN** `memory/context/upstream-merge-workflow.md` 保持不变，并在 Builder 交接中明确说明未产生新增记忆 | 本轮产生并验证了新的可复用 memory 经验，因此无经验分支不适用。 |

## 检查

| 检查 | 命令 | 工作目录 | 状态 | 退出码 | 耗时 |
| --- | --- | --- | --- | ---: | ---: |
| 后端默认测试 | -NoProfile -ExecutionPolicy Bypass -File scripts/test.ps1 ./... | backend | passed | 0 | 186743 ms |
| 后端unit强制新执行（排除已证实的upstream Ollama CAS失败） | -NoProfile -ExecutionPolicy Bypass -File scripts/test.ps1 -tags=unit -skip=^TestOllamaProbeCallback_StaleLongDoesNotOverrideNewShort$ -count=1 ./... | backend | passed | 0 | 235958 ms |
| 后端静态检查 | run ./... | backend | passed | 0 | 48792 ms |
| 前端lint、类型、全量测试及构建 | -NoProfile -Command pnpm run lint:check && pnpm run typecheck && pnpm run test:run && pnpm run build | frontend | passed | 0 | 296529 ms |
| 后端构建 | -NoProfile -Command if (-not (Test-Path -LiteralPath bin)) { throw 'missing bin directory' }; $env:CGO_ENABLED='0'; go build -ldflags='-s -w -X main.Version=0.2.5.1' -trimpath -o bin/server.exe ./cmd/server; exit $LASTEXITCODE | backend | passed | 0 | 5451 ms |
| Ent/Wire两轮生成稳定性 | -NoProfile -Command go generate ./ent; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; go generate ./cmd/server; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; git diff --exit-code -- ent cmd/server/wire_gen.go; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; go generate ./ent; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; go generate ./cmd/server; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; git diff --exit-code -- ent cmd/server/wire_gen.go; exit $LASTEXITCODE | backend | passed | 0 | 59516 ms |
| 原始merge节点、修复边界及版本 | -NoProfile -Command if ((git rev-parse 3fb0a6123^1) -ne '240cd0702ca9eac1ac2d2915bbd789fac0cead32') { throw 'unexpected fork parent' }; if ((git rev-parse 3fb0a6123^2) -ne '86f93c28ee34cc74b629dafb748bd5ac5ca8c5ea') { throw 'unexpected upstream parent' }; if ((git rev-parse HEAD^1) -ne '3fb0a61232246290110cc5cb738cec1aa13e5398') { throw 'unexpected repair parent' }; if ((git show HEAD:backend/cmd/server/VERSION).Trim() -ne '0.2.5.1') { throw 'unexpected version' }; git merge-base --is-ancestor 86f93c28ee34cc74b629dafb748bd5ac5ca8c5ea HEAD; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; git diff 240cd0702 HEAD --check; exit $LASTEXITCODE | . | passed | 0 | 1310 ms |
| 图片缓存映射、释放重放及回退专项 | test ./internal/service -run ^(TestOpenAIImagesPreparedBodyAccountMappingAndReplay\|TestCodexDirectImagesMappingBeforeRouting\|TestCodexDirectImagesHTTPErrorFallbacksOnlyWhenEndpointUnavailable\|TestOpenAISetupTokenImagesUsesOAuthDirectPath)$ -count=1 | backend | passed | 0 | 6030 ms |

### Builder 报告的证据

以下为 Builder 报告，不等同于 Runtime 检查凭据或独立验收结果。

- 正式图片重放回归测试: passed — 修改前生成、JSON编辑、multipart编辑的gpt-image-1映射场景均失败；修改后12个输入组合及各自3种路由通过。
- 原始 overlay 复现: passed — OAuth和SetupToken的4个原始场景均通过。
- Images与请求体专项: passed — service专项通过；handler初次直接go test超时，改用仓库scripts/test.ps1后18秒通过。
- 独立只读复核: passed — 新的reviewer ses_f59cb4d86ffeJllq2GmtwX8ARJ 审查全部修改和调用链，并运行4组相关现有测试通过。
- 完整质量门禁: not-run — 本候选待Runtime统一执行后端默认/unit、lint、前端全部门禁、构建、两轮生成和精确merge边界检查。
- 已知限制: Docker未安装，Testcontainers/integration未运行；unit仍仅排除已在pristine upstream v0.2.5复现的Ollama CAS失败。
- 已知限制: 未推送、未创建tag、未发布、未部署。

## 阻塞项

_无。_

## 风险与跳过的工作

- Docker/Testcontainers 未运行。
- 首次 backend-unit 曾出现 OAuth Images HeapAlloc 116.09 MiB > 108.15 MiB；后续 memory count5 与强制全量 unit 均通过，代码期间未变，该不确定性保留。
- Ollama CAS 与 Images DONE 为 pristine v0.2.5 原有问题，不归因于 candidate。

## 之前的迭代

| 目标周期 | 迭代 | 尝试 | 结果 | 未解决项 | 摘要 | 完成时间 |
| ---: | ---: | ---: | --- | --- | --- | --- |
| 1 | 1 | 1 | pass | — | 独立核实候选 3fb0a612 的 merge 边界、版本、正式 Runtime 门禁、关键 fork 语义和上游原样问题归因；未发现本轮合并或 fork 交互引入的问题。 | 2026-09-15T17:12:22.742Z |
| 1 | 1 | 1 | recovery | — | 用户同意按第二轮审查建议修复：OAuth Images 账号映射跨协议时缓存的 Responses body 被发送到直连 Images 端点。将补正式失败测试，统一最终模型、payload和端点选择，复核释放重放及failover。 | 2026-09-15T17:48:16.857Z |
| 1 | 2 | 1 | execution-error | — | Native Verifier response was invalid: Native verification cannot pass before every required check succeeds | 2026-09-15T18:39:35.728Z |
| 1 | 2 | 2 | pass | — | 最终验证通过。478371caa 修复 OAuth Images 在最终账号模型映射前过早固定协议的问题；正式回归、独立只读审查、Runtime 八项门禁、双父归因、版本与不发布边界均成立。 | 2026-09-15T23:34:55.371Z |



## 结论

最终验证通过。478371caa 修复 OAuth Images 在最终账号模型映射前过早固定协议的问题；正式回归、独立只读审查、Runtime 八项门禁、双父归因、版本与不发布边界均成立。
