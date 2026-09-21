# 目标

把 upstream 在 `v0.2.6` 中发布、随后于 `v0.2.7` 被整体撤销删除的 Codex 292 `x-codex-turn-state` 打票功能重新引入当前 `main` 基线（`0.2.7.1`），在保留 fork 既有定制与 `v0.2.7` 修复的前提下恢复该能力，并补齐账号级开关与配套配置项。

# 范围

## Source coverage

来源为 upstream 仓库中已恢复的 `v0.2.6` release 提交 `49a39b6dc1abed30fd227611e8af1108bc427610`（其功能增量等价于 PR #7315 的 6 个提交，`bc47e212b`…`3c2f05c9`），并参考两个把该功能重新集成到 `v0.2.7` 的第三方 fork 分支（`1057300248/sub2api` 的 `integrate/0.2.6-plus-0.2.7`、`haitun001/sub2api` 的 `integration/v0.2.7-codex292`）。上游 `v0.2.6` 的 Release 产物与 Docker 镜像已删除且不可恢复，本 change 只以源码为来源。

| 单元 | 来源定位 | 读取状态 | 保留语义 | Spec 位置 | 验收 ID | 覆盖状态 |
| --- | --- | --- | --- | --- | --- | --- |
| S1 打票核心 | `backend/internal/service/openai_codex_ticket.go`、`openai_codex_ticket_test.go`、`openai_codex_ticket_lifecycle_test.go` | complete | 合成探针铸造 292 门票、按 `(账号, 模型)` 落库与取用、TTL 与提前刷新、后台常驻 harvester 生命周期、跨模型去重、门票脱敏与不可伪造 | `specs/openai-codex-ticket-harvest/spec.md` | A1, A5, A6, A8 | covered |
| S2 出站注入点 | `openai_gateway_forward.go`、`openai_gateway_messages.go`、`openai_gateway_passthrough.go`、`openai_ws_forwarder_payload.go`、`openai_ws_forwarder_support.go` | complete | 在 `/responses`、Messages 桥、passthrough、WS 路径出站前按出站模型覆盖 `x-codex-turn-state` | 同上 | A1, A8 | covered |
| S3 调度门控 | `openai_gateway_scheduling.go`、`openai_account_runtime_block_fastpath.go`、`openai_account_scheduler.go`、`openai_account_runtime_block_fastpath_test.go`、`openai_account_runtime_transient_test.go`、`openai_guardian_affinity_test.go` | complete | 按与注入同一个出站模型判定门票可用性，含 compact 兜底模型口径一致；缺票时按配置决定是否阻断该账号调度 | 同上 | A4, A8 | covered |
| S4 回合状态守卫 | `openai_codex_turn_state.go` | complete | 下游会话到铸造账号的溯源表、跨账号回带剥离、延迟提交路径的提交点记录与机会式清扫 | 同上 | A8 | covered |
| S5 探针传输 | `backend/internal/repository/http_upstream.go`、`backend/internal/service/http_upstream_profile.go`、`http_upstream_test.go`、`http_upstream_profile_test.go` | complete | 打票专用 profile：HTTP/1.1、禁用 keep-alive、每发新建 CONNECT，使打票代理可按连接轮换出口 IP | 同上 | A1, A6 | covered |
| S6 全局配置 | `backend/internal/config/config.go`、`deploy/config.example.yaml`、`backend/internal/service/domain_constants.go`、`setting_gateway_runtime.go`、`setting_parse.go`、`setting_service.go`、`setting_update.go`、`settings_view.go`、`backend/cmd/server/wire.go`、`wire_gen.go` | complete | 功能总开关（后台热更新）、打票代理、目标长度、TTL、提前刷新、探测间隔、尝试超时、缺票策略、受管模型列表；默认关闭 | 同上 | A3 | covered |
| S7 管理端接口 | `backend/internal/handler/admin/{account_handler.go,account_data.go,setting_handler.go,setting_handler_audit.go,setting_handler_update.go}`、`handler/dto/{settings.go,types.go,mappers.go}`、`backend/internal/handler/wire.go`、`backend/internal/service/admin_account.go` | complete | 设置读写与审计、账号 DTO 携带只读门票状态、导出与编辑路径剥离系统托管的门票物料 | 同上 | A5, A7 | covered |
| S8 门票持久化 | `backend/internal/repository/account_repo.go` | complete | 门票写入不触发调度快照重建；账号编辑不会覆盖或伪造系统托管门票键 | 同上 | A5, A6 | covered |
| S9 前端 | `frontend/src/views/admin/SettingsView.vue`、`frontend/src/api/admin/settings.ts`、`frontend/src/components/account/{EditAccountModal.vue,AccountUsageCell.vue}`、`frontend/src/types/index.ts`、中英文 i18n `admin/{accounts,settings}.ts` | complete | 后台 Codex 设置区的总开关与打票代理字段、账号列表与编辑弹窗的门票状态展示 | 同上 | A7 | covered |
| S10 测试与契约 | 上述 Go 测试、`backend/internal/server/api_contract_test.go`、`frontend/src/views/admin/__tests__/SettingsView.spec.ts`、`frontend/src/components/account/__tests__/AccountUsageCell.spec.ts` | complete | 打票/注入/门控/生命周期/脱敏/设置读写/前端展示的回归覆盖，随实现一并落地并适配当前 `main` | 同上 | A9, A10 | covered |
| S11 账号级开关 | 用户本次新增要求；`v0.2.6` 中不存在账号级开关，仅存在隐式资格过滤与只读状态展示 | complete | 账号级布尔开关，默认关闭，显式开启才参与；全局开关为总开关，全局关闭时账号级开关失效 | 同上 | A2, A4 | covered |
| S12 额外配置项 | 用户本次新增要求；`v0.2.6` 第一版曾存在账号级打票代理覆盖（`codex_harvest_proxy_url`），第二版删除 | complete | 增加探针节流（全局并发上限与探测周期）、打票失败可观测（状态与日志含最近尝试时间与未命中原因）、目标长度/TTL/提前刷新/探测周期/并发上限/总开关/打票代理的后台热更新；不重新引入账号级代理覆盖与账号级模型白名单 | 同上 | A3, A12, A13 | covered |

- 范围只覆盖恢复该能力与本次新增的账号级开关/配置项，不重新设计打票协议，不改变 `v0.2.7` 已经确立的 `x-codex-turn-state` 跨账号剥离守卫。
- 账号级开关为账号 `extra` 上的布尔键（默认关闭），全局开关为总开关；缺票拦截只在两个开关都开启的目标上生效。
- 探针节流为全局配置：探测周期与探针并发上限均无账号级覆盖。
- 上游 `v0.2.6` 的 `HarvestMaxAttempts`/`HarvestFailStreak`/`HarvestOverallTimeoutSeconds` 连抽实现从未接入任何调用链，已在 `d14054afc` 删除，不作为来源单元。

# 非目标

- 不把 `upstream/main` 或 `v0.2.7` 之后的任何 upstream 提交合并进来，不重新引入 `v0.2.6` 中被后续提交修复的缺陷。
- 不恢复 `v0.2.6` 的 Release 二进制、Docker 镜像或 GitHub Release 记录。
- 不修改打票协议本身（探针请求体、探针目标模型、292 长度门限与 `gAAAAA` 前缀判定的既有口径），除非某条配置项被明确采纳。
- 不新增账号级打票代理覆盖与账号级模型白名单，不把账号级开关接入账号批量编辑弹窗。
- 不新增与本能力无关的 Codex 设置、账号字段或前端页面。
- 不创建或推送 release tag，不触发 release workflow，不部署。

# 验收示例

- A1: 恢复后的实现包含打票与注入全链路：后台 harvester 对已开启打票开关且资格符合的账号，按全局探测周期对每个受管模型发起合成探针，仅接受 HTTP 200、长度等于目标长度、且带有 `gAAAAA` 前缀的 `x-codex-turn-state`；有效门票在 `(账号, 模型)` 维度可查，并在受管模型出站请求上覆盖该请求头。
- A2: 每个账号拥有独立的打票开关，默认关闭，存储在账号 `extra` 的 `codex_ticket_enabled` 布尔键；全局开关是总开关，全局关闭时账号级开关一律失效；只有「全局开 + 账号开 + 账号资格符合 + 出站模型属于受管列表」四项同时满足时才打票与注入；账号级关闭（或资格不符）的账号既不参与打票也不注入门票，且不受缺票拦截影响。
- A3: 全局配置项包含功能总开关、打票代理、目标长度、门票 TTL、提前刷新量、探测周期与探针并发上限，其中总开关/代理/目标长度/TTL/提前刷新/探测周期/并发上限均可在后台热更新且无需重启；功能默认关闭，未配置打票代理时不发起探针、不注入门票，并在失败原因中明确体现该原因；不新增账号级打票代理覆盖，也不新增账号级模型白名单。
- A4: 缺票拦截默认开启，且只作用于「全局开关开启 + 账号级开关为真 + 出站模型属于全局受管列表（默认 `gpt-6-astra`、`gpt-5.6-sol`）」的 `(账号, 模型)`；缺票时该账号对该模型不被调度，关闭该账号开关后其调度资格立即恢复；门控与注入使用同一个出站模型口径，`/responses/compact` 的兜底与映射模型不会把实际不需要门票的请求误拦。
- A5: 门票物料对管理员不可伪造也不可越权读取：账号创建/编辑/批量编辑/导入路径都会剥离或忽略客户端提交的门票键，导出与账号接口返回中对门票状态只暴露只读摘要，不泄露 state blob 与代理凭据。
- A6: 门票高频写入不引起调度快照重建或事务放大；打票探针与业务请求互不阻塞，探针连接不复用，账号指标与调度状态不被门票写入污染。
- A7: 管理端可见性符合本次决定：后台可开启/关闭功能并配置打票代理，账号列表与账号编辑弹窗按账号显示每个受管模型的门票可用状态与剩余时间，且不会把只读状态渲染成可写输入。
- A8: 既有 `x-codex-turn-state` 跨账号回带剥离守卫、下游会话溯源、延迟提交路径的提交点记录与指纹收敛语义保持成立，不因本次恢复而回退。
- A9: 恢复的实现与 `v0.2.7` 之后 `main` 上已存在的修复和 fork 定制共存：冲突处保留双方语义，发现回归先以失败测试复现再最小修复闭合。
- A10: 后端默认测试与 unit 测试、后端 lint、前端 ESLint/i18n/单测/类型检查、前后端构建、Ent/Wire 生成稳定性检查全部通过；Docker/Testcontainers 等环境不可用的检查明确记录未运行范围，不记为通过。
- A11: 最终由新的独立只读 Verifier 对全部验收项给出独立结论。
- A12: 探针调度受全局参数约束：探测周期默认 6 秒、探针并发上限默认 8，节流参数均为全局配置且不提供账号级覆盖；同一 `(账号, 模型)` 的并发探测去重，已有有效且未临近过期门票的目标在该周期跳过探测。
- A13: 每个受管 `(账号, 模型)` 的门票状态可读到最近一次尝试时间与未命中原因，至少可区分未配置代理、无访问令牌、传输错误、HTTP 状态非 200、长度不符与前缀不符；该原因同时进入日志，且状态中不包含门票密文与代理凭据。

# 约束与不变量

- 功能默认关闭；开启前后都必须保持网关既有转发语义，未受管模型与不符合资格的账号不受影响。
- 门票与打票代理凭据属于系统托管物料，任何管理员可提交的路径都不得写入或覆盖它们。
- 打票必须走独立代理且不复用连接；业务请求仍使用账号自身代理。
- 配置项只允许通过后台热更新改变那些被明确设计为可热更新的项，其余仍以配置文件为准。
- 账号级开关默认关闭且不影响其他账号；关闭某账号的开关必须立即解除其缺票拦截。
- 缺票拦截与注入必须使用同一个出站模型口径，不得因映射或兜底模型产生不一致判定。
- 无法同时保留上游恢复语义与 fork 定制时暂停并请求用户决定。
- 任何生产代码回归修复必须先有针对该回归的失败测试。

# 决策

- 使用当前干净的 `main`，隔离方式沿用 Runtime 默认的当前目录。
- 恢复方式以 `v0.2.6` 的最终实现（`49a39b6dc`）为语义基准，而不是 `v0.2.6` 第一版；`S12` 中已被上游删除的账号级代理覆盖只有用户明确要求时才重新引入。
- 恢复该能力不以重新合并 `v0.2.6` tag 的方式进行，避免把 `v0.2.6` 之后被修复的问题带回来。
- `Q1` 账号级开关语义：全局开关为总开关，账号级开关默认关闭，每个账号需单独开启才参与；全局关闭时账号级开关失效。
- `Q2` 缺票拦截语义：默认开启（缺票不调度）；作用域为「全局开关开启 + 账号级开关为真 + 出站模型属于全局受管模型列表」，判定粒度为 `(账号, 出站模型)`；非受管模型与未开启该账号的账号不受影响。
- `Q3` 增补配置项：采纳探针节流与打票失败可观测；采纳目标长度、TTL、提前刷新、探测周期、并发上限、总开关与打票代理的后台热更新；不采纳账号级打票代理覆盖、账号级模型白名单与账号级开关进入批量编辑。
- 探针节流参数为全局配置，不提供账号级覆盖。
- 不修改 `backend/cmd/server/VERSION`，版本对齐仍由 release 流程负责。
- 本次只完成实现与验收，不发布或部署。

# 未决问题

- 无。

# 验证预期

- 以 `(账号, 模型)` 为单位验收：打票命中与未命中、门票过期与提前刷新、账号级开关与全局总开关的组合、资格过滤、缺票拦截与 compact 口径一致。
- 以运行为单位的验收：探针并发上限与探测周期生效、同一目标去重、已有门票跳过、未命中原因与最近尝试时间可从状态与日志读到。
- 以管理端为单位的验收：后台设置读写、热更新即时生效、审计记录、账号级开关可写而门票状态只读、账号创建/编辑/批量/导入路径的门票键剥离与导出脱敏。
- 以回归为单位验收：`x-codex-turn-state` 跨账号剥离、指纹收敛、WS 与 Messages 桥路径、调度快照与事务行为不因本次恢复而回退。
- 使用仓库既有的后端与前端质量门禁，并按本机实际可运行范围记录结果。
