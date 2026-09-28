---
generated_from_state_version: 19
---

# 验证

## 当前结果

- 结果: **已归档**
- 验证情况: **已完成检查，验证结果已确认**
- 目标周期: 1
- 迭代: 5
- 验证器尝试次数: 1
- 完成时间: 2026-09-28T07:53:55.420Z
- 摘要: Independent read-only fifth-round review of A1-A30, complete brief/Spec, prior four outcomes, candidate source and current five passed Runtime receipts. The A6/A22 first-send fence and in-flight multi-round behavior, and the A28 immutable 241 plus additive 243 upgrade path, are satisfied.

## 验收

| 编号 | 结果 | 来源 | 验收项 | 原因 |
| --- | --- | --- | --- | --- |
| A1 | passed | brief.md | **WHEN** 账号未配置独立检测模型且全局模型为 `gpt-6-astra` | An absent account override resolves to the global default. |
| A2 | passed | brief.md | **THEN** 后续自动任务使用 `gpt-6-astra`；修改全局模型只影响未配置账号的后续任务。 ### Scenario: 账号模型覆盖全局模型 | Automatic enqueue snapshots the inherited global model for subsequent tasks. |
| A3 | passed | brief.md | **WHEN** 账号配置 `gpt-6-sol` 且全局模型为 `gpt-6-astra` | A configured GPT account override is accepted independently of the global model. |
| A4 | passed | brief.md | **THEN** 后续自动任务使用 `gpt-6-sol`，不使用全局模型。 ### Scenario: 模型配置快照与排队任务 | Automatic enqueue selects and persists the account override, not the global model. |
| A5 | passed | brief.md | **WHEN** 账号模型配置在任务排队后、探针发送前发生变化 | Queued automatic tasks compare persisted override provenance and selected model before the first send. |
| A6 | passed | brief.md | **THEN** 旧排队任务不发送请求并记录失败或跳过原因；已发送任务不被强行取消，完成后保留原模型历史。 ### Scenario: 手动模型列表按平台限定 | The final first-round check after the repository fence records a failed unsent task on override change; completed first rounds continue on their snapshot. |
| A7 | passed | brief.md | **WHEN** 管理员打开 OpenAI 账号的手动检测弹窗或全局 ModelTrace 设置 | Global settings and OpenAI manual modal obtain the ModelTrace model catalog. |
| A8 | passed | brief.md | **THEN** 列表来自 ModelTrace 当前 GPT 模型，包含未被该账号映射配置的模型，但不包含 Claude 等非 OpenAI 模型。 ### Scenario: 直接调用所选模型 | The catalog selects GPT family fingerprints and is not filtered by account mappings. |
| A9 | passed | brief.md | **WHEN** 管理员或自动任务选择 `gpt-6-sol` 发起检测 | Manual and automatic requests retain their selected GPT model ID. |
| A10 | passed | brief.md | **THEN** 探针使用所选账号的凭据、协议、代理，并将 `gpt-6-sol` 作为实际上游模型 ID；不得改用账号映射目标、全局模型或其他账号。 ### Scenario: 所选模型暂不可调用 | Probe payload uses the selected ID with the selected account's credential owner, protocol, and proxy; no mapping substitution. |
| A11 | passed | brief.md | **WHEN** 配置模型仍属于 ModelTrace 列表，但账号凭据、协议或上游不支持该模型 | Catalog validation permits configuration without current account capability or credentials. |
| A12 | passed | brief.md | **THEN** 配置仍可保存；检测实际尝试所选模型，失败时记录脱敏失败原因，不静默换用其他模型。 | Credential and upstream failures become sanitized task history; no fallback target is selected. |
| A13 | passed | specs/modeltrace-degradation-detection/spec.md | 账号和模型隔离 - **WHEN** 管理员选择某 OpenAI 账号及一个当前 ModelTrace GPT 模型发起检测 - **THEN** 探针使用该账号的凭据归属、协议及业务代理，直接发送选择的模型 ID，不得换用其他账号、映射目标或全局模型。 | Probe and transport tests verify account identity, parent credential ownership, proxy and unmodified target. |
| A14 | passed | specs/modeltrace-degradation-detection/spec.md | OpenAI 模型列表 - **WHEN** 管理员打开全局 ModelTrace 设置或 OpenAI 账号的手动检测弹窗 - **THEN** 列表来自当前 ModelTrace GPT 模型，包含未被账号映射配置的模型，不包含 Claude 或其他非 OpenAI 家族模型。 | Global and account-specific model APIs expose the GPT-only fingerprint list without mapping filtering. |
| A15 | passed | specs/modeltrace-degradation-detection/spec.md | 不可调用模型 - **WHEN** 所选模型仍在 ModelTrace GPT 列表中，但账号凭据、协议或上游暂不支持该模型 - **THEN** 配置可以保存，检测实际尝试该模型并记录脱敏失败原因，不改用其他模型。 | Valid catalog choices save without capability checks and failed probes preserve a sanitized failure without substitution. |
| A16 | passed | specs/modeltrace-degradation-detection/spec.md | 单轮与多轮评分 - **WHEN** 分别完成 1、2 或 3 轮有效挑战 - **THEN** 各自使用相应样本数校准，归因第一名等于实际所选模型则记录正常，否则记录降智；概率和指纹库版本可查询。 | All requested valid samples are scored with sample-count calibration and the selected target determines normal versus degraded. |
| A17 | passed | specs/modeltrace-degradation-detection/spec.md | 部分失败 - **WHEN** 请求 3 轮检测但仅 2 轮有效，或实际所选模型返回上游错误 - **THEN** 历史记录为失败并展示有效轮次和脱敏原因，不自动补发、不覆盖账号此前有效结论。 | Partial rounds and upstream errors finish as failures without publishing a new conclusion or hidden retry. |
| A18 | passed | specs/modeltrace-degradation-detection/spec.md | 关闭并重新打开 - **WHEN** 手动任务执行中关闭弹窗再重新打开 - **THEN** 任务继续运行，界面恢复其进度或结果，不因打开动作重复发起模型请求。 | The modal restores active task history after reopening without starting another request. |
| A19 | passed | specs/modeltrace-degradation-detection/spec.md | 热更新和关闭 - **WHEN** 管理员保存有效设置或关闭自动检测 - **THEN** 后续调度使用新配置，关闭后不再启动排队或新自动任务，手动检测仍可用；非法参数拒绝保存且不改变原配置。 | Settings validation rejects invalid updates; hot reads and automatic eligibility honor the global switch while manual starts remain available. |
| A20 | passed | specs/modeltrace-degradation-detection/spec.md | 开关和模型继承 - **WHEN** 全局开启但账号自动开关关闭，或账号未配置模型覆盖 - **THEN** 账号在前一种情况下不自动检测但仍可手动检测；在后一种情况下使用当前全局模型创建后续自动任务。 | Account auto switch gates scheduling; absent override inherits the global model; manual starts ignore automatic switches. |
| A21 | passed | specs/modeltrace-degradation-detection/spec.md | 账号模型覆盖 - **WHEN** 全局模型为 `gpt-6-astra`，账号覆盖模型为 `gpt-6-sol` - **THEN** 该账号后续自动任务使用 `gpt-6-sol`，不使用全局模型。 | Account model selection takes precedence over the global setting at automatic task creation. |
| A22 | passed | specs/modeltrace-degradation-detection/spec.md | 模型配置快照 - **WHEN** 账号模型覆盖在自动任务排队后、探针发送前发生变化 - **THEN** 旧排队任务不发送请求并记录配置变化原因；已发送任务使用创建时模型完成，历史保留其模型。 | Last check before first probe skips changed overrides; after a valid first round no override recheck aborts later rounds, preserving the task snapshot and history. |
| A23 | passed | specs/modeltrace-degradation-detection/spec.md | 账号模型暂不可用 - **WHEN** 账号保存了合法 ModelTrace 模型但当前凭据、协议或上游不能调用它 - **THEN** 保存成功，自动或手动检测实际调用该模型并记录失败，不回退其他模型。 | Unsupported current account credentials or upstream capability fail against the selected model without fallback. |
| A24 | passed | specs/modeltrace-degradation-detection/spec.md | 手动与自动降智 - **WHEN** 已开启隔离策略的账号完成手动或自动检测且有效结论为降智 - **THEN** 整个账号进入临时不可调用状态，其他模型也不被业务调度选中，检测历史记录所选模型。 | Latest successful degraded result applies account-level quarantine and retains the selected model in history. |
| A25 | passed | specs/modeltrace-degradation-detection/spec.md | 隔离期间仍可自动复检 - **WHEN** 账号仅因 ModelTrace 降智处于临时不可调用状态且自动检测开关仍开启 - **THEN** 账号按其有效间隔继续接受使用有效账号模型的 ModelTrace 复检；有效正常结论可解除隔离，失败、超时或继续降智不得放开业务调度。 | Own ModelTrace quarantine does not prevent scheduled recovery probes, while independent account blockers still do. |
| A26 | passed | specs/modeltrace-degradation-detection/spec.md | 手动恢复与策略关闭 - **WHEN** 管理员手动检测得到有效正常结论，或关闭账号的降智隔离策略 - **THEN** 解除仅由 ModelTrace 造成的隔离；若存在其他独立原因，账号仍保持不可调用。 | Successful normal detection or disabling the policy clears only ModelTrace quarantine, retaining other blockers. |
| A27 | passed | specs/modeltrace-degradation-detection/spec.md | 账号级最新有效结论 - **WHEN** 同账号模型 A 有效降智、随后模型 B 有效正常，接着发生一次失败检测 - **THEN** 模型 A 的降智先使账号隔离，模型 B 的正常解除隔离，最后失败不重新隔离；评分历史保留各次模型与结论。 | The latest successful task updates the account summary/quarantine; subsequent failures leave it unchanged. |
| A28 | passed | specs/modeltrace-degradation-detection/spec.md | 来源和持久化 - **WHEN** 同账号先后执行手动与自动检测后刷新页面或重启服务 - **THEN** 30 天内的记录可分页查询并准确区分来源、所选模型、实际请求模型、轮次及结论；非管理员不能读取或触发检测。 | 241 matches its committed original blob exactly; embedded 243 adds model_override for existing databases, repository scans and paginated history include it without exposing secrets; existing admin boundary remains. |
| A29 | passed | specs/modeltrace-degradation-detection/spec.md | 多模型最近结果 - **WHEN** 同账号 Astra 检测降智后 Sol 检测正常，再发生一次请求失败 - **THEN** 状态列显示最近有效的正常结果并标明 Sol 和检测时间，历史保留 Astra 降智及最新失败；点击状态进入该账号历史页签。 | Account status uses the retained latest valid model/time and the history entry point; failed tasks do not replace the summary. |
| A30 | passed | specs/modeltrace-degradation-detection/spec.md | 并发与中断 - **WHEN** 多实例或手动/自动同时请求检测同一账号，或执行中服务中断 - **THEN** 同账号最多一个任务执行，其他触发复用或等待既有任务；中断留下可查询的失败结果，不以正常/降智掩盖故障，也不自动重发已经发送的轮次。 | Database lease, account active-slot and interruption recovery prevent overlapping or replayed tasks; focused checks passed. |

## 检查

| 检查 | 命令 | 工作目录 | 状态 | 退出码 | 耗时 |
| --- | --- | --- | --- | ---: | ---: |
| 后端 ModelTrace regression | test ./internal/service ./internal/repository ./internal/handler ./internal/handler/admin ./cmd/server -count=1 | backend | passed | 0 | 186507 ms |
| Backend vet | vet ./internal/service ./internal/repository ./internal/handler ./internal/handler/admin | backend | passed | 0 | 3486 ms |
| Frontend ModelTrace tests | exec vitest -- --run src/components/account/__tests__/CreateAccountModal.spec.ts src/components/account/__tests__/EditAccountModal.spec.ts src/components/account/__tests__/ModelTraceModal.spec.ts | frontend | passed | 0 | 13464 ms |
| Frontend typecheck | run typecheck | frontend | passed | 0 | 36068 ms |
| Frontend build | run build | frontend | passed | 0 | 100647 ms |

### Builder 报告的证据

以下为 Builder 报告，不等同于 Runtime 检查凭据或独立验收结果。

- 最终 ModelTrace 服务与仓储定向测试: passed — TestModelTrace* 与账号状态查询回归通过，包含发送前变更和已发送多轮任务回归。
- PostgreSQL 集成测试: passed — 手工临时 PostgreSQL 17.6 三项集成测试通过；测试库删除，55479 已确认关闭。
- 迁移与差异检查: passed — 241 原始迁移未修改，243 增量迁移存在；gofmt 和 git diff --check 通过。
- 已知限制: 未执行真实上游账号调用；探针直接模型 ID、失败不回退和账号凭据/代理归属使用传输层测试覆盖。

## 阻塞项

_无。_

## 风险与跳过的工作

- No live upstream account call was authorized. Temporary PostgreSQL integration was reported manually passed for this candidate and port 55479 is closed, but that run is not a formal Runtime receipt; this round independently inspected migration and repository behavior instead.

## 之前的迭代

| 目标周期 | 迭代 | 尝试 | 结果 | 未解决项 | 摘要 | 完成时间 |
| ---: | ---: | ---: | --- | --- | --- | --- |
| 1 | 1 | 1 | fail | A5, A6, A12, A15, A22, A23 | 独立只读核验 30 项：24 passed、6 failed。清除账号覆盖时旧排队任务未失效；凭据缺失时入队前拒绝而非记录失败历史。当前候选绑定的六项 Runtime 检查通过；未调用真实上游，55479 已关闭。 | 2026-09-28T06:18:43.910Z |
| 1 | 2 | 1 | fail | A5, A6, A22 | 独立核验 30 项：27 passed、3 failed（A5/A6/A22）。持久化覆盖快照、延迟凭据解析和 PostgreSQL 列扫描修复有效；自动任务覆盖检查后至实际发送前仍存在配置变更竞态。六项 Runtime 正式检查均 passed。 | 2026-09-28T06:57:06.414Z |
| 1 | 3 | 1 | fail | A6, A22, A28 | Fresh independent read-only third-round verifier reviewed all A1-A30, the complete brief and target Spec, both preserved Runtime failure histories, actual implementation, and all six current-candidate Runtime checks/logs. Result: 27 passed, 3 failed (A6, A22, A28). The post-repo.Check first-send override fence and deferred credential-failure history are implemented, but multi-round in-flight completion regresses and editing applied migration 241 breaks existing-database upgrades. All six Runtime checks passed, including fresh temporary PostgreSQL; no additional checks or code changes were made. | 2026-09-28T07:22:55.759Z |
| 1 | 4 | 0 | recovery | — | Builder handoff Runtime checks failed: backend-modeltrace-postgres | 2026-09-28T07:34:55.990Z |
| 1 | 5 | 1 | pass | — | Independent read-only fifth-round review of A1-A30, complete brief/Spec, prior four outcomes, candidate source and current five passed Runtime receipts. The A6/A22 first-send fence and in-flight multi-round behavior, and the A28 immutable 241 plus additive 243 upgrade path, are satisfied. | 2026-09-28T07:53:55.420Z |



## 结论

Independent read-only fifth-round review of A1-A30, complete brief/Spec, prior four outcomes, candidate source and current five passed Runtime receipts. The A6/A22 first-send fence and in-flight multi-round behavior, and the A28 immutable 241 plus additive 243 upgrade path, are satisfied.
