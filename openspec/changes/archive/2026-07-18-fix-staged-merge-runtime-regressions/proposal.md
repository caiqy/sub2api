## Why

连续合并上游至 `v0.1.159` 时，两处冲突融合遗漏了上游已有行为：部分 OpenAI 入口仍用旧参数上报调度结果，且 `image_storage` 的 Viper 默认键注册被丢失。这会造成模型级瞬时冷却无法被成功请求及时清除、WS 失败终态被误记为成功，以及环境变量配置的异步图片存储无法按预期启用。

## What Changes

- 让 Responses、Messages、Embeddings 与 Responses WebSocket 使用 canonical model 和真实终态上报调度结果。
- 恢复上游 `v0.1.157` 已定义的六个 `image_storage.*` 默认键。
- 增加覆盖调度终态、模型级瞬时状态清理和图片存储配置加载的回归测试。
- 不改变 API、数据库结构、调度策略或对象存储接口。

## Capabilities

### New Capabilities

- `openai-schedule-result-reporting`: 记录 OpenAI 调度结果上报必须保留 canonical model 与上游真实终态的既有行为。
- `image-storage-configuration`: 记录异步图片对象存储配置默认值和已注册环境变量覆盖的既有行为。

### Modified Capabilities

无。

## Impact

影响 OpenAI 网关 handler、调度结果上报调用、图片存储配置默认值及对应测试。不新增依赖，不需要迁移数据。
