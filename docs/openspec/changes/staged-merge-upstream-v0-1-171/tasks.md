## 1. 阶段 0：固定基线与保护面

- [x] 1.1 在 Comet 确认的隔离位置记录不可变 source base、execution base、`VERSION=0.1.169.3`，以及 `v0.1.170`、`v0.1.171` 的 peeled SHA 与严格祖先链；除当前 change 产物和 Comet runtime selection 外的 dirty path 必须阻塞
- [x] 1.2 重新获取 upstream refs，确认 `v0.1.171` 仍为最新正式 tag，记录 tag 后 `upstream/main` 提交并排除出范围；若出现更高 tag，返回 OpenSpec 更新范围
- [x] 1.3 建立两段 changed-files × 本地能力矩阵与冲突台账，覆盖 scheduler/sticky/fallback、网关 HTTP/WS 与 usage、请求体生命周期、alpha-search/composite 路由、prompt audit、subscription quota cycle reset、settings、前端、生成物和 migrations
- [x] 1.4 在当前本地基线上运行聚焦保护测试、`make test`、`make build`、两轮 backend generate 与静态检查；为命中但缺少断言的高风险本地能力补最小保护测试
- [x] 1.5 检查本机 Docker/Testcontainers；可用时运行基线 integration，不可用时记录环境证据、未验证契约和残余风险，且不使用远程服务器补跑

## 2. 分段合入 v0.1.170

- [x] 2.1 使用 `git merge --no-ff --no-commit v0.1.170`，逐文件语义融合实际冲突并创建第二父为固定 tag SHA 的 merge commit；merge commit 不混入后续兼容修复
- [x] 2.2 审查分组利润控制、账号倍率同步、槽位后二次复核和请求级定价时刻，与本地 advanced/layered scheduler、sticky、fallback/WaitPlan、DB recheck、usage billing 和倍率语义的交互；以失败测试驱动必要的最小兼容修复
- [x] 2.3 审查 Anthropic 流式用量、OpenAI WS/流内错误、Responses 工具输出、内容审核代理/最新输入、订阅窗口和 settings 更新，与本地 request-body spooling、统一审计、quota reset/outbox 和前端定制的交互
- [x] 2.4 保留上游 `192_group_profit_control.sql`、`193_group_profit_control_auth_cache_invalidation.sql` 与本地 `192_subscription_cache_invalidation_outbox.sql`，从 schema/provider 源重新生成 Ent/Wire，并验证完整文件名、排序和 checksum
- [x] 2.5 运行 v0.1.170 聚焦测试、本机 full 门禁及适用的本机 integration，关闭能力矩阵 gap 并记录阶段证据后再进入下一 tag

## 3. 分段合入 v0.1.171

- [x] 3.1 使用 `git merge --no-ff --no-commit v0.1.171`，逐文件语义融合实际冲突并创建第二父为固定 tag SHA 的 merge commit；merge commit 不混入后续兼容修复
- [x] 3.2 审查 Codex 出站身份归一化、动态版本同步、账号级自定义 UA 和流内过载有界重试，与本地 HTTP/透传/WS/探针/模型列表/alpha-search、请求体释放、sticky/failover 和错误响应语义的交互
- [x] 3.3 审查腾讯天御/阿里云验证码、认证入口、settings/CSP/前端卡片，以及退款事务、用量失败落库、composite 推理强度、订阅续期、WebSocket 租约和 prompt audit 修复，与本地能力的交互；以失败测试驱动最小兼容修复
- [x] 3.4 运行 v0.1.171 聚焦测试、本机 full 门禁及适用的本机 integration，关闭能力矩阵 gap 并记录阶段证据

## 4. v0.1.171 历史版本与验证（已完成，现被范围扩展取代）

- [x] 4.1 两段全部闭合后将 `backend/cmd/server/VERSION` 一次更新为 `0.1.171.1`，不创建中间过程版本
- [x] 4.2 在最终 source HEAD 重跑全部能力聚焦测试、`make test`、`make build`、两轮 backend generate、静态冲突、unmerged index 与 whitespace 检查
- [x] 4.3 校验两个正式 tag 均为结果 HEAD 祖先、两个 merge 第二父正确，双方 `191_*`、双方 `192_*` 与上游 `193_*` migration 均保留，并记录本机 integration 实际结果或未验证风险
- [x] 4.4 完成本地能力专项 review 与最终验证报告，明确本 change 未推送、未发版、未部署、未操作服务器；发布与部署等待用户另行明确授权

## 5. 已验证未归档状态扩展到 v0.1.172

- [x] 5.1 保留 170/171 已完成任务与 Verify 报告作为历史证据，使旧 Verify 对新增范围失效；重新 fetch upstream refs，固定 `v0.1.172`/`v0.1.173` annotated object 与 peeled SHA、173 为最新正式 tag、严格祖先链、172 的 208/113 和 173 的 352/初步 138 文件面
- [x] 5.2 使用 `git merge --no-ff --no-commit v0.1.172`，逐文件语义融合实际冲突并创建第二父为固定 `155c494964c3ea6ecc31f52679525c1034bf0f16` 的纯 merge commit
- [x] 5.3 以 TDD 审查 OAuth pending 账号接管修复、腾讯验证码 region/ticket/CSP 与本地 Turnstile/Tencent/Aliyun 互斥 provider、OAuth/passkey 和前端 challenge 生命周期的交互
- [x] 5.4 以 TDD 审查金额量化、订阅/usage persistence 与本地 quota receipt/outbox/cache；明确保留新购及用户/管理员手动重置的实际操作时刻锚点和后续 24 小时滚动窗口
- [x] 5.5 以 TDD 审查 upstream response model audit、Codex identity/capacity failover、transport timeout、body replay/release、sticky/final account、WS prewarm、count_tokens、Grok、图片 cooldown 和协议清洗
- [x] 5.6 融合 UsageLog schema/Ent、194/195 migration、单条/批量/best-effort insert、查询筛选和管理端展示，并审查模型广场、错误时间范围及既有本地 frontend 定制
- [x] 5.7 运行 v0.1.172 全部能力聚焦测试、`make test`、版本锁定 build、backend/frontend lint、typecheck、两轮 generate、静态冲突与适用的本机 integration，关闭能力矩阵 gap
- [x] 5.8 保持中间 VERSION `0.1.171.1`，记录 172 阶段门禁、第三个 merge 第二父和 191/192/193/194/195 migration identity，关闭第三阶段能力矩阵后才允许进入 173

## 6. 分段合入 v0.1.173

- [x] 6.1 使用 `git merge --no-ff --no-commit v0.1.173`，逐文件语义融合实际冲突并创建第二父为固定 `29009f0b2ea14edf3b11ae2564fb617ff91a03b4` 的纯 merge commit
- [x] 6.2 以 TDD 融合 Grok SSO/refresh-token、默认文本模型和运行时映射；缺失/未配置的跨客户端映射默认关闭，显式开启与账号显式映射有效，邮箱密码授权 UI 隐藏且服务端硬拒绝
- [x] 6.3 以 TDD 审查 Grok 图片/视频、Voice TTS/STT/Realtime、custom voices、web search 与本地 routing、sticky/failover、body 生命周期、审计和单次计费语义
- [x] 6.4 以 TDD 审查 Grok free 24h 软门禁、team+model 冷却、流式空闲换号、7d/30d 调度阈值、routing hints 与本地 scheduler/usage/settings 热更新；不得改写订阅实际时刻额度锚点
- [ ] 6.5 融合 Channel Monitor V2 被动聚合、V1/V2 互斥开关、普通用户吞吐脱敏、admin 完整指标、rollup/cache/API/UI；默认保持 V1，V2 仅显式启用
- [ ] 6.6 融合 Grok 媒体/Voice/search 定价 schema、Ent、前端价矩阵和 173 的 Channel Monitor 194-206、pricing 217-220 migrations；保留 172 同号 UsageLog 194/195，验证 migration 220 先备份且只清理非 Grok/非 composite 视频价格残值
- [ ] 6.7 运行 v0.1.173 全部能力聚焦测试、`make test`、中间版本 build、backend/frontend lint、typecheck、两轮 generate、静态冲突与适用的本机 integration，关闭第四阶段能力矩阵 gap
- [ ] 6.8 四段全部闭合后将 VERSION 更新为 `0.1.173.1`，校验四个 tag 祖先与 merge 第二父、全部 24 个受保护 migration identity，完成 thorough review 和新的最终 Verify 报告
