## ADDED Requirements

### Requirement: 分层调度保留分组上下文
系统在使用 OpenAI 分层调度器对候选账号执行数据库二次校验时 MUST 传递当前请求的分组 ID，并允许仍属于该分组且满足其他调度条件的账号被选中。

#### Scenario: 已分组账号通过二次校验
- **WHEN** OpenAI 请求指定分组，且分层调度器选中的账号仍属于该分组
- **THEN** 系统继续使用该账号，而不是因缺失分组上下文返回 `503 Service temporarily unavailable`

#### Scenario: 账号已移出分组
- **WHEN** OpenAI 请求指定分组，但候选账号在数据库中已不再属于该分组
- **THEN** 系统拒绝该账号并继续选择其他候选账号
