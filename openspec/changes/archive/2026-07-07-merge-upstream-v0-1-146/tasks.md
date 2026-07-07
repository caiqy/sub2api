## 1. 合并准备

- [x] 1.1 获取 upstream 分支和 tags，确认 `v0.1.146` 可用且当前 `main` 干净。
- [x] 1.2 从当前 `main` 创建临时合并分支。

## 2. 执行上游合并

- [x] 2.1 合并 upstream `v0.1.146` 到临时分支。
- [x] 2.2 处理冲突，优先保留上游更新和本地定制；不可共存语义暂停确认。
- [x] 2.3 复核版本、生成文件和配置差异，避免发布或运行元数据被误改。

## 3. 验证与语义 review

- [x] 3.1 运行 `go test ./...` 并修复合并引入的问题。
- [x] 3.2 运行前端 typecheck、build 和单测，修复合并引入的问题。
- [x] 3.3 专项 review scheduler、sticky、privacy、image capability、runtime setting 热更新和网关透传字段。

## 4. 收尾决策

- [x] 4.1 汇总合并结果、冲突处理、验证输出和本地关键能力 review 结论。
- [x] 4.2 让用户确认是否合回 `main`、是否推送远端。

## 执行摘要

- 目标 tag：`upstream-v0.1.146` -> `d7a6a4513a58b082922dfb8bd80f36cbe6b8a4c4`。
- 隔离分支：`feature/20260707/merge-upstream-v0-1-146`。
- 冲突处理：已融合 upstream 更新和本地 scheduler/sticky/privacy/image/runtime setting/passthrough 定制；未发现需要暂停确认的不可共存语义。
- 版本/生成/配置/migration：`backend/cmd/server/VERSION` 保留 upstream tag 内的 `0.1.145`；无 `go.mod/go.sum`、Ent、migration 差异；`wire_gen.go` 与新构造依赖一致；部署配置保留 upstream 新增 setup timeout 和 URL allowlist 默认说明。
- 验证：`go test ./...` PASS；`pnpm typecheck` PASS；`pnpm build` PASS（仅 Vite chunk/Browserslist 警告）；`pnpm test:run` PASS（157 files / 1175 tests）。
- 专项 review：scheduler/sticky/privacy/image capability/runtime setting 热更新/网关透传字段均通过；review 发现的 Responses capability、Grok `/v1/responses` platform 透传测试缺口和 layered sticky-weighted TopK 偏好语义问题已修复并补回归测试。
- 收尾：用户已确认合回 `main`；已提交 `docs: record full release workflow`，本地 `main` 已合并该分支并删除临时分支，`backend/cmd/server/VERSION` 保持 `0.1.146.1`。
