## Context

本地验证目前出现三种不一致：稳定的 handler unit 断言失败、server unit 测试 fixture 无法满足已扩展的 `UserRepository` 接口、以及 `golangci-lint` 的历史和近期问题。request body spool cleanup 只在并行全套中失败，单独连续运行通过，必须先确认是否存在共享临时文件或异步清理竞争。

## Goals / Non-Goals

**Goals:**
- 使后端默认测试、后端 unit 测试、后端 lint、前端 ESLint、TypeScript 与全量 Vitest 在本地稳定通过。
- 让测试断言遵循 HTTP header 大小写无关语义，并让 fixture 与生产接口契约同步。
- 对每个 lint 诊断修复根因，不降低 linter 覆盖或添加全局排除。

**Non-Goals:**
- 不安装 Docker，不运行或修改 integration/e2e。
- 不改变公开 API、数据库 schema、上游协议或产品能力。
- 不把偶发 spool cleanup 失败视为生产缺陷，除非最小复现能够证明清理路径错误。

## Decisions

- 按失败域而非文件数量分阶段：先修 unit 可执行性与稳定失败，再修 lint，最后从冷缓存做全量复验。这使每一阶段都有独立失败信号。
- HTTP header 测试通过 `http.Header.Get` 或大小写无关比较检查语义，不依赖序列化文本中的 canonicalization 格式。
- server 测试 stub 显式实现新增 repository 方法，返回测试所需的零值或受控值；不修改生产接口以迁就测试。
- spool cleanup 使用重复、并行与临时路径隔离的复现矩阵判断根因；只有稳定复现后才在实际资源所有权边界添加清理。
- lint 按诊断类别分组处理：依赖边界、资源/取消、无效赋值、静态分析与未使用符号；每组运行对应 package 或完整 linter 验证。

## Risks / Trade-offs

- [大范围 lint 修复可能改变资源生命周期] → 每个类别保留聚焦测试，并在合并前运行完整后端默认和 unit 套件。
- [header 测试改写可能掩盖真实头缺失] → 断言值和关键 content type，只有名称比较改为大小写无关。
- [spool 问题可能为测试竞争] → 不在无复现证据时修改生产清理逻辑，先固定测试隔离或生命周期等待条件。
- [本地全量门禁耗时较长] → 仅最终阶段从冷缓存运行一次；开发阶段使用目标 package 验证。
