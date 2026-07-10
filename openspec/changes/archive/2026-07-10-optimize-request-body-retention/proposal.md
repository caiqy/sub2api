## Why

大 `/responses` 请求目前会在 request capture、handler、upstream request capture、ops context 和 usage snapshot 之间产生多份完整 body 副本。上游响应常常需要 10 秒以上，导致完整 request body 在等待期长时间驻留内存，并继续以 usage detail / ops 形式保存在数据库中。对于 50-80MiB 请求，这已经成为独立于内容审计之外的内存和存储放大来源。

## What Changes

- 引入统一的 request body handle，集中管理完整 body、preview、hash 和生命周期。
- 所有 `/responses` 请求都改为只在观测链路保留有界 preview，而不保留完整 request/upstream body 副本。
- 对 `>10MB` 的有效转发 body 启用临时文件模式，让上游发送和重试从文件重开 reader。
- 将 `requestPayloadHash` 提前到转发前计算，避免为了 usage 记录让 handler 长时间持有完整 body。

## Capabilities

### New Capabilities
- `request-body-retention-control`: 控制 `/responses` 请求体在网关、观测和上游转发链路中的驻留方式，降低长时间内存占用。

### Modified Capabilities

## Impact

- 影响 OpenAI `/responses` handler、usage detail capture、ops upstream request capture 和上游 request builder。
- 不改变内容审计语义、不改变上游请求语义、不改变计费与 usage 指纹语义。
- 不新增数据库 schema；超大 body 的完整正文不再进入 usage detail。
