---
generated_from_state_version: 21
---

# 验证

## 当前结果

- 结果: **已归档**
- 验证情况: **已完成检查，验证结果已确认**
- 目标周期: 1
- 迭代: 5
- 验证器尝试次数: 1
- 完成时间: 2026-09-21T02:20:38.929Z
- 摘要: 新的独立只读最终 Verifier 完整核对 A1-A46，全部 passed。iteration5 五项后端正式检查全绿，前端与Ent/Wire证据未受后续改动影响；当前Codex差异为93个任务内文件，两个测试稳定性修复均为独立提交。

## 验收

| 编号 | 结果 | 来源 | 验收项 | 原因 |
| --- | --- | --- | --- | --- |
| A1 | passed | brief.md | A1: 恢复后的实现包含打票与注入全链路：后台 harvester 对已开启打票开关且资格符合的账号，按全局探测周期对每个受管模型发起合成探针，仅接受 HTTP 200、长度等于目标长度、且带有 `gAAAAA` 前缀的 `x-codex-turn-state`；有效门票在 `(账号, 模型)` 维度可查，并在受管模型出站请求上覆盖该请求头。 | 打票、校验、存取及出站注入实现与证据未变。 |
| A2 | passed | brief.md | A2: 每个账号拥有独立的打票开关，默认关闭，存储在账号 `extra` 的 `codex_ticket_enabled` 布尔键；全局开关是总开关，全局关闭时账号级开关一律失效；只有「全局开 + 账号开 + 账号资格符合 + 出站模型属于受管列表」四项同时满足时才打票与注入；账号级关闭（或资格不符）的账号既不参与打票也不注入门票，且不受缺票拦截影响。 | 账号开关、全局开关、资格及模型门控成立。 |
| A3 | passed | brief.md | A3: 全局配置项包含功能总开关、打票代理、目标长度、门票 TTL、提前刷新量、探测周期与探针并发上限，其中总开关/代理/目标长度/TTL/提前刷新/探测周期/并发上限均可在后台热更新且无需重启；功能默认关闭，未配置打票代理时不发起探针、不注入门票，并在失败原因中明确体现该原因；不新增账号级打票代理覆盖，也不新增账号级模型白名单。 | 全局配置、热更新及代理缺失行为成立。 |
| A4 | passed | brief.md | A4: 缺票拦截默认开启，且只作用于「全局开关开启 + 账号级开关为真 + 出站模型属于全局受管列表（默认 `gpt-6-astra`、`gpt-5.6-sol`）」的 `(账号, 模型)`；缺票时该账号对该模型不被调度，关闭该账号开关后其调度资格立即恢复；门控与注入使用同一个出站模型口径，`/responses/compact` 的兜底与映射模型不会把实际不需要门票的请求误拦。 | 缺票门控与 compact 出站模型口径一致。 |
| A5 | passed | brief.md | A5: 门票物料对管理员不可伪造也不可越权读取：账号创建/编辑/批量编辑/导入路径都会剥离或忽略客户端提交的门票键，导出与账号接口返回中对门票状态只暴露只读摘要，不泄露 state blob 与代理凭据。 | 门票物料写入边界与脱敏成立。 |
| A6 | passed | brief.md | A6: 门票高频写入不引起调度快照重建或事务放大；打票探针与业务请求互不阻塞，探针连接不复用，账号指标与调度状态不被门票写入污染。 | 独立传输及调度中性持久化成立。 |
| A7 | passed | brief.md | A7: 管理端可见性符合本次决定：后台可开启/关闭功能并配置打票代理，账号列表与账号编辑弹窗按账号显示每个受管模型的门票可用状态与剩余时间，且不会把只读状态渲染成可写输入。 | 管理端设置、账号开关及只读状态成立。 |
| A8 | passed | brief.md | A8: 既有 `x-codex-turn-state` 跨账号回带剥离守卫、下游会话溯源、延迟提交路径的提交点记录与指纹收敛语义保持成立，不因本次恢复而回退。 | 跨账号守卫与既有语义保持。 |
| A9 | passed | brief.md | A9: 恢复的实现与 `v0.2.7` 之后 `main` 上已存在的修复和 fork 定制共存：冲突处保留双方语义，发现回归先以失败测试复现再最小修复闭合。 | 当前 93 文件范围复核通过；Ollama 独立修复已移出差异。 |
| A10 | passed | brief.md | A10: 后端默认测试与 unit 测试、后端 lint、前端 ESLint/i18n/单测/类型检查、前后端构建、Ent/Wire 生成稳定性检查全部通过；Docker/Testcontainers 等环境不可用的检查明确记录未运行范围，不记为通过。 | iteration5 backend-default、unit、lint、build、diff 全部 exit0；前端2649、lint/build及Ent/Wire350 hash沿用未受后续改动影响的正式证据。内存断言保持20MiB输入与12MiB阈值。 |
| A11 | passed | brief.md | A11: 最终由新的独立只读 Verifier 对全部验收项给出独立结论。 | 新的独立只读 Verifier 已完成全量判断。 |
| A12 | passed | brief.md | A12: 探针调度受全局参数约束：探测周期默认 6 秒、探针并发上限默认 8，节流参数均为全局配置且不提供账号级覆盖；同一 `(账号, 模型)` 的并发探测去重，已有有效且未临近过期门票的目标在该周期跳过探测。 | 周期、并发、去重及新鲜票跳过成立。 |
| A13 | passed | brief.md | A13: 每个受管 `(账号, 模型)` 的门票状态可读到最近一次尝试时间与未命中原因，至少可区分未配置代理、无访问令牌、传输错误、HTTP 状态非 200、长度不符与前缀不符；该原因同时进入日志，且状态中不包含门票密文与代理凭据。 | 失败摘要、尝试时间及脱敏日志成立。 |
| A14 | passed | specs/openai-codex-ticket-harvest/spec.md | 账号级默认关闭 - **WHEN** 一个符合资格的账号从未设置 `codex_ticket_enabled` - **THEN** 该账号不参与打票、不注入门票，也不受缺票拦截影响，即使全局开关处于开启状态 | 账号级默认关闭成立。 |
| A15 | passed | specs/openai-codex-ticket-harvest/spec.md | 全局总开关关闭 - **WHEN** 全局开关关闭，且多个账号的 `codex_ticket_enabled` 为真 - **THEN** 系统不发起任何探针、不注入任何门票，也不对任何账号执行缺票拦截 | 全局关闭停止探测、注入与门控。 |
| A16 | passed | specs/openai-codex-ticket-harvest/spec.md | 全局与账号均开启 - **WHEN** 全局开关开启，且账号 `codex_ticket_enabled` 为真 - **THEN** 该账号对其受管模型参与打票与注入 | 双开关开启后的探测与注入成立。 |
| A17 | passed | specs/openai-codex-ticket-harvest/spec.md | 资格过滤优先于账号开关 - **WHEN** 某账号不是 OpenAI OAuth-like（oauth 或 setup-token）、属于凭证影子账号，或状态不是 active - **THEN** 即使 `codex_ticket_enabled` 为真，该账号也不参与打票，也不被记入门票状态 | OAuth-like、非影子及 active 资格过滤成立。 |
| A18 | passed | specs/openai-codex-ticket-harvest/spec.md | 修改后即时生效 - **WHEN** 管理员在后台修改打票代理或探测周期并保存 - **THEN** 后续探测周期与后续业务请求按新值执行，且不需要重启或重新部署 | 运行时配置下一周期生效。 |
| A19 | passed | specs/openai-codex-ticket-harvest/spec.md | 非法配置被拒绝 - **WHEN** 管理员提交非法打票代理 URL（非 http/https/socks5/socks5h、缺少主机、含路径、查询或片段、端口越界） - **THEN** 系统拒绝保存并返回可读错误，不改变已存配置 | 代理 URL 原子校验成立。 |
| A20 | passed | specs/openai-codex-ticket-harvest/spec.md | 默认关闭 - **WHEN** 一个全新部署未做任何门票配置 - **THEN** 系统不发起探针、不注入门票、不执行缺票拦截 | 全新部署默认关闭。 |
| A21 | passed | specs/openai-codex-ticket-harvest/spec.md | 未配置代理时不打票 - **WHEN** 全局开关与账号级开关均开启，但打票代理为空 - **THEN** 系统不发起任何探针、不注入门票，并把该原因记录为可读的失败原因 | 无代理不探测、不注入并记录原因。 |
| A22 | passed | specs/openai-codex-ticket-harvest/spec.md | 探针形态 - **WHEN** 系统为某账号的受管模型发起打票尝试 - **THEN** 请求为目标端点的合成响应请求，携带该账号的访问令牌与 Codex 身份头，对 gpt-6/astra 类模型强制不低于内置最低版本的版本头，并只读取响应头而不读取响应体 | 探针请求形态与身份头成立。 |
| A23 | passed | specs/openai-codex-ticket-harvest/spec.md | 连接不复用 - **WHEN** 同一账号同一模型连续发起两次打票尝试 - **THEN** 两次尝试使用不同的底层连接，且打票代理参数与业务请求代理互相独立 | HTTP/1.1 非复用专用传输成立。 |
| A24 | passed | specs/openai-codex-ticket-harvest/spec.md | 长度不符不计入 - **WHEN** 探针返回 HTTP 200，但请求头值长度不等于目标长度，或缺少 `gAAAAA` 前缀 - **THEN** 系统丢弃该值、记录未命中原因，并在后续周期继续重试 | HTTP、长度及前缀严格判定成立。 |
| A25 | passed | specs/openai-codex-ticket-harvest/spec.md | 到期与提前刷新 - **WHEN** 已存门票在提前刷新量之内即将过期 - **THEN** 系统在下一次探测周期重新打票，并在新门票写入前继续使用尚未过期的旧门票 | TTL 与提前刷新成立。 |
| A26 | passed | specs/openai-codex-ticket-harvest/spec.md | 持续未命中不封顶 - **WHEN** 某账号的某受管模型长期未命中 - **THEN** 系统按探测周期持续重试，不设重试上限，也不因连续失败永久退出重试 | 持续未命中继续轮转重试。 |
| A27 | passed | specs/openai-codex-ticket-harvest/spec.md | 覆盖陈旧回带值 - **WHEN** 受管模型请求携带客户端回带的旧门票，且该账号已有有效门票 - **THEN** 出站请求头上的该值被替换为该账号的有效门票 | 有效门票覆盖旧回带值。 |
| A28 | passed | specs/openai-codex-ticket-harvest/spec.md | 跨账号回带被剥离 - **WHEN** 客户端回带的门票由其他账号铸造 - **THEN** 该值在该次出站请求上被剥离，不与其他账号的门票混用 | HTTP 与 WS 跨账号回带剥离成立。 |
| A29 | passed | specs/openai-codex-ticket-harvest/spec.md | 不受管模型不注入 - **WHEN** 出站模型不属于受管列表 - **THEN** 系统不注入也不移除该请求头 | 非受管模型不注入。 |
| A30 | passed | specs/openai-codex-ticket-harvest/spec.md | compact 口径一致 - **WHEN** 请求为 `/responses/compact` 且其出站模型被改写为非受管兜底模型 - **THEN** 门控与注入都不按客户端原始模型判定，该请求既不因缺票被拦截，也不会被注入门票 | compact 门控和注入口径一致。 |
| A31 | passed | specs/openai-codex-ticket-harvest/spec.md | 缺票不调度 - **WHEN** 全局与账号开关均开启，账号对某受管模型没有有效门票，且缺票拦截为默认开启 - **THEN** 该账号不被调度到该模型，且管理端可读到该模型处于被拦截状态 | fail-closed 缺票门控成立。 |
| A32 | passed | specs/openai-codex-ticket-harvest/spec.md | 关闭开关立即恢复 - **WHEN** 某账号因缺票被拦截，管理员随后关闭该账号的打票开关 - **THEN** 该账号对该模型的调度资格立即恢复，不再受门票影响 | 关闭账号开关立即恢复调度。 |
| A33 | passed | specs/openai-codex-ticket-harvest/spec.md | 非受管模型不受影响 - **WHEN** 账号对某模型的出站模型不属于受管列表 - **THEN** 该账号对该模型的调度不受门票是否存在影响 | 非受管模型不受缺票影响。 |
| A34 | passed | specs/openai-codex-ticket-harvest/spec.md | 并发上限生效 - **WHEN** 某轮探测需要发起的尝试数超过全局并发上限 - **THEN** 系统不超过该上限并发发探针，其余尝试留待后续周期 | 并发上限成立。 |
| A35 | passed | specs/openai-codex-ticket-harvest/spec.md | 已有门票跳过 - **WHEN** 某 `(账号, 模型)` 已有有效且未临近过期的门票 - **THEN** 该周期不对其发起探测 | 新鲜门票跳过探测。 |
| A36 | passed | specs/openai-codex-ticket-harvest/spec.md | 同一目标去重 - **WHEN** 同一 `(账号, 模型)` 在同一时间被多次触发探测 - **THEN** 实际最多只有一次探测在飞，其余调用等待或跳过 | 同账号模型并发探测去重。 |
| A37 | passed | specs/openai-codex-ticket-harvest/spec.md | 未命中原因可查 - **WHEN** 某账号的某受管模型连续未命中 - **THEN** 管理端可读到最近一次尝试时间与具体未命中原因，而不只是「未就绪」 | 状态暴露最近尝试与原因。 |
| A38 | passed | specs/openai-codex-ticket-harvest/spec.md | 未配置代理可辨识 - **WHEN** 全局开关与账号开关均开启但打票代理为空 - **THEN** 账号状态中该模型的未命中原因明确指向未配置代理 | 无代理原因可辨识。 |
| A39 | passed | specs/openai-codex-ticket-harvest/spec.md | 状态不含密文 - **WHEN** 管理端读取账号门票状态 - **THEN** 返回内容只包含摘要字段，不包含门票密文与打票代理凭据 | 状态不含门票或代理密文。 |
| A40 | passed | specs/openai-codex-ticket-harvest/spec.md | 管理员无法伪造门票 - **WHEN** 管理员在创建或编辑账号时提交 `codex_turn_ticket:<模型>` 或历史遗留打票代理键 - **THEN** 系统丢弃这些值，账号上只保留系统自身写入的门票 | 管理员无法伪造系统门票。 |
| A41 | passed | specs/openai-codex-ticket-harvest/spec.md | 导出脱敏 - **WHEN** 导出或备份账号数据 - **THEN** 门票密文与打票代理凭据不出现在导出内容中 | 导出脱敏成立。 |
| A42 | passed | specs/openai-codex-ticket-harvest/spec.md | 高频写入不放大调度 - **WHEN** 系统周期性写入门票 - **THEN** 该写入不引起调度快照重建，也不引起不必要的持久化事务 | 门票写入不放大调度事务。 |
| A43 | passed | specs/openai-codex-ticket-harvest/spec.md | 后台配置入口 - **WHEN** 管理员打开后台设置页 - **THEN** 可看到功能总开关与打票代理字段，代理密码以掩码形式回显，留空保存表示不修改已存值 | 设置入口与代理掩码保留语义成立。 |
| A44 | passed | specs/openai-codex-ticket-harvest/spec.md | 账号页面可读写分离 - **WHEN** 管理员打开账号编辑弹窗或账号列表 - **THEN** 可读写账号级打票开关，且只能只读查看每个受管模型的门票状态与剩余时间 | 账号开关可写、状态只读。 |
| A45 | passed | specs/openai-codex-ticket-harvest/spec.md | 守卫不回退 - **WHEN** 既有跨账号回带剥离与溯源相关测试执行 - **THEN** 全部通过，且新增门票逻辑不改变其判定 | 既有守卫语义与定向测试保持。 |
| A46 | passed | specs/openai-codex-ticket-harvest/spec.md | 发现回归先复现 - **WHEN** 在恢复或冲突处理中发现行为回归 - **THEN** 先新增针对该回归的失败测试，再以最小修复闭合，并重跑受影响与完整门禁 | Codex 回归均有红绿闭合；当前 Antigravity 失败无 Codex 因果。 |

## 检查

| 检查 | 命令 | 工作目录 | 状态 | 退出码 | 耗时 |
| --- | --- | --- | --- | ---: | ---: |
| 最终默认全量 | -NoProfile -ExecutionPolicy Bypass -File scripts/test.ps1 -p=1 ./... -count=1 | backend | passed | 0 | 445815 ms |
| 最终 unit 全量 | -NoProfile -ExecutionPolicy Bypass -File scripts/test.ps1 -tags=unit -p=1 ./... -count=1 | backend | passed | 0 | 665483 ms |
| 最终后端 lint | run ./... | backend | passed | 0 | 141841 ms |
| 最终后端构建 | build ./... | backend | passed | 0 | 10428 ms |
| 最终差异检查 | diff HEAD --check | . | passed | 0 | 318 ms |

### Builder 报告的证据

以下为 Builder 报告，不等同于 Runtime 检查凭据或独立验收结果。

- 失败复现: passed — iteration4 backend-default-final 在普通客户端 keepalive 断言失败；最小定向 count=5 复现同一偶发。
- SSE 定向重复验证: passed — 两个普通/Go GenAI 场景在 2200ms 窗口下 count=10 全部通过，44.204s。
- 范围与差异检查: passed — 两个独立测试修复均已单独提交，不在 Codex diff；当前 93 个唯一文件，git diff HEAD --check 通过。
- 已知限制: Docker/Testcontainers 与 race 因本机环境限制未运行。

## 阻塞项

_无。_

## 风险与跳过的工作

- Docker/Testcontainers 与 race 因环境限制未运行。

## 之前的迭代

| 目标周期 | 迭代 | 尝试 | 结果 | 未解决项 | 摘要 | 完成时间 |
| ---: | ---: | ---: | --- | --- | --- | --- |
| 1 | 1 | 1 | fail | A9, A10, A28, A46 | 独立只读Verifier ses_f3f8b0f1cffecbBKZQ1LpUfmVI覆盖A1–A46：42 passed、2 failed、2 blocked。WS构造器缺少已知跨账号回带剥离；正式默认全量仍失败。下一轮保留WS先失败再修复证据并真实重跑默认全量。 | 2026-09-20T20:28:17.243Z |
| 1 | 2 | 1 | fail | A10 | 独立只读Verifier ses_f3f51e5e7ffeZl332pd426tukk完整核对46项：45 passed、1 failed（A10）。WS修复和红绿证据成立。unit全量及串行补验均在OAuth文本释放内存断言失败；需修测试harvester清理遗漏并定位内存保留来源。 | 2026-09-20T21:36:43.855Z |
| 1 | 3 | 1 | pass | — | 新的独立只读最终Verifier ses_f3f1615ccffe2ahHI3D0TIVFU4核对完整46项全部passed。源码、第三轮Runtime五项exit0、未受影响前端2649测试与lint/build、Ent/Wire350hash稳定、WS红绿和A10根因及消融证据一致。保留20MiB请求及12MiB阈值，无放宽断言；Docker/race未运行明确记录。 | 2026-09-20T22:30:56.518Z |
| 1 | 3 | 1 | recovery | — | 范围审查发现一处独立的Ollama 429测试稳定性修复；将其从Codex候选移出，重新验证范围纯净的候选。 | 2026-09-21T01:06:43.332Z |
| 1 | 4 | 1 | fail | A10 | 独立最终 Verifier 判定 45 passed、1 failed；仅 A10 因既有 Antigravity SSE keepalive 定时测试失败。该失败与 Codex 93 文件差异无因果，但严格全绿条件未满足。 | 2026-09-21T01:44:23.235Z |
| 1 | 5 | 1 | pass | — | 新的独立只读最终 Verifier 完整核对 A1-A46，全部 passed。iteration5 五项后端正式检查全绿，前端与Ent/Wire证据未受后续改动影响；当前Codex差异为93个任务内文件，两个测试稳定性修复均为独立提交。 | 2026-09-21T02:20:38.929Z |



## 结论

新的独立只读最终 Verifier 完整核对 A1-A46，全部 passed。iteration5 五项后端正式检查全绿，前端与Ent/Wire证据未受后续改动影响；当前Codex差异为93个任务内文件，两个测试稳定性修复均为独立提交。
