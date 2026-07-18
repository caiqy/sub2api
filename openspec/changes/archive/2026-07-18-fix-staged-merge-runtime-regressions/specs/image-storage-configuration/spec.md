## ADDED Requirements

### Requirement: 图片存储配置保留已定义默认值
系统 MUST 注册上游已定义的 `image_storage.enabled`、`region`、`prefix`、`force_path_style`、`presign_expiry_hours` 与 `max_download_bytes` 默认值。

#### Scenario: 未显式配置时应用默认值
- **WHEN** 配置文件未提供上述图片存储可选字段
- **THEN** 系统使用 `false`、`auto`、`images/`、`false`、`24` 与 `33554432` 作为对应默认值

#### Scenario: 已注册字段接受环境变量覆盖
- **WHEN** 部署通过对应 `IMAGE_STORAGE_*` 环境变量覆盖已注册字段
- **THEN** 配置加载结果使用环境变量值而非默认值
