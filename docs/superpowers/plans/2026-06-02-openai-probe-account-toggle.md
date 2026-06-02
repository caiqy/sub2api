# OpenAI 账号级探活开关 / 探活模型 + 手动恢复不再探活 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 OpenAI 账号增加 `openai_probe_enabled` / `openai_probe_model` 两个 extra 配置项，实现"关闭探活后立即恢复"的运维路径，并删除手动恢复时被立即探活打回原状的 bug。

**Architecture:** 字段存 `accounts.extra` JSON（与 `openai_compact_mode` 同层）。探活开关在 4 个入口处加 guard（markPenalized / bootstrapRegister / Reattach / tick），关闭账号不进 probe.entries。账号更新时若 probe_enabled 翻转 false，复用 `ClearTempUnschedulable` + `ClearAccountSchedulingBlock` 链路解封；若当前 temp 状态来源是 `layered_probe` 则同步清 DB。手动恢复路径删除 `probeImmediatelyAfterManualRecovery` 调用与函数本身。

**Tech Stack:** Go (backend) / Vue 3 + TypeScript (frontend) / PostgreSQL (`accounts.extra` JSONB)

**Spec:** `docs/superpowers/specs/2026-06-02-openai-probe-account-toggle-and-manual-recovery-design.md`

---

## 文件结构

### 后端

| 路径 | 操作 | 责任 |
|---|---|---|
| `backend/internal/service/account.go` | 修改 | 增加 `IsOpenAIProbeEnabled()` / `GetOpenAIProbeModel()` getter |
| `backend/internal/service/openai_account_probe.go` | 修改 | 4 处 guard、`resolveProbeModel` 优先级、删 `probeImmediatelyAfterManualRecovery` |
| `backend/internal/service/openai_account_scheduler_layered.go` | 修改 | `markPenalized` 调用前 guard |
| `backend/internal/service/openai_gateway_service.go` | 修改 | `Reattach...` guard、新增 `DropProbeEntry`、删 `getManualRecoveryProbeAccount` |
| `backend/internal/service/admin_service.go` | 修改 | `UpdateAccount` 末尾增加"关闭即恢复"钩子 |
| `backend/internal/service/account_openai_probe_config_test.go` | **新建** | getter 单测 |
| `backend/internal/service/admin_service_probe_toggle_test.go` | **新建** | "关闭即恢复"集成测试 |
| `backend/internal/service/openai_account_probe_test.go` | 修改 | 增加 4 处 guard 单测、`resolveProbeModel` 优先级单测 |
| `backend/internal/service/openai_account_probe_manual_recovery_test.go` | 修改 | 删除立即探活相关用例，新增"不再发探活"用例 |

### 前端

| 路径 | 操作 | 责任 |
|---|---|---|
| `frontend/src/components/account/CreateAccountModal.vue` | 修改 | OpenAI 配置区增加两个字段 + `buildOpenAIExtra` 处理 |
| `frontend/src/components/account/EditAccountModal.vue` | 修改 | 同上 |
| `frontend/src/components/account/BulkEditAccountModal.vue` | 修改 | 同上 |
| `frontend/src/i18n/locales/zh.ts` | 修改 | 新增 i18n 键 |
| `frontend/src/i18n/locales/en.ts` | 修改 | 新增 i18n 键 |

---

## 实现顺序

按 7 个阶段，每阶段独立可提交、可回滚：

1. 后端 Account getter（数据模型层，无副作用）
2. 探活模型选择优先级（`resolveProbeModel`，单点改动）
3. 4 处入口 guard（探活开关核心生效）
4. 删除手动恢复立即探活
5. `DropProbeEntry` 接口 + UpdateAccount 钩子（关闭即恢复闭环）
6. 前端 i18n 键
7. 前端三个 Modal

---

## 阶段 1：后端 Account getter

### Task 1：新增 `IsOpenAIProbeEnabled` / `GetOpenAIProbeModel`

**Files:**
- Modify: `backend/internal/service/account.go:660-668`（在 `GetOpenAICompactMode` 之后插入两个新 getter）
- Test: `backend/internal/service/account_openai_probe_config_test.go`（**新建**）

**说明：** 缺省 / nil / 类型错误时 `IsOpenAIProbeEnabled` 一律返回 `true`（保守默认，向后兼容）；非 OpenAI 平台返回 `true`（其他平台不会调用 probe，返回值不影响行为，但语义上"未启用 OpenAI probe = 不参与"也可，这里选 true 与现有 compact mode 对非 OpenAI 返回 auto 的保守默认一致）。`GetOpenAIProbeModel` 缺省返回 `""`，调用方据此走原回退逻辑。

- [ ] **Step 1：写 getter 失败测试**

新建文件 `backend/internal/service/account_openai_probe_config_test.go`，内容：

```go
//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccount_IsOpenAIProbeEnabled(t *testing.T) {
	cases := []struct {
		name string
		acc  *Account
		want bool
	}{
		{name: "nil account", acc: nil, want: true},
		{name: "non-openai platform", acc: &Account{Platform: PlatformAnthropic, Extra: map[string]any{"openai_probe_enabled": false}}, want: true},
		{name: "openai with nil extra", acc: &Account{Platform: PlatformOpenAI}, want: true},
		{name: "openai with key absent", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{}}, want: true},
		{name: "openai explicit true", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_enabled": true}}, want: true},
		{name: "openai explicit false", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_enabled": false}}, want: false},
		{name: "openai non-bool type falls back to true", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_enabled": "false"}}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.acc.IsOpenAIProbeEnabled())
		})
	}
}

func TestAccount_GetOpenAIProbeModel(t *testing.T) {
	cases := []struct {
		name string
		acc  *Account
		want string
	}{
		{name: "nil account", acc: nil, want: ""},
		{name: "nil extra", acc: &Account{Platform: PlatformOpenAI}, want: ""},
		{name: "key absent", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{}}, want: ""},
		{name: "empty string", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_model": ""}}, want: ""},
		{name: "non-string type", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_model": 123}}, want: ""},
		{name: "explicit value", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_model": "gpt-image-2"}}, want: "gpt-image-2"},
		{name: "value with surrounding whitespace returns raw", acc: &Account{Platform: PlatformOpenAI, Extra: map[string]any{"openai_probe_model": "  gpt-image-2  "}}, want: "  gpt-image-2  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.acc.GetOpenAIProbeModel())
		})
	}
}
```

注：`GetOpenAIProbeModel` 不去空白；空白裁剪在调用方 `resolveProbeModel` 处理（spec 第 5 节示例如此）。这样 getter 保持纯字符串读取，调用方控制语义。

- [ ] **Step 2：运行测试确认失败**

```
go test -tags=unit ./backend/internal/service -run "TestAccount_IsOpenAIProbeEnabled|TestAccount_GetOpenAIProbeModel" -v
```

Expected: 编译失败，`a.IsOpenAIProbeEnabled undefined` / `a.GetOpenAIProbeModel undefined`。

- [ ] **Step 3：实现 getter**

在 `backend/internal/service/account.go` 第 668 行（`GetOpenAICompactMode` 函数闭合的 `}` 之后、`OpenAICompactSupportKnown` 之前）插入：

```go
// IsOpenAIProbeEnabled reports whether the layered probe subsystem may
// take over this account. Missing or non-bool extra values fall back to
// true to preserve backward compatibility for accounts created before
// this option existed.
func (a *Account) IsOpenAIProbeEnabled() bool {
	if a == nil || !a.IsOpenAI() || a.Extra == nil {
		return true
	}
	v, ok := a.Extra["openai_probe_enabled"]
	if !ok {
		return true
	}
	enabled, ok := v.(bool)
	if !ok {
		return true
	}
	return enabled
}

// GetOpenAIProbeModel returns the explicit probe model configured on the
// account. Empty string means "fall back to the default selection logic
// in resolveProbeModel" (model_mapping first key, then gpt-4o-mini).
func (a *Account) GetOpenAIProbeModel() string {
	if a == nil || a.Extra == nil {
		return ""
	}
	v, ok := a.Extra["openai_probe_model"].(string)
	if !ok {
		return ""
	}
	return v
}
```

- [ ] **Step 4：运行测试确认通过**

```
go test -tags=unit ./backend/internal/service -run "TestAccount_IsOpenAIProbeEnabled|TestAccount_GetOpenAIProbeModel" -v
```

Expected: 全部 PASS（13 个子测试）。

- [ ] **Step 5：提交**

```
git add backend/internal/service/account.go backend/internal/service/account_openai_probe_config_test.go
git commit -m "feat(account): add IsOpenAIProbeEnabled and GetOpenAIProbeModel getters"
```

---

## 阶段 2：探活模型选择优先级

### Task 2：`resolveProbeModel` 增加显式配置优先

**Files:**
- Modify: `backend/internal/service/openai_account_probe.go:551-572`
- Test: `backend/internal/service/openai_account_probe_test.go`（追加测试）

**说明：** 显式配置（trim 后非空）覆盖一切；空字符串 / 纯空白维持原 model_mapping → gpt-4o-mini 逻辑。

- [ ] **Step 1：写失败测试**

在 `backend/internal/service/openai_account_probe_test.go` **末尾**追加（不要插到现有测试中间）：

```go
func TestResolveProbeModel_ExplicitOverridesModelMapping(t *testing.T) {
	probe := &openAIAccountProbe{}
	account := &Account{
		Platform: PlatformOpenAI,
		Extra:    map[string]any{"openai_probe_model": "gpt-image-2"},
		Credentials: map[string]any{
			"model_mapping": map[string]any{"gpt-4o": "upstream-foo"},
		},
	}
	require.Equal(t, "gpt-image-2", probe.resolveProbeModel(account))
}

func TestResolveProbeModel_EmptyExplicitFallsBackToMapping(t *testing.T) {
	probe := &openAIAccountProbe{}
	account := &Account{
		Platform: PlatformOpenAI,
		Extra:    map[string]any{"openai_probe_model": ""},
		Credentials: map[string]any{
			"model_mapping": map[string]any{"gpt-4o": "upstream-foo"},
		},
	}
	require.Equal(t, "gpt-4o", probe.resolveProbeModel(account))
}

func TestResolveProbeModel_WhitespaceExplicitFallsBackToMapping(t *testing.T) {
	probe := &openAIAccountProbe{}
	account := &Account{
		Platform: PlatformOpenAI,
		Extra:    map[string]any{"openai_probe_model": "   "},
		Credentials: map[string]any{
			"model_mapping": map[string]any{"gpt-4o": "upstream-foo"},
		},
	}
	require.Equal(t, "gpt-4o", probe.resolveProbeModel(account))
}

func TestResolveProbeModel_EmptyExplicitNoMappingFallsBackToDefault(t *testing.T) {
	probe := &openAIAccountProbe{}
	account := &Account{Platform: PlatformOpenAI}
	require.Equal(t, probeDefaultFallbackModel, probe.resolveProbeModel(account))
}

func TestResolveProbeModel_ExplicitOverridesEvenWithoutMapping(t *testing.T) {
	probe := &openAIAccountProbe{}
	account := &Account{
		Platform: PlatformOpenAI,
		Extra:    map[string]any{"openai_probe_model": "custom-model"},
	}
	require.Equal(t, "custom-model", probe.resolveProbeModel(account))
}
```

注：如果 `openai_account_probe_test.go` 头部已经 import `"github.com/stretchr/testify/require"` 则不需要额外加 import；若没有，导入它。

- [ ] **Step 2：运行测试确认失败**

```
go test -tags=unit ./backend/internal/service -run "TestResolveProbeModel_" -v
```

Expected: `TestResolveProbeModel_ExplicitOverridesModelMapping` 失败（返回 `gpt-4o` 而非 `gpt-image-2`）；`TestResolveProbeModel_ExplicitOverridesEvenWithoutMapping` 失败（返回 `gpt-4o-mini` 而非 `custom-model`）；其他三个 PASS（fall-through 行为已经正确）。

- [ ] **Step 3：实现**

修改 `backend/internal/service/openai_account_probe.go:551-572`：

```go
// resolveProbeModel 为探活请求选择模型。
// 优先级：账号 extra.openai_probe_model（trim 后非空） > model_mapping 第一个非通配键 > gpt-4o-mini。
func (p *openAIAccountProbe) resolveProbeModel(account *Account) string {
	if account == nil {
		return probeDefaultFallbackModel
	}
	if explicit := strings.TrimSpace(account.GetOpenAIProbeModel()); explicit != "" {
		return explicit
	}
	mapping := account.GetModelMapping()
	if len(mapping) > 0 {
		// 排序以保证确定性
		keys := make([]string, 0, len(mapping))
		for k := range mapping {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if k != "*" && strings.TrimSpace(k) != "" {
				return k
			}
		}
	}
	return probeDefaultFallbackModel
}
```

`strings` 已经在 import 列表里，无需新增 import。

- [ ] **Step 4：运行测试确认通过**

```
go test -tags=unit ./backend/internal/service -run "TestResolveProbeModel_" -v
```

Expected: 5 个测试全部 PASS。

- [ ] **Step 5：跑一遍 probe 包的全部已有测试，确认没回归**

```
go test -tags=unit ./backend/internal/service -run "Probe" -v
```

Expected: 全部 PASS。

- [ ] **Step 6：提交**

```
git add backend/internal/service/openai_account_probe.go backend/internal/service/openai_account_probe_test.go
git commit -m "feat(probe): explicit openai_probe_model overrides default selection"
```

---

## 阶段 3：4 处入口 guard

阶段 3 把"`openai_probe_enabled=false` 的账号不进 probe 子系统"落到 4 个入口。每个 sub-task 独立改一处 + 独立单测 + 独立提交，便于回滚。

### Task 3.1：`markPenalized` 调用前 guard（运行期选号触发）

**Files:**
- Modify: `backend/internal/service/openai_account_scheduler_layered.go:309-318`
- Test: `backend/internal/service/openai_account_probe_test.go`（追加测试）

**说明：** `markPenalized` 自身签名只接收 `accountID int64`、不持有账号对象，guard 必须放在调用方（`layeredOpenAIAccountScheduler` 候选评估循环）。关闭账号既不调 `markPenalized` 也不调 `clearPenaltyReasons`（避免误清其他来源留下的 entry 状态）。

- [ ] **Step 1：写失败测试**

在 `backend/internal/service/openai_account_probe_test.go` 末尾追加：

```go
func TestMarkPenalized_KeepsExistingSemanticsRegardlessOfAccountConfig(t *testing.T) {
	// 契约：markPenalized 自身不感知账号开关，调用方（layered scheduler 候选循环）负责 guard。
	// 本测试锁定 markPenalized 的现有语义不被破坏。
	probe := newOpenAIAccountProbe(nil, newOpenAIAccountRuntimeStats())
	t.Cleanup(probe.stop)

	probe.markPenalized(42, nil, true, false)
	_, ok := probe.entries.Load(int64(42))
	require.True(t, ok, "markPenalized must keep its existing semantics")
}
```

注：`markPenalized` 本身不持有账号对象，guard 在 layered scheduler 调用点。`scheduler` 的循环在测试基础设施里要构造完整 service / candidates 比较重，因此实际行为验证我们走 layered scheduler 测试。先在 `openai_account_probe_test.go` 占住一个语义提醒测试，主要的 guard 行为单测放在 layered scheduler 测试文件里：

新建测试文件 `backend/internal/service/openai_account_scheduler_probe_guard_test.go`：

```go
//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLayeredScheduler_MarkPenalizedSkipsProbeDisabledAccount(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	probe := newOpenAIAccountProbe(nil, stats)
	t.Cleanup(probe.stop)

	scheduler := &layeredOpenAIAccountScheduler{
		probe: probe,
		stats: stats,
	}

	// 模拟"账号关闭探活但被 evaluateRuntimePenalty 判定为 errorPenalized"的场景。
	// scheduler 候选循环里的 guard 必须先于 markPenalized 调用。
	disabled := &Account{
		ID:       77,
		Platform: PlatformOpenAI,
		Extra:    map[string]any{"openai_probe_enabled": false},
	}

	scheduler.applyProbeRegistration(disabled, true /*errorPenalized*/, false /*ttftPenalized*/, nil /*groupID*/)

	_, present := probe.entries.Load(int64(77))
	require.False(t, present, "probe-disabled account must not be registered to probe.entries")
}

func TestLayeredScheduler_MarkPenalizedRunsForProbeEnabledAccount(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	probe := newOpenAIAccountProbe(nil, stats)
	t.Cleanup(probe.stop)

	scheduler := &layeredOpenAIAccountScheduler{
		probe: probe,
		stats: stats,
	}

	enabled := &Account{
		ID:       78,
		Platform: PlatformOpenAI,
	}

	scheduler.applyProbeRegistration(enabled, true, false, nil)

	_, present := probe.entries.Load(int64(78))
	require.True(t, present, "probe-enabled account must be registered as before")
}

func TestLayeredScheduler_MarkPenalizedSkipsClearForProbeDisabledAccount(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	probe := newOpenAIAccountProbe(nil, stats)
	t.Cleanup(probe.stop)

	scheduler := &layeredOpenAIAccountScheduler{
		probe: probe,
		stats: stats,
	}

	// 预先放一个 entry（模拟其他来源已留下的状态），关闭探活的账号不应触发 clearPenaltyReasons。
	probe.entries.Store(int64(79), &openAIAccountProbeEntry{accountID: 79})

	disabled := &Account{
		ID:       79,
		Platform: PlatformOpenAI,
		Extra:    map[string]any{"openai_probe_enabled": false},
	}

	scheduler.applyProbeRegistration(disabled, false, false, nil)

	_, present := probe.entries.Load(int64(79))
	require.True(t, present, "probe-disabled account must not have its existing entry cleared")
}
```

注：上面用了一个新方法 `applyProbeRegistration` —— 这是为了让 guard 逻辑可独立测试而抽出的小工具方法（不强求，下一步直接在原位 inline 也可，但抽方法更干净，候选循环更短）。

- [ ] **Step 2：运行测试确认失败**

```
go test -tags=unit ./backend/internal/service -run "TestLayeredScheduler_MarkPenalized" -v
```

Expected: 编译失败，`scheduler.applyProbeRegistration undefined`。

- [ ] **Step 3：实现**

打开 `backend/internal/service/openai_account_scheduler_layered.go`，找到第 309-318 行的候选循环：

```go
for _, c := range candidates {
    eval := s.evaluateRuntimePenalty(c.account.ID, groupMinTTFT, hasGroupMin)
    acc := s.applyPenaltyToAccount(c.account, eval)

    if eval.ErrorPenalized || eval.TTFTPenalized {
        s.probe.markPenalized(c.account.ID, req.GroupID, eval.ErrorPenalized, eval.TTFTPenalized)
    } else {
        s.probe.clearPenaltyReasons(c.account.ID)
    }
    ...
}
```

替换其中的 5 行（从 `if eval.ErrorPenalized` 到 `}`）为：

```go
        s.applyProbeRegistration(c.account, eval.ErrorPenalized, eval.TTFTPenalized, req.GroupID)
```

然后在文件末尾（紧邻 `Stop()` 方法之后，最后 `}` 之前）追加新方法：

```go
// applyProbeRegistration 把候选账号的运行时惩罚结果转化为 probe 注册动作。
// 关闭探活（openai_probe_enabled=false）的账号既不会被 markPenalized，
// 也不会被 clearPenaltyReasons —— 避免误动其他来源留下的 entry 状态。
func (s *layeredOpenAIAccountScheduler) applyProbeRegistration(account *Account, errorPenalized, ttftPenalized bool, groupID *int64) {
	if s == nil || s.probe == nil || account == nil {
		return
	}
	if !account.IsOpenAIProbeEnabled() {
		return
	}
	if errorPenalized || ttftPenalized {
		s.probe.markPenalized(account.ID, groupID, errorPenalized, ttftPenalized)
		return
	}
	s.probe.clearPenaltyReasons(account.ID)
}
```

- [ ] **Step 4：运行测试确认通过**

```
go test -tags=unit ./backend/internal/service -run "TestLayeredScheduler_MarkPenalized|TestMarkPenalized_" -v
```

Expected: 4 个测试全部 PASS。

- [ ] **Step 5：跑 layered scheduler 全部测试，确认没回归**

```
go test -tags=unit ./backend/internal/service -run "LayeredScheduler" -v
```

Expected: 全部 PASS。

- [ ] **Step 6：提交**

```
git add backend/internal/service/openai_account_scheduler_layered.go backend/internal/service/openai_account_scheduler_probe_guard_test.go backend/internal/service/openai_account_probe_test.go
git commit -m "feat(scheduler): skip probe registration for accounts with probe disabled"
```

---

### Task 3.2：`bootstrapRegister` + `rehydrateTempUnschedulableEntries` guard（启动恢复）

**Files:**
- Modify: `backend/internal/service/openai_account_probe.go:138-163`（`bootstrapRegister` 内部 guard）
- Modify: `backend/internal/service/openai_account_probe.go:165-198`（`rehydrateTempUnschedulableEntries` 过滤循环增加跳过日志）
- Test: `backend/internal/service/openai_account_probe_test.go`（追加测试）

**说明：** 双层防御 —— `bootstrapRegister` 内部加 guard（任何调用方关闭账号都拦下），`rehydrateTempUnschedulableEntries` 过滤循环里也加显式 guard 让跳过日志可观测。

- [ ] **Step 1：写失败测试**

在 `backend/internal/service/openai_account_probe_test.go` 末尾追加：

```go
func TestBootstrapRegister_SkipsProbeDisabledAccount(t *testing.T) {
	probe := newOpenAIAccountProbe(nil, newOpenAIAccountRuntimeStats())
	t.Cleanup(probe.stop)

	disabled := &Account{
		ID:       91,
		Platform: PlatformOpenAI,
		Extra:    map[string]any{"openai_probe_enabled": false},
	}

	probe.bootstrapRegister(disabled, time.Now(), 60*time.Second)

	_, present := probe.entries.Load(int64(91))
	require.False(t, present, "bootstrapRegister must skip probe-disabled accounts")
}

func TestBootstrapRegister_RegistersProbeEnabledAccount(t *testing.T) {
	probe := newOpenAIAccountProbe(nil, newOpenAIAccountRuntimeStats())
	t.Cleanup(probe.stop)

	enabled := &Account{
		ID:       92,
		Platform: PlatformOpenAI,
	}

	probe.bootstrapRegister(enabled, time.Now(), 60*time.Second)

	_, present := probe.entries.Load(int64(92))
	require.True(t, present, "bootstrapRegister must keep registering probe-enabled accounts")
}
```

注：`time` 已经在文件 import 中。

- [ ] **Step 2：运行测试确认失败**

```
go test -tags=unit ./backend/internal/service -run "TestBootstrapRegister_Skips|TestBootstrapRegister_Registers" -v
```

Expected: `TestBootstrapRegister_SkipsProbeDisabledAccount` 失败（entry 仍被注册）。

- [ ] **Step 3：实现 `bootstrapRegister` guard**

修改 `backend/internal/service/openai_account_probe.go:138-141`：

```go
func (p *openAIAccountProbe) bootstrapRegister(account *Account, now time.Time, cooldown time.Duration) {
	if p == nil || account == nil || account.ID <= 0 || p.stopped.Load() {
		return
	}
```

替换为：

```go
func (p *openAIAccountProbe) bootstrapRegister(account *Account, now time.Time, cooldown time.Duration) {
	if p == nil || account == nil || account.ID <= 0 || p.stopped.Load() {
		return
	}
	if !account.IsOpenAIProbeEnabled() {
		return
	}
```

- [ ] **Step 4：在 `rehydrateTempUnschedulableEntries` 过滤循环里加显式跳过日志**

修改 `backend/internal/service/openai_account_probe.go:184-188`：

```go
		if !account.IsOpenAI() || !account.IsActive() || !account.Schedulable {
			skipped++
			slog.Debug("probe: startup rehydrate skipped account", "account_id", account.ID, "reason", "not_bootstrap_eligible")
			continue
		}
```

在该 if 块之后、`if account.TempUnschedulableUntil == nil` 之前插入：

```go
		if !account.IsOpenAIProbeEnabled() {
			skipped++
			slog.Debug("probe: startup rehydrate skipped account", "account_id", account.ID, "reason", "probe_disabled_for_account")
			continue
		}
```

- [ ] **Step 5：运行测试确认通过**

```
go test -tags=unit ./backend/internal/service -run "TestBootstrapRegister_|TestStartupRehydrate" -v
```

Expected: 全部 PASS。如有现有 `TestStartupRehydrate*` 测试用了 disabled 账号，行为应符合 skip 路径；若全部用默认账号则不受影响。

- [ ] **Step 6：提交**

```
git add backend/internal/service/openai_account_probe.go backend/internal/service/openai_account_probe_test.go
git commit -m "feat(probe): skip bootstrap and startup rehydrate for probe-disabled accounts"
```

---

### Task 3.3：`ReattachLayeredProbeTempUnschedAccount` guard（运行时再接管）

**Files:**
- Modify: `backend/internal/service/openai_gateway_service.go:589-618`
- Test: `backend/internal/service/openai_gateway_service_test.go`（追加测试）

**说明：** 这条路径在账号状态运行期变更时被调用（schedulerSnapshot 的 OpenAIAccountChangeHandler）。关闭探活的账号即使被运行期事件触发也不应被 reattach。

- [ ] **Step 1：写失败测试**

打开 `backend/internal/service/openai_gateway_service_test.go`，在文件末尾追加：

```go
func TestOpenAIGatewayService_ReattachSkipsProbeDisabledAccount(t *testing.T) {
	now := time.Now()
	future := now.Add(10 * time.Minute)
	reason, err := buildLayeredProbeTempUnschedReason("consecutive_failures", 3)
	require.NoError(t, err)

	repo := &startupRecoveryRepoStub{tempUnschedAccounts: []Account{{
		ID:                      201,
		Platform:                PlatformOpenAI,
		Type:                    AccountTypeAPIKey,
		Status:                  StatusActive,
		Schedulable:             true,
		TempUnschedulableUntil:  &future,
		TempUnschedulableReason: reason,
		Extra:                   map[string]any{"openai_probe_enabled": false},
	}}}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	snapshot := &SchedulerSnapshotService{}
	svc := &OpenAIGatewayService{accountRepo: repo, cfg: cfg, schedulerSnapshot: snapshot}

	svc.StartOpenAIBackgroundRecovery()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	scheduler := svc.getOpenAIAccountScheduler()
	layered, ok := scheduler.(*layeredOpenAIAccountScheduler)
	require.True(t, ok)

	// startup 阶段就应被跳过（Task 3.2 的功能），这里再触发一次 runtime reattach 验证 guard。
	require.NoError(t, snapshot.handleAccountEvent(context.Background(), ptrInt64(201), nil))

	_, present := layered.probe.entries.Load(int64(201))
	require.False(t, present, "ReattachLayeredProbeTempUnschedAccount must skip probe-disabled accounts")
}
```

- [ ] **Step 2：运行测试确认失败**

```
go test -tags=unit ./backend/internal/service -run "TestOpenAIGatewayService_ReattachSkips" -v
```

Expected: 失败（entry 被 reattach）。

- [ ] **Step 3：实现 guard**

修改 `backend/internal/service/openai_gateway_service.go:605-607`：

```go
	if !account.IsOpenAI() || !account.IsActive() || !account.Schedulable {
		return
	}
```

替换为：

```go
	if !account.IsOpenAI() || !account.IsActive() || !account.Schedulable {
		return
	}
	if !account.IsOpenAIProbeEnabled() {
		return
	}
```

- [ ] **Step 4：运行测试确认通过**

```
go test -tags=unit ./backend/internal/service -run "TestOpenAIGatewayService_Reattach" -v
```

Expected: 全部 PASS（含原有 reattach 测试）。

- [ ] **Step 5：提交**

```
git add backend/internal/service/openai_gateway_service.go backend/internal/service/openai_gateway_service_test.go
git commit -m "feat(probe): skip runtime reattach for probe-disabled accounts"
```

---

### Task 3.4：`tick` 防御性 guard（最后防线）

**Files:**
- Modify: `backend/internal/service/openai_account_probe.go:301-309`
- Test: `backend/internal/service/openai_account_probe_test.go`（追加测试）

**说明：** 防御性兜底 —— 配置可能在运行中被改，tick 在执行前再检查一次，发现孤儿 entry 就清掉。

- [ ] **Step 1：写失败测试**

在 `backend/internal/service/openai_account_probe_test.go` 末尾追加：

```go
func TestProbeTick_RemovesEntryWhenAccountProbeDisabled(t *testing.T) {
	repo := &probeTickAccountRepoStub{
		account: &Account{
			ID:          303,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
			Extra:       map[string]any{"openai_probe_enabled": false},
		},
	}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeIntervalSeconds = 60
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeCooldownSeconds = 1
	svc := &OpenAIGatewayService{accountRepo: repo, cfg: cfg}

	probe := newOpenAIAccountProbe(svc, newOpenAIAccountRuntimeStats())
	t.Cleanup(probe.stop)

	// 模拟一个孤儿 entry：之前账号是 enabled、被 markPenalized；之后被改成 disabled。
	probe.entries.Store(int64(303), &openAIAccountProbeEntry{
		accountID:   303,
		penalizedAt: time.Now().Add(-2 * time.Second), // 已过 cooldown
	})

	probe.tick()

	_, present := probe.entries.Load(int64(303))
	require.False(t, present, "tick must remove orphan entries for probe-disabled accounts")
}
```

`probeTickAccountRepoStub` 是一个最简的 repo stub，需要在测试文件里新增（如果还没有等价 stub）：在 `openai_account_probe_test.go` 顶部 `import` 之后、第一个测试之前查找已有 stub 类型；若没有可复用的，新增：

```go
type probeTickAccountRepoStub struct {
	mockAccountRepoForGemini
	account *Account
}

func (r *probeTickAccountRepoStub) GetByID(ctx context.Context, id int64) (*Account, error) {
	if r.account != nil && r.account.ID == id {
		return r.account, nil
	}
	return nil, ErrAccountNotFound
}
```

注：复用 `mockAccountRepoForGemini`（位于 `gemini_multiplatform_test.go:17`）的内嵌方式，规避补全 AccountRepository 接口里的所有方法。

`OpenAIGatewayService.getSchedulableAccount` 内部会读 `schedulerSnapshot` 或 `accountRepo`，本测试因为 `schedulerSnapshot` 为 nil 会走 repo 路径（具体见 service 实现）；如果发现 tick 走了 schedulerSnapshot 路径而绕过 repo，需在 stub 上同时覆盖 snapshot —— 实现到 step 3 时再核对，先按 repo 路径写。

- [ ] **Step 2：运行测试确认失败**

```
go test -tags=unit ./backend/internal/service -run "TestProbeTick_RemovesEntryWhenAccountProbeDisabled" -v
```

Expected: 失败（entry 没被清）。

- [ ] **Step 3：实现 guard**

修改 `backend/internal/service/openai_account_probe.go:301-309`：

```go
		if err != nil || account == nil || !account.IsOpenAI() {
			p.entries.Delete(accountID)
			return true
		}
		// 如果账号已被管理员标记为不可调度，移除 entry
		if !account.Schedulable {
			p.entries.Delete(accountID)
			return true
		}
```

在 `if !account.Schedulable` 块之后插入：

```go
		// 防御性 guard：账号探活开关被运行中改为关闭时，清理孤儿 entry。
		if !account.IsOpenAIProbeEnabled() {
			p.entries.Delete(accountID)
			slog.Debug("probe: tick removed orphan entry for probe-disabled account", "account_id", accountID)
			return true
		}
```

- [ ] **Step 4：运行测试确认通过**

```
go test -tags=unit ./backend/internal/service -run "TestProbeTick_" -v
```

Expected: PASS。

如果 step 3 中发现 tick 走了 schedulerSnapshot 路径绕过 repo，调整 stub 让它注入空 snapshot（具体看 `getSchedulableAccount` 的实现路径）。

- [ ] **Step 5：跑全部 probe 测试，确认没回归**

```
go test -tags=unit ./backend/internal/service -run "Probe" -v
```

Expected: 全部 PASS。

- [ ] **Step 6：提交**

```
git add backend/internal/service/openai_account_probe.go backend/internal/service/openai_account_probe_test.go
git commit -m "feat(probe): defensive tick guard removes orphan entries for disabled accounts"
```

---

## 阶段 4：删除手动恢复立即探活

### Task 4：移除 `probeImmediatelyAfterManualRecovery` 与所有调用

**Files:**
- Modify: `backend/internal/service/openai_account_probe.go:839-921`（`applyManualRecovery` 删除立即探活调用 + 整体删除 `probeImmediatelyAfterManualRecovery`）
- Modify: `backend/internal/service/openai_gateway_service.go:923-930`（删除 `getManualRecoveryProbeAccount`）
- Modify: `backend/internal/service/openai_account_runtime_block_fastpath.go:111-133`（重命名/简化 `probeAccountAfterManualTempUnschedulableClear`）
- Modify: `backend/internal/service/openai_account_probe_manual_recovery_test.go`（重写：删立即探活相关用例，新增「不再发探活」用例）

**说明：** 立即探活会在上游真不可达时把刚恢复的账号秒回临时不可用。这次彻底删除 —— 手动恢复 = 直接恢复，不验证。`ClearAccountSchedulingBlock` 仍调 `applyManualRecovery`（清 EWMA + 清 entry），但不再发探活。

- [ ] **Step 1：写新测试（替代旧用例）**

完整重写 `backend/internal/service/openai_account_probe_manual_recovery_test.go`，把它替换为：

```go
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type manualRecoverySnapshotCacheStub struct {
	account *Account
}

func (c *manualRecoverySnapshotCacheStub) GetSnapshot(context.Context, SchedulerBucket) ([]*Account, bool, error) {
	return nil, false, nil
}

func (c *manualRecoverySnapshotCacheStub) SetSnapshot(context.Context, SchedulerBucket, []Account) error {
	return nil
}

func (c *manualRecoverySnapshotCacheStub) GetAccount(context.Context, int64) (*Account, error) {
	if c.account != nil {
		cloned := *c.account
		return &cloned, nil
	}
	return nil, errors.New("stale snapshot missing account")
}

func (c *manualRecoverySnapshotCacheStub) SetAccount(context.Context, *Account) error { return nil }

func (c *manualRecoverySnapshotCacheStub) DeleteAccount(context.Context, int64) error { return nil }

func (c *manualRecoverySnapshotCacheStub) UpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}

func (c *manualRecoverySnapshotCacheStub) TryLockBucket(context.Context, SchedulerBucket, time.Duration) (bool, error) {
	return true, nil
}

func (c *manualRecoverySnapshotCacheStub) UnlockBucket(context.Context, SchedulerBucket) error {
	return nil
}

func (c *manualRecoverySnapshotCacheStub) ListBuckets(context.Context) ([]SchedulerBucket, error) {
	return nil, nil
}

func (c *manualRecoverySnapshotCacheStub) GetOutboxWatermark(context.Context) (int64, error) {
	return 0, nil
}

func (c *manualRecoverySnapshotCacheStub) SetOutboxWatermark(context.Context, int64) error {
	return nil
}

type manualRecoveryProbeRepo struct {
	stubOpenAIAccountRepo
	clearCalls int
	setCalls   int
}

func (r *manualRecoveryProbeRepo) ClearTempUnschedulable(context.Context, int64) error {
	r.clearCalls++
	return nil
}

func (r *manualRecoveryProbeRepo) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	r.setCalls++
	return nil
}

func TestProbeManualRecoveryDoesNotSendImmediateProbe(t *testing.T) {
	upstream := &openAIHTTPUpstreamRecorder{}
	repo := &manualRecoveryProbeRepo{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{{
		ID:          91,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "oauth-token",
			"expires_at":   "2999-01-01T00:00:00Z",
		},
	}}}}
	probe := &openAIAccountProbe{
		service: &OpenAIGatewayService{
			accountRepo:  repo,
			httpUpstream: upstream,
		},
		stats:  newOpenAIAccountRuntimeStats(),
		ctx:    context.Background(),
		stopCh: make(chan struct{}),
	}
	defer probe.stop()
	entry := &openAIAccountProbeEntry{accountID: 91}
	entry.dbFlagSet.Store(true)
	entry.ttftPenalized.Store(true)

	probe.applyManualRecovery(91, entry)

	require.Nil(t, upstream.lastReq, "manual recovery must not send any probe request")
	require.Equal(t, 1, repo.clearCalls, "manual recovery still clears DB temp unschedulable")
	require.Equal(t, 0, repo.setCalls, "manual recovery must not re-mark temp unschedulable")
}

func TestProbeManualRecoveryRemovesEntry(t *testing.T) {
	repo := &manualRecoveryProbeRepo{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{{
		ID:          92,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
	}}}}
	probe := &openAIAccountProbe{
		service: &OpenAIGatewayService{accountRepo: repo},
		stats:   newOpenAIAccountRuntimeStats(),
		ctx:     context.Background(),
		stopCh:  make(chan struct{}),
	}
	defer probe.stop()
	entry := &openAIAccountProbeEntry{accountID: 92}
	entry.dbFlagSet.Store(true)
	probe.entries.Store(int64(92), entry)

	probe.applyManualRecovery(92, entry)

	_, present := probe.entries.Load(int64(92))
	require.False(t, present, "manual recovery removes the probe entry")
}

func TestOpenAIManualTempUnschedulableClearDoesNotTriggerProbe(t *testing.T) {
	upstream := &openAIHTTPUpstreamRecorder{}
	repo := &manualRecoveryProbeRepo{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{{
		ID:          95,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
	}}}}
	probe := &openAIAccountProbe{
		stats:  newOpenAIAccountRuntimeStats(),
		ctx:    context.Background(),
		stopCh: make(chan struct{}),
	}
	defer probe.stop()
	svc := &OpenAIGatewayService{
		accountRepo:     repo,
		httpUpstream:    upstream,
		openaiScheduler: &layeredOpenAIAccountScheduler{probe: probe},
	}
	probe.service = svc
	svc.openaiAccountRuntimeBlockUntil.Store(int64(95), time.Now().Add(time.Minute))

	svc.ClearAccountSchedulingBlock(95)

	require.Nil(t, upstream.lastReq, "ClearAccountSchedulingBlock must not trigger any probe request")
	_, blockPresent := svc.openaiAccountRuntimeBlockUntil.Load(int64(95))
	require.False(t, blockPresent, "ClearAccountSchedulingBlock still clears the runtime block")
}

func TestOpenAIManualTempUnschedulableClearWithoutLayeredProbeIsNoop(t *testing.T) {
	svc := &OpenAIGatewayService{}
	svc.openaiAccountRuntimeBlockUntil.Store(int64(96), time.Now().Add(time.Minute))

	svc.ClearAccountSchedulingBlock(96)

	_, ok := svc.openaiAccountRuntimeBlockUntil.Load(int64(96))
	require.False(t, ok, "ClearAccountSchedulingBlock clears runtime block even when no probe is configured")
}
```

注：旧的 5 个测试中，3 个验证「立即探活会发生」、1 个验证「失败时重新标记」、1 个验证「失败时保留 entry」—— 全部基于已删除的行为，所以整体替换是干净的。`manualRecoverySnapshotCacheStub` / `manualRecoveryProbeRepo` 保留并简化（删去 `until` 字段，新增 `clearCalls` / `setCalls` 计数）。

- [ ] **Step 2：运行新测试确认失败**

```
go test ./backend/internal/service -run "TestProbeManualRecovery|TestOpenAIManualTempUnschedulableClear" -v
```

Expected: `TestProbeManualRecoveryDoesNotSendImmediateProbe` 失败（探活仍发生），`TestOpenAIManualTempUnschedulableClearDoesNotTriggerProbe` 失败。

- [ ] **Step 3：删除 `probeImmediatelyAfterManualRecovery` 调用**

修改 `backend/internal/service/openai_account_probe.go:874-880`。当前代码：

```go
	if p.stats != nil {
		p.stats.resetAccount(accountID)
	}
	p.probeImmediatelyAfterManualRecovery(accountID)
	if stored, ok := p.entries.Load(accountID); !ok || stored == entry {
		p.entries.Delete(accountID)
	}
```

替换为（删除 `probeImmediatelyAfterManualRecovery` 那一行）：

```go
	if p.stats != nil {
		p.stats.resetAccount(accountID)
	}
	if stored, ok := p.entries.Load(accountID); !ok || stored == entry {
		p.entries.Delete(accountID)
	}
```

- [ ] **Step 4：删除 `probeImmediatelyAfterManualRecovery` 函数本身**

删除 `backend/internal/service/openai_account_probe.go:889-921`（整个 `probeImmediatelyAfterManualRecovery` 函数）。

删除后该位置下一个函数应该直接是 `getManualRecoveryProbeAccount`（在 `openai_gateway_service.go` 里），所以本文件该位置后面紧接 `setTempUnschedulable`（line 932）。检查 `setTempUnschedulable` 仍被 `probeAccount` 的失败分支（line 439）调用 —— 保留。

- [ ] **Step 5：删除 `getManualRecoveryProbeAccount`**

删除 `backend/internal/service/openai_gateway_service.go:923-930`（整个 `getManualRecoveryProbeAccount` 函数）。

确认无其他调用方：

```
grep -rn "getManualRecoveryProbeAccount" backend/
```

Expected: 0 个匹配（应该只剩下被删的两处：定义 + 原 `probeImmediatelyAfterManualRecovery` 中的引用，这两处都已经删了）。

- [ ] **Step 6：简化 `probeAccountAfterManualTempUnschedulableClear`**

打开 `backend/internal/service/openai_account_runtime_block_fastpath.go:111-133`。

当前 `ClearAccountSchedulingBlock` 注释和实现：

```go
func (s *OpenAIGatewayService) ClearAccountSchedulingBlock(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	s.openaiAccountRuntimeBlockUntil.Delete(accountID)
	// 清除 TempUnschedulable/运行时 block 后，layered probe 需要立即对账号做一次有界探活，
	// 避免管理员恢复或成功恢复路径还要等待下一次 probe tick 才重新验证账号可用性。
	// 注意：这里是同步触发，调用方可能阻塞到 probe timeout 上限；非 layered scheduler/probe 缺失时会直接 no-op。
	s.probeAccountAfterManualTempUnschedulableClear(accountID)
}

func (s *OpenAIGatewayService) probeAccountAfterManualTempUnschedulableClear(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	s.openaiSchedulerMu.Lock()
	scheduler, _ := s.openaiScheduler.(*layeredOpenAIAccountScheduler)
	s.openaiSchedulerMu.Unlock()
	if scheduler == nil || scheduler.probe == nil {
		return
	}
	scheduler.probe.applyManualRecovery(accountID, nil)
}
```

整段替换为：

```go
func (s *OpenAIGatewayService) ClearAccountSchedulingBlock(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	s.openaiAccountRuntimeBlockUntil.Delete(accountID)
	// 清除运行时 block 后，让 layered probe 同步清掉对应 entry 与运行时 EWMA，
	// 避免下一轮 tick 还按陈旧的惩罚状态对待该账号。手动恢复不再发探活。
	s.dropProbeEntryAfterManualClear(accountID)
}

func (s *OpenAIGatewayService) dropProbeEntryAfterManualClear(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	s.openaiSchedulerMu.Lock()
	scheduler, _ := s.openaiScheduler.(*layeredOpenAIAccountScheduler)
	s.openaiSchedulerMu.Unlock()
	if scheduler == nil || scheduler.probe == nil {
		return
	}
	scheduler.probe.applyManualRecovery(accountID, nil)
}
```

仅注释 + 函数名改了，`applyManualRecovery(accountID, nil)` 调用保留 —— 因为 `applyManualRecovery` 已经在 step 3 里被简化为「清 EWMA + 清 entry，不发探活」。

- [ ] **Step 7：运行新测试确认通过**

```
go test ./backend/internal/service -run "TestProbeManualRecovery|TestOpenAIManualTempUnschedulableClear" -v
```

Expected: 全部 PASS（4 个新测试）。

- [ ] **Step 8：跑全部 probe 与 service 测试，确认没回归**

```
go test ./backend/internal/service -v
```

Expected: 全部 PASS。重点关注：
- 没有引用已删除的 `probeImmediatelyAfterManualRecovery` / `getManualRecoveryProbeAccount` 报编译错误
- 没有引用旧函数名 `probeAccountAfterManualTempUnschedulableClear` 报编译错误
- 现有 OAuth 401 / 429 fallback / keyword 规则相关测试全部正常

- [ ] **Step 9：提交**

```
git add backend/internal/service/openai_account_probe.go backend/internal/service/openai_gateway_service.go backend/internal/service/openai_account_runtime_block_fastpath.go backend/internal/service/openai_account_probe_manual_recovery_test.go
git commit -m "fix(probe): remove immediate probe after manual recovery"
```

---

## 阶段 5：关闭即恢复闭环

阶段 5 把"账号 probe_enabled 翻转为 false"这个事件接到现有的 ClearTempUnschedulable / ClearAccountSchedulingBlock 链路上。两层语义（spec 第 4 节）：第 1 层一定执行（清 entry 幂等无副作用），第 2 层仅当当前 temp 来源 = layered_probe 时执行（清 DB temp flag）。

### Task 5.1：`OpenAIGatewayService.DropProbeEntry` 入口

**Files:**
- Modify: `backend/internal/service/openai_gateway_service.go`（在 `ReattachLayeredProbeTempUnschedAccount` 附近增加新方法）
- Test: `backend/internal/service/openai_gateway_service_test.go`（追加测试）

**说明：** 给 admin service 一个轻量入口去清 probe entry，不暴露 probe 内部 sync.Map。`DropProbeEntry` 是 idempotent 的：entry 不存在 / 非 layered scheduler 时直接 no-op。

- [ ] **Step 1：写失败测试**

在 `backend/internal/service/openai_gateway_service_test.go` 末尾追加：

```go
func TestOpenAIGatewayService_DropProbeEntry_RemovesExistingEntry(t *testing.T) {
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	svc := &OpenAIGatewayService{cfg: cfg}

	scheduler := svc.getOpenAIAccountScheduler()
	layered, ok := scheduler.(*layeredOpenAIAccountScheduler)
	require.True(t, ok)
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	layered.probe.entries.Store(int64(411), &openAIAccountProbeEntry{accountID: 411})

	svc.DropProbeEntry(411)

	_, present := layered.probe.entries.Load(int64(411))
	require.False(t, present, "DropProbeEntry must remove the entry")
}

func TestOpenAIGatewayService_DropProbeEntry_NoopWhenEntryAbsent(t *testing.T) {
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	svc := &OpenAIGatewayService{cfg: cfg}
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	// 不应 panic、不应触发任何下游
	require.NotPanics(t, func() { svc.DropProbeEntry(412) })
}

func TestOpenAIGatewayService_DropProbeEntry_NoopForNonLayeredScheduler(t *testing.T) {
	svc := &OpenAIGatewayService{
		openaiScheduler: &defaultOpenAIAccountScheduler{},
	}

	require.NotPanics(t, func() { svc.DropProbeEntry(413) })
}

func TestOpenAIGatewayService_DropProbeEntry_NoopForZeroID(t *testing.T) {
	svc := &OpenAIGatewayService{}
	require.NotPanics(t, func() { svc.DropProbeEntry(0) })
	require.NotPanics(t, func() { svc.DropProbeEntry(-1) })
}
```

- [ ] **Step 2：运行测试确认失败**

```
go test ./backend/internal/service -run "TestOpenAIGatewayService_DropProbeEntry" -v
```

Expected: 编译失败，`svc.DropProbeEntry undefined`。

- [ ] **Step 3：实现**

打开 `backend/internal/service/openai_gateway_service.go`。在 `ReattachLayeredProbeTempUnschedAccount` 函数之后（line 618 之后、`billingDeps()` 之前）插入：

```go
// DropProbeEntry 从 layered probe 的 runtime entry 表中移除指定账号。
// idempotent：entry 不存在 / 非 layered scheduler / 探活子系统未初始化时直接 no-op。
// 用于账号配置翻转（例如关闭 openai_probe_enabled）后清除孤儿 entry，
// 调用方负责后续的 DB 状态清理（ClearTempUnschedulable 等）。
func (s *OpenAIGatewayService) DropProbeEntry(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	s.openaiSchedulerMu.Lock()
	scheduler, _ := s.openaiScheduler.(*layeredOpenAIAccountScheduler)
	s.openaiSchedulerMu.Unlock()
	if scheduler == nil || scheduler.probe == nil {
		return
	}
	scheduler.probe.entries.Delete(accountID)
}
```

注：访问 `s.openaiScheduler` 走 `openaiSchedulerMu`，与 `dropProbeEntryAfterManualClear` 保持同样的并发模式（Task 4 step 6）。

- [ ] **Step 4：运行测试确认通过**

```
go test ./backend/internal/service -run "TestOpenAIGatewayService_DropProbeEntry" -v
```

Expected: 4 个测试全部 PASS。

- [ ] **Step 5：提交**

```
git add backend/internal/service/openai_gateway_service.go backend/internal/service/openai_gateway_service_test.go
git commit -m "feat(gateway): add DropProbeEntry idempotent entry remover"
```

---

### Task 5.2：`adminServiceImpl` 持有 `OpenAIGatewayService` 引用

**Files:**
- Modify: `backend/internal/service/admin_service.go`（成员 + 构造器 + setter）
- Modify: `backend/internal/service/wire.go`（注入连接）

**说明：** Task 5.3 的 UpdateAccount 钩子需要调 `OpenAIGatewayService.DropProbeEntry`，但当前 `adminServiceImpl` 不持有 gateway 引用。先用最小侵入方式注入 —— 加一个 `OpenAIProbeController` 接口（仅声明 `DropProbeEntry`），让 admin service 持有该接口而非具体类型，wire.go 用 `wire.Bind` 把 `*OpenAIGatewayService` 绑定到这个接口。

- [ ] **Step 1：先看现有 admin_service.go 构造器和成员位置**

```
grep -n "type adminServiceImpl struct" backend/internal/service/admin_service.go
grep -n "runtimeBlocker " backend/internal/service/admin_service.go
grep -n "func NewAdminService" backend/internal/service/admin_service.go
```

记下位置（应该在 line 540-600 范围内）。Step 3 用到。

- [ ] **Step 2：写失败测试（接口定义存在性 + DropProbeEntry 被调用）**

在 `backend/internal/service/admin_service_clear_error_test.go` 同目录新建 `backend/internal/service/admin_service_probe_toggle_test.go`：

```go
//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type probeControllerRecorder struct {
	droppedIDs []int64
}

func (r *probeControllerRecorder) DropProbeEntry(accountID int64) {
	r.droppedIDs = append(r.droppedIDs, accountID)
}

func TestAdminService_UpdateAccount_ProbeToggleOff_DropsProbeEntry(t *testing.T) {
	repo := &probeToggleRepoStub{
		account: &Account{
			ID:       501,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Extra:    map[string]any{"openai_probe_enabled": true}, // 翻转前
		},
	}
	probeCtrl := &probeControllerRecorder{}
	blocker := &runtimeBlockRecorder{}
	svc := &adminServiceImpl{
		accountRepo:        repo,
		runtimeBlocker:     blocker,
		openaiProbeControl: probeCtrl,
	}

	// 翻转后的 extra：probe_enabled=false
	_, err := svc.UpdateAccount(context.Background(), 501, &UpdateAccountInput{
		Extra: map[string]any{"openai_probe_enabled": false},
	})
	require.NoError(t, err)

	require.Equal(t, []int64{501}, probeCtrl.droppedIDs, "DropProbeEntry must be called once with account ID")
}

func TestAdminService_UpdateAccount_ProbeToggleNoChange_DoesNotDropEntry(t *testing.T) {
	repo := &probeToggleRepoStub{
		account: &Account{
			ID:       502,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Extra:    map[string]any{"openai_probe_enabled": false}, // 已经是 false
		},
	}
	probeCtrl := &probeControllerRecorder{}
	svc := &adminServiceImpl{
		accountRepo:        repo,
		runtimeBlocker:     &runtimeBlockRecorder{},
		openaiProbeControl: probeCtrl,
	}

	_, err := svc.UpdateAccount(context.Background(), 502, &UpdateAccountInput{
		Extra: map[string]any{"openai_probe_enabled": false}, // 没翻转
	})
	require.NoError(t, err)

	require.Empty(t, probeCtrl.droppedIDs, "DropProbeEntry must not be called when probe_enabled did not flip to false")
}

func TestAdminService_UpdateAccount_ProbeToggleOnAgain_DoesNotDropEntry(t *testing.T) {
	repo := &probeToggleRepoStub{
		account: &Account{
			ID:       503,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Extra:    map[string]any{"openai_probe_enabled": false}, // 翻转前 false
		},
	}
	probeCtrl := &probeControllerRecorder{}
	svc := &adminServiceImpl{
		accountRepo:        repo,
		runtimeBlocker:     &runtimeBlockRecorder{},
		openaiProbeControl: probeCtrl,
	}

	// 翻回 true（重新启用）
	_, err := svc.UpdateAccount(context.Background(), 503, &UpdateAccountInput{
		Extra: map[string]any{}, // probe_enabled 缺省 = true（默认）
	})
	require.NoError(t, err)

	require.Empty(t, probeCtrl.droppedIDs, "flipping probe_enabled back to true (default) must not drop entry")
}

func TestAdminService_UpdateAccount_ProbeToggleOff_LayeredProbeSource_ClearsTempUnsched(t *testing.T) {
	until := time.Now().Add(20 * time.Minute)
	probeReason, err := buildLayeredProbeTempUnschedReason("consecutive_failures", 5)
	require.NoError(t, err)

	repo := &probeToggleRepoStub{
		account: &Account{
			ID:                      504,
			Platform:                PlatformOpenAI,
			Type:                    AccountTypeAPIKey,
			Status:                  StatusActive,
			Extra:                   map[string]any{"openai_probe_enabled": true},
			TempUnschedulableUntil:  &until,
			TempUnschedulableReason: probeReason,
		},
	}
	probeCtrl := &probeControllerRecorder{}
	blocker := &runtimeBlockRecorder{}
	svc := &adminServiceImpl{
		accountRepo:        repo,
		runtimeBlocker:     blocker,
		openaiProbeControl: probeCtrl,
	}

	_, err = svc.UpdateAccount(context.Background(), 504, &UpdateAccountInput{
		Extra: map[string]any{"openai_probe_enabled": false},
	})
	require.NoError(t, err)

	require.Equal(t, []int64{504}, probeCtrl.droppedIDs, "Layer 1: DropProbeEntry called")
	require.Equal(t, 1, repo.clearTempUnschedCalls, "Layer 2: ClearTempUnschedulable called when source=layered_probe")
	require.Equal(t, []int64{504}, blocker.clearedIDs, "Layer 2: ClearAccountSchedulingBlock called")
}

func TestAdminService_UpdateAccount_ProbeToggleOff_NonLayeredSource_DoesNotClearTempUnsched(t *testing.T) {
	until := time.Now().Add(20 * time.Minute)

	repo := &probeToggleRepoStub{
		account: &Account{
			ID:                      505,
			Platform:                PlatformOpenAI,
			Type:                    AccountTypeOAuth,
			Status:                  StatusActive,
			Extra:                   map[string]any{"openai_probe_enabled": true},
			TempUnschedulableUntil:  &until,
			TempUnschedulableReason: `{"version":1,"source":"oauth_401","kind":"token_invalid"}`, // 非 layered_probe
		},
	}
	probeCtrl := &probeControllerRecorder{}
	blocker := &runtimeBlockRecorder{}
	svc := &adminServiceImpl{
		accountRepo:        repo,
		runtimeBlocker:     blocker,
		openaiProbeControl: probeCtrl,
	}

	_, err := svc.UpdateAccount(context.Background(), 505, &UpdateAccountInput{
		Extra: map[string]any{"openai_probe_enabled": false},
	})
	require.NoError(t, err)

	require.Equal(t, []int64{505}, probeCtrl.droppedIDs, "Layer 1 always runs")
	require.Equal(t, 0, repo.clearTempUnschedCalls, "Layer 2 must NOT clear non-layered_probe temp state")
	require.Empty(t, blocker.clearedIDs, "Layer 2 must NOT clear blocker for non-layered_probe state")
}

func TestAdminService_UpdateAccount_ProbeToggleOff_NoTempState_OnlyDropsEntry(t *testing.T) {
	repo := &probeToggleRepoStub{
		account: &Account{
			ID:       506,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Extra:    map[string]any{"openai_probe_enabled": true},
		},
	}
	probeCtrl := &probeControllerRecorder{}
	blocker := &runtimeBlockRecorder{}
	svc := &adminServiceImpl{
		accountRepo:        repo,
		runtimeBlocker:     blocker,
		openaiProbeControl: probeCtrl,
	}

	_, err := svc.UpdateAccount(context.Background(), 506, &UpdateAccountInput{
		Extra: map[string]any{"openai_probe_enabled": false},
	})
	require.NoError(t, err)

	require.Equal(t, []int64{506}, probeCtrl.droppedIDs, "Layer 1 runs even when no temp state")
	require.Equal(t, 0, repo.clearTempUnschedCalls)
	require.Empty(t, blocker.clearedIDs)
}

// probeToggleRepoStub 是一个最小化的 AccountRepository stub，
// 仅覆盖 UpdateAccount 钩子用到的方法。
type probeToggleRepoStub struct {
	mockAccountRepoForGemini
	account               *Account
	clearTempUnschedCalls int
}

func (r *probeToggleRepoStub) GetByID(ctx context.Context, id int64) (*Account, error) {
	if r.account != nil && r.account.ID == id {
		cloned := *r.account
		return &cloned, nil
	}
	return nil, ErrAccountNotFound
}

func (r *probeToggleRepoStub) Update(ctx context.Context, account *Account) error {
	r.account = account
	return nil
}

func (r *probeToggleRepoStub) ClearTempUnschedulable(ctx context.Context, id int64) error {
	r.clearTempUnschedCalls++
	if r.account != nil {
		r.account.TempUnschedulableUntil = nil
		r.account.TempUnschedulableReason = ""
	}
	return nil
}
```

- [ ] **Step 3：运行测试确认失败**

```
go test -tags=unit ./backend/internal/service -run "TestAdminService_UpdateAccount_ProbeToggle" -v
```

Expected: 编译失败 —— `adminServiceImpl` 没有 `openaiProbeControl` 字段；`OpenAIProbeController` 接口未定义。

- [ ] **Step 4：在 admin_service.go 中定义接口 + 加成员 + 加 setter**

打开 `backend/internal/service/admin_service.go:540-600`。找到 `type adminServiceImpl struct` 定义，在 `runtimeBlocker AccountRuntimeBlocker` 字段下方加：

```go
	openaiProbeControl OpenAIProbeController
```

在文件靠近顶部的 type 区域（`AccountRuntimeBlocker` interface 附近，或者 `type adminServiceImpl struct` 之前）加接口定义：

```go
// OpenAIProbeController 是 admin service 用于通知 OpenAI probe 子系统的窄接口。
// 仅声明 admin 路径需要的方法，避免直接持有 *OpenAIGatewayService 形成循环依赖。
type OpenAIProbeController interface {
	DropProbeEntry(accountID int64)
}
```

注：如果 `AccountRuntimeBlocker` 在 `ratelimit_service.go` 中定义，仿照同样位置；admin_service.go 自己声明也可以，Go 接口是 structural 的。

找到 `NewAdminService` 构造函数，在它的参数列表里加 `openaiProbeControl OpenAIProbeController`，并在函数体里赋值给 `s.openaiProbeControl`。

如果有专门的 setter pattern（比如 `SetXxx`），按 setter 风格加：

```go
func (s *adminServiceImpl) SetOpenAIProbeController(c OpenAIProbeController) {
	s.openaiProbeControl = c
}
```

具体走哪种风格看 `runtimeBlocker` 当前是怎么注入的（Step 1 已经看过）—— 跟它保持一致。

- [ ] **Step 5：在 wire.go 中绑定**

打开 `backend/internal/service/wire.go`。找到 `wire.Bind(new(AccountRuntimeBlocker), new(*OpenAIGatewayService))` 那一行（line 565 附近），在它下方加：

```go
	wire.Bind(new(OpenAIProbeController), new(*OpenAIGatewayService)),
```

如果 admin service 是用 setter 注入的，找到 `AdminService` 的 provider 函数（通常在 wire.go 或 service_set.go），在 `svc.SetAccountRuntimeBlocker(...)` 旁边加 `svc.SetOpenAIProbeController(openaiGateway)` 调用。

如果 admin service 是构造器注入的（`NewAdminService(..., openaiProbeControl, ...)`），wire.go 会自动 resolve（因为有 wire.Bind）。

- [ ] **Step 6：跑全 build 确认编译通过**

```
go build ./backend/...
```

Expected: 编译通过。如果 wire 报错说 OpenAIProbeController 未提供 provider，说明 step 5 的 wire.Bind 没正确加；如果 admin_service_test.go 现有测试编译失败，说明现有测试构造 `&adminServiceImpl{...}` 时没初始化新字段 —— 不是问题，因为 `openaiProbeControl == nil` 时 Task 5.3 的钩子会做 nil check。但要确保现有测试在新字段引入后不再 panic（nil 接口调方法会 panic，所以钩子里必须 nil check）。

- [ ] **Step 7：现有测试不应回归**

```
go test ./backend/internal/service -v
```

Expected: 现有 admin_service 测试全部 PASS（钩子还没接，所以 5.2 Step 2 的新测试仍然失败 —— 这是预期的，5.3 才接钩子）。

- [ ] **Step 8：提交**

```
git add backend/internal/service/admin_service.go backend/internal/service/wire.go backend/internal/service/admin_service_probe_toggle_test.go
git commit -m "feat(admin): inject OpenAIProbeController into adminServiceImpl"
```

---

### Task 5.3：`UpdateAccount` 末尾增加"关闭即恢复"钩子

**Files:**
- Modify: `backend/internal/service/admin_service.go:2695-2712`（在 `accountRepo.Update` 之后、`return updated, nil` 之前增加钩子）

**说明：** 钩子分两层 —— 第 1 层一定执行（清 entry，幂等）；第 2 层判定 `source=layered_probe` 才清 DB temp + blocker。

- [ ] **Step 1：跑 5.2 写好的测试，确认它们都 fail**

```
go test -tags=unit ./backend/internal/service -run "TestAdminService_UpdateAccount_ProbeToggle" -v
```

Expected: 6 个测试中至少 5 个失败（Layer 1 / Layer 2 都没接）。

- [ ] **Step 2：在 UpdateAccount 末尾加钩子**

打开 `backend/internal/service/admin_service.go:2573` 的 `UpdateAccount`。

第一步：在函数开头（`account, err := s.accountRepo.GetByID(ctx, id)` 这一行返回 account 之后），保存翻转前的 probe_enabled 状态：

```go
	wasProbeEnabledBefore := account.IsOpenAIProbeEnabled()
```

放在 `wasOveragesEnabled := account.IsOveragesEnabled()` 那一行下方。

第二步：在 `s.accountRepo.Update(ctx, account)` 之后、`if input.GroupIDs != nil` 之前（line 2695-2700 之间）插入钩子：

```go
	// 关闭即恢复：probe_enabled 从 true 翻转为 false 时，
	// 第 1 层一定执行（清 probe entry），第 2 层仅当当前 temp 状态来源 = layered_probe 时清 DB temp flag。
	s.applyProbeToggleSideEffects(ctx, account, wasProbeEnabledBefore)
```

第三步：在文件末尾（其他方法之后）新增 helper 方法：

```go
// applyProbeToggleSideEffects 处理 probe_enabled 翻转产生的副作用。
// 仅当 OpenAI 账号的 probe_enabled 从 true 翻转为 false 时触发。
//
// 第 1 层（一定执行）：DropProbeEntry —— 幂等清除 layered probe 内存表中的 entry。
// 第 2 层（仅当 DB temp 来源 = layered_probe 时执行）：
//   - ClearTempUnschedulable
//   - ClearAccountSchedulingBlock
//
// 其他来源的 temp 状态（OAuth 401 / 429 / keyword 规则等）由各自模块自行恢复，钩子不动它们。
func (s *adminServiceImpl) applyProbeToggleSideEffects(ctx context.Context, account *Account, wasEnabledBefore bool) {
	if account == nil {
		return
	}
	if account.Platform != PlatformOpenAI {
		return
	}
	// 仅 true → false 翻转才触发
	if !wasEnabledBefore || account.IsOpenAIProbeEnabled() {
		return
	}

	// 第 1 层：清 probe entry（幂等）
	if s.openaiProbeControl != nil {
		s.openaiProbeControl.DropProbeEntry(account.ID)
	}

	// 第 2 层：仅当当前 temp 来源 = layered_probe 才清 DB temp + blocker
	if account.TempUnschedulableUntil == nil {
		return
	}
	parsed, ok := parseTempUnschedReason(account.TempUnschedulableReason)
	if !ok || parsed.Source != "layered_probe" {
		return
	}

	if err := s.accountRepo.ClearTempUnschedulable(ctx, account.ID); err != nil {
		slog.Warn("admin: probe toggle off failed to clear temp unschedulable",
			"account_id", account.ID, "error", err)
		return
	}
	if s.runtimeBlocker != nil {
		s.runtimeBlocker.ClearAccountSchedulingBlock(account.ID)
	}
	slog.Info("admin: probe toggle off cleared layered_probe temp unschedulable",
		"account_id", account.ID)
}
```

确认 `slog` 已在 admin_service.go 的 import 中（如果没有则加 `"log/slog"`）。`parseTempUnschedReason` 在 `openai_account_probe.go` 同 package，直接调用即可。

- [ ] **Step 3：运行新测试确认通过**

```
go test -tags=unit ./backend/internal/service -run "TestAdminService_UpdateAccount_ProbeToggle" -v
```

Expected: 6 个测试全部 PASS。

- [ ] **Step 4：跑 admin service 全部测试，确认没回归**

```
go test -tags=unit ./backend/internal/service -run "AdminService" -v
```

Expected: 全部 PASS（含原有 ClearAccountError 等测试）。

- [ ] **Step 5：跑 service 包全部测试**

```
go test ./backend/internal/service -v
```

Expected: 全部 PASS。

- [ ] **Step 6：提交**

```
git add backend/internal/service/admin_service.go
git commit -m "feat(admin): clear layered_probe temp state when probe_enabled flips to false"
```

---

## 阶段 6：前端 i18n 键

### Task 6：新增 zh / en 文案

**Files:**
- Modify: `frontend/src/i18n/locales/zh.ts:3768`（在 `modelRestrictionDisabledByPassthrough` 之前插入）
- Modify: `frontend/src/i18n/locales/en.ts:3625`（在 `modelRestrictionDisabledByPassthrough` 之前插入）

**说明：** 6 个新 key —— `probeEnabled` / `probeEnabledDesc` / `probeEnabledOffHint` / `probeModel` / `probeModelPlaceholder` / `probeModelHint`。挂在 `admin.accounts.openai.*` 路径下，与 `compactMode` 同级。

- [ ] **Step 1：写 zh.ts 新键**

打开 `frontend/src/i18n/locales/zh.ts`，找到 line 3768 的 `modelRestrictionDisabledByPassthrough: ...`。在它**上方**插入：

```ts
        probeEnabled: '故障自动检查',
        probeEnabledDesc:
          '开启时该账号会被分层调度器健康检查；关闭后该账号不再被自动检查，已有的探活临时不可用状态会立即清除。',
        probeEnabledOffHint:
          '关闭后该账号不再进入分层调度器的自动健康检查；当前若处于自动检查导致的临时不可用状态，保存后会立即恢复调度。其他来源的临时不可用（限流、401、流超时等）仍会按各自规则生效与恢复。',
        probeModel: '自检模型',
        probeModelPlaceholder: '留空使用默认逻辑',
        probeModelHint:
          '填写后该模型将用于账号健康检查；图像专用上游建议填写实际支持的模型（如 gpt-image-2）。留空时按 model_mapping 首键回退到 gpt-4o-mini。',
```

注意：缩进是 8 个空格（与现有 `compactMode` 同级 4 层缩进）；末尾有逗号。

- [ ] **Step 2：写 en.ts 新键**

打开 `frontend/src/i18n/locales/en.ts`，找到 line 3625 的 `modelRestrictionDisabledByPassthrough: ...`。在它**上方**插入：

```ts
        probeEnabled: 'Auto health check',
        probeEnabledDesc:
          'When enabled, this account participates in the layered scheduler probe. Disable to stop automatic checks; existing probe-induced temp unschedulable state is cleared immediately.',
        probeEnabledOffHint:
          'When disabled, this account is not enrolled in the layered scheduler probe. If it is currently in a probe-induced temp unschedulable state, saving will restore scheduling immediately. Temp unschedulable from other sources (rate limit, 401, stream timeout, etc.) is unaffected.',
        probeModel: 'Probe model',
        probeModelPlaceholder: 'Leave blank to use default selection',
        probeModelHint:
          'When set, the layered probe sends health checks using this model. For image-only upstreams, set to a model the upstream actually supports (e.g. gpt-image-2). When blank, falls back to model_mapping first key, then gpt-4o-mini.',
```

- [ ] **Step 3：跑前端类型检查 + lint**

```
npm --prefix frontend run typecheck
```

Expected: 0 错误（i18n 文件是纯 TS 对象，类型应该自洽）。

如果项目里没有 `typecheck` script，用 `npm --prefix frontend run lint` 或 `npm --prefix frontend run build` 验证。

- [ ] **Step 4：提交**

```
git add frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts
git commit -m "i18n: add probe enabled and probe model account fields"
```

---

## 阶段 7：前端三个 Modal

阶段 7 把两个新字段接到 Create / Edit / BulkEdit 三个 Modal。Create 是先驱者（最详细），Edit 和 BulkEdit 复用同一套 ref + 同一套 buildOpenAIExtra 逻辑、UI 略有差异。

### Task 7.1：CreateAccountModal 增加两个字段

**Files:**
- Modify: `frontend/src/components/account/CreateAccountModal.vue`

**说明：** 三件事 ——
1. `<script setup>` 顶部 ref 区加两个新 ref
2. `buildOpenAIExtra` 函数里加两个新键的写入逻辑
3. template 在 OpenAI compact 配置区下方加两个字段的 UI

- [ ] **Step 1：找到现有 ref 定义位置**

```
grep -n "openAICompactMode = ref" frontend/src/components/account/CreateAccountModal.vue
grep -n "openAIResponsesMode = ref" frontend/src/components/account/CreateAccountModal.vue
```

记下行号（应在 `<script setup>` 区域）。新 ref 紧邻它们插入。

- [ ] **Step 2：加两个新 ref**

在 `openAICompactMode` ref 定义之后插入：

```ts
const openAIProbeEnabled = ref<boolean>(true)
const openAIProbeModel = ref<string>('')
```

- [ ] **Step 3：在 `buildOpenAIExtra` 中加写入逻辑**

打开 `frontend/src/components/account/CreateAccountModal.vue:4393-4407`。当前代码片段：

```ts
  if (openAICompactMode.value !== 'auto') {
    extra.openai_compact_mode = openAICompactMode.value
  } else {
    delete extra.openai_compact_mode
  }

  if (
    accountCategory.value === 'apikey' &&
    openAITextGenerationCapabilityEnabled.value &&
    openAIResponsesMode.value !== 'auto'
  ) {
    extra.openai_responses_mode = openAIResponsesMode.value
  } else {
    delete extra.openai_responses_mode
  }
```

在 `openai_responses_mode` 块之后、函数末尾的 `return Object.keys(extra).length > 0 ? extra : undefined` 之前插入：

```ts
  // openai_probe_enabled：默认 true 时不写入（保留缺省语义）
  if (openAIProbeEnabled.value === false) {
    extra.openai_probe_enabled = false
  } else {
    delete extra.openai_probe_enabled
  }

  // openai_probe_model：仅在非空时写入
  const probeModel = openAIProbeModel.value.trim()
  if (probeModel !== '') {
    extra.openai_probe_model = probeModel
  } else {
    delete extra.openai_probe_model
  }
```

- [ ] **Step 4：在 template 中加 UI**

打开 `frontend/src/components/account/CreateAccountModal.vue:2685-2723`（OpenAI Compact 配置区 `<div v-if="form.platform === 'openai' && (accountCategory === 'oauth-based' || accountCategory === 'apikey')">`）。

在该区块的**闭合 `</div>` 之前**（即 line 2722-2723 之间）插入新的字段块：

```vue
        <!-- 故障自动检查（probe enabled）+ 自检模型（probe model） -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4">
          <div class="flex items-center justify-between gap-4">
            <div class="flex-1">
              <label class="input-label mb-0">{{ t('admin.accounts.openai.probeEnabled') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.openai.probeEnabledDesc') }}
              </p>
            </div>
            <div class="flex items-center">
              <button
                type="button"
                @click="openAIProbeEnabled = !openAIProbeEnabled"
                :class="[
                  'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                  openAIProbeEnabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-dark-500'
                ]"
              >
                <span
                  :class="[
                    'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                    openAIProbeEnabled ? 'translate-x-6' : 'translate-x-1'
                  ]"
                />
              </button>
            </div>
          </div>
          <p
            v-if="!openAIProbeEnabled"
            class="rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
          >
            {{ t('admin.accounts.openai.probeEnabledOffHint') }}
          </p>
          <div>
            <label class="input-label">{{ t('admin.accounts.openai.probeModel') }}</label>
            <input
              v-model="openAIProbeModel"
              type="text"
              class="input"
              :placeholder="t('admin.accounts.openai.probeModelPlaceholder')"
            />
            <p class="input-hint">{{ t('admin.accounts.openai.probeModelHint') }}</p>
          </div>
        </div>
```

注：开关按钮的样式参考现有 `codexCLIOnlyEnabled` 的 toggle（同文件内可搜 `translate-x-` 找到现成模板）。如果项目有统一的 `<Toggle>` 组件，按现有 compactMode 区域的风格使用即可。

- [ ] **Step 5：手动验证 UI**

```
npm --prefix frontend run dev
```

打开浏览器创建新 OpenAI 账号，确认：
- 新字段在 OpenAI compact 配置区下方显示
- toggle 默认 ON
- 关闭 toggle 时显示橙色提示文案
- 输入框 placeholder 文案正确
- 切换 platform 到非 OpenAI 后字段消失

- [ ] **Step 6：跑前端类型检查**

```
npm --prefix frontend run typecheck
```

Expected: 0 错误。

- [ ] **Step 7：提交**

```
git add frontend/src/components/account/CreateAccountModal.vue
git commit -m "feat(account-modal): probe enabled and probe model fields in CreateAccountModal"
```

---

### Task 7.2：EditAccountModal 增加两个字段 + 加载已有值

**Files:**
- Modify: `frontend/src/components/account/EditAccountModal.vue`

**说明：** Edit 比 Create 多两件事 —— (1) 从 `account.extra` 加载现有值到 ref；(2) 写回 extra 时仍走相同的 buildOpenAIExtra 逻辑。

- [ ] **Step 1：找到 EditAccountModal 中现有 ref + 加载逻辑**

```
grep -n "openAICompactMode = ref" frontend/src/components/account/EditAccountModal.vue
grep -n "openAICompactMode.value =" frontend/src/components/account/EditAccountModal.vue
grep -n "extra.openai_compact_mode" frontend/src/components/account/EditAccountModal.vue
```

记下三个位置：ref 定义（顶部）、加载（watch / setup 中读取 account.extra）、写入（buildOpenAIExtra 或类似函数）。

- [ ] **Step 2：加两个新 ref**

在 `openAICompactMode` ref 定义之后加：

```ts
const openAIProbeEnabled = ref<boolean>(true)
const openAIProbeModel = ref<string>('')
```

- [ ] **Step 3：加加载逻辑**

找到 `EditAccountModal.vue:2701` 附近 `openAICompactMode.value = (extra?.openai_compact_mode as OpenAICompactMode) || 'auto'` 这一行。在它之后加：

```ts
    openAIProbeEnabled.value = extra?.openai_probe_enabled !== false
    openAIProbeModel.value = typeof extra?.openai_probe_model === 'string' ? extra.openai_probe_model : ''
```

注：`!== false` 让缺省值 / true 都视为开启，与后端 `IsOpenAIProbeEnabled` 默认 true 语义一致。

- [ ] **Step 4：加写入逻辑**

找到 `EditAccountModal.vue:3569-3578` 附近 `extra.openai_compact_mode = openAICompactMode.value` 那段。在 `delete extra.openai_responses_mode` 之后插入（与 Create Modal 完全一致的逻辑）：

```ts
    if (openAIProbeEnabled.value === false) {
      extra.openai_probe_enabled = false
    } else {
      delete extra.openai_probe_enabled
    }
    const probeModel = openAIProbeModel.value.trim()
    if (probeModel !== '') {
      extra.openai_probe_model = probeModel
    } else {
      delete extra.openai_probe_model
    }
```

- [ ] **Step 5：加 UI**

打开 `EditAccountModal.vue:1627-1660` 找到 OpenAI compact 配置区 `<div v-if="account?.platform === 'openai' && (account?.type === 'oauth' || account?.type === 'apikey')">`。

在该区块**结尾的 `</div>` 之前**插入与 Task 7.1 Step 4 完全相同的 template 片段（toggle + hint + probe model 输入框）。注：在 EditAccountModal 中变量名相同（`openAIProbeEnabled` / `openAIProbeModel`），可直接复用。

- [ ] **Step 6：手动验证**

```
npm --prefix frontend run dev
```

打开浏览器，编辑现有 OpenAI 账号，确认：
- 新字段显示且默认值正确（已存量账号 toggle = ON、probeModel = 空）
- 修改 toggle 为 OFF、保存
- 重新打开同一账号，toggle 仍是 OFF
- 设置 probeModel = `gpt-image-2`、保存、重新打开仍是 `gpt-image-2`
- 设置 probeModel = 空、保存、extra 中没有 `openai_probe_model` 键（用浏览器 DevTools 看 PUT 请求）

- [ ] **Step 7：跑前端类型检查**

```
npm --prefix frontend run typecheck
```

Expected: 0 错误。

- [ ] **Step 8：提交**

```
git add frontend/src/components/account/EditAccountModal.vue
git commit -m "feat(account-modal): probe enabled and probe model fields in EditAccountModal"
```

---

### Task 7.3：BulkEditAccountModal 增加两个字段

**Files:**
- Modify: `frontend/src/components/account/BulkEditAccountModal.vue`

**说明：** BulkEdit 字段必须支持"未修改"语义（不像 Create/Edit 那样总有值）—— 通常用一个外层 `enabled` ref 控制是否加入本次批量更新。

- [ ] **Step 1：看现有 BulkEdit 是怎么处理 `openai_compact_mode` 的**

```
grep -n "openAICompactMode" frontend/src/components/account/BulkEditAccountModal.vue
grep -n "openai_compact_mode" frontend/src/components/account/BulkEditAccountModal.vue
```

记下：toggle ref、值 ref、UI 位置（line 835 附近）、写入逻辑（line 1552 附近）。

- [ ] **Step 2：加两组新 ref（toggle + 值）**

在现有 `openAICompactMode` 相关 ref 之后加：

```ts
const updateOpenAIProbeEnabled = ref<boolean>(false) // 是否包含本字段进入批量更新
const openAIProbeEnabled = ref<boolean>(true)         // 实际值（仅当 update 为 true 时写入）

const updateOpenAIProbeModel = ref<boolean>(false)
const openAIProbeModel = ref<string>('')
```

- [ ] **Step 3：加写入逻辑**

找到 `BulkEditAccountModal.vue:1552` 附近 `extra.openai_compact_mode = openAICompactMode.value` 块。仿照同样模式追加：

```ts
    if (updateOpenAIProbeEnabled.value) {
      if (openAIProbeEnabled.value === false) {
        extra.openai_probe_enabled = false
      } else {
        // BulkEdit 的"重置为默认"语义需要显式删除键，让后端走默认 true
        extra.openai_probe_enabled = true
      }
    }
    if (updateOpenAIProbeModel.value) {
      const probeModel = openAIProbeModel.value.trim()
      if (probeModel !== '') {
        extra.openai_probe_model = probeModel
      } else {
        // BulkEdit 写入空字符串清除该字段（前端 UI 上等同"清除自定义模型"）
        extra.openai_probe_model = ''
      }
    }
```

注：BulkEdit 与 Create/Edit 不同，因为它必须支持"明确清空" —— 用 `extra.openai_probe_enabled = true`（而非 delete）来表达"批量重置为开启"。后端 `UpdateExtra` 对 true 与缺省都视为启用，因此语义等价。

- [ ] **Step 4：加 UI**

找到 `BulkEditAccountModal.vue:835-880` 现有的 compactMode 字段区域。仿照同样模式（顶部一个 checkbox `update*` 控制是否启用本字段、下方一个 toggle/input 编辑值）追加：

```vue
      <!-- 故障自动检查 -->
      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <label class="flex items-center gap-2">
          <input v-model="updateOpenAIProbeEnabled" type="checkbox" class="rounded" />
          <span class="text-sm font-medium">
            {{ t('admin.accounts.openai.probeEnabled') }}
          </span>
        </label>
        <p class="mt-1 ml-6 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.openai.probeEnabledDesc') }}
        </p>
        <div v-if="updateOpenAIProbeEnabled" class="ml-6 mt-2">
          <button
            type="button"
            @click="openAIProbeEnabled = !openAIProbeEnabled"
            :class="[
              'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
              openAIProbeEnabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-dark-500'
            ]"
          >
            <span
              :class="[
                'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                openAIProbeEnabled ? 'translate-x-6' : 'translate-x-1'
              ]"
            />
          </button>
        </div>
      </div>

      <!-- 自检模型 -->
      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <label class="flex items-center gap-2">
          <input v-model="updateOpenAIProbeModel" type="checkbox" class="rounded" />
          <span class="text-sm font-medium">
            {{ t('admin.accounts.openai.probeModel') }}
          </span>
        </label>
        <p class="mt-1 ml-6 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.openai.probeModelHint') }}
        </p>
        <div v-if="updateOpenAIProbeModel" class="ml-6 mt-2">
          <input
            v-model="openAIProbeModel"
            type="text"
            class="input"
            :placeholder="t('admin.accounts.openai.probeModelPlaceholder')"
          />
        </div>
      </div>
```

确认插入位置在 BulkEdit 的 OpenAI 字段分组下。

- [ ] **Step 5：手动验证**

```
npm --prefix frontend run dev
```

打开浏览器，进入账号列表 → 多选 OpenAI 账号 → 批量编辑：
- 不勾选两个字段 checkbox 直接保存 → 后端不应收到 `openai_probe_enabled` / `openai_probe_model` 键（DevTools 看 PUT 请求）
- 勾选「故障自动检查」并切换为 OFF → 保存后所选账号都进入 disabled 状态
- 勾选「自检模型」、填 `gpt-image-2` → 所选账号都设置该模型

- [ ] **Step 6：跑前端类型检查**

```
npm --prefix frontend run typecheck
```

Expected: 0 错误。

- [ ] **Step 7：提交**

```
git add frontend/src/components/account/BulkEditAccountModal.vue
git commit -m "feat(account-modal): probe enabled and probe model in BulkEditAccountModal"
```

---

## 端到端验证

阶段 7 完成后做一次完整的回归测试，确认 spec "验证标准"全部满足。

### Task 8：端到端回归

- [ ] **Step 1：跑后端全测**

```
go test ./backend/... -v
```

Expected: 全部 PASS（含本次新增的所有测试）。

- [ ] **Step 2：跑前端全测**

```
npm --prefix frontend test
npm --prefix frontend run typecheck
npm --prefix frontend run build
```

Expected: 0 错误。

- [ ] **Step 3：构建后端二进制**

```
go build -o /tmp/sub2api-test ./backend/cmd/server
```

Expected: 编译成功。

- [ ] **Step 4：（可选）本地起服务跑一遍 dmit 场景**

如果有本地 Postgres，启动服务，用以下方式验证 spec 验证标准第 5 项：
1. 创建一个 OpenAI APIKey 账号，base_url 指向一个本地 mock 的 OpenAI 兼容服务（可用 `httpbin` + 自定义路由），让它对 `/v1/responses` 的 `gpt-4o-mini` 请求始终返回 400
2. 等 ProbeMaxFailures 次失败后账号被打上 layered_probe 临时不可用
3. 编辑账号关闭 `openai_probe_enabled` 保存
4. 立即查询账号状态：`temp_unschedulable_until` 应为空、可调度

如果没有本地 Postgres，跳过 Step 4，依赖单测覆盖。

- [ ] **Step 5：写发布说明 / changelog 条目（如果项目维护 CHANGELOG）**

如果项目有 `CHANGELOG.md`，加一条：

```
- feat(openai): per-account probe enable toggle and probe model override; manual recovery no longer triggers immediate probe
```

如果没有，跳过此步。

---

## 完成

8 个阶段全部完成后：

- 后端：4 处 guard、resolveProbeModel 优先级、关闭即恢复钩子、删手动恢复立即探活
- 前端：i18n + Create/Edit/BulkEdit 三个 Modal
- 测试：getter 单测、4 处 guard 单测、resolveProbeModel 单测、关闭即恢复集成测试、手动恢复回归测试

预期 commit 数：14（含 README/changelog 0-1 个）。预期总改动行数约 800-1000 行（含测试）。
