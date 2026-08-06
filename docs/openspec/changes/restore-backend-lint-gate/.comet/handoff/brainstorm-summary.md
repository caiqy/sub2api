# Brainstorm Summary

- Change: restore-backend-lint-gate
- Date: 2026-08-06

## 已确认事实与约束

- 基线为 `main@b576f73a22c4bf23d61727fc93950766a7e33929`。
- Go 1.26.5、golangci-lint 2.9.0 与现有配置下，uncapped 结果为 144 issues：140 `ineffassign`、3 `staticcheck`、1 `unused`，共 39 个文件。
- upstream 原始 `v0.1.169` 与 `v0.1.171` 在相同 gate 下均为 0 issues，债务属于 fork-local 演进。
- 不修改 lint 配置、CI、Go 版本、Make target，不使用 suppression、`--new` 或 issue cap 隐藏问题。
- 必须保持 `local-test-gates` 与 `request-body-retention-control` 既有规范，尤其是 25 分支 handler/upstream 阻塞内存矩阵、独立 async image 测试、spool、retry/failover、审计和计费语义。
- 本 change 不包含 upstream 合并、版本、前端、部署或无关重构。

## 候选技术方案

1. 候选 A（推荐）：删除静态不可观察的局部 `nil` 赋值；拆分混合赋值并保留 owner/结构体字段清理；仅当现有内存测试回退时收窄作用域或提取职责相符的小函数。
2. 候选 B：即使测试不回退，也把所有命中路径强制改造成窄作用域/helper，以代码结构显式表达生命周期。意图更显眼，但会显著扩大 39 文件 diff 和回归面。
3. 候选 C：用统一清零 helper、无意义读取或 `//nolint` 绕过分析。该方案不解决真实控制流问题且违反已确认约束，排除。

## 关键取舍与风险

- 候选 A 的 diff 最小，依赖 Go 的 last-use liveness 与现有运行时矩阵证明；风险是某些平台/编译器路径的保留行为只能靠测试发现。
- 候选 B 更显式，但为 lint remediation 引入大量控制流改造，审查与行为风险高于收益。
- 混合赋值不能机械整行删除；必须逐项保留可观察 ownership 清理。

## 测试策略

- 每批以全量 uncapped lint 中属于该批的命中集合作为 TDD RED；运行时行为当前正确，不新增人为失败的实现细节测试。
- 每批修改后重新运行全量 uncapped lint，确认剩余集合只减少，并运行对应 package 测试。
- 请求体相关批次运行现有 25 分支内存保留矩阵、独立 async image 测试及 spool/retry/failover 聚焦测试。
- 最终运行 uncapped lint 0 issues、backend 默认/unit 测试和仓库级 `make test` exit 0。

## 确认的批次与失败处理

1. handler/routes 与 QF1003。
2. 通用 Gateway、Anthropic、Bedrock 与 Antigravity，包括归属这些路径的测试源。
3. OpenAI、Gemini、Grok 与 unused 方法。

- 每批保存全量 uncapped 输出并核对目标集合，不依赖同 package 内脆弱的文件级 lint 参数。
- 内存矩阵失败时，只对失败路径改用窄作用域或职责相符的小函数。
- 出现基线外问题时，先判断是否由当前批次引入；不得静默扩大范围。

## Spec Patch

无。当前 change 仅恢复既有规范符合性。

## 已确认的技术方案

- 采用候选 A：最小删除优先。删除静态不可观察的局部清零；拆分混合赋值并保留 owner/结构体字段清理；仅当现有测试发现实际回退时收窄作用域或提取小函数。

## 待确认

无。技术方案、TDD RED、批次、失败处理与测试策略均已由用户确认。
