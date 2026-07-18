## Context

`scanUsageLog` 按固定顺序扫描 UsageLog 字段。`usageLogSelectColumns` 是所有读取路径共享的固定 SELECT 列表；当前两者相差 `image_input_tokens` 和 `image_input_cost`，而现有 sqlmock 测试手工维护了正确列，未覆盖共享常量与扫描目标之间的契约。

## Goals / Non-Goals

**Goals:**

- 恢复所有使用 `usageLogSelectColumns` 的查询读取。
- 用最小测试直接约束 SELECT 列数与扫描目标数一致。

**Non-Goals:**

- 不修改 UsageLog 数据模型、数据库 schema、API 响应或计费逻辑。
- 不重构固定列扫描机制。

## Decisions

- 在 `image_output_cost` 后补入两个缺失列，使顺序与 `scanUsageLog` 保持一致；这是唯一需要的生产代码修改。
- 使用记录扫描器收到的目标数量测试共享列契约。相比继续扩充手工 sqlmock 行，该测试能直接捕获本次遗漏。

## Risks / Trade-offs

- 固定字符串和扫描列表仍需同步维护 → 回归测试在两侧数量再次漂移时立即失败。
