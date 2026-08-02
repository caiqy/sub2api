## Why

当前主线运行版本为 `0.1.165.4`，上游已发布 `v0.1.166`、`v0.1.168`、`v0.1.169` 三个后续正式 tag；其中 `v0.1.169` 修复 GHSA-vrxq-qm4h-6hgg，受影响版本覆盖当前基线。需要沿用此前分段合并方式吸收全部上游演进，同时保护本地调度、网关、审计、订阅额度和前端定制。

## What Changes

- 从当前 `main`（`0.1.165.4`）创建隔离实施位置，依次以 `--no-ff --no-commit` 合入 `v0.1.166`、`v0.1.168`、`v0.1.169`；`v0.1.167` 没有正式 tag，不建立独立阶段。
- 实施前重新获取 upstream refs，固定三个 tag 的 peeled SHA 与严格祖先链；若出现比 `v0.1.169` 更新的正式 tag，暂停并更新 change 范围，不静默扩展，也不合入 tag 之后的 `upstream/main` 提交。
- 每段建立 changed-files × 本地能力矩阵、冲突台账和阶段门禁；冲突默认融合上游修复与本地定制，无法共存的业务语义交由用户决定。
- 专项保护 GHSA 路径校验、prompt/security audit、OpenAI 网关与 WebSocket 计费、scheduler/sticky/fallback、请求体生命周期、settings 热更新、subscription quota reset、migration、生成代码和前端本地能力。
- 每段及最终阶段运行本机 `make test`、`make build`、聚焦测试、两轮 Ent/Wire 生成稳定性、静态冲突检查；本机 Docker 可用时运行 Docker-backed integration，不可用时明确记录未验证项和残余风险，不使用远程服务器补跑。
- 最终将 `backend/cmd/server/VERSION` 更新为 `0.1.169.1`，并验证三个正式 tag 均为结果 HEAD 的祖先。
- 不推送、不打 tag、不触发 GitHub Actions Release、不发布或构建镜像、不部署、不操作任何服务器，也不移除生产 Nginx 临时盾。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `upstream-release-sync`: 将分段同步目标推进到 `v0.1.166`、`v0.1.168`、`v0.1.169`，并为用户明确选择的本机验证模式规定 integration 可用性与残余风险记录方式。

## Impact

- Git 历史：新增 3 个按正式 tag 顺序形成的 `--no-ff` merge 节点，以及必要的聚焦兼容修复与验证证据提交。
- 后端：面板 API 限流、Passkey、模型广场、OpenAI/Anthropic/Gemini 兼容与计费、安全审计、代理断流熔断、上游 URL 路径校验、settings/repository 更新语义和 token refresh。
- 前端：Passkey、模型广场、订阅与渠道展示、settings、移动端及相关本地页面定制。
- 数据与部署文件：新增 `191_passkey_credentials.sql`，更新 release 资源打包、Docker/Caddy/compose 安全配置及依赖锁文件；不执行生产 migration 或部署。
- 本地保护面：继承既有 `upstream-release-sync` 能力清单，并增加本地 subscription quota cycle reset、统一 prompt audit、安全路径校验与 `v0.1.169` 代理断流策略的专项交互审查。
- 验证边界：完整本机自动门禁和能力级 review；Docker-backed integration 仅在本机环境可用时执行，未执行项必须进入最终报告的残余风险。
