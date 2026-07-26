## Why

截至 2026-07-26，本地主线已包含上游 `v0.1.159`，`backend/cmd/server/VERSION` 为 `0.1.159.6`，并叠加长期本地定制；上游已发布 `v0.1.160` ~ `v0.1.165` 六个后续 release tag，覆盖 security-audit prompt 审计、客户端 IP 请求头体系、composite group routing、OpenAI Live gateway 等大量新能力与修复（每段 24~114 commits、133~257 文件，并新增 12 个 SQL migration 文件）。需要按既有 staged-merge 经验逐版本合入，在不回归本地定制能力的前提下吸收上游演进。

## What Changes

- 从 `main`（`0.1.159.6`）切出临时合并分支，依次使用 `--no-ff` 合入 `v0.1.160`、`v0.1.161`、`v0.1.162`、`v0.1.163`、`v0.1.164`、`v0.1.165` 六个上游 tag。
- 实施前重新 fetch upstream tags 并确认 `v0.1.165` 仍是最新正式 release；若出现更新 tag，暂停并更新 change 范围，不静默跳过。当前 `upstream/main` 仅比 `v0.1.165` 多一个 `VERSION` 同步提交，不作为 release 合并目标。
- 每段合并后运行 **full 门禁**（根目录 `make test` + `make build`、Docker-backed integration、Ent/Wire 两次生成稳定性、migration 集合兼容性、冲突标记检查），并按本地能力矩阵做能力级审查；全部通过后才进入下一段。
- 冲突处理优先"保留上游变更 + 保留本地定制"共存；真实语义冲突无法共存时暂停等待用户决定。
- 对无文本冲突但可能语义覆盖的重点区域（scheduler/sticky、image capability、privacy、runtime 热更新、composite routing 与本地调度定制的交互、OpenAI Live 与本地 OpenAI 定制的交互）做专项 review。
- **BREAKING** 延续既有决定：本地 `openai-first-token-timeout` 契约保持已移除状态，后续 tag 不得恢复旧实现或兼容别名。
- 最终将 `backend/cmd/server/VERSION` 规范为 `0.1.165.1` 并完成 full verify；**不包含**推送、打 tag、发版或部署（验证通过归档后另行发版）。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `upstream-release-sync`: 将分段合并推进到 `v0.1.160`~`v0.1.165` 六个正式 tag；每段从"轻门禁"升级为"full 门禁"（`make test` + `make build` + Docker-backed integration + 生成稳定性/migration 兼容性检查），最终仍需整体 full verify。

## Impact

- Git 历史：新增 6 个按顺序关联正式 tag 的 `--no-ff` merge 节点，以及必要的本地兼容修复提交。
- 后端：security-audit prompt 审计、客户端 IP 请求头解析与可信代理、模型级临时冷却、Grok media/视频代理与 client tool 缓存、composite group routing、ollama Cloud 用量、OpenAI Live gateway、email alias 注册查重、billing 计费口径调整、优雅关停 Cleanup、golang.org/x 依赖升级。
- 前端：客户端 IP 设置界面、step-up 2FA 开关、S3 备份/image storage 配置卡、移动端与 iOS 适配、审计 UI、Alipay deep link、axios/postcss 安全升级。
- 数据库：新增 12 个 migration 文件（172、181-190，其中有两个 `186_*`）；上游 `172_composite_model_routes.sql`、`181_prompt_audit.sql` 分别与本地既有 `172_video_per_second_billing_metadata.sql`、`181_group_duplicate_operation_id.sql` 同号不同名，必须验证按完整文件名记录的迁移执行器在新库和升级库上的行为，不得仅检查编号连续性。
- 本地保护清单：以主 spec 的本地关键能力专项 review 清单为下限，并重点覆盖 advanced/layered scheduler、fallback/WaitPlan 与 DB recheck、Grok/platform sticky、privacy 与 image capability、runtime 热更新、OpenAI prompt cache reuse、网关透传、body replay/request spooling、公开分组屏蔽、用户菜单隐藏、admin 资源控制、前端翻译、subscription quota 原子重置、settings JSON backfill、local test gates。
- 验证：6 段独立 full 门禁记录 + 最终 full verify 报告。
