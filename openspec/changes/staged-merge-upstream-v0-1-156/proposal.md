## Why

本地分支与上游自 `v0.1.151` 后已分别积累大量提交，并共同修改网关、调度、运行时设置和生成代码等高风险区域。一次性合入 `v0.1.156` 难以定位无文本冲突的语义回归，因此需要先补齐本地能力保护门禁，再按正式 release tag 分段集成并逐段验证。

## What Changes

- 在首次 merge 前建立强制阶段 0：验证当前本地测试基线，将本地能力映射到现有测试，并只补充上游改动触及且缺少行为断言的高风险测试。
- 在隔离分支依次使用 `--no-ff` 合入 `v0.1.152`、`v0.1.153`、`v0.1.155` 和 `v0.1.156`，每段记录冲突决策并运行受影响能力及全部本地保护测试。
- 对无文本冲突的调用链、配置、生成代码和 migration 进行能力级审查；除明确例外外，本地能力不得回归，无法共存时暂停等待用户决定。
- **BREAKING** 在 `v0.1.156` 阶段完整移除本地 `openai-first-token-timeout` 实现、运行时配置、管理端 UI、测试和相关文档契约，完全采用上游的 HTTP 首输出超时与客户端 WebSocket 首消息超时语义。
- 最终执行后端与前端全量门禁、生成代码一致性、migration、冲突标记、Git 祖先关系及能力矩阵验证；不包含推送、发布或部署。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `upstream-release-sync`: 将单次 release 合并扩展为可按多个正式 tag 分段集成，并增加合并前测试保护门禁、逐段能力验证和明确例外管理。
- `openai-first-token-timeout`: 移除本地首 Token 超时契约，改为完全跟随上游 `v0.1.156` 提供的超时行为。

## Impact

- Git 历史：新增四个按顺序关联正式 tag 的 merge 节点，以及必要的本地兼容修复提交。
- 后端：重点影响 OpenAI 网关、WebSocket、调度与 Sticky、运行时设置、请求体生命周期、内容审计、图片能力、Wire/Ent 和 migrations。
- 前端：重点影响管理设置、账号与资源控制、用量与图片功能、i18n 及相关测试。
- 配置与 API：删除本地 `openai_text_first_token_timeout`、`openai_image_first_token_timeout` 及其运行时设置/API/UI；采用上游配置字段和默认行为。
- 验证：沿用无 Docker 的本地质量门禁，并增加本次上游同步专用的能力映射、逐段验证和最终全量审查记录。
