## Why

settings service 拆分后，legacy JSON 中缺失字段会被 Go 零值覆盖，回退了既有默认值回填行为，导致升级后的历史配置改变运行语义。

## What Changes

- overload cooldown、stream timeout 与 rectifier getter 在反序列化前使用各自默认配置初始化结果。
- 保留已有校验、错误回退与显式字段覆盖行为。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- 无。

## Impact

- 仅修改 `backend/internal/service/setting_features.go`。
- 不修改公开 API、配置格式、数据库 schema、依赖或产品 spec。
