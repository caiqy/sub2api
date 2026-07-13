# Task 13 Report

- 增加串行受控端侧矩阵：identity JSON、gzip JSON、multipart 分别覆盖精确解码后 5MB、10MB、12MB。
- 每例记录并断言客户端/模拟上游 SHA-256；上游阻塞时按 raw、effective、multipart form 临时目录分别断言。12MB raw body spool 存在；JSON effective 目录为空；multipart fixture 的 `multipart-*` form 文件在三档均存在，返回后所有目录为空。
- 阻塞 JSON 与 multipart upstream 在启动前注册 `sync.Once` release 和带超时的 `t.Cleanup`，此前置断言失败也会释放 handler goroutine。
- JSON 使用真实 OpenAI Embeddings handler，gzip 以解压后正文核验；multipart 使用真实 Grok Images handler，确认完整文件和文本 multipart body 原样到达上游。usage detail 和 ops wrapper 均按 JSON 结构解析，明确断言 `kind`、原始 `size`、`truncated`、multipart 文件内容省略，以及 preview 不超过生产 `openAIResponsesRequestBodyPreviewLimitBytes`。
- 本轮使用临时 D 盘 Go cache 重跑：`go test ./internal/handler -run '^TestRequestBodySizeMatrix$' -count=1 -v`（9/9），`go test ./... -count=1`。前端也重跑 `pnpm test:run`（157 files / 1183 tests）、`pnpm typecheck`、`pnpm build`，均 PASS。
