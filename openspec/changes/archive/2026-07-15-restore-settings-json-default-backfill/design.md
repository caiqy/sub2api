## Context

`f2966530c` 将 settings getter 拆分到 `setting_features.go` 时改为对零值结构体执行 `json.Unmarshal`，丢失了 `9095a21c6` 已实现的 legacy JSON 默认字段回填语义。

## Goals / Non-Goals

**Goals:**
- 恢复缺失 JSON 字段沿用当前默认值的既有行为。
- 保持显式配置字段、校验与错误处理不变。

**Non-Goals:**
- 不改变 settings JSON schema 或默认值。
- 不新增迁移、helper 或测试抽象。

## Decisions

- 三个 getter 直接以现有 `Default*Settings()` 返回值初始化，再执行 `json.Unmarshal`。
- 复用已有回归测试，不新增重复测试。

## Risks / Trade-offs

- 显式保存的零值仍由 JSON 覆盖默认值；只有缺失字段获得默认值，兼容现有配置语义。
