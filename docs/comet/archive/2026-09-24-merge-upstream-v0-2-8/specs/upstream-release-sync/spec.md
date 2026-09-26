# Upstream Release Sync Specification

## Purpose

将正式上游 release 按准确 tag 边界集成到本地 fork，保留双方行为并验证回归，不越过已授权的发布版本。

## Requirements

### Requirement: 精确合入 v0.2.8

从已包含 `v0.2.7` 的 fork 基线，在隔离分支执行一个 `--no-ff` merge；第一父为 `0c03d2ef2f9f63d164d59ec459f2d5387196ec57`，唯一上游第二父为 `v0.2.8` 的 peeled commit `fd80b08c90b55edcad5b00171b53f08721d30da1`。不得合并滚动分支或 tag 之后的上游提交。

#### Scenario: 可追溯的发布边界

- **WHEN** 在当前 fork 基线上完成合并
- **THEN** 结果有独立 merge 节点，包含精确的 `v0.2.8` 祖先，不包含之后的上游提交

### Requirement: 保留上游能力及 fork 定制

合并应保留上游新模型支持、OpenCode Go 用量窗口与自动刷新、推理力度计费、备份保留策略、内容审计、插件账号元数据、协议转换、网关与管理端/前端修复，并保持 fork 的 scheduler、sticky/fallback、DB recheck、请求体生命周期、审计、每请求计费、运行时设置、插件及网关透传、前端定制、Codex 打票与 OpenCode 会话支持。管理员用量备注只能由管理员查看；用户 token 排名导出保留 fork 的用户名、排序与样式。

内容审计继续对同一请求扫描有界的多轮用户/助手正文，排除 system/tool 和工具结果，不缩为仅最新用户轮次。语义 API 扫描过滤客户端 reminder，关键词扫描仍检查其中的客户端文本。

OpenCode Go 用量的同 key 组快照与刷新节流遵循 `v0.2.8` 原有行为；fork 的代理回退不另外定义这一新能力的快照失效策略。

#### Scenario: 上游行为与本地行为有重叠

- **WHEN** upstream 修改了上述定制所在文件或调用路径
- **THEN** 以相关 upstream 测试、fork 回归测试与能力级源码审查证明双方行为仍成立；若无法共存则暂停请求用户决定

### Requirement: 合并回归审计

合并后检查冲突标记及语义重叠，运行受影响测试、后端默认和 unit 测试、lint 与构建、前端 ESLint/i18n/单测/类型检查与构建，并检查 migrations、Ent/Wire 两轮生成稳定性；由独立只读 Verifier 审查验收。对环境不支持的检查明确记录未运行范围。

#### Scenario: 合并引入回归

- **WHEN** 检查或专项审查发现合并或 fork 定制引入的问题
- **THEN** 先用失败测试复现，再最小修复并重跑相关及完整检查

#### Scenario: 上游 tag 原样问题

- **WHEN** 问题在 `v0.2.8` commit 中原样存在且未受合并或 fork 定制影响
- **THEN** 记录证据但不扩大本次修复范围

#### Scenario: 本机不支持外部检查

- **WHEN** Docker/Testcontainers、race detector 或其他检查缺少运行条件
- **THEN** 其余门禁仍执行，无法执行的检查记为未运行而非通过

### Requirement: 版本与交付边界

合并后的 `backend/cmd/server/VERSION` 为 `0.2.8.1`；不创建或推送 tag，不触发 release workflow，不部署。仅在获得验证的新经验时增量更新合并经验文档，否则保持原状。

#### Scenario: 集成完成但不发布

- **WHEN** 合并及可执行验证均完成
- **THEN** 版本为 `0.2.8.1`，保留可审计的集成提交，且没有任何发布或部署动作

#### Scenario: 没有新增的可复用经验

- **WHEN** 本轮经验没有超出现有合并经验文档
- **THEN** 文档保持不变，Builder 交接注明没有新经验
