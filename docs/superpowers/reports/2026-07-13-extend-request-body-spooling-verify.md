# extend-request-body-spooling 验证报告

## 结论

| 维度 | 结果 |
| --- | --- |
| 完整性 | PASS：OpenSpec 16/16，实施计划 16/16 |
| 正确性 | PASS：全部 requirement 与 scenario 有实现和回归证据 |
| 一致性 | PASS：实现符合 OpenSpec design 与 Superpowers Design Doc |
| 安全与资源 | PASS：解压上限、脱敏、503/413/400、ownership 与 cleanup 均通过 |

最终评估：无 CRITICAL 或 IMPORTANT 问题，可以进入归档阶段。

## 实现覆盖

- Anthropic Messages/Responses、OpenAI Chat/Embeddings、Gemini 三种 action、OpenAI/Grok Images 与 Videos 均使用共享 request body coordinator。
- 严格大于 10MB 才落盘；5MB preview；压缩请求按解压后大小判断并受 64MB 上限保护。
- raw/effective handle 支持 retry/failover 重放，替换与请求终止时清理 owned handle 和 multipart form。
- usage/ops 仅保存有界 preview 或媒体 metadata，不保存完整 JSON、base64、data URL 或文件正文。
- spool create/write/close/open/read 失败不降级到内存，并按协议返回 503；请求过大返回 413；格式错误保持 400。
- multipart 文本 part 保持既有 20MB 单 part 边界；10MB/20MB 成功，20MB+1 返回 413。等待前释放 form/DTO 大文本，同时保持 moderation 与 sticky session 语义。
- JSON 媒体请求继续使用内容派生 sticky hash，显式 session 信号优先；multipart 使用释放前冻结的 fallback。

## 场景证据

- 5MB/10MB/12MB identity、gzip、multipart 共 9 例受控矩阵通过，客户端与上游 SHA-256 一致。
- 12MB 阻塞窗口采样：GC 后 heap 约 3.64-3.68MiB；raw/form spool 尺寸与请求一致；上游仅保留 size/hash。
- success、4xx、5xx、cancel、stream interruption 均验证响应、usage/billing 语义和 raw/effective/form cleanup。
- multipart 20MB 文本在 OpenAI API-key/OAuth 与 Grok generate/edit/video 路径验证等待期无长期大字符串引用。

## 最终命令

| 命令 | 结果 |
| --- | --- |
| `go -C backend test ./... -count=1` | PASS |
| `pnpm --dir frontend build` | PASS |
| `pnpm test:run`（`frontend`） | PASS：157 files / 1183 tests |
| `pnpm typecheck`（`frontend`） | PASS |

Build guard 已记录组合命令并通过。前端 build 仅有既有 chunk/dynamic-import 警告，无构建错误。

## 重新验证

- 用户在首次归档确认时选择重新执行完整 Verify。
- 四个重型命令并行执行时，后端一个 5 秒 failover 测试因资源竞争超时并暂时占用 spool 文件；该用例隔离运行约 6.3 秒通过，随后后端全量独占运行通过。
- 重新执行的前端 build、157 files / 1183 tests 与 typecheck 均通过。
- 最终 OpenSpec 完整审查无 CRITICAL、IMPORTANT 或 WARNING，结论为 ready for archive。
- 计划文件中的 `base-ref` 是实施计划创建时的代码基线；Comet `base_ref` 是 change 生命周期基线，两者用途不同，提交区间评估使用 Comet 基线。

## 已知非阻断项

- `-tags unit` 的 service 全包仍受变更前 Grok/Codex 测试漂移阻断；默认标签全量通过。
- race 验证环境缺少 CGO/C 编译器；本 change 的 ownership、并发释放与 panic/cancel 路径由定向测试和 thorough review 覆盖。
- Task 14 的临时 instrumentation 已删除，证据保存在 `.superpowers/sdd/task-14-report.md`；工作树未保留临时代码。
