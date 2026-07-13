## MODIFIED Requirements

### Requirement: 合并后验证本地关键能力
维护流程 SHALL 在合并后执行自动验证，并专项 review 本地关键能力是否仍然成立。

#### Scenario: 自动验证通过
- **WHEN** 合并完成且无冲突残留
- **THEN** 维护流程运行后端测试、前端单测、前端类型检查和构建验证

#### Scenario: 本地关键能力专项 review
- **WHEN** 自动验证完成
- **THEN** 维护流程复核 scheduler、sticky、privacy、image capability、runtime setting 热更新、网关透传字段和大输入请求体保留、落盘与释放语义等本地关键能力
