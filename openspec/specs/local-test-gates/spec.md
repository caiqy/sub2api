# local-test-gates Specification

## Purpose
定义无需 Docker 的本地代码质量门禁、重复与并发测试稳定性及跨平台资源清理要求，确保开发工作站在合并前获得一致且可复现的验证结果。
## Requirements
### Requirement: 本地代码质量门禁
仓库 MUST 使以下本地验证命令在不依赖 Docker 的开发环境中通过：后端默认测试、后端 unit 测试、后端 `golangci-lint`、前端 ESLint、前端 TypeScript 检查和前端全量 Vitest。integration/e2e 不属于本地代码质量门禁。

#### Scenario: 完整本地验证
- **WHEN** 开发者在依赖已安装的本地工作区执行定义的后端与前端验证命令
- **THEN** 所有命令以退出码 0 结束，且后端 unit 测试不会因测试 fixture 接口缺口而编译失败

#### Scenario: 不使用 Docker 的验证环境
- **WHEN** 开发者的工作站未安装 Docker
- **THEN** 本地代码质量门禁仍可完整执行，且不会要求启动 integration/e2e

### Requirement: 测试协议断言与资源清理
测试 MUST 断言 HTTP 协议语义而非 header 序列化大小写格式；request body 临时 spool 文件的清理测试 MUST 在重复和并行执行时具有稳定结果。

#### Scenario: HTTP header canonicalization
- **WHEN** 上游请求 header 由 Go 的 `http.Header` canonicalize
- **THEN** 相关测试仍验证关键 header 值和 content type，而不因名称大小写变化失败

#### Scenario: 并行 spool 清理
- **WHEN** request body spool cleanup 测试与同 package 的其他测试并行执行
- **THEN** 测试能够区分自身创建的临时文件，并稳定验证正确的清理生命周期
