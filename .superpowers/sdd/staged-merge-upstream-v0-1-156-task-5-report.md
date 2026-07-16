# Task 5 v0.1.152 合并与冲突融合记录（OpenSpec 2.1）

## 状态

- 结果：`DONE`。
- merge commit：`4ffe039a4399f8cbac1f83df32b709afda777ffe` `merge: upstream v0.1.152`。
- 第一父：`f10199795fd5fe4ef54c99553149177612179756`（任务准备提交）。
- 第二父：`b73d8c3efe01a290eaaa9326b6e40ece02c67a0e`（唯一允许的 `v0.1.152^{}`）。
- 本报告随后以独立提交 `docs: record v0.1.152 merge decisions` 记录；该提交只包含本文件。

## 前置核验与命令记录

| 命令 | 结果 |
| --- | --- |
| `git status --short` | merge 前唯一未跟踪项为允许的 `.comet/current-change.json`。 |
| `git status --branch --short` | 分支为 `feature/20260716/staged-merge-upstream-v0-1-156`。 |
| `git log -1 --format='%H %s'` | `f10199795fd5fe4ef54c99553149177612179756 chore: prepare staged merge task 5`。 |
| `git rev-parse 'v0.1.152^{}'` | `b73d8c3efe01a290eaaa9326b6e40ece02c67a0e`，与任务固定 SHA 一致。 |
| `git cat-file -t v0.1.152` | `tag`，确认目标为 annotated tag。 |
| `git tag -v v0.1.152` | 本机缺失 tag 签名，校验失败；不影响 tag 对象类型与 peel SHA 的确定性核验。 |
| Task 4 阶段 0 报告和 `openspec/changes/staged-merge-upstream-v0-1-156/tasks.md` | 阶段 0 的 1.1 至 1.4 已勾选，`make test`、前端 build、Ent/Wire 生成检查和 `git diff --check` 均有通过记录。 |
| `git merge-base --is-ancestor v0.1.152 HEAD` | merge 前返回非零，确认 tag 尚未是当前分支祖先。 |
| `git diff --name-status HEAD...v0.1.152` | 确认本次仅需处理该 tag 与当前分支的差异。 |
| `git merge --no-ff v0.1.152 -m "merge: upstream v0.1.152"` | 进入冲突状态；未 abort、squash 或 cherry-pick。 |
| `git diff --name-only --diff-filter=U`、`git diff --cc -- <path>`、`git show :1:<path>`、`git show :2:<path>`、`git show :3:<path>` | 逐路径核对共同祖先、第一父本地实现和 tag 上游实现。 |
| CodeGraph `context`、`impact(openAIFirstTokenWatchdog)`、`callers(resolveOpenAIUpstreamEndpoint)` | 确认首 Token watchdog 覆盖 Responses-to-Chat fallback，真实上游端点解析由 Responses、WebSocket、cyber-policy 记录调用。 |
| `go generate ./ent` | 基于已自动合并的 `backend/ent/schema/group.go` 重生成 Ent，修复 runtime descriptor 索引。 |
| `gofmt -w <10 个冲突 Go 文件>` | 格式化必要融合路径。 |
| `git ls-files -u`、`git diff --name-only --diff-filter=U`、`rg -n '^(<<<<<<<|>>>>>>>)' backend frontend` | 输出为空。 |
| `git diff --cached --check` | 输出为空。 |
| `git commit --no-edit` | 创建上述 merge commit。 |
| `git show -s --format='%H%n%P%n%s' HEAD`、`git rev-parse HEAD^2` | merge SHA、两个父节点、subject 和第二父均符合要求。 |
| `git merge-base --is-ancestor upstream/main HEAD` | 返回非零，`upstream/main` 不是结果祖先。 |
| `git log --oneline v0.1.152..upstream/main --not HEAD` | 列出所有 tag 后 upstream/main 提交，均未进入结果。 |
| `git show --check --format=fuller --stat HEAD` | 输出无 whitespace 错误；commit 仅为 merge 树与冲突融合。 |

## 冲突台账

共 `15` 个文本冲突；无其他未合并路径。

| 路径 | 分类 | 第一父本地语义 | tag 上游语义 | 融合结论 | Task 6 验证 |
| --- | --- | --- | --- | --- | --- |
| `backend/cmd/server/VERSION` | 版本依赖 | 本地四段版本 `0.1.151.2`。 | tag 内为 `0.1.151`。 | 保留本地版本，避免在 staged merge 擅自改变本地发布编号。 | 复核最终版本与 tag/发布流程。 |
| `backend/ent/mutation.go` | 生成代码/接口演进 | `user_concurrency_*` mutation 字段。 | `web_search_price_per_call` mutation 字段。 | 按合并 schema 重生成，两个字段集合均存在，`Fields` 容量为 50。 | Ent 生成可复现检查。 |
| `backend/ent/runtime/runtime.go` | 生成代码/接口演进 | 用户并发 descriptor 偏移。 | 网页搜索价格 descriptor 偏移。 | 重生成后所有后续 descriptor 索引后移一位，未发生同位读取。 | Ent 生成可复现检查。 |
| `backend/internal/handler/openai_chat_completions.go` | 网关接口演进/本地定制 | failover 耗尽 usage、usage detail snapshot、raw Chat endpoint 推导。 | 模型不存在错误分类和 forwarding result 的真实 endpoint。 | 保留 failover 分支与 detail snapshot；无 failover 时使用上游分类；采用真实 endpoint resolver。 | Chat Completions 无账号、failover、usage detail 和 raw Chat 路径测试。 |
| `backend/internal/handler/openai_gateway_handler.go` | 网关接口演进/本地定制 | Responses/Messages failover、detail snapshot、调度保护。 | 模型不存在错误分类和真实 endpoint。 | failover 分支保留；无 failover 时使用上游错误分类；所有 usage 记录使用真实 endpoint。 | Responses、Messages、WebSocket usage 与 no-account 定向测试。 |
| `backend/internal/server/routes/gateway.go` | 路由/本地定制 | Responses 路由捕获 usage detail，Grok 拒绝 Responses WebSocket。 | 新增 `/alpha/search` 路由。 | 保留 middleware 和 Grok 拒绝，新增 alpha/search 并同样接入 usage detail。 | 路由契约及 alpha/search 计费测试。 |
| `backend/internal/service/api_key_auth_cache_impl.go` | 缓存版本依赖 | snapshot v17 已涵盖 blocked group、批量图像/视频价格。 | v15 新增网页搜索单次价格。 | 保持 v17，并明确其快照覆盖网页搜索价格。 | API Key cache 序列化、失效与网页搜索价格测试。 |
| `backend/internal/service/openai_gateway_grok.go` | 调度/配额融合 | pool-mode 可重试状态不临时下线账号。 | 将 Grok quota snapshot 写入运行时与持久限流状态。 | 先保留 pool-mode 早退，其他状态继续写入 quota snapshot。 | Grok pool-mode、429/quota snapshot 测试。 |
| `backend/internal/service/openai_gateway_responses_chat_fallback.go` | 首 Token 保护/协议桥 | SSE 首 Token watchdog、超时记录和客户端写入保护。 | custom/tool_search/namespace 工具往返还原。 | 两者共存；watchdog 仍覆盖 fallback 串流，工具元数据完整传入转换器。 | 首 Token fallback 超时、custom/tool_search/namespace 流式与非流式测试。 |
| `backend/internal/service/openai_gateway_service.go` | usage/接口演进 | 异步 usage 的深拷贝快照。 | 实际上游 endpoint context 和网页搜索计数字段。 | 保留快照函数，同时加入 endpoint context API 和 `WebSearchCalls`。 | usage 异步快照与实际 endpoint 记录测试。 |
| `backend/internal/service/openai_oauth_passthrough_test.go` | 测试融合 | 断言账户注入 header、转发 header 和 body 字段。 | 断言 `x-codex-beta-features=remote_compaction_v2`。 | 同一用例保留全部三类断言。 | 运行该文件的 passthrough 回归测试。 |
| `frontend/src/components/account/CreateAccountModal.vue` | 前端本地定制 | `getDefaultBaseUrl` 统一覆盖平台默认 URL。 | Grok API Key 默认 xAI URL。 | 保留 helper，已覆盖 Grok 且避免手写分支遗漏其他本地平台。 | CreateAccountModal Grok/API Key 测试。 |
| `frontend/src/components/account/EditAccountModal.vue` | 前端本地定制 | `getDefaultBaseUrl` 统一覆盖 OpenAI、Grok、Antigravity 等。 | Grok API Key 默认 xAI URL。 | 保留 helper，维持完整平台覆盖。 | EditAccountModal Grok/API Key 测试。 |
| `frontend/src/i18n/locales/en/admin/settings.ts` | 前端接口演进 | 手工用户 ID 输入文案。 | 邮箱模糊搜索 selector 文案。 | 采用上游 selector 键；底层仍选择同一用户 ID，配合已合入 selector 组件。 | Fast/Flex 用户 selector locale 测试。 |
| `frontend/src/i18n/locales/zh/admin/settings.ts` | 前端接口演进 | 手工用户 ID 输入中文文案。 | 邮箱模糊搜索 selector 中文文案。 | 采用上游 selector 键，与英文 locale 和组件契约一致。 | Fast/Flex 用户 selector locale 测试。 |

## 自审与风险

- merge commit 不含验证报告、OpenSpec task、Comet progress、`.comet/current-change.json` 或其他 `.superpowers/` 文件；本报告是 merge 后唯一单独文档提交内容。
- 本阶段未运行行为回归、完整后端测试或前端 build，未新增测试、未做无冲突能力审查或 merge 后语义修复；这些均由 Task 6 按 TDD 和阶段测试矩阵执行。
- 本地首 Token 超时仍受保护：其 watchdog 及 fallback 覆盖路径保留；未提前进行仅获批准于 v0.1.156 的移除。
- 风险信号：本 tag 触及网关、协议转换、Ent schema/生成物、Grok 配额、API Key cache、计费与前端设置；生成物和无文本冲突行为仍需 Task 6 复核。
- 顾虑：本机无法验证 annotated tag 签名；已使用对象类型与固定 peel SHA 作为合并身份依据。`VERSION` 选择保留本地四段版本，需在最终发布阶段再次复核。
