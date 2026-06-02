# 会话上下文：Conversation Flow 注入过滤 + Refined Console 视觉重设计

> 创建时间：2026-05-27
> 最后更新：2026-05-28
> 分支：`main`（已合入）
> 最新相关 commit：`09a4ab52 feat: conversation timeline layout & parser improvements`
> 最新版本：`v0.1.131.2`

## 项目背景

Sub2API 前端 admin usage detail modal 中的「Conversation Flow」tab，用于展示 API 请求/响应的对话内容。

## 已完成的工作

### 第一阶段：Webgui-like Message/Part 模型重构（早期会话完成）

将初版 per-node timeline cards 重构为 webgui 风格的 `Message -> parts` 模型。相关 commits 在 `feat/conversation-flow-timeline` 远程分支（已 push 但未 merge PR）。

### 第二阶段：注入过滤 + Refined Console 视觉（commit `2753bc1e`）

基于 spec `docs/superpowers/specs/2026-05-26-conversation-injection-and-refined-style-design.md` 和 plan `docs/superpowers/plans/2026-05-26-conversation-injection-and-refined-style.md`，执行了 8 个 Task：

| Task | 内容 | 状态 |
|------|------|------|
| 1 | Types + `formatHumanBytes` helper | ✅ |
| 2 | Parser: system prompt extraction | ✅ |
| 3 | Parser: injection splitting | ✅ |
| 4 | Parser: reasoning merge + outputSize | ✅ |
| 5 | i18n keys + format.ts copy updates | ✅ |
| 6 | New components: SystemPromptBar + InjectionPart | ✅ |
| 7 | Visual redesign: MessageRow + ReasoningPart + ToolPart | ✅ |
| 8 | Integration verification | ✅ |

### 第三阶段：布局优化 + Parser 增强（commit `09a4ab52`，v0.1.131.2）

| 改动 | 说明 |
|------|------|
| Timeline 居中收窄 | `max-w-[45rem]` 居中容器 + 边框 + 滚动条常驻（`overflow-y-scroll` + `scrollbar-gutter: stable`） |
| 推理标签重命名 | "· 推理" → "思考内容" + 右箭头 chevron 指示展开 |
| AI 输出占满宽度 | assistant message shell 改为 `w-full`，工具/代码块不再受 max-width 限制 |
| 文件工具仅显示文件名 | `extractFilename()` 从完整路径提取文件名，如 `编辑：app.ts` |
| 任务列表渲染 | todowrite/todoread 展开后渲染带状态图标 + 优先级标签的任务列表 UI |
| SSE 流式响应解析 | `parseSSEBody()` 支持从 SSE 事件流重组完整响应，不再 fallback 为 Raw Response |
| 空 reasoning 跳过 | `summary: []` 的 reasoning item 静默跳过，不生成 Raw 卡片 |
| Dev 预览页 | `/dev/conversation-preview` 路由 + mock 数据页面，用于快速视觉验证 |

**验证**：`pnpm typecheck` ✅ | `pnpm test:run` → 46 tests passed (parseConversationPayload) ✅

## 关键设计决策

1. **System prompt 提取**：developer/system 角色消息从 `messages[]` 移入 `flow.systemPrompt`，顶部独立折叠条展示
2. **Injection 拆分**：用户消息中白名单 XML tag（`EXTREMELY_IMPORTANT`, `EXTREMELY-IMPORTANT`, `SUBAGENT-STOP`, `system-reminder`, `reminder`, `important`）提取为 `injection` part，默认折叠
3. **Reasoning 合并**：连续 reasoning parts 合并为 1 个，记录 `metadata.segments`
4. **Tool outputSize**：计算 formatted output 的 bytes/lines，header 右侧显示
5. **i18n key 命名**：用 `reasoningMeta.*` 和 `toolMeta.*` 避免与已有标量 key `conversation.reasoning` / `conversation.tool` 冲突
6. **视觉风格**：neutral gray refined console 风格；user bubble 用 `bg-gray-100` + `border-l-4 border-primary-500`
7. **居中列宽**：`max-w-[45rem]`（720px），在 `max-w-7xl`（1280px）弹窗内居中，两侧对称留白
8. **SSE 解析策略**：优先 `response.done` 完整对象 → `output_item.done` 重组 → delta 拼接重组
9. **文件名提取**：`read/write/edit/multiedit` 工具标题只显示文件名，不显示完整路径

## 当前状态

- **分支**：`main`，working tree clean
- **最新版本**：`v0.1.131.2`（已发布，Release workflow success）
- **local-serv-ai**：已更新到最新版，容器 healthy
- **远程分支** `origin/feat/conversation-flow-timeline`：包含第一阶段旧 commits，可清理

## 改动文件清单（第三阶段，commit `09a4ab52`）

```
frontend/src/components/common/conversation/ConversationTimeline.vue      — 居中容器 + 边框 + 滚动条样式
frontend/src/components/common/conversation/ConversationMessageRow.vue    — assistant shell 改 w-full
frontend/src/components/common/conversation/ConversationReasoningPart.vue — "思考内容" + chevron arrow
frontend/src/components/common/conversation/ConversationToolPart.vue      — todowrite 任务列表 UI
frontend/src/utils/conversation/toolDisplay.ts                            — extractFilename() 仅显示文件名
frontend/src/utils/conversation/parseConversationPayload.ts               — parseSSEBody() + 空 reasoning 跳过
frontend/src/utils/conversation/__tests__/parseConversationPayload.spec.ts — 更新文件名断言
frontend/src/i18n/locales/zh.ts                                           — reasoningMeta.collapsedLabel → '思考内容'
frontend/src/router/index.ts                                              — /dev/conversation-preview 路由
frontend/src/views/dev/ConversationPreview.vue                            — 新建 mock 数据预览页
```

## 可能的后续工作

1. **远程分支清理**：`origin/feat/conversation-flow-timeline` 已过时，可删除
2. **白名单扩展**：如有新的 injection tag 需要识别，修改 `INJECTION_TAG_WHITELIST` 常量即可
3. **SSE 解析覆盖**：当前 SSE 解析覆盖了 OpenAI Responses API 和 Chat Completions 流式格式，如遇其他格式可扩展
4. **Dev 预览页清理**：`/dev/conversation-preview` 路由和 `ConversationPreview.vue` 仅用于开发调试，生产环境可选择移除

## 重要约束（延续）

- `opencode.json`、`CLAUDE.md`、`memory/context/release-workflow.md` 是外部脏文件，不应混入功能提交
- 发版只能基于当前 HEAD 已包含的最高上游三段式 tag 递增四段式版本
- 前端调试预览流程见 `memory/context/frontend-debug-preview.md`
