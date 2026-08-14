# Outcome

管理员可在使用记录的“延迟”单元格内直接看到模型生成阶段的输出速度，用于区分首字等待和持续输出性能，同时保持现有宽表的横向尺寸。

# Scope

- 仅管理员使用记录表格展示输出速度。
- 在现有“首字 / 总耗时”下增加第三行“速度”，值采用 `tok/s`。
- 前端使用现有 `output_tokens`、`duration_ms`、`first_token_ms` 计算：`output_tokens * 1000 / (duration_ms - first_token_ms)`。
- 速度保留一位小数；缺少必要计时字段、结果非有限数或 `duration_ms <= first_token_ms` 时显示 `-`。
- 补充中英文标签和针对性前端测试。

# Non-goals

- 不新增后端字段、数据库列或 API 契约。
- 不在普通用户使用记录中展示输出速度。
- 不为输出速度增加独立列、排序、筛选、健康阈值或颜色等级。
- 不修改 Excel 导出内容。

# Acceptance examples

- A1：管理员记录的 `output_tokens=2016`、`duration_ms=34520`、`first_token_ms=5960` 时，“延迟”单元格第三行显示“速度 70.6 tok/s”。
- A2：管理员记录缺少 `first_token_ms` 或 `duration_ms`，或 `duration_ms <= first_token_ms` 时，第三行仍保持布局并显示“速度 -”。
- A3：输出 Token 为 0 且计时区间有效时显示“速度 0.0 tok/s”。
- A4：管理员表格不增加新列，现有首字/总耗时健康色条和文本颜色行为保持不变。
- A5：普通用户使用记录不出现“速度”行，后端响应结构保持不变。

# Constraints and invariants

- 输出速度分母必须是生成阶段耗时，即总耗时减首字耗时，并将毫秒换算为秒。
- 展示使用等宽数字，速度采用独立但无等级含义的天蓝色文本；不可仅依赖颜色表达指标含义。
- 复用现有 `UsageTable`，通过默认关闭的展示属性限制管理员范围，避免复制表格组件。

# Decisions

- 采用视觉方案 A：在延迟单元格中纵向增加第三行，不新增独立列。
- 方案 B（合并到总耗时行）因扩大延迟列宽且长数值拥挤而不采用。
- 方案 C（独立列）因继续拉宽管理表格而不采用。
- 速度由前端按已有字段即时计算，避免维护可推导的后端字段。

# Open questions

无。

# Verification expectations

- 运行 `UsageTable` 针对性 Vitest，覆盖有效计算、零输出和无效计时区间。
- 运行管理员与用户 `UsageView` 针对性 Vitest，确认管理员开启且用户端保持隐藏。
- 运行前端 TypeScript 检查或等价构建检查，确认模板和属性类型正确。
