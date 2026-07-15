---
comet_change: restore-local-test-gates
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-15-restore-local-test-gates
status: final
---

# 恢复本地测试门禁设计

## 目标与边界

本变更让 `make test` 在不依赖 Docker 的工作站完整执行以下检查：

1. `go test ./...`
2. `go test -tags=unit ./...`
3. `golangci-lint run ./...`
4. 前端 `lint:check`、`typecheck` 与全量 `test:run`

integration/e2e、Docker、公开 API、数据库 schema、依赖版本和产品行为均不在范围内。OpenSpec delta spec 是验收标准；本文只说明实现边界和验证方法。

## 当前基线与根因分类

默认 Go 测试和前端全量检查已通过。unit 全量测试存在三类确定性问题和一个待确认的并行问题：

| 域 | 根因 | 处理边界 |
| --- | --- | --- |
| Anthropic failed-usage | 测试将 HTTP header 的序列化名称作为协议契约 | 仅改测试为语义断言，保留关键 header 值检查 |
| Images failover | 所有账号失败后没有稳定写出预期网关错误响应 | 修复 handler 的最终响应路径并保留 failover 行为 |
| server fixture | test stub 落后于 `UserRepository` 接口 | 仅补齐 test stub 的受控实现，不收缩生产接口 |
| request body spool | 仅在并行全套中偶现，测试以共享 temp 和时间窗口推断文件归属 | 先隔离测试资源并复现；没有证据不改生产生命周期 |

lint 报告按依赖边界、资源关闭与 context cancel、无效赋值、静态分析和未使用符号五组处理。现有规则、阈值和 ignore 不变。

## 测试修复设计

### HTTP 协议断言

failed-usage 测试将通过 `http.Header.Get` 或等价的大小写无关查找读取 header。测试继续断言请求携带预期 API key、版本/header 值和 content type；只移除对 Go canonicalization 文本格式的依赖。这样测试失败仍表示协议值缺失或错误，而不是 map key 展示形式改变。

### Images failover 耗尽响应

handler 保持既有账号选择和 failover 顺序。所有候选账号都失败时，最终分支使用已有的网关错误构造/写入路径向 `gin.Context` 输出非空、可识别的错误响应，并返回该错误，不继续尝试写成功结果。测试验证状态码、错误消息和尝试次数，不扩大该端点的外部响应契约。

### Repository test fixture

各 package 私有的 `stubUserRepo` 显式实现当前被测试路径需要的 `GetBlockedGroups` 与 `GetHiddenUIResources`。默认返回空切片或空配置及 `nil` 错误；仅测试需要非空数据的用例在 stub 上提供最小可配置值。生产 `UserRepository` 不因测试编译问题而回退。

### Request body spool 生命周期

spool 测试为每个测试实例使用 `t.TempDir()`，并只观察该目录或当前 handle 记录的路径。验证矩阵为：

1. 单测试重复运行，确认没有残留。
2. 与同 package 测试并行运行，确认测试不会看见其他测试文件。
3. unit 全套运行，确认行为在真实并发下稳定。

若隔离后不再失败，根因是测试共享资源，生产代码不改。若隔离后仍稳定复现残留，沿 `RequestBodyHandle` 的创建者和关闭者追踪唯一所有权：只在拥有文件的 cleanup 路径补充关闭或删除，并添加聚焦回归测试。不得用 sleep、全局 temp 扫描或放宽断言隐藏竞争。

## 静态检查修复设计

每次只处理一个诊断类别，并在该类别完成后运行受影响 package 的测试与完整 linter：

| 类别 | 修复原则 |
| --- | --- |
| `depguard` | 将受限依赖替换为仓库允许的本地封装或已存在的标准库/API；不新增依赖。 |
| `errcheck` | 对资源关闭、写入和关键调用显式处理错误；cleanup 中保留原始业务错误优先级。 |
| `govet` / `staticcheck` | 按诊断含义修正 API 用法、格式化、context 或可疑控制流。 |
| `ineffassign` | 删除无效赋值，确保保留的赋值影响后续结果。 |
| `unused` | 删除无调用者的私有代码、导入和测试辅助项；不为保留代码制造调用点。 |

若某项 lint 修复影响资源所有权、错误返回或并发，必须先写或调整最小回归测试。纯删除和机械替换只复用现有测试。

## 门禁入口

根目录 `Makefile` 是唯一的一键本地入口：

- `test-backend` 继续委托 `backend` 的默认测试与 lint。
- 新增 `test-backend-unit`，委托 `backend` 的 `test-unit`。
- `test-frontend` 保留 ESLint 与 TypeScript，并由现有 critical Vitest 列表改为 `pnpm --dir frontend run test:run`。
- `test` 依次聚合这三项；不引用 `test-integration`、`test-e2e` 或 Docker 命令。

保留 `test-frontend-critical` 作为快速开发命令，避免将它误当作完整门禁。

## 验证策略

实施期间先运行最小失败测试，再运行对应 package。静态检查修复按类别运行 `golangci-lint run ./...`，避免将新诊断延后到最终阶段。

最终验证从清理 Go test/build cache 后开始，并记录每个命令的退出状态：

```text
make -C backend test
make -C backend test-unit
pnpm --dir frontend run lint:check
pnpm --dir frontend run typecheck
pnpm --dir frontend run test:run
make test
```

最终 `make test` 是门禁验收命令；前面的细分命令用于定位失败域。Docker 不可用不应阻塞任何上述命令。

## 风险与回退边界

- header 修改只能放宽名称大小写，不能减少关键值断言。
- failover 修改限于耗尽后的最终错误写入，不改变选择、重试或计费路径。
- spool 生产改动需要稳定复现证据；否则仅修复测试隔离。
- lint 修复不得通过 `.golangci.yml`、build tag、忽略列表或降低检查范围绕过诊断。
- 发现需要公开 API、schema、Docker 或 integration/e2e 改动时，停止并回到 OpenSpec 重新确认范围。
