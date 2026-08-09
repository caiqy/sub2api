## Why

当前 change 已完成 `v0.1.170`、`v0.1.171` 合并及 Verify，但尚未归档；上游随后发布正式 tag `v0.1.172` 和 `v0.1.173`。172 相对 171 新增 54 commits / 208 files，其中 113 个路径与当前 fork 相对 171 的本地演进重叠；173 相对 172 新增 120 commits / 352 files，初步与当前 fork 重叠 138 个路径。早期 `300 files` 来自 GitHub Compare API 的 files 数组上限，完整本地 Git tree diff 是范围事实源。两个 tag 包含 OAuth 账号接管高危修复、Grok/xAI 与 Channel Monitor V2、网关/计费/订阅语义变化和连续 schema/migration，因此旧 Verify 对扩展后的最终范围失效，必须追加第三、第四个受审 release 阶段。

## What Changes

- 从固定的当前 `main` 基线依次以 `--no-ff --no-commit` 合入 `v0.1.170`、`v0.1.171`、`v0.1.172`、`v0.1.173`，每个正式 tag 建立独立 merge 节点；不合入 `v0.1.173` 之后的 `upstream/main` 提交。
- 172 merge 前重新获取 upstream refs，固定四个 tag 的 annotated object、peeled SHA、严格祖先链、changed-files × 本地能力矩阵和冲突台账；若出现更高正式 tag，暂停并更新 change 范围，不静默扩展。
- 每段分别完成冲突语义融合、聚焦回归修复、能力级审查和本机质量门禁，再进入下一段；merge commit 只承载上游树和完成 merge 所必需的冲突融合，兼容修复使用独立提交。
- 专项保护本地 scheduler/sticky/fallback、OpenAI HTTP/WS 和 usage、请求体 spooling/释放、alpha-search/composite 路由、统一 prompt/security audit、subscription quota cycle reset、settings 热更新、前端定制、生成代码和 migration identity。
- 审查 `v0.1.170` 的分组利润控制、账号倍率同步、流式用量结算、内容审核代理和订阅窗口修复，与本地调度、计费、审计和额度语义的交互。
- 审查 `v0.1.171` 的验证码服务商、Codex 出站身份与版本同步、过载重试、退款/用量落库、composite 推理强度、WebSocket 租约和 prompt audit 修复，与本地能力的交互。
- 审查 `v0.1.172` 的 OAuth pending 账号接管修复、upstream response model audit、Codex/capacity failover、transport timeout、协议清洗、计费量化、腾讯验证码区域适配和前端修复；保留本地订阅日窗口的实际操作时刻锚点，不接受上游 midnight 滚动语义。
- 审查 `v0.1.173` 的 Grok SSO/refresh-token、模型映射、媒体/Voice/search、free 24h 软门禁、调度阈值、Channel Monitor V2、定价矩阵和前端设置；跨客户端模型映射默认关闭，密码授权硬禁用，Channel Monitor 默认 V1 且 V2 显式切换。
- 保留既有双方 `191_*`、双方 `192_*`、上游 `193_*`、172 的 UsageLog `194_*`/`195_*`，并新增 173 同号的 Channel Monitor `194_*`/`195_*`、`196_*` 至 `206_*`、`217_*` 至 `220_*` 的完整文件名与既有身份，不重命名历史 migration。
- 每段及最终阶段运行受影响能力聚焦测试、`make test`、`make build`、两轮 backend generate 稳定性和静态冲突检查；本机 Docker/Testcontainers 可用时运行 integration，不可用时如实记录未验证契约与残余风险。
- 四段全部闭合后将 `backend/cmd/server/VERSION` 更新为 `0.1.173.1`，并验证四个目标 tag 都是结果 HEAD 的祖先且 merge 第二父精确匹配固定 tag SHA。
- 不推送、不打 tag、不触发 GitHub Actions Release、不发布或构建镜像、不部署，也不操作任何服务器、数据库、Redis 或 Nginx；后续发布与部署需用户另行明确授权。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `upstream-release-sync`: 将分段同步目标继续推进到 `v0.1.173`，除 172 的 OAuth pending 安全、响应模型审计和实际时刻额度锚点外，增加 Grok 授权/映射/媒体/调度、Channel Monitor V2、同号 194/195 与 196-206/217-220 migration 的专项验收要求。

## Impact

- Git 历史：在已完成的两个 merge 节点后新增 `v0.1.172`、`v0.1.173` 两个 `--no-ff` merge 节点，以及必要的聚焦兼容修复、验证证据和最终版本提交。
- 后端：既有调度/网关/审计/订阅保护面，以及 OAuth pending 安全、响应模型审计、Grok 授权/模型/媒体/Voice/search/free gate、Channel Monitor V2、金额量化、capacity failover、transport timeout、协议清洗和 usage persistence。
- 前端：安全与认证设置、Grok 账号与定价矩阵、Channel Monitor V1/V2、退款与用量展示、模型广场及既有本地菜单、设置、订阅、渠道和移动端定制。
- 数据与配置：保留既有 migration identity，新增上游 UsageLog 194/195、Channel Monitor 194-206、Grok pricing 217-220、Ent 生成物、CSP 与 deploy 配置；本 change 不执行生产 migration 或部署。
- 验证边界：完整本机自动门禁和能力级 review；Docker-backed integration 仅在本机环境可用时执行，未执行项必须进入最终报告的残余风险。
