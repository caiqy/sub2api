# Outcome

在管理员用户编辑弹窗增加按用户关闭 Cyber 历史超限自动拦截的开关，解决曾出现 Cyber 事件的同来源后续长会话跨模型被本地 403 拒绝的问题。用户要求使用 Comet 开发。此需求只控制 `transcript_lookup_limit_exceeded` 这一拒绝策略。

# Scope

管理员编辑现有用户时展示“Cyber 历史超限拦截”，默认开启以保持现有行为。说明文案为：来源存在 Cyber 封禁标记时，历史匹配候选超限将直接拒绝请求。关闭后仍检查明确命中的会话封禁。保存后影响该用户所有 API Key，并沿用现有鉴权缓存失效机制。

关闭开关后，不因超过 256 条历史候选直接拒绝，仍执行显式会话键检查和现有最多保留 256 个历史前缀的有界匹配。超出保留范围的更早前缀不承诺匹配，本次不扩大候选数量或 Redis 查询工作量。真正命中现有检查的封禁仍返回原本的本地错误。开关适用于共享检查链路下的 HTTP 与 WebSocket 入口，包含 OpenAI 兼容、DeepSeek 等已走该检查的请求。

使用用户级布尔字段持久化豁免状态，默认不豁免；管理 API 使用可选字段区分未提供、开启与关闭。普通用户无权修改该安全策略。

## 目录结构

### 新建

- backend/migrations/ — 添加用户策略布尔字段，现有用户默认保持自动拦截。
- docs/comet/changes/user-cyber-overflow-toggle/specs/user-cyber-overflow-policy/spec.md — 完整目标规格。
- 相关现有测试模块内的必要回归用例 — 覆盖持久化、缓存、默认与开关切换行为。

### 修改

- backend/ent/schema/user.go 及 backend/ent/ 生成文件 — 用户字段定义及代码生成。
- backend/internal/service/ — User、用户更新字段掩码及管理服务、API Key 鉴权快照与共享 Cyber 判断，传递用户豁免状态。
- backend/internal/repository/ — 用户读写映射与 API Key 鉴权查询投影，保存和加载该字段。
- backend/internal/handler/admin/user_handler.go 与 backend/internal/handler/dto/ — 管理员更新请求和用户响应。
- backend/internal/handler/openai_cyber_allowlist.go 与共享会话查找调用链 — 确保所有已有入口使用当前用户策略。
- frontend/src/components/admin/user/UserEditModal.vue — 复用已有表单样式添加可访问开关、提示、初始化与提交。
- frontend/src/types/、frontend/src/api/admin/users.ts 与 frontend/src/i18n/locales/ — 同步管理 API 类型与中英文文案。
- backend/internal/ 与 frontend/src/ 相关现有测试文件 — 定向回归验证。

### 删除

无。

### 明确不建

- 独立策略引擎、额外全局开关、额外依赖或新的管理页面 — 当前用户编辑和既有安全检查链路即可承载。
- 用户自助设置、用户批量开关、修改全局 TTL、生产部署脚本 — 不属于本次用户级编辑功能。

# Non-goals

不关闭上游 Cyber 检测、普通内容审核、违规累计或自动封禁，不复用会关闭更多保护的 Cyber 白名单来替代本开关。不删除 Redis 会话封禁记录，不更改默认 256 候选上限，不保证检查已被截断的更早历史前缀。不发布版本、不部署或改动 dmit 生产配置。

# Acceptance examples

- A1：管理员编辑用户时可查看、切换并保存“Cyber 历史超限拦截”；重新打开显示已保存状态，取消不保存；控件具有可见标签、说明和键盘可操作性，中英文文案齐全。
- A2：新建及已有用户缺省保持现有自动拦截；管理更新省略字段不改变原值，显式开启、关闭都可持久化及读回；普通用户不能通过自助接口修改该字段。
- A3：开关开启且来源范围活跃时，超过 256 个历史候选仍返回 transcript_lookup_limit_exceeded；来源范围不活跃时，不因历史超限单独拦截。
- A4：开关关闭且来源范围活跃时，超过 256 个历史候选不再仅因溢出拒绝；显式会话封禁及保留的历史候选命中仍拦截，Redis 查询继续有界。
- A5：开关只影响对应用户并覆盖其所有 API Key；鉴权冷加载、缓存命中及保存后的缓存失效/重载均保留正确状态，不因旧版本快照丢失用户策略。
- A6：已有 HTTP 与 WebSocket 共享检查路径均传递用户策略，切换模型或兼容平台时遵循同一用户设置；原有全局开关、白名单与上游 Cyber 记录行为保持兼容。

# Constraints and invariants

当前工作区干净，使用 Runtime 默认 current 隔离方式，在 main 分支当前目录开发。宿主提供独立 reviewer 子代理及公开 Runtime 命令，实施后由新的独立只读 Verifier 领取并回报验收 Action。Shape 确认前不改普通实现。验证优先复用项目既有测试，不新增依赖；任何临时数据库在验证结束后清理，仅停止本任务创建的实例。

# Verification expectations

覆盖管理员字段映射和存取、用户更新字段掩码、鉴权快照往返及失效、Cyber 溢出策略启停及精确命中、HTTP/WebSocket 调用链、前端表单初始化及提交。运行相关 Go 定向测试、前端定向 Vitest、TypeScript 检查及生产构建，真实失败和未运行检查如实交给独立验收。用户可见结果由 Comet 独立验收后再请求接受。
