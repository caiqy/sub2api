## Why

当前主线运行版本为 `0.1.169.3`，上游已发布后续正式 tag `v0.1.170` 和 `v0.1.171`。两版累计修改 392 个文件，并与本地调度、网关、请求体生命周期、安全审计、订阅额度和前端定制形成 151 个路径重叠，需要沿用上一轮分段合并和能力级审查方法完成可审计升级。

## What Changes

- 从固定的当前 `main` 基线依次以 `--no-ff --no-commit` 合入 `v0.1.170`、`v0.1.171`，每个正式 tag 建立独立 merge 节点；不合入 `v0.1.171` 之后的 `upstream/main` 提交。
- 首次 merge 前重新获取 upstream refs，固定两个 tag 的 peeled SHA、严格祖先链、changed-files × 本地能力矩阵和冲突台账；若出现更高正式 tag，暂停并更新 change 范围，不静默扩展。
- 每段分别完成冲突语义融合、聚焦回归修复、能力级审查和本机质量门禁，再进入下一段；merge commit 只承载上游树和完成 merge 所必需的冲突融合，兼容修复使用独立提交。
- 专项保护本地 scheduler/sticky/fallback、OpenAI HTTP/WS 和 usage、请求体 spooling/释放、alpha-search/composite 路由、统一 prompt/security audit、subscription quota cycle reset、settings 热更新、前端定制、生成代码和 migration identity。
- 审查 `v0.1.170` 的分组利润控制、账号倍率同步、流式用量结算、内容审核代理和订阅窗口修复，与本地调度、计费、审计和额度语义的交互。
- 审查 `v0.1.171` 的验证码服务商、Codex 出站身份与版本同步、过载重试、退款/用量落库、composite 推理强度、WebSocket 租约和 prompt audit 修复，与本地能力的交互。
- 保留上游 `192_group_profit_control.sql`、`193_group_profit_control_auth_cache_invalidation.sql` 和本地已发布 `192_subscription_cache_invalidation_outbox.sql` 的完整文件名与既有身份，不因数字前缀重复而重命名历史 migration。
- 每段及最终阶段运行受影响能力聚焦测试、`make test`、`make build`、两轮 backend generate 稳定性和静态冲突检查；本机 Docker/Testcontainers 可用时运行 integration，不可用时如实记录未验证契约与残余风险。
- 两段全部闭合后将 `backend/cmd/server/VERSION` 更新为 `0.1.171.1`，并验证两个目标 tag 都是结果 HEAD 的祖先且 merge 第二父精确匹配固定 tag SHA。
- 不推送、不打 tag、不触发 GitHub Actions Release、不发布或构建镜像、不部署，也不操作任何服务器、数据库、Redis 或 Nginx；后续发布与部署需用户另行明确授权。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `upstream-release-sync`: 将分段同步目标推进到 `v0.1.170` 和 `v0.1.171`，增加同号 `192_*` migration 共存、利润控制/倍率同步、Codex 身份/过载策略、验证码与本地能力交互的专项验收要求。

## Impact

- Git 历史：新增两个按正式 tag 顺序形成的 `--no-ff` merge 节点，以及必要的聚焦兼容修复、验证证据和最终版本提交。
- 后端：调度与利润门禁、上游倍率探测、OpenAI/Anthropic/Gemini/Grok/Antigravity 网关与计费、Codex 身份、验证码/auth、prompt/security audit、订阅与退款、settings/repository、WebSocket 和请求体生命周期。
- 前端：安全与认证设置、账号管理、分组利润控制、退款与用量展示、模型广场及既有本地菜单、设置、订阅、渠道和移动端定制。
- 数据与配置：新增上游 `192`/`193` migration、Ent/Wire 生成物、Go/pnpm 依赖、CSP 与 deploy 配置；本 change 不执行生产 migration 或部署。
- 验证边界：完整本机自动门禁和能力级 review；Docker-backed integration 仅在本机环境可用时执行，未执行项必须进入最终报告的残余风险。
