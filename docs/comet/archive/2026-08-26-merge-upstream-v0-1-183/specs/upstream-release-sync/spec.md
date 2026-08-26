# Upstream Release Sync Specification

## Purpose

将已获授权的 upstream 正式 release 按 tag 顺序安全集成到本地 fork，在每一版边界完成可追溯的回归审计，保留 upstream 行为和 fork 定制，并在未获授权的下一 tag 前停止。

## Requirements

### Requirement: 精确合并 v0.1.183

维护流程必须以 `v0.1.183` 的 peeled commit `e8cb019fabf8b55199436229044cbf9aa7a82564` 为唯一上游 merge 父；不得合并滚动分支、其他 tag 或该 tag 之后的 upstream 提交。

#### Scenario: 单一可追溯 merge 节点

- **WHEN** 当前基线已包含 `v0.1.182`
- **THEN** 结果 `--no-ff` merge 的第一父为合并前 fork HEAD，第二父为 `e8cb019fabf8b55199436229044cbf9aa7a82564`，`v0.1.183` 是结果 HEAD 的祖先，且没有 post-`v0.1.183` upstream 提交

### Requirement: 保留 v0.1.183 upstream 行为

合并结果必须保留 Responses custom tool-call ID 的类型化还原和反向映射、邮箱换绑别名去重与并发守卫、OpenAI sticky 容量溢出绑定、OAuth 429 配额暂停、Codex session ID affinity、channel monitor v2 composite 聚合、Kimi 并发 403 可恢复处理、Antigravity token clamp 及该 tag 的版本更新。

#### Scenario: Responses custom tool-call 往返

- **WHEN** 客户端 custom tool-call 项经 HTTP 或 WebSocket 桥接发送到上游并再次恢复
- **THEN** 客户端 ID 使用与项类型匹配的前缀，上游 function-call ID 可逆还原，未知前缀不被猜测或破坏

#### Scenario: OpenAI 调度与限额更新

- **WHEN** sticky 账号容量溢出或 OAuth 账号收到配额耗尽 429
- **THEN** sticky 绑定按 upstream 行为处理且配额耗尽账号被暂停，不破坏 fork 的 fallback、WaitPlan、DB recheck、并发槽释放或审计计费边界

### Requirement: 保留 fork 核心定制

冲突和无文本冲突的语义审查必须保持 scheduler、sticky/fallback、DB recheck、请求体重放与清理、审计、每请求计费边界、运行时设置、网关透传和前端定制语义；无法共存时必须请求用户决定。

#### Scenario: upstream 修改重叠调用链

- **WHEN** upstream 更新触及上述本地能力所在文件或调用路径
- **THEN** 合并结果必须由现有或新增回归测试、或明确源码审查证明双方行为仍成立

### Requirement: 验证、版本与发布边界闭合

每版必须完成冲突标记检查、受影响测试、后端默认与 unit 测试、后端 lint、前端 ESLint/单测/类型检查、前后端构建与 Ent/Wire 两次生成稳定性检查；最终必须确认版本为 `0.1.183.1`，并由独立只读 Verifier 验收。

#### Scenario: Docker 不可用

- **WHEN** 本机没有 Docker/Testcontainers
- **THEN** 必须执行其余门禁，并明确记录未运行的 migration/repository integration 及受影响契约，不得将其记为通过

#### Scenario: 集成完成但不发布

- **WHEN** `v0.1.183` 合并及全部可执行验证通过
- **THEN** 项目保留可审计的本地集成提交和 `0.1.183.1` 版本文件，不创建或推送 Git tag、不触发 release workflow、不部署，并在任何后续 upstream tag 或 post-`v0.1.183` 提交前停止等待用户授权
