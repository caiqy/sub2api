---
generated_from_state_version: 8
---

# 验证

## 当前结果

- 结果: **已归档**
- 验证情况: **已完成检查，验证结果已确认**
- 目标周期: 1
- 迭代: 1
- 验证器尝试次数: 1
- 完成时间: 2026-09-26T08:11:16.562Z
- 摘要: 独立只读 Verifier 已核查 brief、完整 Spec、A1-A17、实际实现及候选绑定的 Runtime 回执，最后参考 Builder 交接。未发现阻断验收的代码问题，17 项全部通过。正式 8 项检查通过：后端六包聚焦、后端构建、前端 12 文件 283 测试、类型、i18n，以及独立追加的真实 PostgreSQL、临时目录生产构建、HTTP 禁重定向和管理员认证回归。未修改候选代码，未启停 PostgreSQL，未执行 accept-result、归档或 Git 提交。

## 验收

| 编号 | 结果 | 来源 | 验收项 | 原因 |
| --- | --- | --- | --- | --- |
| A1 | passed | brief.md | A1：从账号更多操作打开检测弹窗，选择支持的模型及次数（初始为 1），确认后看到进度和结果。 | 已核查更多操作到 ModelTraceModal 的连接、默认 1 轮、确认后才 POST、进度和结果展示；正式前端回归包含 gpt-5.6-sol 及 3 轮提交。 |
| A2 | passed | brief.md | A2：手动与自动检测记录均出现在第二个历史页签，记录账号、触发来源、时间、模型、轮次、结果、归因概率、耗时及失败原因，刷新页面后仍可查询。 | 第二页签按账号分页读取 SQL 历史，展示来源、时间、选择及目标模型、请求/有效轮次、结论、概率、耗时、失败和版本；正式 PostgreSQL 持久化检查通过。 |
| A3 | passed | brief.md | A3：网关设置保存自动检测配置后无需重启；关闭时不再创建自动检测，手动检测仍可运行。 | 网关设置独立持久化；扫描逐账号读取最新配置，执行前再次校验；关闭自动不影响手动 Create，热更新及关闭回归通过。 |
| A4 | passed | brief.md | A4：账号状态第二行展示检测结论，点击直接打开该账号历史页签。 | AccountsView 状态列第二行接入 ModelTraceStatus，点击以 history 页签打开所选账号弹窗；已有有效结果的模型和时间通过提示及无障碍标签提供。 |
| A5 | passed | brief.md | A5：网络、认证、限流、无效输出等错误记录为检测失败，不伪装为正常或降智。 | HTTP 非 200、传输、鉴权、限流、超时及无效/未完成输出都进入 failed；成功终态才允许评分，错误不会伪装为正常或降智，正式探针回归通过。 |
| A6 | passed | brief.md | A6：轮次为 1 时可以正常评分；多轮按有效输出对应的校准参数聚合，并显示请求轮次与有效轮次。 | ScoreSamples 使用有效输出逐回答评分及样本数 1/2/3 对应 beta，任务要求所有请求轮次有效才聚合；上游固定样本概率对照及不足样本检查通过。 |
| A7 | passed | brief.md | A7：并发手动与自动任务不会重复测试同一个账号；账号凭据、代理密码及 Cookie 不进入历史或接口。 | 账号行锁和唯一活动任务索引防止手动/自动并发重复，跨实例租约和发送前围栏保护执行；SQL/API 不保存原始输出或凭据，错误仅保留固定脱敏类别；真实 PostgreSQL 并发复用检查通过。 |
| A8 | passed | brief.md | A8：每个账号有可持久化的独立自动检测开关，新增及存量缺失配置均视为关闭；关闭账号不被自动检测，但仍可手动检测。 | 创建/编辑独立开关写入 extra，前端默认 false，后端仅布尔 true 合格，缺失和字符串真值均关闭；关闭自动仍可手动，正式服务及账号表单测试通过。 |
| A9 | passed | specs/modeltrace-degradation-detection/spec.md | 账号和模型隔离 - **WHEN** 管理员选择某 OpenAI 账号及一个支持的模型发起检测 - **THEN** 探针使用该账号的凭据归属、协议及业务代理，记录选择模型与有效目标模型；不得换用其他账号并将其结果归入所选账号。 | ResolveModelTraceTarget 保留所选账号路由身份、解析影子凭据、映射及透传规则，ProbeModelTrace 使用业务代理和对应协议，发送前拒绝不支持目标；协议、影子代理、Sol、映射及能力回归通过。 |
| A10 | passed | specs/modeltrace-degradation-detection/spec.md | 单轮与多轮评分 - **WHEN** 分别完成 1、2 或 3 轮有效挑战 - **THEN** 各自使用相应样本数校准，归因第一名等于有效目标则记录正常，否则记录降智；概率和指纹库版本可查询。 | 独立读取固定上游 commit 的 fingerprint-core.js，核对本地边缘/有序指纹聚合和 softmax 校准；正式 1/2/3 轮固定样本概率对照通过，归因第一名按有效目标直接判断，版本可查询。 |
| A11 | passed | specs/modeltrace-degradation-detection/spec.md | 部分失败 - **WHEN** 请求 3 轮检测但仅 2 轮有效，或上游返回错误 - **THEN** 历史记录为失败并展示有效轮次和脱敏原因，不自动补发、不覆盖账号此前有效结论。 | 第 3 轮失败时保留 2 个有效轮次并立即结束，不补发；Finish 强制清空失败结论且不更新摘要；正式部分失败测试及真实 PostgreSQL 失败不覆盖检查通过。 |
| A12 | passed | specs/modeltrace-degradation-detection/spec.md | 关闭并重新打开 - **WHEN** 手动任务执行中关闭弹窗再重新打开 - **THEN** 任务继续运行，界面恢复其进度或结果，不因打开动作重复发起模型请求。 | 任务使用服务生命周期上下文，与弹窗及 HTTP 创建请求解绑；关闭只停止前端轮询，重新打开读取 active 或最近结果，generation 防迟到响应；恢复及分页终态回归通过。 |
| A13 | passed | specs/modeltrace-degradation-detection/spec.md | 热更新和关闭 - **WHEN** 管理员保存有效设置或关闭自动检测 - **THEN** 后续调度使用新配置，关闭后不再启动排队或新自动任务，手动检测仍然可用；非法参数拒绝保存且不改变原配置。 | 服务端模型、轮次和 5-10080 分钟边界先校验后保存；非法参数不改变原配置，排队自动任务执行前重新检查资格，新模型/轮次仅用于后续任务。 |
| A14 | passed | specs/modeltrace-degradation-detection/spec.md | 开关组合 - **WHEN** 全局开启但账号独立开关关闭，或全局关闭而账号开关开启 - **THEN** 该账号不被自动检测，但管理员仍可发起手动检测；只有两个开关均开启且账号符合资格时才自动检测。 | 已从 comet-state.yaml 补齐本项；全局、账号和调度资格组合均在 modelTraceAutoEligible 校验，手动路径不受自动开关限制；正式组合测试覆盖缺失及非布尔值。 |
| A15 | passed | specs/modeltrace-degradation-detection/spec.md | 来源和持久化 - **WHEN** 同账号先后执行手动与自动检测后刷新页面或重启服务 - **THEN** 30 天内的记录可分页查询并准确区分来源、模型、轮次及结论；非管理员不能读取或触发检测。 | 已从 comet-state.yaml 补齐本项；SQL 历史区分 manual/auto、分页及 30 天保留，清理不抹掉独立摘要；全部检测路由位于 AdminAuth 管理员组，非管理员在中间件被拒绝，账号删除清理及认证正式检查通过。 |
| A16 | passed | specs/modeltrace-degradation-detection/spec.md | 多模型最近结果 - **WHEN** 同账号 Astra 检测降智后 Sol 检测正常，再发生一次请求失败 - **THEN** 状态列显示最近有效的正常结果并标明 Sol 和检测时间，历史保留 Astra 降智及最新失败；点击状态进入该账号历史页签。 | 成功 Finish 以递增任务 ID 更新最近有效摘要，不按模型聚合，失败不触碰摘要；状态显示最新模型及时间，历史保留此前结果和失败；Sol 可选且可提交，正式摘要保护及状态入口测试通过。 |
| A17 | passed | specs/modeltrace-degradation-detection/spec.md | 并发与中断 - **WHEN** 多实例或手动/自动同时请求检测同一账号，或执行中服务中断 - **THEN** 同账号最多一个任务执行，其他触发复用或等待既有任务；中断留下可查询的失败结果，不以正常/降智掩盖故障，也不自动重发已经发送的轮次。 | 持久化活动任务唯一、集群并发上限和 owner/deadline 围栏防重叠；失效 owner 不接管重放，保留 100 秒发送排空窗口后明确中断失败；数据库恢复、取消、正常退出及真实 PostgreSQL 失效租约检查通过。 |

## 检查

| 检查 | 命令 | 工作目录 | 状态 | 退出码 | 耗时 |
| --- | --- | --- | --- | ---: | ---: |
| 后端检测回归 | test ./internal/service ./internal/repository ./internal/handler/admin ./internal/handler/dto ./internal/pkg/modeltrace ./cmd/server -run ModelTrace\|UpstreamAstraReference\|ChallengeAndInvalidSamples\|Cleanup\|Wire -count=1 | backend | passed | 0 | 27506 ms |
| 后端构建 | build ./... | backend | passed | 0 | 8538 ms |
| 前端检测回归 | -NoProfile -Command pnpm test:run src/components/account/__tests__/ModelTraceModal.spec.ts src/components/account/__tests__/ModelTraceStatus.spec.ts src/views/admin/__tests__/ModelTraceSettings.spec.ts src/api/admin/__tests__/modeltrace.spec.ts src/views/admin/__tests__/AccountsView.lite.spec.ts src/components/account/__tests__/CreateAccountModal.spec.ts src/components/account/__tests__/EditAccountModal.spec.ts src/views/admin/__tests__/AccountsView.sparkShadow.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/SettingsView.gatewayRuntime.spec.ts src/i18n/__tests__/localesMessageCompile.spec.ts src/i18n/__tests__/localesNoKeyCollision.spec.ts; exit $LASTEXITCODE | frontend | passed | 0 | 21720 ms |
| 前端类型检查 | -NoProfile -Command pnpm typecheck; exit $LASTEXITCODE | frontend | passed | 0 | 36107 ms |
| 翻译完整性 | -NoProfile -Command pnpm check:i18n; exit $LASTEXITCODE | frontend | passed | 0 | 26626 ms |
| Independent PostgreSQL persistence leases retention deletion | -NoProfile -Command $env:SUB2API_MODELTRACE_POSTGRES='1'; $env:GOPROXY='off'; go test -tags modeltrace_postgres -run '^TestModelTracePersistenceLeasesRetentionAndDeletion$' -count=1 -v ./internal/repository; exit $LASTEXITCODE | backend | passed | 0 | 13027 ms |
| Frontend production build to isolated temporary directory | -NoProfile -Command if (!(Test-Path -LiteralPath 'C:/Users/caiqy/AppData/Local/Temp/opencode')) { exit 1 }; pnpm exec vite build --outDir C:/Users/caiqy/AppData/Local/Temp/opencode/modeltrace-verifier-build-e305c9e8 --emptyOutDir; exit $LASTEXITCODE | frontend | passed | 0 | 42506 ms |
| HTTP redirect prevention and administrator authentication regression | test -tags unit ./internal/repository ./internal/server/middleware -run TestHTTPUpstreamDoCanDisableRedirectsPerRequest\|TestAdminAuthJWTValidatesTokenVersion -count=1 -v | backend | passed | 0 | 21513 ms |

### Builder 报告的证据

以下为 Builder 报告，不等同于 Runtime 检查凭据或独立验收结果。

- 后端聚焦测试: passed — service/repository/admin handler/DTO/core/server的ModelTrace及相关回归通过
- 前端相关测试: passed — 12文件283项通过，包含Sol选择提交
- 类型/i18n/ESLint/前端构建: passed — 前端Builder完成typecheck、check:i18n、改动文件ESLint与临时目录构建
- 后端构建与vet: passed — go build ./...及受影响包go vet通过，Wire生成通过
- 真实PostgreSQL集成: passed — 127.0.0.1:55479专用临时PostgreSQL17.6实例，modeltrace_postgres tag原集成测试通过，测试数据库已删除
- 已知限制: 未用真实上游账号消耗额度实测；样本对照不能证明线上识别准确率。
- 已知限制: 便携PostgreSQL专用集成入口需要显式SUB2API_MODELTRACE_POSTGRES=1与本地测试实例；未跑整个Docker集成套件。
- 已知限制: 浏览器核查使用模拟API，非真实后端联调。

## 阻塞项

_无。_

## 风险与跳过的工作

- 未使用真实上游账号发送探针或消耗额度；固定样本与模拟传输验证不能证明线上模型识别准确率，也不能作为模型身份的确定性证明。
- 未启动真实业务服务进行浏览器到后端的端到端联调；前端验收依据组件测试、实际源码和生产构建，现有 Builder 浏览器记录仅作为线索。
- 真实 PostgreSQL 正式检查覆盖专用隔离数据库、迁移、并发创建、租约、保留、摘要和删除，但未运行整个 Docker 集成套件；外部可选 OAuth 插件及真实上游传输环境未作实测。
- 前端测试/构建存在 Browserslist 数据陈旧、既有 router-link stub、jsdom 请求及分包/体积警告，检查均成功退出，不据此宣称全库无警告。

## 之前的迭代

| 目标周期 | 迭代 | 尝试 | 结果 | 未解决项 | 摘要 | 完成时间 |
| ---: | ---: | ---: | --- | --- | --- | --- |
| 1 | 1 | 1 | pass | — | 独立只读 Verifier 已核查 brief、完整 Spec、A1-A17、实际实现及候选绑定的 Runtime 回执，最后参考 Builder 交接。未发现阻断验收的代码问题，17 项全部通过。正式 8 项检查通过：后端六包聚焦、后端构建、前端 12 文件 283 测试、类型、i18n，以及独立追加的真实 PostgreSQL、临时目录生产构建、HTTP 禁重定向和管理员认证回归。未修改候选代码，未启停 PostgreSQL，未执行 accept-result、归档或 Git 提交。 | 2026-09-26T08:11:16.562Z |



## 结论

独立只读 Verifier 已核查 brief、完整 Spec、A1-A17、实际实现及候选绑定的 Runtime 回执，最后参考 Builder 交接。未发现阻断验收的代码问题，17 项全部通过。正式 8 项检查通过：后端六包聚焦、后端构建、前端 12 文件 283 测试、类型、i18n，以及独立追加的真实 PostgreSQL、临时目录生产构建、HTTP 禁重定向和管理员认证回归。未修改候选代码，未启停 PostgreSQL，未执行 accept-result、归档或 Git 提交。
