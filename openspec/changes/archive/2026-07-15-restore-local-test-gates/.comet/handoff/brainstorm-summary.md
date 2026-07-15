# Brainstorm Summary

- Change: restore-local-test-gates
- Date: 2026-07-15

## 确认的技术方案

以单一完整 change 恢复本地代码质量门禁。先恢复 unit 可执行性和稳定失败，再按诊断类别修复全部现有 lint，最后通过统一入口从冷缓存做全量验证。Docker、integration/e2e 不属于本地门禁。

- failed-usage 测试从结构化 header 读取关键值，避免依赖 Go canonical header 名称。
- Images failover 耗尽分支必须写出网关错误体；server fixture 显式补齐当前 `UserRepository` 方法。
- spool cleanup 测试使用 `t.TempDir()` 和自身 handle 路径隔离临时文件；仅在隔离后稳定复现时调整生产清理。
- `make test` 覆盖默认 Go、unit Go、lint、前端 ESLint、TypeScript 和全量 Vitest。

## 关键取舍与风险

- 不放宽 linter 或添加 ignore；每项诊断必须修复根因。
- HTTP header 测试按协议语义比较，不依赖 Go 序列化后的 header 名称大小写。
- request body spool cleanup 目前只在并行全套中失败；先建立重复和并行复现矩阵，稳定复现前不改生产资源生命周期。
- server fixture 显式跟随 `UserRepository` 接口扩展，不回退或缩减生产接口。
- lint 保持现有配置，按依赖边界、资源/取消、无效赋值、静态分析和未使用符号分组修复。

## 测试策略

- 每个失败域先运行最小 package/test 名称验证 RED/GREEN。
- 每类 lint 修复后运行受影响 package 测试和 `golangci-lint run ./...`。
- 最终从冷缓存运行后端默认测试、后端 unit、后端 lint、前端 ESLint、TypeScript 与全量 Vitest。

## Spec Patch

无。
