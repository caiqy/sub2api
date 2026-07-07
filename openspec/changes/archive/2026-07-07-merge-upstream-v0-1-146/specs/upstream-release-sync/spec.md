## ADDED Requirements

### Requirement: 确认上游合并目标
维护流程 SHALL 在合并前确认本地当前版本、upstream 最新 release tag、以及目标分支或 tag 的选择理由。

#### Scenario: 选择 upstream release tag
- **WHEN** upstream 存在比本地当前定制版本更新的 release tag
- **THEN** 维护流程记录目标 tag，并说明为何不默认使用 `upstream/main`

### Requirement: 在隔离分支执行合并
维护流程 SHALL 从干净的本地主线创建临时分支执行上游合并，除非用户明确选择其他隔离方式。

#### Scenario: 创建临时合并分支
- **WHEN** 本地 `main` 干净且已确认目标 upstream tag
- **THEN** 维护流程在临时分支中执行合并，不直接改写 `main`

### Requirement: 保留上游更新和本地定制
维护流程 MUST 在冲突处理时优先保留上游修复和本地定制能力；若两者语义不能共存，必须暂停等待用户确认。

#### Scenario: 冲突能力可以共存
- **WHEN** upstream 更新和本地定制修改同一文件但行为可以同时成立
- **THEN** 合并结果同时保留上游更新和本地定制语义

#### Scenario: 冲突能力不能共存
- **WHEN** upstream 更新和本地定制在同一行为上存在不可共存语义
- **THEN** 维护流程停止自动处理并请求用户选择保留策略

### Requirement: 合并后验证本地关键能力
维护流程 SHALL 在合并后执行自动验证，并专项 review 本地关键能力是否仍然成立。

#### Scenario: 自动验证通过
- **WHEN** 合并完成且无冲突残留
- **THEN** 维护流程运行后端测试、前端类型检查和构建验证

#### Scenario: 本地关键能力专项 review
- **WHEN** 自动验证完成
- **THEN** 维护流程复核 scheduler、sticky、privacy、image capability、runtime setting 热更新和网关透传字段等本地关键能力
