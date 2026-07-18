## ADDED Requirements

### Requirement: 调度结果保留模型与真实终态
系统 MUST 使用账号实际采用的 canonical model 上报 OpenAI 调度结果，并 MUST 根据上游终止事件区分成功与失败。

#### Scenario: 成功请求清除对应模型的瞬时状态
- **WHEN** OpenAI API Key 账号在某个 canonical model 上成功完成请求
- **THEN** 系统清除该账号与该 canonical model 对应的瞬时失败状态

#### Scenario: WS 失败终态不记为成功
- **WHEN** 上游 WS 返回 `response.failed`、`response.incomplete` 或 `response.cancelled` 终态
- **THEN** 系统将该次调度结果上报为失败

#### Scenario: WS 完成终态记为成功
- **WHEN** 上游 WS 返回 `response.completed` 或 `response.done` 终态
- **THEN** 系统将该次调度结果上报为成功
