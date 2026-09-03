# Memory

## Me

Sub2API — AI API 网关平台，用平台签发的 API Key 让 Claude Code/Codex/Gemini CLI 访问上游账号池，负责鉴权、计费、负载均衡、故障转移。技术栈：Go 1.25（Gin+Ent+Wire）+ Vue 3（Vite+Tailwind）+ PostgreSQL + Redis。

## People

| Who | Role |
|-----|------|

→ Full list: `memory/glossary.md`, profiles: `memory/people/`

## Terms

| Term | Meaning |
|------|---------|
| 网关(Gateway) | 接收 AI 客户端请求、转发到上游的核心层 |
| 账号(Account) | 上游 AI 订阅/凭证（OAuth 或 API Key 类型），被调度的资源 |
| 分组(Group) | 账号的逻辑集合，绑定 platform，决定调度策略 |
| 渠道(Channel) | 上游服务/模型供给的配置单元，承载模型映射与定价 |
| 粘性会话(Sticky) | 同一会话连续请求绑定到同一账号 |
| apicompat | 协议互转包，以 OpenAI Responses 为枢纽格式 |

→ Full glossary: `knowledge-base/reference/glossary.md`

## Projects

| Name | What |
|------|------|
| knowledge-base/ | 项目知识库（Diátaxis 框架），入口 `knowledge-base/README.md` |

→ Details: `memory/projects/`

## Knowledge Base

> 完整知识库位于 `knowledge-base/`，入口文件 `knowledge-base/README.md`。以下为快速导航。

### 架构与理解

- 架构总览 → `knowledge-base/explanation/architecture-overview.md`
- 上游协议兼容（apicompat） → `knowledge-base/explanation/upstream-protocol-compat.md`
- 横切关注点（幂等/并发/限流/缓存） → `knowledge-base/explanation/cross-cutting-concerns.md`

### 能力参考（31 个业务能力）

- 能力总索引 → `knowledge-base/reference/capabilities-index.md`
- 各能力详情 → `knowledge-base/reference/business/<name>.md`
- 术语表 → `knowledge-base/reference/glossary.md`
- API 契约来源 → `knowledge-base/reference/api-docs.md`

### 仓库参考

- 后端(Go) → `knowledge-base/reference/repositories/backend.md`
- 前端(Vue) → `knowledge-base/reference/repositories/frontend.md`
- 部署与CI → `knowledge-base/reference/repositories/deploy-and-ci.md`

### 操作配方

- 开发者任务 → `knowledge-base/how-to/for-developers.md`
- 管理员任务 → `knowledge-base/how-to/for-administrators.md`
- 运行时核验清单 → `knowledge-base/how-to/verify-runtime-workflows.md`

### 新人入门

- 教程：从零跑通 → `knowledge-base/tutorials/getting-started.md`

### 架构决策

- ADR 索引 → `knowledge-base/adr/README.md`

## Preferences

- 将高频约定、长期偏好或重要协作规则记录在这里。
- 发布本地版本时，只能基于当前 `HEAD` 已包含的最高上游三段式 tag 递增四段式版本；不要基于尚未合入的上游 tag 发 `.1`。
- 发版后需要使用 `gh` 跟进 Release workflow 结果，不要只校验远端 tag。
- Release 只有一个入口：推送目标 tag 后，用 `workflow_dispatch` 对该 tag 触发一次完整 Release，并核验二进制 assets 和 `checksums.txt`；推送 tag 本身不会触发发布。
- 镜像只能通过 GitHub Actions 构建；禁止在 dmit-serv-ai 等生产服务器上执行镜像构建，服务器仅拉取并运行 CI 发布的镜像。
- Comet/OpenSpec 文档沿用模板标题和必需关键字；业务填充内容尽量使用中文。该约束同步写入 `openspec/config.yaml`。
- 前端调试预览 → `memory/context/frontend-debug-preview.md`
- 发版流程 → `memory/context/release-workflow.md`
- 上游合并流程 → `memory/context/upstream-merge-workflow.md`
- dmit 服务器 sub2api 更新 → `memory/context/dmit-sub2api-update.md`
- local-serv-ai 服务器 sub2api 更新 → `memory/context/local-sub2api-update.md`

<comet-ambient-resume>
<!-- Managed by Comet. Edits inside this block may be replaced by comet init/update. -->
<!-- Contract: comet.resume_probe.v2 -->

## Comet Ambient Resume

在这个仓库中，开始处理需要改动或调查的任务前，如果可能存在活跃 Comet workflow，把当前用户请求传入只读探针：`comet resume-probe . --stdin --json`。

- 如果用户通过宿主明确调用任意 Comet Skill（例如 `@comet`、`/comet`、`@comet-native` 或 `/comet-hotfix`），显式调用优先于本恢复协议；不要运行 resume probe，直接进入被调用的 Skill。
- 如果用户通过宿主明确调用的是非 Comet 的 Skill 或斜杠命令，任务意图已由该调用明确：不要运行 resume probe，直接执行该 Skill。
- 如果你正在 Comet 流程内（包括正在等待用户回复你在流程中提出的问题），不要运行 resume probe；把这类回复（例如方案/选项选择）当作当前 change 的继续，直接按用户的选择推进。
- 只信任返回的 `workflow`、`skill` 和 `entrySource`；它们只由项目配置或无配置兼容回退决定。不得扫描或切换另一套 workflow。
- 如果 probe 返回 `auto_resume`，简短说明选中的 active change，并进入 `nextCommand` 指向的永久入口。不要把状态命令当作恢复入口直接推进。
- 如果 probe 返回 `ask_user`，只问一个简短问题并等待用户回复。
- 如果当前请求未明确调用 Comet Skill，且 probe 返回 `out_of_scope` 或 `none`，不要进入 Comet workflow。
- `out_of_scope` 或 `none` 只表示不要因为这个新请求进入 Comet workflow；它绝不表示要暂停或退出一个已在进行的 Comet 流程。
- 如果配置或状态无效且没有 `nextCommand`，停止并报告原因；不要猜测另一个 workflow。
- 不能只因为存在 active change 就把无关任务挂到该 change。Native 的未提交改动由 Native 入口检查，不由探针自动归因。
</comet-ambient-resume>
