# Task 9 Report

## 实现

- OpenAI Images 与 Grok media 请求改用现有 `requestBodyCoordinator`：超过 10MB 的 raw body 在上游调用期间落盘，并在 handler 返回后清理。
- OpenAI JSON inline data URL 与 Grok multipart source/mask 回归测试验证：上游完整接收 body，usage/ops preview 不包含正文或文件 bytes。
- coordinator 回归测试以标准库 `multipart.Writer` 和专用 tempdir 验证 raw spool、effective spool、multipart temp 文件在 cleanup 后删除。

## TDD

- RED: `TestOpenAIImages_InlineSpoolKeepsRawBodyAndOmitsSnapshots` 在改动前因 raw spool 目录为空失败。
- RED: `TestGrokMedia_MultipartSpoolPreservesFilesAndOmitsSnapshots` 在改动前因 raw spool 目录为空失败。

## 验证

- `go test ./internal/handler -run 'Test(OpenAIImages|GrokMedia).*(Multipart|Inline|Mask|Source|Spool)' -count=1`
- `go test ./internal/handler -run 'Test(OpenAIImages|GrokMedia)' -count=1`
- `go test ./internal/handler -count=1`
- `go test -tags unit ./internal/handler -count=1`

## 已记录漂移

`go test -tags unit ./internal/service -run 'Test.*Grok' -count=1` 在未执行本任务代码路径前编译失败：缺少 `shouldAutoPauseGrokAccountByQuota`，且 `OpenAIGatewayService` 不再有 `grokTokenProvider` 字段。按 Task 9 约束，仅记录，未修改 service unit 测试或生产代码。
