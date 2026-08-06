## Context

参见 `proposal.md` 的 Why。基线 `main@b576f73a` 在 Go 1.26.5、golangci-lint 2.9.0 和现有 `backend/.golangci.yml` 下产生 144 项 uncapped lint 问题：140 项 `ineffassign`、3 项 `staticcheck`、1 项 `unused`，分布于 39 个文件。140 项中的绝大多数是请求体内存优化期间加入的局部 `[]byte = nil`；静态分析确认赋值后的值没有被读取，但其运行时驻留意图仍需由 `request-body-retention-control` 的内存生命周期测试验证。

现有 `request_body_memory_retention_test.go` 已覆盖 25 条 handler/upstream 阻塞分支，并分别比较 2MB 与 8.9MB 请求在 GC 后的保留增长；独立 async image 测试内部重复 3 轮。该矩阵和相关 spool/retry 测试是语义保持的事实源；lint 配置、工具版本和 CI 命令不得改变。

## Goals / Non-Goals

**Goals:**

- 在 unchanged gate 下把 uncapped lint 结果降为 0 issues，并使仓库级 `make test` 退出 0。
- 保持请求体 ownership、spool、retry、failover、审计、计费和上游 wire body 行为。
- 让每项修复保持局部、可审查，并留下按风险分组的验证证据。

**Non-Goals:**

- 不重写请求体架构，不引入通用清零抽象或新依赖。
- 不调整 lint rule、issue cap、CI workflow、Go 版本或 Make target。
- 不处理 upstream 合并、版本发布、前端、部署或无关代码整理。

## Decisions

### 1. 以 uncapped 清单作为闭包，不使用增量过滤

基线与最终验证均执行 `golangci-lint run ./... --max-issues-per-linter 0 --max-same-issues 0`。默认输出只显示 54 项且具体 `ineffassign` 样本不稳定，不能证明 144 项已全部关闭；`--new` 也会把已有债务排除在验收外。

替代方案是依赖默认 `golangci-lint run ./...`，但它只适合最终 CI 一致性复核，不适合 remediation 清单闭包。

### 2. 对无效清零采用删除优先，并保留真实 ownership 变更

对最后一次读取后仅给局部 `[]byte` 赋 `nil` 的语句，直接删除无效赋值，依赖 Go 编译器基于最后使用点的 liveness，而不增加 `dropBytes` 一类仅用于躲避分析器的 helper。对同时清理结构体字段、输入对象或其他仍可观察状态的多重赋值，拆分表达式并只删除被报告的局部赋值，保留真实 ownership 释放。

若现有内存保留测试证明某一路径因删除而回退，则将物化 body 的工作收进窄作用域或现有职责相符的小函数，使大 slice 在进入上游等待前离开作用域；不得用 `//nolint`、反射、指针清零 helper 或无意义读取绕过 linter。

替代方案是机械替换所有 `= nil` 或统一封装清零 helper。前者会误删可观察状态，后者只是隐藏静态分析问题并扩大 API 面，均不采用。

### 3. 其余四项按工具建议做语义等价最小修改

- 将两个嵌套条件位置和一个状态条件改为 tagged switch，保持分支顺序、默认分支和返回值不变。
- 删除确认无静态、接口或动态调用方的私有 `sendCCUpstreamRequest` 方法；不保留 dead wrapper。
- 不借此整理相邻代码或改变错误文本、HTTP status、日志、header 和 payload。

### 4. 按风险域分批修复和验证

实现分三批：handler/routes（含其 QF1003 测试源）；通用 Gateway、Anthropic、Bedrock 与 Antigravity（含归属这些路径的测试源）；OpenAI/Gemini/Grok 与 unused 方法。每批从全量 uncapped lint 中确认该批 RED 集合，再修改并运行对应 package 测试；涉及 request body 的批次还运行内存保留、spool、retry/failover 聚焦测试。全部批次结束后依次运行全量 uncapped lint、backend 默认/unit 测试和仓库级 `make test`。

分批仅用于审查和失败定位，最终验收仍以全仓命令为准，不能将未显示问题留给后续 change。

默认 changed-file allowlist 仅含 baseline manifest 的 39 个文件。若真实回退暴露现有测试无法覆盖的行为缺口，必须先更新 design、baseline manifest 与 tasks，经范围审查后才可修改额外 backend Go 测试。

## Risks / Trade-offs

- [删除局部清零后大 body 跨上游等待存活] → 先保留 owner/handle 清理，只删除值流上无后续读取的赋值；运行 25 分支内存保留矩阵和独立 async image 测试，失败时改用窄作用域而非 suppression。
- [多重赋值中误删结构体字段清理] → 逐项拆分，仅移除 linter 指向的局部变量，使用 spool/retry/failover 测试验证 ownership。
- [39 文件批量改动掩盖业务变化] → 按风险域小批提交，禁止顺手重构，并逐批运行 package 测试与 diff review。
- [默认 lint 输出掩盖残留] → 最终必须记录 uncapped 0 issues，再运行与 CI 一致的默认 gate。
- [主分支在实施期间漂移] → change 固定 `base_ref=b576f73a`；若 base 改变，重新生成 baseline 清单并审视范围，不能静默吸收。

## Migration Plan

此 change 无数据或部署迁移。完成后先将修复分支集成到 `main`，再恢复 `staged-merge-upstream-v0-1-171`，显式更新并重新确认其 immutable source base，然后从 Task 4 完整重跑基线 gate。回滚时按提交边界撤销 lint remediation，并把依赖 change 保持 blocked；不得在 gate 非绿时继续 upstream 合并。
