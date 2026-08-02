## 1. 阶段 0：固定基线与保护面

- [x] 1.1 在 Comet 确认的隔离位置记录不可变 source base、仅允许包含当前 change 规划产物的 execution base、`VERSION=0.1.165.4`，以及 `v0.1.166`、`v0.1.168`、`v0.1.169` 的 peeled SHA 和严格祖先链；除 Comet runtime selection 外的 dirty path 必须阻塞
- [x] 1.2 重新获取 upstream refs，确认 `v0.1.169` 仍为最新正式 tag，记录 tag 后 `upstream/main` 提交并排除出范围
- [x] 1.3 建立三段 changed-files × 本地能力矩阵与冲突台账，覆盖 scheduler/sticky/fallback、网关与 WebSocket、prompt audit、请求体生命周期、subscription quota cycle reset、前端、生成物和 migrations
- [ ] 1.4 在当前本地基线上运行聚焦保护测试、`make test`、`make build`、两轮 backend generate 与静态检查；为命中但缺少断言的高风险本地能力补最小保护测试
- [ ] 1.5 检查本机 Docker/Testcontainers；可用时运行基线 integration，不可用时记录环境证据、未验证契约和残余风险，且不使用远程服务器补跑

## 2. 分段合入 v0.1.166

- [ ] 2.1 使用 `git merge --no-ff --no-commit v0.1.166`，逐文件融合冲突并创建第二父为固定 tag SHA 的 merge commit
- [ ] 2.2 审查面板 API 限流、settings 部分更新、WebSocket 每轮模型计费、composite routing 与本地调度/usage/网关定制的交互；以失败测试驱动必要的最小兼容修复
- [ ] 2.3 运行 v0.1.166 聚焦测试、本机 full 门禁及适用的本机 integration，关闭能力矩阵 gap 并记录阶段证据

## 3. 分段合入 v0.1.168

- [ ] 3.1 使用 `git merge --no-ff --no-commit v0.1.168`，逐文件融合冲突并创建第二父为固定 tag SHA 的 merge commit
- [ ] 3.2 审查 Passkey、模型广场、repository scoped updates、prompt audit 配置恢复、OpenAI Live store 容错与本地功能交互；保留 `191_passkey_credentials.sql` 和本地 `191_subscription_quota_advance_receipts.sql`
- [ ] 3.3 运行 v0.1.168 聚焦测试、本机 full 门禁及适用的本机 integration，验证或明确记录双方 191 migration 的新库/升级库风险并关闭阶段证据

## 4. 分段合入 v0.1.169

- [ ] 4.1 使用 `git merge --no-ff --no-commit v0.1.169`，逐文件融合冲突并创建第二父为固定 tag SHA 的 merge commit
- [ ] 4.2 审查 GHSA-vrxq-qm4h-6hgg 路径片段闭集校验、代理断流熔断 fail-open、release 资源、Qwen3Guard、count_tokens、pricing 与本地网关/审计/调度定制的交互
- [ ] 4.3 运行 v0.1.169 安全与行为聚焦测试、本机 full 门禁及适用的本机 integration，关闭能力矩阵 gap 并记录阶段证据

## 5. 最终版本与验证

- [ ] 5.1 三段全部闭合后将 `backend/cmd/server/VERSION` 一次更新为 `0.1.169.1`，不创建中间过程版本
- [ ] 5.2 在最终 source HEAD 重跑全部能力聚焦测试、`make test`、`make build`、两轮 backend generate、静态冲突与 whitespace 检查
- [ ] 5.3 校验三个正式 tag 均为结果 HEAD 祖先、三个 merge 第二父正确、双方 191 与本地 192 migration 均保留，并记录本机 integration 实际结果或未验证风险
- [ ] 5.4 完成本地能力专项 review 与最终验证报告，明确本 change 未推送、未发版、未部署、未操作服务器，生产 Nginx 临时盾仍保留
