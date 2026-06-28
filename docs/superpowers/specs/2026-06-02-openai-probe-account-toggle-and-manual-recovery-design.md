# OpenAI 账号级探活开关 / 探活模型 + 手动恢复不再探活 设计

## 背景

当前 OpenAI layered scheduler 的探活子系统（`openAIAccountProbe`）在生产环境暴露了两个相关但独立的问题，dmit 服务器上的 `1codex-pomo-image` 账号同时命中了这两个问题，陷入永不恢复的 temp-unschedulable 死循环。

### 问题 1：探活模型与账号上游不兼容时永不恢复

`resolveProbeModel(account)` 选模型时优先取 `model_mapping` 第一个非通配键，取不到则回退固定的 `gpt-4o-mini`。但部分 OpenAI APIKey 账号指向第三方 OpenAI 兼容上游，且仅服务特定模型（例如图像专用上游只支持 `gpt-image-2`）。这类账号：

- 无 `model_mapping` 配置 → 回退到 `gpt-4o-mini`
- 上游对 `gpt-4o-mini` 返回 400 / 503 / 超时 → 探活永远失败
- 探活连续失败达 `ProbeMaxFailures` 后写 DB temp-unschedulable
- 探活 tick 仍每 60s 跑，仍失败，仍调 `setTempUnschedulable` → DB 窗口被反复刷新到 `now + 30min`
- 真实流量（`gpt-image-2`）实际上工作正常，但账号永远不可调度

当前设计中没有「这个账号不该被探活」或「这个账号该用别的模型探活」的表达方式。

### 问题 2：手动恢复反被立即探活打回原状

`applyManualRecovery` 在清掉 DB temp flag、清 EWMA、删 entry 之后，会同步触发 `probeImmediatelyAfterManualRecovery`。该立即探活若失败，会立刻：

1. 调 `setTempUnschedulable` 重新写 DB temp flag（30 分钟窗口）
2. `p.entries.Store` 一个 `consecutiveFail=1` 的新 entry

结果：管理员手动恢复 = 立刻被验证否决 = 账号瞬间又回到 temp-unschedulable。生产日志中能看到该模式连续多轮发生。

设计意图是「手动恢复后立即验证账号是否真的可用」，但当上游真不可达 / 探活模型本身不兼容时，验证必然失败，进而推翻管理员的恢复决策。语义上验证不应有权否决管理员的显式操作。

## 目标

1. 为 OpenAI 账号增加两个账号级配置项：
   - **故障自动检查（probe enabled）**：账号级开关，关闭后该账号不再进入 layered probe 子系统
   - **自检模型（probe model）**：账号级显式探活模型，留空则维持原选模逻辑
2. 关闭「故障自动检查」时，若账号当前正处于 `source=layered_probe` 的 temp-unschedulable 状态，立即清除该标记并恢复调度
3. 删除手动恢复时的「立即重测」逻辑，让手动恢复就是恢复
4. 提供一键修复 dmit `1codex-pomo-image` 类问题账号的运维路径
5. 完全向后兼容：默认启用探活，存量账号行为不变

## 非目标

- 不改临时不可用窗口期内每 60s 探活、失败重置 30 分钟窗口的现有行为
- 不引入「到期前体检」/「预过期探活」等新生命周期阶段
- 不重写 layered scheduler 选号逻辑、运行时 EWMA 惩罚体系
- 不动其他平台（Claude / Gemini / Antigravity），probe 是 OpenAI 专属机制
- 不改其他来源的 temp-unschedulable 写入路径（OAuth 401、429 fallback、keyword 规则、流超时等）
- 不做旧数据迁移、不引入新 DB 列
- 不改 admin 端「清除错误」按钮的现有行为（已经是直接清，不走探活）

## 方案选型

### 「探活开关」拦截层 —— 选择方案 1

候选：

1. **入口拦截**：所有把账号注册进 probe 的入口加 guard，关闭的账号根本不进 `entries` map。同时复用现有 `ClearTempUnschedulable` 链路实现「关闭即恢复」
2. **tick 层 gating**：账号照常注册，tick 执行前判断开关、关闭则早退
3. **运行时惩罚层硬排除**：在 `evaluateRuntimePenalty` / `markPenalized` 处整体跳过

最终选 **方案 1**。

理由：
- 关闭的账号语义上「完全不进探活体系」，方案 1 最贴合
- 入口拦截后 entry 不存在，不需要额外处理 entry 的存留 / dbFlagSet 状态机
- 「关闭即恢复」可直接复用 `accountRepo.ClearTempUnschedulable` + `runtimeBlocker.ClearAccountSchedulingBlock`
- 拦截点虽然有 4 处，但每处都是单行 guard，单测覆盖明确
- 方案 3 会让账号在选号阶段也不再受 error/TTFT 惩罚影响，副作用过大

### 「探活模型」优先级 —— 显式配置最优先

候选：

1. **显式配置最优先，空则维持原逻辑**（model_mapping 首键 → `gpt-4o-mini`）
2. 显式配置最优先，空则不探活
3. 显式配置仅作默认回退，低于 model_mapping

选方案 1。理由：

- 完全向后兼容：留空时行为与现状一致，不影响存量账号
- 语义清晰：账号管理员显式表达"用这个模型探我"时，覆盖一切默认推断
- 将「是否探活」与「用什么模型探活」解耦为两个正交维度，更符合最小惊讶

### 「手动恢复」如何处理 —— 删除立即探活

候选：

1. 立即探活但不回写惩罚（之前讨论过的"非惩罚性"）
2. **彻底删除手动恢复时的立即探活**
3. 用 probe_enabled 开关 gating

选方案 2。理由：

- 用户明确反馈：手动恢复就是恢复，不应该走探活
- 探活在临时不可用窗口期内每 60s 已经在跑，「上线前体检」需求由该机制覆盖
- 立即探活的设计意图（验证账号真的可用）在上游真不可达时无法成立，反而成为否决管理员决策的工具
- 方案 1 仍然保留了立即探活的代码路径，未来可能再次被改成有副作用，不如彻底删干净

## 设计概述

### 1. 数据模型

两个字段都存在 `accounts.extra` JSON 中，与现有 `openai_compact_mode` / `openai_responses_mode` 同层。不新增 DB 列。

| 键 | 类型 | 默认（缺省时） | 含义 |
|---|---|---|---|
| `openai_probe_enabled` | bool | `true` | 是否允许该账号进入 layered probe 子系统 |
| `openai_probe_model` | string | `""` | 显式探活模型；留空走原逻辑 |

**默认值语义**：
- `openai_probe_enabled` 缺省 = `true`，与现有行为完全一致
- 显式写入 `false` 才视为关闭
- 这样存量账号无需迁移，新建账号默认开启

### 2. 后端 Account getter

在 `backend/internal/service/account.go` 的 OpenAI getter 区（`GetOpenAICompactMode` 附近）增加：

```go
// IsOpenAIProbeEnabled 是否允许 layered probe 接管该账号。
// 缺省视为 true，保持向后兼容。
func (a *Account) IsOpenAIProbeEnabled() bool

// GetOpenAIProbeModel 返回显式配置的探活模型；留空表示走原选模逻辑。
func (a *Account) GetOpenAIProbeModel() string
```

非 OpenAI 平台、Extra 为 nil、未设置该键时，`IsOpenAIProbeEnabled()` 返回 `true`；类型不是 bool 时也返回 `true`（保守默认）。

### 3. 探活入口拦截（4 处 guard）

所有把账号写入 `probe.entries` 的入口都需要 guard。这是方案 1 的核心。

#### 3.1 `markPenalized`（运行期选号触发）

`backend/internal/service/openai_account_scheduler_layered.go:314`：

```go
if eval.ErrorPenalized || eval.TTFTPenalized {
    s.probe.markPenalized(c.account.ID, req.GroupID, eval.ErrorPenalized, eval.TTFTPenalized)
}
```

调用前判断 `c.account.IsOpenAIProbeEnabled()`。关闭的账号：不调 `markPenalized`、也不调 `clearPenaltyReasons`（避免动到其他来源留下的状态）。该账号的运行时惩罚仍由 EWMA 自然衰减，但 probe 不再接管。

#### 3.2 `bootstrapRegister`（启动恢复）

`openai_account_probe.go:138 bootstrapRegister` 内部加 guard：参数中的 `account` 关闭探活时直接 return。同时调用方 `rehydrateTempUnschedulableEntries`（line 165）的过滤循环里，识别到 `openai_probe_enabled=false` 的账号也跳过并写跳过日志（reason 字段标 `probe_disabled_for_account`）。

#### 3.3 `ReattachLayeredProbeTempUnschedAccount`（运行时再接管）

`openai_gateway_service.go:589`：在 `account.IsActive() && account.Schedulable` 检查之后增加 `account.IsOpenAIProbeEnabled()` 检查。

#### 3.4 `tick` 防御性跳过

`openai_account_probe.go:281 tick` 的 `entries.Range` 循环里，`getSchedulableAccount` 返回账号后增加一道防御性 guard：若该账号 `IsOpenAIProbeEnabled() == false`，从 `entries` 中删除并跳过本轮。

这是「双保险」：理论上方案 1 的入口拦截已经保证关闭探活的账号不会出现在 entries 里，但配置可能在运行中被改（关闭探活），tick 需要能识别并清理已存在的孤儿 entry。

### 4. 关闭即恢复（账号更新触发点）

挂在 `adminServiceImpl.UpdateAccount`（`backend/internal/service/admin_service.go:2573`），在账号 extra 更新写入完成后增加。

**前置条件**：本次更新让账号的 `IsOpenAIProbeEnabled()` 从 `true` 翻转为 `false`。前置条件不满足时整段逻辑跳过。

**前置条件满足时分两层动作**：

第 1 层（一定执行 —— 清 runtime entry）：
- 调用 `OpenAIGatewayService.DropProbeEntry(accountID)`（新增的轻量入口）从 probe `entries` 中删除该账号 entry
- 这一步是无副作用、幂等的：entry 不存在时是 no-op，DB 状态不动

第 2 层（仅当 DB temp 状态来源是 layered_probe 时执行 —— 解封 DB 状态）：
- 判定方式：取最新账号的 `TempUnschedulableReason`，经 `parseTempUnschedReason` 识别 `source=layered_probe` 才进入
- 动作：
  1. `accountRepo.ClearTempUnschedulable(ctx, id)`
  2. `runtimeBlocker.ClearAccountSchedulingBlock(id)` （若实例存在）
  3. 写一条 INFO 日志：`probe: account toggle disabled, cleared layered_probe temp unschedulable`

**不触发整段逻辑的场景**：
- `openai_probe_enabled` 没翻转（包括从 false 翻 true 的反向操作）
- 账号本身不是 OpenAI 平台

**触发第 1 层但不触发第 2 层的场景**：
- 当前 DB temp 状态来源不是 layered_probe（例如 OAuth 401、429 fallback、keyword 规则、流超时写入的 temp 状态）→ 不动 DB，让原来源自己的恢复路径处理
- 账号当前没有 temp 标记 → 不需要清，跳过即可

第 2 层复用现有 `adminServiceImpl.ClearAccountError` 已经在用的清除链路，行为一致、可观测性一致。

### 5. 探活模型选择

修改 `openai_account_probe.go:553 resolveProbeModel`：

```go
func (p *openAIAccountProbe) resolveProbeModel(account *Account) string {
    if account == nil {
        return probeDefaultFallbackModel
    }
    // 新增：显式配置最优先
    if explicit := strings.TrimSpace(account.GetOpenAIProbeModel()); explicit != "" {
        return explicit
    }
    // 以下维持原逻辑：model_mapping 首键 → gpt-4o-mini
    mapping := account.GetModelMapping()
    ...
}
```

仅一处改动。空字符串保持原逻辑、非空覆盖一切。

### 6. 删除手动恢复立即探活

删除 / 改造点：

1. `openai_account_probe.go:877` `applyManualRecovery` 中删除对 `p.probeImmediatelyAfterManualRecovery(accountID)` 的调用
2. 整个 `probeImmediatelyAfterManualRecovery` 函数（`openai_account_probe.go:889-921`）删除
3. `getManualRecoveryProbeAccount`（`openai_gateway_service.go:923-930`）若再无其他调用方，一并删除
4. 相关测试（`openai_account_probe_test.go` 中针对 immediate probe 的测试用例）删除或改写

`applyManualRecovery` 保留功能：清 DB temp flag、清 EWMA、删 entry、写日志。这些都不变。

**行为变化**：
- 手动恢复后，账号立即变为可调度
- 若上游真不可达，下一笔真实流量打到该账号时会被 `evaluateRuntimePenalty` 重新打惩罚，正常进 probe 流程
- 整体收敛性不变，只是从「立即验证」改为「随真实流量验证」

### 7. 前端落点

OpenAI 账号高级配置区（与 `openai_compact_mode` 并排），三个 Modal 都需要改：

- `frontend/src/components/account/CreateAccountModal.vue`
- `frontend/src/components/account/EditAccountModal.vue`
- `frontend/src/components/account/BulkEditAccountModal.vue`

**字段呈现**：

1. **故障自动检查**（toggle / switch）
   - 默认 ON
   - 关闭时显示提示文案：「关闭后该账号不再被自动健康检查；当前若处于自动检查导致的临时不可用状态，保存后会立即恢复调度。」

2. **自检模型**（text input，可选）
   - placeholder 文案：「留空使用默认逻辑」
   - 帮助文案：「填写后该模型将用于账号健康检查；图像专用上游建议填 `gpt-image-2`」

**仅 OpenAI 平台显示**：BulkEdit 中已有 `openai_compact_mode` 的 platform 判断逻辑，照搬即可。两个新字段对 OpenAI APIKey 和 OAuth 两类账号都显示（与 `openai_compact_mode` 一致），因为 probe 子系统对 APIKey 和 OAuth 都生效。

**`buildOpenAIExtra()` 组装**（`CreateAccountModal.vue:4356`、`EditAccountModal.vue` 类似位置）：
- `openai_probe_enabled` 仅在用户显式设为 `false` 时写入；为 `true` 时 `delete extra.openai_probe_enabled`（默认行为）
- `openai_probe_model` 仅在非空时写入，否则 delete

这与 `openai_compact_mode` / `openai_responses_mode` 现有的处理范式完全一致。

**i18n 键**（`zh.ts` / `en.ts` 的 `admin.accounts.openai.*`）：
- `probeEnabled` / `probeEnabledDesc` / `probeEnabledOffHint`
- `probeModel` / `probeModelPlaceholder` / `probeModelHint`

### 8. 兼容性 / 升级路径

- DB schema 不变
- 存量账号 extra 中无新键 → `IsOpenAIProbeEnabled()` 返回 `true`、`GetOpenAIProbeModel()` 返回 `""` → 行为与升级前完全一致
- 唯一的行为变化是「手动恢复不再立即探活」，这对未中招的账号无可观测影响（探活立即成功的账号原本走 success 分支，与「不探活」结果等价）；对中招账号是修复

## 代码结构

### 主要改动位置

- `backend/internal/service/account.go`
  - 增加 `IsOpenAIProbeEnabled()` / `GetOpenAIProbeModel()` getter
- `backend/internal/service/openai_account_probe.go`
  - `bootstrapRegister` 增加 guard
  - `rehydrateTempUnschedulableEntries` 过滤循环增加 guard 和跳过日志
  - `tick` 防御性 guard
  - `resolveProbeModel` 增加显式配置优先逻辑
  - 删除 `probeImmediatelyAfterManualRecovery`
  - `applyManualRecovery` 删除立即探活调用
- `backend/internal/service/openai_account_scheduler_layered.go`
  - `markPenalized` 调用前增加 guard
- `backend/internal/service/openai_gateway_service.go`
  - `ReattachLayeredProbeTempUnschedAccount` 增加 guard
  - 删除 `getManualRecoveryProbeAccount`（若无其他调用方）
  - 增加 `DropProbeEntry(accountID)` 入口供 admin service 调用
- `backend/internal/service/admin_service.go`
  - `UpdateAccount` 在 extra 写入后挂「关闭即恢复」钩子
- 前端
  - `CreateAccountModal.vue` / `EditAccountModal.vue` / `BulkEditAccountModal.vue` 增加两个字段
  - `buildOpenAIExtra` 处理新键
  - i18n（zh / en）增加键

### 责任边界

- **Account（model）**：仅暴露配置 getter，不感知调度
- **probe 子系统**：尊重账号 `IsOpenAIProbeEnabled()` 决定是否接管，自身仍按原算法运行
- **scheduler layered**：调用 probe 前先问账号开关
- **admin service（UpdateAccount）**：账号配置翻转的副作用编排者，唯一持有「关闭即恢复」的判定逻辑
- **frontend**：UI 表达 + extra 组装，不引入业务规则

## 测试设计

### 1. Account getter 单测

- `IsOpenAIProbeEnabled` 在 nil account / nil extra / 缺键 / 非 bool 类型 / 显式 true / 显式 false 各场景下的返回值
- `GetOpenAIProbeModel` 在缺键 / 空字符串 / 非空字符串 / 非 string 类型下的返回值

### 2. probe 入口拦截单测

针对 `markPenalized` / `bootstrapRegister` / `ReattachLayeredProbeTempUnschedAccount` / `tick` 四处入口分别验证：

- 账号 `openai_probe_enabled=false` → entry 不被创建 / 已有 entry 被清除
- 账号 `openai_probe_enabled=true` 或缺省 → 行为不变（与现有测试保持一致）

### 3. 探活模型选择单测

`resolveProbeModel` 至少覆盖：

- 显式配置非空 → 返回显式值
- 显式配置空 + 有 model_mapping → 返回首键
- 显式配置空 + 无 model_mapping → 返回 `gpt-4o-mini`
- 显式配置为纯空白 → 视为空、走原逻辑
- 显式配置覆盖 model_mapping（即使 model_mapping 也有值，显式配置仍优先）

### 4. 关闭即恢复集成测试

`adminServiceImpl.UpdateAccount` 测试（按两层语义分别覆盖）：

第 1 层（清 entry）：
- 账号 probe_enabled 从 true 翻转为 false → `DropProbeEntry` 被调用一次（无论 entry 是否存在）
- 账号 probe_enabled 没翻转 / 从 false 翻 true / 非 OpenAI 账号 → `DropProbeEntry` 不被调用

第 2 层（解封 DB）：
- 账号当前 temp 状态 source=layered_probe + probe_enabled 翻转为 false → `ClearTempUnschedulable` + `ClearAccountSchedulingBlock` 被调用，且写入预期日志
- 账号当前 temp 状态 source!=layered_probe（例如 OAuth401 来源）+ probe_enabled 翻转为 false → `ClearTempUnschedulable` 不被调用
- 账号当前无 temp 标记 + probe_enabled 翻转为 false → `ClearTempUnschedulable` 不被调用
- 账号当前 temp 状态 source=layered_probe，但 probe_enabled 没翻转 → 第 2 层不触发

### 5. 手动恢复路径回归

- `applyManualRecovery` 调用后，DB temp flag 已清、entry 已删、EWMA 已清、**不再有任何探活请求发出**
- 删除对应原有「立即探活成功 / 失败」分支的测试用例

### 6. 端到端场景验证

模拟 dmit `1codex-pomo-image` 场景：

- 账号 OpenAI APIKey、无 model_mapping、上游对 `gpt-4o-mini` 返回 400
- 默认配置（probe_enabled=true）下，账号被 probe 反复标记 → 重现死循环
- 关闭 probe_enabled → 立即恢复、tick 不再标记
- 替代路径：保留 probe_enabled=true、设置 probe_model=`gpt-image-2`（mock 上游对该模型返回 200） → 探活成功、自动恢复

## 验证标准

实现完成后，应能证明：

1. 默认配置下所有现有 OpenAI 账号行为与升级前完全一致（向后兼容）
2. 关闭账号 `openai_probe_enabled` 后：
   - 账号不再被 markPenalized / bootstrap / reattach
   - 已有 entry 被清理
   - 若当前 temp 状态来源是 layered_probe，立即恢复调度
   - 其他来源的 temp 状态不被误清
3. 设置 `openai_probe_model` 后，探活请求发送的模型为该值
4. 手动恢复（`applyManualRecovery`）调用后不再发出任何探活请求
5. dmit `1codex-pomo-image` 类账号通过任一路径（关开关 / 配模型）可恢复正常调度
6. 现有 layered scheduler / dual-penalty / startup rehydrate / 其他 temp-unsched 来源逻辑均未被改动

## 风险与取舍

### 风险 1：关闭探活后该账号的运行时惩罚不再被 probe 接管恢复

关闭探活的账号若被 EWMA 打上 error/TTFT 惩罚，没有 probe 主动恢复，需要等 EWMA 自然衰减。这是有意识的取舍：

- 关闭探活的账号本身就是「我知道这个账号探活有问题，但它真实流量正常」的场景
- 该账号被运行时惩罚意味着真实流量也有问题，让 EWMA 自然衰减是合理的
- 如果完全不希望被惩罚，应通过其他手段（专属分组、独占调度）解决，不在本设计范围

### 风险 2：手动恢复不再立即探活，管理员失去快速反馈

管理员点恢复后无法立刻知道账号是否真的可用，需要等下一笔真实流量。取舍：

- 这正是用户明确的需求：恢复就是恢复，不该被验证否决
- 探活在 temp 窗口期内每 60s 仍会跑（不变），实际可用性反馈延迟最多 60s
- 想要主动验证可用性，应通过独立的「测试账号」按钮（已有功能），不应耦合在恢复路径里

### 风险 3：tick 防御性 guard 与入口拦截的语义重叠

入口拦截已经保证关闭的账号不进 entries，tick 防御性 guard 看似冗余。保留它是因为：

- 配置可能在运行中被改（账号刚被 markPenalized 后立刻被改成 `probe_enabled=false`，第 4 节"关闭即恢复"会清 entry 但存在竞态窗口）
- tick guard 是 idempotent 的最后防线，代价极低（一次 bool 检查）
- 可观测性：通过日志能确认 tick 在 guard 处确实跳过了哪些孤儿 entry

### 风险 4：扩展更多 probe 配置项时 extra 键膨胀

未来如果探活需要更多账号级配置（间隔、超时等），extra 中会出现一组 `openai_probe_*` 键。当前两个键尚不构成膨胀，但需注意若超过 4-5 个时考虑收敛为嵌套对象 `openai_probe: {...}`。本期不做。

## 结论

本次设计采用「**extra JSON 字段 + 入口拦截 + 复用现有清除链路 + 删除立即探活**」组合方案：

- 两个新字段（`openai_probe_enabled` / `openai_probe_model`）完全照搬 `openai_compact_mode` 的成熟范式
- 探活开关在 4 个入口处拦截，关闭账号不进 probe 体系
- 「关闭即恢复」挂在 `UpdateAccount` 上，复用 `ClearTempUnschedulable` 链路，提供一键修复
- 探活模型留空走原逻辑，显式配置最优先，向后完全兼容
- 删除手动恢复时的立即探活，让恢复就是恢复

这样既能修复 dmit 上 `1codex-pomo-image` 类问题账号的死循环，也修复了手动恢复被立即探活打回原状的设计缺陷，且不改动任何现有正常工作的链路。
