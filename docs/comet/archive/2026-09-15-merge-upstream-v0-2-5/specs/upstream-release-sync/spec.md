# Upstream Release Sync Specification

## Purpose

将已获授权的 upstream 正式 release 按 tag 顺序安全集成到本地 fork，在 tag 边界完成可追溯的冲突与回归审查，保留 upstream 行为和 fork 定制，并在未获授权的下一 tag 前停止。

## Requirements

### Requirement: 精确合并 v0.2.5

维护流程必须以 `v0.2.5` 的 peeled commit `86f93c28ee34cc74b629dafb748bd5ac5ca8c5ea` 为唯一上游 merge 父，生成独立 `--no-ff` merge；不得合并滚动分支、其他 tag 或该 tag 之后的 upstream 提交。

#### Scenario: 单一可追溯 merge 节点

- **WHEN** 当前 fork 基线已包含 `v0.2.4`
- **THEN** 结果 merge 节点的第二父为 `86f93c28ee34cc74b629dafb748bd5ac5ca8c5ea`，`v0.2.5` 是结果 HEAD 的祖先，且结果不包含 post-`v0.2.5` upstream 提交

### Requirement: 在 tag 边界闭合审计

合并后必须完成冲突标记检查、受影响能力测试、本机完整质量门禁、构建和独立只读回归审计。发现由合并、冲突处理或 fork 定制引入的问题时，必须先用失败测试和最小修复闭合；upstream tag 原样存在的问题不自动扩大为本 change 的修复范围。

#### Scenario: 合并审计发现 fork 回归

- **WHEN** 测试、构建或能力级源码审查发现合并结果破坏 fork 既有行为
- **THEN** 流程必须停留在 `v0.2.5` 边界，以失败测试和最小修复闭合问题后重新执行受影响及完整门禁

#### Scenario: 发现 upstream 原样问题

- **WHEN** 审查发现问题在 `v0.2.5` peeled commit 中已原样存在，且本次合并与 fork 定制没有扩大或改变该问题
- **THEN** 记录该问题及证据但不在本 change 主动修复，不得以修复 upstream 既有问题为由扩大范围

### Requirement: 保留 v0.2.5 的 upstream 行为

合并结果必须保留 OpenAI/Codex WebSocket 执行作用域、池生命周期、抢占与心跳，原生 Images、Responses Lite namespace、模型映射、配额窗口、OpenCode Go，Antigravity/Gemini、DeepSeek、Ollama Cloud、Grok、协议转换、调度与计费，以及订阅/API Key 批量操作、认证、站点模式、监控、Ops、代理、支付和相关前端能力。

#### Scenario: v0.2.5 能力验证

- **WHEN** `v0.2.5` merge 完成并准备交付
- **THEN** 该 tag 的主要能力由对应 upstream 测试、fork 回归测试和能力级源码审查覆盖

### Requirement: 保留 fork 核心定制

冲突和无文本冲突语义审查必须保持 scheduler、sticky/fallback、DB recheck、请求体重放与清理、审计、每请求计费边界、运行时设置、插件边界、网关透传和前端定制语义；还必须保持管理员用量用户备注的管理员可见性与普通用户隐私边界，以及用户 token 排名 Excel 导出的用户名、排序和样式。无法共存时必须请求用户决定。

#### Scenario: upstream 修改重叠调用链

- **WHEN** upstream 更新触及上述 fork 能力所在文件或调用路径
- **THEN** 合并结果必须由现有或新增回归测试、或明确源码审查证明双方行为仍成立

#### Scenario: 管理员备注和导出定制保持

- **WHEN** 合并后的管理员用量接口、普通用户用量接口和用户 token 排名导出被验证
- **THEN** 管理员仍可查看用户备注、普通用户响应不包含备注，且导出的用户名、排序和样式符合 fork 当前行为

### Requirement: 验证、版本与发布边界闭合

最终必须完成冲突标记检查、受影响测试、后端默认与 unit 测试、后端 lint、前端 ESLint/i18n/单测/类型检查、前后端构建、migrations 与 Ent/Wire 两次生成稳定性检查；版本必须为 `0.2.5.1`，并由新的独立只读 Verifier 验收。

最终还必须复核现有项目记忆，并仅将本轮实际验证出的新合并经验增量写入 `memory/context/upstream-merge-workflow.md`；不得为了产出变更而重复或臆造经验。

#### Scenario: 环境不支持部分检查

- **WHEN** 本机缺少 Docker/Testcontainers 或其他必要运行条件
- **THEN** 必须执行其余门禁并明确记录未运行范围，不得将其记为通过

#### Scenario: 集成完成但不发布

- **WHEN** `v0.2.5` 合并及全部可执行验证通过
- **THEN** 项目保留可审计的集成提交和 `0.2.5.1` 版本文件，不创建或推送 Git tag、不触发 release workflow、不部署，并在任何后续 upstream tag 或 post-`v0.2.5` 提交前停止等待用户授权

#### Scenario: 本轮没有新的可复用经验

- **WHEN** 本轮实际过程没有超出现有项目记忆的有效原则
- **THEN** `memory/context/upstream-merge-workflow.md` 保持不变，并在 Builder 交接中明确说明未产生新增记忆
