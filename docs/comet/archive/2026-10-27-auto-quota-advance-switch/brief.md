# 目标

为订阅分组和用户订阅增加「额度用尽自动扣减剩余有效天数后重置额度」的能力。
- 管理端在订阅分组设置中可配置 `auto_quota_advance`（默认关闭），控制该分组是否支持额度用尽自动提前重置。
- 当且仅当管理端分组开启该能力时，用户端在「我的订阅」卡片中为每个具体订阅提供 `auto_quota_advance` 开关（默认关闭），由用户自主决定是否开启。
- 网关在请求鉴权与限额预检阶段拦截超额请求前夕，若分组与用户订阅均开启了自动重置，且当前有且仅有一个窗口达到门槛且超额，且剩余有效时间足以扣减，则原子扣减距下次自然重置所需的剩余时间并重置该窗口用量，放行本次请求并记录结构化审计日志。
- 多窗口超额冲突、剩余时间不足扣除或任一开关未开启时，不触发自动重置并维持正常 429 拦截。
- 保留现有的用户手动提前重置按钮作为补充入口。

# 范围

## 目录结构

### 新建
- `backend/migrations/*_add_auto_quota_advance.sql`: 为 `groups` 和 `user_subscriptions` 表添加 `auto_quota_advance` 字段。
- `backend/internal/service/subscription_auto_advance_test.go`: 自动重置与用户开关单元测试。

### 修改
- `backend/ent/schema/group.go`: 添加 `auto_quota_advance` 字段定义。
- `backend/ent/schema/user_subscription.go`: 添加 `auto_quota_advance` 字段定义。
- `backend/internal/domain/group.go` / `user_subscription.go`: 领域模型增加相应字段。
- `backend/internal/service/subscription_service.go`: 增加用户切换自动重置偏好逻辑、续费状态保留逻辑及网关自动重置核心逻辑。
- `backend/internal/handler/subscription_handler.go`: 增加用户更新订阅自动重置偏好的 API 端点。
- `backend/internal/handler/admin/group_handler.go`: 分组创建与更新支持 `auto_quota_advance`。
- `backend/internal/server/middleware/api_key_auth.go`: 在限额拦截前集成自动重置检查与放行。
- `frontend/src/views/admin/GroupsView.vue`: 分组创建/编辑表单增加自动重置开关与说明。
- `frontend/src/views/user/SubscriptionsView.vue`: 订阅卡片增加用户自动重置开关及状态展示。
- `frontend/src/api/subscriptions.ts`: 增加更新订阅自动重置偏好的 API 方法。
- `frontend/src/i18n/locales/*/admin/overview.ts` & `frontend/src/i18n/locales/*/misc.ts`: 补充中英文文案。

### 删除
无。

### 明确不建
- 不建立定时扫描自动重置的后台任务（严格基于网关拦截时按需触发，避免浪费用户闲置期有效期）。
- 不建立多窗口级联自动扣减机制（严格沿用既有单窗口安全规格）。

# 非目标
- 不改变单窗口提前重置的基本公平性与计算逻辑（多窗口同时超额时不自动跨窗口合并重置）。
- 不改变一次性日额度订阅不可重置的规则。
- 不改变管理员手动重置或调整有效期的行为。
- 不支持在剩余有效期不足以扣除剩余周期时长时透支扣减为过期状态。
- 不在请求计费结算（Post-billing）时强制提前触发重置。

# 验收示例

- A1: 管理员可以在创建或更新订阅分组时配置「额度用尽自动重置」（`auto_quota_advance`），新分组默认关闭（false）。
- A2: 当分组未开启 `auto_quota_advance` 或未开启 `allow_quota_advance` 时，用户端不可开启该订阅的自动重置，后端更新接口拒绝变更。
- A3: 当分组开启了 `allow_quota_advance` 和 `auto_quota_advance` 时，用户可以在订阅卡片切换开启/关闭其订阅的 `auto_quota_advance`，新订阅默认关闭（false）。
- A4: 用户订阅续费、充值延期或天数调整时，该订阅原有的 `auto_quota_advance` 开关状态保持不变。
- A5: 当分组和用户订阅均开启了自动重置，且用户请求因唯一达到门槛的窗口超额时，网关原子执行扣减剩余时间并重置该窗口，请求成功放行，并记录审计日志。
- A6: 当存在多个窗口同时超额，或剩余订阅时间不足以扣除本次重置所需时长时，不执行自动重置，请求按正常的 429 限额超限拦截。
- A7: 手动「提前重置额度」入口与弹窗功能保持正常可用，不受自动重置开关开启与否的破坏。
