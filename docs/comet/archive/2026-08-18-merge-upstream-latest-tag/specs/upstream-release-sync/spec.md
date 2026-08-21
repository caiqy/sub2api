# Upstream Release Sync Specification

## Purpose

将上游正式 release 安全地集成到本地定制分支，保留双方行为并留下可审计的验证结论。

## Requirements

### Requirement: 精确选择正式 release tag

维护流程必须在合并前确认本地版本、upstream 最新正式 release tag 及其 peeled SHA；不得以滚动的 `upstream/main` 代替已确认的 release tag。

#### Scenario: 合入 v0.1.177

- **WHEN** 当前本地 VERSION 为 `0.1.176.4`，upstream 最新正式 tag 为 `v0.1.177`
- **THEN** 流程必须以 `073e92d17178a1ccdb0a27017f572f10c9c7ab62` 为唯一上游合并父提交，并排除该 tag 之后的 `upstream/main` 提交

### Requirement: 隔离分支和可追溯 merge 节点

维护流程必须从干净的本地 `main` 创建临时集成分支，在该分支执行 `--no-ff` merge；不得直接改写 `main`。

#### Scenario: 建立 v0.1.177 merge 节点

- **WHEN** 合并开始时本地基线和 index 均干净
- **THEN** 结果 merge 节点的第一父必须是合并前临时集成分支 HEAD，第二父必须是 `v0.1.177` 的 peeled SHA，且 `v0.1.177` 必须是结果 HEAD 的祖先

### Requirement: 先保护本地能力，再处理上游变更

首次上游 merge 前必须通过既有本地质量门禁，并为被上游触及而缺少行为断言的高风险本地能力补充最小回归测试。冲突和无文本冲突的语义审查必须同时保留上游更新与本地定制；无法共存时必须暂停请求用户选择。

#### Scenario: 上游与本地调用链重叠

- **WHEN** 上游更新和本地定制修改同一文件或调用链
- **THEN** 结果必须保持 scheduler、平台 sticky、fallback/WaitPlan、DB recheck、审计、请求体重放与清理、每请求最多一次计费、运行时设置及前端本地定制语义，除非用户明确批准某项能力移除

### Requirement: 保留 v0.1.177 的上游行为

合并结果必须保留 `v0.1.177` 的 OpenAI compaction、Codex turn-state 与 opt-in fingerprint 行为，以及分组用量日汇总、时区 migration 和管理端相关更新，同时不绕过本地能力保护。

#### Scenario: OpenAI 调用链集成

- **WHEN** compaction、turn-state、fingerprint、网关转发或 usage 路径发生合并
- **THEN** 原生与 legacy compaction 路由、远端 compaction v2、`x-codex-turn-state` 的账号隔离、请求体生命周期、审计和计费边界都必须可由自动测试或专项审查证明

#### Scenario: 分组用量与 migration 集成

- **WHEN** 日汇总和时区 migration 进入本地 repository、service 与管理端调用链
- **THEN** 新库和已有本地记录升级路径不得破坏现有统计、缓存、权限或界面定制

### Requirement: 验证和版本闭合

每个合并阶段必须完成冲突标记检查、受影响能力测试、Ent/Wire 两次生成稳定性检查、`make test` 和 `make build`。最终阶段必须执行后端默认与 unit 测试、后端 lint、前端 ESLint、前端单测、前端类型检查和构建验证，并记录能力级审查证据。

#### Scenario: Docker 不可用的本机验证

- **WHEN** Docker/Testcontainers 在本机不可用
- **THEN** 流程必须执行其余本机门禁，并在验收报告中记录未执行的 Docker-backed migration/repository integration、环境原因和受影响契约；这些 integration 不得记为通过

#### Scenario: 集成完成但尚未发布

- **WHEN** 全部可执行门禁和专项审查完成
- **THEN** `backend/cmd/server/VERSION` 必须为 `0.1.177.1`，流程不得创建或推送 Git tag、触发 release workflow 或部署
