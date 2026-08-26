# Upstream Release Sync Specification

## Purpose

将已获授权的连续 upstream 正式 release 按 tag 顺序安全集成到本地 fork，在每一版边界完成可追溯的回归审计，保留 upstream 行为和 fork 定制，并在未获授权的下一 tag 前停止。

## Requirements

### Requirement: 精确且顺序地合并正式 tag

维护流程必须依次以 `v0.1.180`、`v0.1.181`、`v0.1.182` 的 peeled commit 为唯一上游合并父，不得跳版、合并滚动分支或包含 `v0.1.182` 之后的 upstream 提交。

#### Scenario: 三版形成独立 merge 节点

- **WHEN** 当前基线已包含 `v0.1.179`
- **THEN** 三次 `--no-ff` merge 的第二父必须依次为 `c40edb4070a9274e8c23f161b4ed552051b14698`、`3af5443b224823ae507a50c7b113aa50604409c8`、`5a7d469622911a6b1291a692376df5fa03f9ac2e`，且结果不包含 `v0.1.182` 之后的 upstream 提交

### Requirement: 每版先审计再推进

每个 tag 合并后必须先完成冲突检查、受影响能力测试、本机完整质量门禁、构建和独立只读回归审计；发现的问题必须修复并复验，前一版未闭合不得合并下一版。

#### Scenario: 单版审计失败

- **WHEN** 某版测试、构建或审计发现行为回归
- **THEN** 流程必须停留在该版，使用最小回归测试和修复闭合问题，不得继续合并下一 tag

### Requirement: 保留 fork 核心定制

冲突和无文本冲突的语义审查必须保持 scheduler、sticky/fallback、DB recheck、请求体重放与清理、审计、每请求计费边界、运行时设置、网关透传和前端定制语义；无法共存时必须请求用户决定。

#### Scenario: upstream 修改重叠调用链

- **WHEN** upstream 更新触及上述本地能力所在文件或调用路径
- **THEN** 合并结果必须由现有/新增回归测试或明确源码审查证明双方行为仍成立

### Requirement: 保留三版 upstream 行为

集成结果必须保留 `v0.1.180` 的插件/Fast mode/模型列表与协议兼容更新、`v0.1.181` 的 Gemini/Grok/Responses 修复，以及 `v0.1.182` 的 Responses Lite/WS、OAuth 图片提示、Kimi K3、Antigravity、Anthropic cache TTL 计费、OpenCode reset、monitor 平台归因和支付余额刷新修复。

#### Scenario: 逐版能力验证

- **WHEN** 每个 tag 合并完成
- **THEN** 该 tag 的主要行为必须由对应 upstream 测试、fork 回归测试和能力级审查覆盖，后续 tag 不得掩盖前一版未验证结果

### Requirement: 验证、版本和发布边界闭合

每版必须执行受影响测试、后端默认与 unit 测试、后端 lint、前端 ESLint/单测/类型检查以及前后端构建；最终还必须确认 Ent/Wire 生成稳定、merge 拓扑正确、`VERSION` 为 `0.1.182.1` 且没有发布动作。

#### Scenario: Docker 不可用

- **WHEN** 本机没有 Docker/Testcontainers
- **THEN** 必须执行其余门禁，并明确记录未运行的 migration/repository integration 及受影响契约，不得将其记为通过

#### Scenario: 集成完成但不发布

- **WHEN** `v0.1.180`、`v0.1.181`、`v0.1.182` 均审计通过
- **THEN** 项目保留可审计的本地集成提交和 `0.1.182.1` 版本文件，不创建或推送 tag、不触发 release workflow、不部署，并在任何后续 upstream tag 或 post-`v0.1.182` 提交前停止等待用户授权
