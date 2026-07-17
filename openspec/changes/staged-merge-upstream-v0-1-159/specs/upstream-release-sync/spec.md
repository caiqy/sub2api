## ADDED Requirements

### Requirement: 合并前建立本地能力保护门禁
维护流程 MUST 在首次上游 merge 前验证当前本地质量门禁，将本地能力映射到现有行为测试，并为上游目标触及且缺少保护的高风险本地能力补充最小回归测试。该门禁未通过时 MUST NOT 开始上游 merge。

#### Scenario: 当前本地基线稳定
- **WHEN** 维护流程尚未合入首个目标 tag
- **THEN** 后端与前端既定本地质量门禁 MUST 在当前本地 `HEAD` 上通过，或将既有失败明确标记为阻塞

#### Scenario: 高风险本地能力缺少行为断言
- **WHEN** 本地独有能力所在路径被目标 release 修改，且现有测试不能断言该能力的关键行为
- **THEN** 维护流程 MUST 在首次 merge 前添加可复现的最小回归测试

### Requirement: 按正式 release tag 分段集成
维护流程 SHALL 允许将一个最终上游 release 目标拆为具有严格祖先顺序的多个正式 tag 阶段。每个阶段 MUST 完成冲突处理、能力审查和阶段验证后，才能进入下一阶段。

#### Scenario: 顺序合入多个 tag
- **WHEN** 用户选择按 `v0.1.152`、`v0.1.153`、`v0.1.155`、`v0.1.156` 分段集成
- **THEN** 维护流程 MUST 按该顺序建立独立 merge 节点，且不得跳过尚未完成验证的前置阶段

#### Scenario: 某阶段首次出现本地能力回归
- **WHEN** 阶段验证发现阶段 0 已保护的本地能力不再成立
- **THEN** 维护流程 MUST 在当前 release 区间内保留失败证据并完成最小修复，不得继续合入下一 tag

## MODIFIED Requirements

### Requirement: 保留上游更新和本地定制
维护流程 MUST 在冲突处理和无文本冲突的语义审查中优先保留上游修复和本地定制能力。仅当用户在了解行为差异后明确批准某项本地能力移除时，维护流程 MAY 将其登记为例外；其他无法共存的语义 MUST 暂停等待用户确认。

#### Scenario: 冲突能力可以共存
- **WHEN** upstream 更新和本地定制修改同一文件或调用链但行为可以同时成立
- **THEN** 合并结果 MUST 同时保留上游更新和本地定制语义

#### Scenario: 用户明确批准能力移除
- **WHEN** upstream 仅部分覆盖本地能力，且用户在获知缺失范围和行为差异后仍明确选择完全采用上游
- **THEN** 维护流程 MAY 删除该本地能力，但 MUST 在 proposal、delta spec、任务和验证报告中记录例外范围

#### Scenario: 未批准的能力不能共存
- **WHEN** upstream 更新和本地定制存在不可共存语义，且该能力不在已批准例外中
- **THEN** 维护流程 MUST 停止自动处理并请求用户选择保留策略

### Requirement: 合并后验证本地关键能力
维护流程 SHALL 在每个分段 merge 后运行该阶段受影响能力与全部本地保护测试，并在最终阶段执行完整自动验证和本地能力专项 review。测试通过 MUST NOT 替代能力级审查结论。

#### Scenario: 分段自动验证通过
- **WHEN** 一个目标 tag 的 merge、冲突处理和兼容修复完成
- **THEN** 维护流程 MUST 运行阶段 0 建立的保护测试和该 tag 触及能力的定向验证，再决定是否进入下一阶段

#### Scenario: 最终自动验证通过
- **WHEN** 最终目标 tag 合并完成且无冲突残留
- **THEN** 维护流程 MUST 运行后端默认与 unit 测试、后端 lint、前端 ESLint、前端单测、前端类型检查和构建验证

#### Scenario: 本地关键能力专项 review
- **WHEN** 最终自动验证完成
- **THEN** 维护流程 MUST 逐项复核 scheduler、各平台 sticky、fallback/WaitPlan、DB recheck、privacy、image capability、runtime setting 热更新、网关透传字段、请求体重放与清理、用户资源控制、前端本地功能、版本依赖、生成代码和 migrations，并记录每项证据
