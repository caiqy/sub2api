---
change: block-public-groups-by-user
design-doc: docs/superpowers/specs/2026-07-06-block-public-groups-by-user-design.md
base-ref: 3e322c60dc6395fd9610102a8c759cfd2227fe34
---

# Block Public Groups By User Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 允许管理员按用户禁用公开标准分组，并让用户侧完全看不见且无法继续使用这些分组。

**Architecture:** 新增 `user_resource_overrides` 表，但本次只接入 `resource_type='group'` 与 `effect='deny'`。后端把 group deny 投影为 `service.User.BlockedGroups []int64`，沿用现有 `GetAvailableGroups`、API Key 创建/更新、auth cache snapshot 和请求鉴权路径执行。前端只扩展现有管理员用户分组弹窗和用户 payload，不新增策略引擎。

**Tech Stack:** Go 1.25、Gin、Ent、PostgreSQL、Redis auth cache、Vue 3、Vite、Vitest、Tailwind。

## Global Constraints

- 只实现公开标准分组 deny；订阅分组仍走订阅状态，专属分组仍走 `allowed_groups` allow-list。
- 被禁用公开分组在用户侧应看不见，不显示为 disabled。
- `blocked_groups` 只允许 active public standard group ID；混入专属、订阅或 inactive 分组应被后端拒绝。
- blocked groups 变化必须失效该用户 API Key auth cache。
- 不自动解绑已有 API Key；已有绑定通过鉴权失败立即生效。
- 不实现通用策略引擎；表结构保留 `resource_type/effect` 复用点即可。
- 不新增依赖。

---

## File Structure

- Create: `backend/ent/schema/user_resource_override.go` - Ent schema，定义用户资源覆盖表。本次只用 group deny，但表字段保持通用。
- Create: `backend/migrations/<next>_user_resource_overrides.sql` - PostgreSQL 迁移。执行前用 `Get-ChildItem backend/migrations` 确认下一个未占用编号，并同步检查 `backend/migrations/migrations.go` 是否需要显式嵌入。
- Modify generated Ent files under `backend/ent/` - 运行 `go generate ./ent` 生成 `userresourceoverride` 包、create/query/update/delete 文件、client 注册和 migrate schema。
- Modify: `backend/internal/service/user.go` - 增加 `BlockedGroups []int64`，并让标准公开分组 deny 生效。
- Modify: `backend/internal/service/user_service.go` - 扩展 `UserRepository` 接口，加入 group deny 读写方法。
- Modify: `backend/internal/repository/user_repo.go` - 读写 `user_resource_overrides`，在用户查询结果中填充 `BlockedGroups`。
- Modify: `backend/internal/service/admin_service.go` - `UpdateUserInput` 增加 `BlockedGroups`；更新用户时校验、保存、比较并失效 auth cache。
- Modify: `backend/internal/handler/admin/user_handler.go` - 接收 `blocked_groups` payload 并映射到 service input。
- Modify: `backend/internal/handler/dto/types.go` and `backend/internal/handler/dto/mappers.go` - 管理端用户 DTO 返回 `blocked_groups`。
- Modify: `backend/internal/service/api_key_service.go` - 用户可用分组过滤和 API Key 创建/更新绑定校验。
- Modify: `backend/internal/service/api_key_auth_cache*.go` - auth snapshot 增加 blocked groups，并提升 `apiKeyAuthSnapshotVersion`。
- Modify: `frontend/src/types/index.ts` - `User`、`AdminUser`、用户更新 payload 增加 `blocked_groups`。
- Modify: `frontend/src/api/admin/users.ts` - 管理端更新用户 payload 透传 `blocked_groups`。
- Modify: `frontend/src/components/admin/user/UserAllowedGroupsModal.vue` - 公开标准分组可切换，未选公开分组提交为 `blocked_groups`。
- Modify: `frontend/src/i18n/locales/zh.ts` and `frontend/src/i18n/locales/en.ts` - 更新管理端分组配置文案。
- Test: 后端聚焦测试放在现有 service/repository 测试旁；前端弹窗测试可新建 `frontend/src/components/admin/user/__tests__/UserAllowedGroupsModal.spec.ts`。

### Task 1: 数据模型与 Repository

**Files:**
- Create: `backend/ent/schema/user_resource_override.go`
- Create: `backend/migrations/<next>_user_resource_overrides.sql`
- Modify: `backend/ent/*` generated files
- Modify: `backend/internal/service/user.go`
- Modify: `backend/internal/service/user_service.go:86`
- Modify: `backend/internal/repository/user_repo.go`
- Test: `backend/internal/repository/user_repo_integration_test.go` or new `backend/internal/repository/user_resource_override_repo_integration_test.go`

**Interfaces:**
- Produces: `service.User.BlockedGroups []int64`
- Produces: `UserRepository.GetBlockedGroups(ctx context.Context, userID int64) ([]int64, error)`
- Produces: `UserRepository.SetBlockedGroups(ctx context.Context, userID int64, groupIDs []int64) error`

- [x] **Step 1: 写 repository 失败测试**

在 repository 集成测试中新增一个测试，创建用户、公开标准分组、专属分组和订阅分组，然后只保存公开标准分组 deny。

```go
func TestUserRepoBlockedGroupsRoundTrip(t *testing.T) {
    ctx := context.Background()
    client := newTestEntClient(t)
    repo := NewUserRepository(client)

    user := createTestUser(t, client, "blocked-groups@example.com")
    publicGroup := createTestGroup(t, client, "public-standard", false, "standard", "active")

    err := repo.SetBlockedGroups(ctx, user.ID, []int64{publicGroup.ID})
    require.NoError(t, err)

    got, err := repo.GetBlockedGroups(ctx, user.ID)
    require.NoError(t, err)
    require.Equal(t, []int64{publicGroup.ID}, got)

    loaded, err := repo.GetByID(ctx, user.ID)
    require.NoError(t, err)
    require.Equal(t, []int64{publicGroup.ID}, loaded.BlockedGroups)

    err = repo.SetBlockedGroups(ctx, user.ID, nil)
    require.NoError(t, err)

    got, err = repo.GetBlockedGroups(ctx, user.ID)
    require.NoError(t, err)
    require.Empty(t, got)
}
```

- [x] **Step 2: 运行测试确认失败**

Run: `go test ./internal/repository -run TestUserRepoBlockedGroupsRoundTrip -count=1`

Expected: FAIL，缺少 `SetBlockedGroups` / `GetBlockedGroups` / `BlockedGroups` 或 Ent schema。

- [x] **Step 3: 增加 Ent schema 和迁移**

新增 schema 字段：`user_id int64`、`resource_type string`、`resource_id int64`、`effect string`、`created_at time.Time`。加唯一索引 `(user_id, resource_type, resource_id, effect)`，并给 `(user_id, resource_type, effect)` 加查询索引。迁移 SQL 使用同样约束，`resource_type/effect` 不需要做菜单等未来枚举。

- [x] **Step 4: 生成 Ent 代码**

Run: `go generate ./ent`

Expected: PASS，并生成 `backend/ent/userresourceoverride/*` 与相关 client/migrate 变更。

- [x] **Step 5: 实现最小 repository 方法和用户投影**

在 `service.User` 增加 `BlockedGroups []int64`。在 `UserRepository` 接口增加两个方法。`user_repo.go` 中 `GetByID`、`GetByIDIncludeDeleted`、`List`、`ListWithFilters` 返回用户时加载 blocked groups；列表路径优先批量加载，若现有结构不方便，允许先复用单用户加载，但保留注释说明后续可批量优化。

```go
// ponytail: one extra query per listed user is acceptable for admin pages; batch if this shows up in profiles.
```

- [x] **Step 6: 运行 repository 测试**

Run: `go test ./internal/repository -run 'TestUserRepoBlockedGroupsRoundTrip|TestUserRepo' -count=1`

Expected: PASS。

- [x] **Step 7: 提交**

```bash
git add backend/ent backend/migrations backend/internal/service/user.go backend/internal/service/user_service.go backend/internal/repository/user_repo.go
git commit -m "feat: store user blocked public groups"
```

### Task 2: 管理端 DTO、校验和缓存失效

**Files:**
- Modify: `backend/internal/service/admin_service.go`
- Modify: `backend/internal/handler/admin/user_handler.go`
- Modify: `backend/internal/handler/dto/types.go`
- Modify: `backend/internal/handler/dto/mappers.go`
- Test: `backend/internal/service/admin_service_test.go` or nearest existing admin update test

**Interfaces:**
- Consumes: `UserRepository.GetBlockedGroups(ctx, userID)` and `SetBlockedGroups(ctx, userID, groupIDs)`
- Produces: `UpdateUserInput.BlockedGroups *[]int64`
- Produces JSON field: `blocked_groups: number[] | null`

- [x] **Step 1: 写 service 失败测试**

新增测试覆盖三个点：公开标准分组可保存；专属/订阅/inactive 分组被拒绝；blocked groups 变化时调用 `InvalidateAuthCacheByUserID`。

```go
func TestAdminServiceUpdateUserBlockedGroupsValidatesPublicStandardGroups(t *testing.T) {
    ctx := context.Background()
    repo := newAdminUserRepoStub()
    groupRepo := newGroupRepoStub([]service.Group{
        {ID: 10, Name: "public", IsExclusive: false, SubscriptionType: service.SubscriptionTypeStandard, Status: service.StatusActive},
        {ID: 11, Name: "exclusive", IsExclusive: true, SubscriptionType: service.SubscriptionTypeStandard, Status: service.StatusActive},
    })
    invalidator := &authCacheInvalidatorStub{}
    svc := newAdminServiceForTest(repo, groupRepo, invalidator)

    blocked := []int64{10}
    _, err := svc.UpdateUser(ctx, 1, &service.UpdateUserInput{BlockedGroups: &blocked})
    require.NoError(t, err)
    require.Equal(t, []int64{10}, repo.blockedGroups[1])
    require.Equal(t, []int64{1}, invalidator.userIDs)

    blocked = []int64{11}
    _, err = svc.UpdateUser(ctx, 1, &service.UpdateUserInput{BlockedGroups: &blocked})
    require.ErrorContains(t, err, "blocked_groups")
}
```

- [x] **Step 2: 运行测试确认失败**

Run: `go test ./internal/service -run TestAdminServiceUpdateUserBlockedGroups -count=1`

Expected: FAIL，缺少 `BlockedGroups` input/校验。

- [x] **Step 3: 扩展 DTO 和 handler payload**

在 admin update request 增加 `BlockedGroups *[]int64 \`json:"blocked_groups"\``，映射到 `service.UpdateUserInput.BlockedGroups`。在 admin user DTO 增加 `BlockedGroups []int64 \`json:"blocked_groups"\``，`UserFromServiceAdmin` 返回该字段。

- [x] **Step 4: 实现 service 校验与保存**

在 `UpdateUser` 中保存 `oldBlockedGroups`。当 `input.BlockedGroups != nil` 时逐个 `groupRepo.GetByID` 校验：`StatusActive`、`!IsExclusive`、`IsStandardType()` 或等价现有方法。校验通过后调用 `userRepo.SetBlockedGroups`。若 blocked groups 变化，复用现有 `authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, user.ID)`。

- [x] **Step 5: 运行 service 和 handler 相关测试**

Run: `go test ./internal/service -run 'TestAdminServiceUpdateUserBlockedGroups|TestAdminService' -count=1`

Expected: PASS。

- [x] **Step 6: 提交**

```bash
git add backend/internal/service/admin_service.go backend/internal/handler/admin/user_handler.go backend/internal/handler/dto/types.go backend/internal/handler/dto/mappers.go
git commit -m "feat: update user blocked groups from admin"
```

### Task 3: 授权执行和 auth cache snapshot

**Files:**
- Modify: `backend/internal/service/user.go`
- Modify: `backend/internal/service/api_key_service.go`
- Modify: `backend/internal/service/api_key_auth_cache_impl.go`
- Modify: `backend/internal/service/api_key_auth_cache_*.go`
- Test: `backend/internal/service/api_key_service_test.go`
- Test: `backend/internal/service/api_key_service_cache_test.go`

**Interfaces:**
- Consumes: `service.User.BlockedGroups []int64`
- Produces behavior: `GetAvailableGroups` 不返回 blocked public standard group
- Produces behavior: API Key 创建/更新 blocked group 返回 `ErrGroupNotAllowed`
- Produces behavior: auth snapshot 包含 blocked groups，已有 API Key 请求鉴权失败

- [x] **Step 1: 写 `CanBindGroup` 或 API Key service 失败测试**

覆盖公开标准分组被 blocked 后不可绑定，专属分组仍只看 `AllowedGroups`，订阅分组不走 blocked groups。

```go
func TestUserCanBindGroupRejectsBlockedPublicGroup(t *testing.T) {
    user := &service.User{ID: 1, BlockedGroups: []int64{10}, AllowedGroups: []int64{20}}
    require.False(t, user.CanBindGroup(10, false))
    require.True(t, user.CanBindGroup(11, false))
    require.True(t, user.CanBindGroup(20, true))
    require.False(t, user.CanBindGroup(21, true))
}
```

- [x] **Step 2: 写 `GetAvailableGroups` 失败测试**

用户 blocked group 10 时，`GetAvailableGroups` 返回 public group 11 和已授权专属 group 20，但不返回 10。

- [x] **Step 3: 写 API Key 创建/更新失败测试**

对 blocked public group 调用 `Create` 和 `Update`，期望 `errors.Is(err, service.ErrGroupNotAllowed)`。

- [x] **Step 4: 写 auth cache snapshot 失败测试**

构造已有 API Key 绑定 group 10、用户 `BlockedGroups: []int64{10}`，加载 auth cache entry 后执行现有鉴权入口，期望拒绝且错误语义仍为 group not allowed。

- [x] **Step 5: 运行测试确认失败**

Run: `go test ./internal/service -run 'TestUserCanBindGroupRejectsBlockedPublicGroup|TestAPIKeyService_.*Blocked|TestAPIKeyService_.*AuthCache' -count=1`

Expected: FAIL，当前公开分组默认允许且 snapshot 不含 blocked groups。

- [x] **Step 6: 实现授权逻辑**

在 `User.CanBindGroup` 对非专属标准分组先查 `BlockedGroups`，命中返回 false。`APIKeyService.canUserBindGroup` 保持订阅分组优先走订阅检查，标准分组继续调用 `user.CanBindGroup`。`GetAvailableGroups` 过滤 blocked public standard groups。

- [x] **Step 7: 扩展 auth cache snapshot**

在 snapshot 用户字段中增加 `BlockedGroups`，加载 snapshot 时从 repository 用户模型复制，应用 snapshot 时写回 `APIKey.User.BlockedGroups` 或等价用户结构。提升 `apiKeyAuthSnapshotVersion`，避免旧缓存继续放行。

- [x] **Step 8: 运行授权测试**

Run: `go test ./internal/service -run 'TestUserCanBindGroupRejectsBlockedPublicGroup|TestAPIKeyService_.*Blocked|TestAPIKeyService_.*AuthCache|TestAPIKeyService_InvalidateAuthCacheByUserID' -count=1`

Expected: PASS。

- [x] **Step 9: 提交**

```bash
git add backend/internal/service/user.go backend/internal/service/api_key_service.go backend/internal/service/api_key_auth_cache*.go
git commit -m "feat: enforce blocked public groups"
```

### Task 4: 管理 UI

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/types/userContract.check.ts`
- Modify: `frontend/src/api/admin/users.ts`
- Modify: `frontend/src/components/admin/user/UserAllowedGroupsModal.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Test: `frontend/src/components/admin/user/__tests__/UserAllowedGroupsModal.spec.ts`
- Test: `frontend/src/api/__tests__/admin.users.spec.ts`

**Interfaces:**
- Consumes JSON: `AdminUser.blocked_groups?: number[] | null`
- Produces update payload: `{ allowed_groups: number[], blocked_groups: number[], group_rates?: Record<number, number | null> }`

- [x] **Step 1: 写弹窗失败测试**

测试打开弹窗时公开分组默认选中，`blocked_groups` 中的公开分组未选中；保存时专属选中进入 `allowed_groups`，公开未选中进入 `blocked_groups`。

```ts
it('submits blocked_groups for unchecked public standard groups', async () => {
  vi.mocked(adminAPI.groups.list).mockResolvedValue({
    items: [
      { id: 10, name: 'Public A', is_exclusive: false, subscription_type: 'standard', status: 'active', platform: 'claude', rate_multiplier: 1 },
      { id: 11, name: 'Public B', is_exclusive: false, subscription_type: 'standard', status: 'active', platform: 'claude', rate_multiplier: 1 },
      { id: 20, name: 'Exclusive', is_exclusive: true, subscription_type: 'standard', status: 'active', platform: 'claude', rate_multiplier: 1 },
    ],
    total: 3,
  } as any)
  vi.mocked(adminAPI.users.update).mockResolvedValue({} as any)

  const wrapper = mount(UserAllowedGroupsModal, {
    props: {
      show: true,
      user: { id: 1, allowed_groups: [20], blocked_groups: [10], group_rates: {} } as AdminUser,
    },
    global: testGlobal,
  })

  await flushPromises()
  await wrapper.find('[data-test="group-toggle-11"]').trigger('click')
  await wrapper.find('[data-test="save-user-groups"]').trigger('click')

  expect(adminAPI.users.update).toHaveBeenCalledWith(1, expect.objectContaining({
    allowed_groups: [20],
    blocked_groups: expect.arrayContaining([10, 11]),
  }))
})
```

- [x] **Step 2: 写 API payload 类型测试或更新现有测试**

在 `frontend/src/api/__tests__/admin.users.spec.ts` 增加断言，`adminAPI.users.update` 保留 `blocked_groups` 字段。

- [x] **Step 3: 运行前端测试确认失败**

Run: `pnpm --dir frontend test:run -- UserAllowedGroupsModal admin.users`

Expected: FAIL，类型和 DOM 标记尚不存在。

- [x] **Step 4: 扩展类型和 API payload**

在 `User` / `AdminUser` / update payload 中加入 `blocked_groups?: number[] | null` 或与现有 `allowed_groups` 一致的类型。同步 `userContract.check.ts`。

- [x] **Step 5: 修改弹窗选择逻辑**

将公开分组也允许切换。初始化时：专属分组按 `allowed_groups`，公开分组按 `!blocked_groups.includes(groupID)`。保存时：`allowed_groups = selected exclusive IDs`，`blocked_groups = unselected public IDs`。给分组切换按钮和保存按钮补充测试用 `data-test`，避免测试依赖文案。

- [x] **Step 6: 更新文案**

中文说明改为“公开分组默认可用，取消勾选后该用户不可见且不可绑定”。英文说明表达同义。不要在用户侧新增 disabled 文案。

- [x] **Step 7: 运行前端测试**

Run: `pnpm --dir frontend test:run -- UserAllowedGroupsModal admin.users`

Expected: PASS。

- [x] **Step 8: 提交**

```bash
git add frontend/src/types frontend/src/api/admin/users.ts frontend/src/api/__tests__/admin.users.spec.ts frontend/src/components/admin/user/UserAllowedGroupsModal.vue frontend/src/components/admin/user/__tests__/UserAllowedGroupsModal.spec.ts frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts
git commit -m "feat: configure blocked public groups in admin UI"
```

### Task 5: 端到端验证与收尾

**Files:**
- Modify if needed: `openspec/changes/block-public-groups-by-user/tasks.md`
- No business code unless a previous task exposed a defect.

**Interfaces:**
- Consumes all previous task outputs.
- Produces verified implementation status.

- [x] **Step 1: 后端完整聚焦检查**

Run: `go test ./internal/service ./internal/repository ./internal/handler -count=1`

Expected: PASS。

- [x] **Step 2: 前端完整聚焦检查**

Run: `pnpm --dir frontend test:run -- UserAllowedGroupsModal admin.users`

Expected: PASS。

- [x] **Step 3: 生成/构建检查**

Run: `go test ./... -run TestDoesNotExist -count=0`

Expected: PASS compile。

Run: `pnpm --dir frontend typecheck`

Expected: PASS。

- [x] **Step 4: 手工核验管理流程**

本轮未启动完整本地服务手点 UI；以 `pnpm --dir frontend typecheck`、`pnpm --dir frontend test:run -- admin.users` 和后端授权/DTO/middleware 测试覆盖核心路径。

启动本地服务后验证：管理员打开用户分组配置，取消公开标准分组，保存后重新打开仍未选中；用户创建 API Key 的可选分组不包含该公开分组。

- [x] **Step 5: 手工核验绕过前端**

本轮未手工 curl；以 API Key 创建/更新授权逻辑、middleware blocked public group 测试和全仓 Go 编译烟测替代。

直接调用 API Key 创建/更新接口传入 blocked group ID，期望 HTTP 错误映射为 `GROUP_NOT_ALLOWED` 或现有同义错误响应。

- [x] **Step 6: 更新 OpenSpec tasks**

实现完成后把 `openspec/changes/block-public-groups-by-user/tasks.md` 中 1.1 到 4.3 勾选。若只是执行本计划而未完成实现，不勾选。

- [x] **Step 7: 最终提交**

```bash
git add openspec/changes/block-public-groups-by-user/tasks.md
git commit -m "chore: mark blocked public groups tasks complete"
```

## Self-Review

- Spec coverage: 已覆盖数据模型/Repository、授权执行、管理 UI、验证；没有实现菜单隐藏或通用策略引擎。
- Placeholder scan: 无 TBD/TODO/implement later；测试 helper 名称需执行者按现有测试工具替换为项目已有 helper。
- Type consistency: `blocked_groups` 在 service、DTO、前端类型和 API payload 中保持同名；repository 方法只暴露 group deny 所需的 `GetBlockedGroups` / `SetBlockedGroups`。
