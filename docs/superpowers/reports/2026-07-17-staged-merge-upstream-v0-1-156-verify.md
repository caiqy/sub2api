# 分段合并上游 v0.1.156 验证报告

## 结论

`staged-merge-upstream-v0-1-156` 完整验证通过，无 CRITICAL 或 IMPORTANT 问题。正式实施证据见 [分段合并验证报告](./2026-07-16-staged-merge-upstream-v0-1-156-validation.md)。

| 维度 | 结果 |
| --- | --- |
| 完整性 | OpenSpec `23/23`；upstream-sync `4/4` requirements、`10/10` scenarios；first-token `7/7` removed requirements |
| 正确性 | `11 protected / 4 manual / 1 approved-removal / 0 gap` |
| 一致性 | OpenSpec design 与技术设计一致，无 design drift |
| Git 拓扑 | 四个 `--no-ff` merge 顺序和固定 second parent 均正确；`upstream/main` 不是结果祖先 |
| 安全与边界 | 未发现新增安全问题；bounded first-output staging 与 spool 清理仍受保护 |

## Fresh 验证

- `make test`：exit `0`；后端默认/unit/lint 与前端 ESLint/typecheck/Vitest 通过，Vitest `181` files、`1405` tests。
- `pnpm --dir frontend run build`：exit `0`；Vite `970` modules。
- `make -C backend generate` 后 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go`：exit `0`。
- `git diff --check`、cached diff、unmerged index：clean。
- OpenSpec checked count：`23`；OpenSpec 与实施计划 unchecked count：`0`。
- Verify 阶段独立 full review：PASS，无 CRITICAL/IMPORTANT；无 spec/design drift。

## Requirement 与设计核对

- 阶段 0 在首个 merge 前完成基线、能力矩阵与必要回归保护。
- `v0.1.152`、`v0.1.153`、`v0.1.155`、`v0.1.156` 形成四个顺序独立 merge；回归修复均为后续普通提交。
- 唯一批准移除为本地 first-token timeout 全链路；旧配置/API/UI/错误/日志/usage/watchdog/兼容别名均已退场。
- 保留 native HTTP first-output 默认关闭、高 effort override、`first_output_timeout`、failover、`HandleStreamTimeout`，以及客户端 WebSocket first-message/read/write timeout。
- scheduler、Sticky、fallback/WaitPlan、DB recheck、协议转换与透传、privacy、image capability、runtime hot reload、请求体生命周期、用户资源控制与前端本地能力均有自动或人工证据。
- `VERSION=0.1.156.1`；Ent/Wire 生成稳定；migration 顺序、重复编号与 checksum compatibility 已复核。

## 残余与未执行项

- Docker-backed migration integration 因本机无 Docker 未执行，不记为 PASS；部署前必须在 Docker CI 运行。
- `backend/internal/service/openai_gateway_grok_test.go:2159` 的 EOF whitespace warning 继承自 `v0.1.156`。
- Windows root `make build` 存在非门禁兼容限制；要求的 `make test` 与 frontend build 已通过。
- 未执行 push、release、deploy、merge-to-main 或 archive。

## 分支处置

用户选择保留 `feature/20260716/staged-merge-upstream-v0-1-156` 原状，稍后处理；未合并、未推送、未删除分支或工作区。
