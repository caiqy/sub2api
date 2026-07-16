---
comet_change: staged-merge-upstream-v0-1-156
role: technical-design
canonical_spec: openspec
---

# 分段合并上游 v0.1.156 技术设计

## 1. 背景与约束

本地基线为 `main@d1cc02502`，与 `origin/main` 同步。上游最终目标 `v0.1.156` peel commit 为 `12f991dde8a58e183d4bd16a87ef6fd0df714757`；相对共同基线，本地与上游分别有 541 和 300 个独有提交，上游 release 区间涉及 503 个文件。高风险变化集中在网关、调度、运行时设置、前端管理功能、Wire/Ent 和 migrations，与本地定制高度重叠。

Git 文本冲突只能发现同一文本区域的竞争修改，无法发现拆文件后的字段漏迁、新入口绕过旧条件、缓存结构不完整或生成 provider 丢失。设计目标不是宣称“可以全面检测所有回归”，而是先建立本地行为基线，再把首次回归压缩到单个 release 区间，并要求每项本地能力有自动或人工证据。

本次不包含 `v0.1.156` 后的 `upstream/main` 提交，不推送、不发布、不部署，不重写本地历史，也不新增通用 merge 或测试框架。

## 2. Git 拓扑与提交边界

### 2.1 分段目标

在用户选择的隔离 feature 分支或 worktree 中按以下顺序执行：

| 阶段 | 上游区间 | 提交数 | 文件数 |
|---|---|---:|---:|
| 1 | `v0.1.151..v0.1.152` | 56 | 128 |
| 2 | `v0.1.152..v0.1.153` | 41 | 97 |
| 3 | `v0.1.153..v0.1.155` | 71 | 238 |
| 4 | `v0.1.155..v0.1.156` | 132 | 253 |

每个 tag 使用独立 `--no-ff` merge 节点。tag merge commit 包含该 tag 的上游树与必要冲突解决；merge 后由测试或能力审查发现的语义回归使用后续普通提交修复。阶段 0 的测试补齐也在第一个 merge 节点之前形成普通提交。

该边界使审查者可以区分上游引入、冲突融合和本地兼容修复。相邻 tag 即使重复冲突，也不得机械复用 ours/theirs；只复用已验证的业务决策。

### 2.2 阶段状态机

```text
固定 tag peel SHA
  -> --no-ff merge 与冲突台账
  -> 完成 merge commit
  -> 能力审查与失败测试
  -> 最小兼容修复提交
  -> 阶段保护测试通过
  -> 允许下一个 tag
```

未授权且无法共存的语义是阻塞状态，必须等待用户决定。任何阶段未完成能力审查或验证时，不得创建下一 tag merge 节点。

## 3. 阶段 0 本地能力保护门禁

### 3.1 基线验证

首次 merge 前至少执行：

```text
root:     make test
frontend: pnpm build
backend:  Ent 与 Wire 生成结果一致性检查
```

`make test` 委托当前仓库已定义的后端默认测试、unit 测试、`golangci-lint`、前端 ESLint、typecheck 和 Vitest。基线失败必须先诊断并收敛，不得带入第一个 merge 后再归因给上游。

### 3.2 能力矩阵来源

能力清单取以下来源的并集：

1. `openspec/specs/` 中的本地主规格。
2. `knowledge-base/` 和 `memory/context/upstream-merge-workflow.md` 记录的本地关键能力。
3. 上次 `v0.1.151` 合并验证报告的能力矩阵与已修复回归。
4. `v0.1.151..HEAD` 的本地一方提交，特别是用户资源控制、平台 Sticky、请求体重放与清理、运行时设置和本地测试门禁。

每行至少包含：能力名称、本地行为契约、入口与调用链、关键文件、触及该能力的 tag、现有自动测试、人工审查点、阶段结果和证据位置。

### 3.3 缺口判定

矩阵状态仅使用：

- `protected`：存在直接行为测试，且阶段 0 基线通过。
- `gap`：上游目标会触及该能力，但现有测试不能断言关键行为；首次 merge 前必须补最小测试。
- `manual`：生成代码、migration、版本依赖或跨层契约无法只靠行为测试确认，需要结构或生成审查。
- `approved-removal`：用户明确批准移除；本次仅允许本地首 Token 超时使用。

不追求逐行覆盖率。新增测试必须同时满足本地独有、目标 release 触及、缺少行为断言三个条件，并优先覆盖 previous-response/session Sticky、fallback/WaitPlan、DB recheck、热更新、重试、终止错误、body replay/cleanup 等边界路径。

## 4. 冲突与无冲突语义审查

### 4.1 冲突台账

每个冲突文件记录：

| 字段 | 内容 |
|---|---|
| 类别 | 上游修复、本地定制、接口/配置演进、版本依赖、生成代码或 migration |
| ours | 合并前本地行为与依赖调用链 |
| theirs | 当前 tag 引入的上游行为 |
| 结果 | 最终融合语义或已批准移除 |
| 证据 | 测试命令、生成检查或人工审查结论 |

两边可共存时最小融合。不能共存且不属于首 Token 例外时，停止自动解决并提交用户决策。

### 4.2 无文本冲突检查

对每段 `git diff <前一 tag>..<当前 tag>` 的 changed files，与能力矩阵关键文件求交集。若上游修改入口、条件、DTO、配置解析、运行时缓存、provider、schema 或生成结果，即使 Git 没有报告冲突，也必须检查相关调用方和影响范围；结构追踪优先使用 CodeGraph context、impact 或 trace，具体行为由现有或新增测试验证。

重点能力包括 scheduler、各平台 Sticky、fallback/WaitPlan、DB recheck、Messages/Responses/Chat 转换、透传字段、终止 usage、privacy 与内容审计、image capability、运行时设置热更新、请求体重放与清理、用户资源控制、前端本地功能和本地测试门禁。

## 5. 首 Token 超时替换

本地首 Token 超时在前三个 tag 阶段仍为 `protected`，只在 `v0.1.156` merge 后切换为 `approved-removal`。最终必须删除：

- HTTP SSE 与 WebSocket 上游首输出 watchdog；
- 文本 30 秒与明确生图 600 秒分档；
- `openai_text_first_token_timeout`、`openai_image_first_token_timeout` 的配置、运行时存储、DTO、API 和管理端 UI；
- `first_token_timeout` 专用错误、日志、失败 usage 与 Ops 逻辑；
- 本地专用测试和不再成立的兼容文档。

结果完全采用上游 `v0.1.156`：native HTTP Responses 的 `openai_first_output_timeout_seconds`、高 reasoning effort 覆盖、默认关闭、`first_output_timeout`、failover 与 `HandleStreamTimeout` 行为，以及客户端 WebSocket 首消息超时。上游没有等价的 WebSocket 上游首输出 watchdog，这是用户知情批准的能力移除，不得在兼容修复中恢复。

删除后扫描旧配置键、错误类型、结构化日志事件、watchdog 与本地文件符号；业务代码不得残留兼容别名。上游配置、实现和测试必须保留并通过。

## 6. 分阶段验证

每个 tag 阶段执行：

1. 未合并文件、冲突标记和 `git diff --check`。
2. 阶段 0 固定的全部本地保护测试。
3. 当前 tag changed files 对应的包、组件和契约测试。
4. 冲突文件对应的聚焦测试或生成检查。
5. 能力矩阵逐项结论与残余风险记录。

阶段测试失败时保留可复现失败，再做最小修复；不得先继续下一 tag。基础设施或环境限制必须单独记录，不得将“未执行”写成“通过”。

最终阶段执行：

- 根目录 `make test`；
- 前端 `pnpm build`；
- Ent 与 Wire 重新生成后无非预期 diff，并验证重复生成稳定；
- migration 文件、编号冲突、幂等性和 runner 顺序复核；
- `VERSION`、Go/前端依赖、配置默认值和发布元数据复核；
- `git diff --check`、冲突标记扫描和干净工作树检查；
- `git merge-base --is-ancestor` 验证四个 tag 均为结果祖先；
- 完整能力矩阵和 thorough review。

## 7. 错误处理与回退

- 阶段 0 基线失败：停在首次 merge 前。
- 文本冲突可融合：完成最小融合并添加直接证据。
- 未授权语义不可共存：暂停等待用户决定。
- 阶段测试失败：停在当前 tag，修复并重跑该阶段门禁。
- 生成代码不稳定：回到 schema/provider 源修复，不手工维护不可复现生成结果。
- 最终验证失败：不得进入 finishing-branch 或 archive。

合并尚未进入 `main` 时，回退方式是放弃隔离分支或 worktree。进入 `main` 后按阶段 revert 对应 merge commit 及其后的兼容修复提交，不改写已推送历史。推送、发布与部署始终属于后续独立决策和流程。

## 8. 完成条件

四个 tag 按顺序成为结果祖先；23 项 OpenSpec 任务完成；首 Token 本地能力按批准范围完整退场；其余能力矩阵无未解释回归；冲突、生成物、migration、版本依赖和全量门禁均有可复现证据；验证报告明确记录未执行项与残余风险。
