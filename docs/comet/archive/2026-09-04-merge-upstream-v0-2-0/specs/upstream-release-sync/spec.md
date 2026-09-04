# Upstream Release Sync Specification

## Purpose

将已获授权的 upstream 正式 release 按 tag 顺序安全集成到本地 fork，在每一版边界完成可追溯的冲突与回归审查，保留 upstream 行为和 fork 定制，并在未获授权的下一 tag 前停止。

## Requirements

### Requirement: 按顺序精确合并三个正式 tag

维护流程必须依次以 `v0.1.184`、`v0.1.185`、`v0.2.0` 的 peeled commit `e98ef32eb29aecd30d1def615912ec4dc93173f3`、`2ac784c51a5d0925b324efef2ba6b3446c364781`、`aa236488351eb71e120fc2b6fb32e36b0374c918` 为上游 merge 父，各生成一个独立 `--no-ff` merge；不得直接合并滚动分支或 `v0.2.0` 之后的 upstream 提交。

#### Scenario: 三个可追溯 merge 节点

- **WHEN** 当前基线已包含 `v0.1.183`
- **THEN** Git 第一父历史依次出现以三个 peeled commit 为第二父的 merge 节点，最终 HEAD 包含三个 tag 且不包含 post-`v0.2.0` upstream 提交

### Requirement: 每版先审计闭合再推进

每个 tag 合并后必须先完成冲突检查、受影响能力测试、本机完整质量门禁、构建和独立只读回归审计；发现的问题必须先用失败测试和最小修复闭合，前一版未闭合不得合并下一 tag。

#### Scenario: 单版审计失败

- **WHEN** 某版测试、构建或审计发现行为回归
- **THEN** 流程必须停留在该版修复并复验，不得继续合并下一 tag

### Requirement: 保留各 tag 的 upstream 行为

合并结果必须保留 `v0.1.184` 的网关、WebSocket、用量、模型目录、调度、计费与前端修复，`v0.1.185` 的价格目录覆盖、数据驱动长上下文计费、数据库重试与 Codex 能力处理，以及 `v0.2.0` 的分组 Fast/reasoning policy、Kimi Responses、fallback 清理、调度快照、持久化迁移与模型定价 UI。

#### Scenario: 逐版能力验证

- **WHEN** 一个 tag 的 merge 完成并准备进入下一 tag
- **THEN** 该 tag 的主要行为由 upstream 测试、fork 回归测试和能力级源码审查覆盖，后续 tag 不得掩盖前一版未验证结果

### Requirement: 保留 fork 核心定制

冲突和无文本冲突的语义审查必须保持 scheduler、sticky/fallback、DB recheck、请求体重放与清理、审计、每请求计费边界、运行时设置、插件边界、网关透传和前端定制语义；无法共存时必须请求用户决定。

#### Scenario: upstream 修改重叠调用链

- **WHEN** upstream 更新触及上述本地能力所在文件或调用路径
- **THEN** 合并结果必须由现有或新增回归测试、或明确源码审查证明双方行为仍成立

### Requirement: 验证、版本与发布边界闭合

最终必须完成冲突标记检查、受影响测试、后端默认与 unit 测试、后端 lint、前端 ESLint/单测/类型检查、前后端构建与 Ent/Wire 两次生成稳定性检查；版本必须为 `0.2.0.1`，并由独立只读 Verifier 验收。

最终还必须复核现有项目记忆，并仅将本轮实际验证出的新合并经验增量写入 `memory/context/upstream-merge-workflow.md`；不得为了产出变更而重复或臆造经验。

#### Scenario: 环境不支持部分检查

- **WHEN** 本机缺少 Docker/Testcontainers 或其他必要运行条件
- **THEN** 必须执行其余门禁并明确记录未运行范围，不得将其记为通过

#### Scenario: 集成完成但不发布

- **WHEN** 三个 tag 合并及全部可执行验证通过
- **THEN** 项目保留可审计的集成提交和 `0.2.0.1` 版本文件，不创建或推送 Git tag、不触发 release workflow、不部署，并在任何后续 upstream tag 或 post-`v0.2.0` 提交前停止等待用户授权

#### Scenario: 本轮没有新的可复用经验

- **WHEN** 本轮实际过程没有超出现有项目记忆的有效原则
- **THEN** `memory/context/upstream-merge-workflow.md` 保持不变，并在 Builder 交接中明确说明未产生新增记忆
