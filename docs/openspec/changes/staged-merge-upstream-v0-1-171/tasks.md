## 1. 阶段 0：固定基线与保护面

- [x] 1.1 在 Comet 确认的隔离位置记录不可变 source base、execution base、`VERSION=0.1.169.3`，以及 `v0.1.170`、`v0.1.171` 的 peeled SHA 与严格祖先链；除当前 change 产物和 Comet runtime selection 外的 dirty path 必须阻塞
- [x] 1.2 重新获取 upstream refs，确认 `v0.1.171` 仍为最新正式 tag，记录 tag 后 `upstream/main` 提交并排除出范围；若出现更高 tag，返回 OpenSpec 更新范围
- [x] 1.3 建立两段 changed-files × 本地能力矩阵与冲突台账，覆盖 scheduler/sticky/fallback、网关 HTTP/WS 与 usage、请求体生命周期、alpha-search/composite 路由、prompt audit、subscription quota cycle reset、settings、前端、生成物和 migrations
- [ ] 1.4 在当前本地基线上运行聚焦保护测试、`make test`、`make build`、两轮 backend generate 与静态检查；为命中但缺少断言的高风险本地能力补最小保护测试
- [ ] 1.5 检查本机 Docker/Testcontainers；可用时运行基线 integration，不可用时记录环境证据、未验证契约和残余风险，且不使用远程服务器补跑

## 2. 分段合入 v0.1.170

- [ ] 2.1 使用 `git merge --no-ff --no-commit v0.1.170`，逐文件语义融合实际冲突并创建第二父为固定 tag SHA 的 merge commit；merge commit 不混入后续兼容修复
- [ ] 2.2 审查分组利润控制、账号倍率同步、槽位后二次复核和请求级定价时刻，与本地 advanced/layered scheduler、sticky、fallback/WaitPlan、DB recheck、usage billing 和倍率语义的交互；以失败测试驱动必要的最小兼容修复
- [ ] 2.3 审查 Anthropic 流式用量、OpenAI WS/流内错误、Responses 工具输出、内容审核代理/最新输入、订阅窗口和 settings 更新，与本地 request-body spooling、统一审计、quota reset/outbox 和前端定制的交互
- [ ] 2.4 保留上游 `192_group_profit_control.sql`、`193_group_profit_control_auth_cache_invalidation.sql` 与本地 `192_subscription_cache_invalidation_outbox.sql`，从 schema/provider 源重新生成 Ent/Wire，并验证完整文件名、排序和 checksum
- [ ] 2.5 运行 v0.1.170 聚焦测试、本机 full 门禁及适用的本机 integration，关闭能力矩阵 gap 并记录阶段证据后再进入下一 tag

## 3. 分段合入 v0.1.171

- [ ] 3.1 使用 `git merge --no-ff --no-commit v0.1.171`，逐文件语义融合实际冲突并创建第二父为固定 tag SHA 的 merge commit；merge commit 不混入后续兼容修复
- [ ] 3.2 审查 Codex 出站身份归一化、动态版本同步、账号级自定义 UA 和流内过载有界重试，与本地 HTTP/透传/WS/探针/模型列表/alpha-search、请求体释放、sticky/failover 和错误响应语义的交互
- [ ] 3.3 审查腾讯天御/阿里云验证码、认证入口、settings/CSP/前端卡片，以及退款事务、用量失败落库、composite 推理强度、订阅续期、WebSocket 租约和 prompt audit 修复，与本地能力的交互；以失败测试驱动最小兼容修复
- [ ] 3.4 运行 v0.1.171 聚焦测试、本机 full 门禁及适用的本机 integration，关闭能力矩阵 gap 并记录阶段证据

## 4. 最终版本与验证

- [ ] 4.1 两段全部闭合后将 `backend/cmd/server/VERSION` 一次更新为 `0.1.171.1`，不创建中间过程版本
- [ ] 4.2 在最终 source HEAD 重跑全部能力聚焦测试、`make test`、`make build`、两轮 backend generate、静态冲突、unmerged index 与 whitespace 检查
- [ ] 4.3 校验两个正式 tag 均为结果 HEAD 祖先、两个 merge 第二父正确，双方 `191_*`、双方 `192_*` 与上游 `193_*` migration 均保留，并记录本机 integration 实际结果或未验证风险
- [ ] 4.4 完成本地能力专项 review 与最终验证报告，明确本 change 未推送、未发版、未部署、未操作服务器；发布与部署等待用户另行明确授权
