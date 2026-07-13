# Task11 Review Round1

基线：`d224ee1f1`。

- Grok media recorder 逐 attempt 比较独立 fixture 的完整 body、SHA-256、Content-Type、method 与 path。
- 覆盖 generate、edit、video create、video status、拒绝、4xx、5xx failover、pool-mode 同账号 retry 与取消。
- usage detail 与 ops 上游 body 均必须存在，且不包含 base64/data URL、multipart 文件正文或二进制 sentinel。
- video status 以 GET 验证，并以不可创建的 spool 目录证明不创建 request body handle。
- 修复 pool-mode Grok 在 429 同账号 retry 前被临时下线的问题；拒绝路径现在也写入脱敏 ops preview。

验证：

- `go test ./internal/handler -run 'TestGrok(Media_GenerateEditVideoRejectUpstreamFailoverPreserveRequestSemantics|Media_MultipartSpoolPreservesFilesAndOmitsSnapshots|Media_MultipartEditTextSourcesRebuildUpstreamJSON|VideoStatus_UsesNoRequestBodyHandle)$' -count=1 -timeout 120s`
- `go test ./internal/handler ./internal/service -count=1 -timeout 180s`
- `go test ./internal/handler -count=1 -timeout 180s`
