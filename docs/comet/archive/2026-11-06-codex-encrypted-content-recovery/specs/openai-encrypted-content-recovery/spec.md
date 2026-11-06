# OpenAI Encrypted Content Recovery Specification

## Purpose

在 Responses 网关遭遇不兼容历史密文时安全恢复可恢复请求，保持历史、工具配对和已有重试语义。账号健康统计豁免不在本次需求内。

## Requirements

### Requirement: 协议感知清理

系统 SHALL 复用顶层 reasoning、compaction、compaction_summary 清理约定，识别已知安全 content[] 片段。根据类型和位置判断，普通文本、元数据、未知类型及无密文输入保持，保留既有顶层 reasoning content:null 兼容。

#### Scenario: 混合历史

Acceptance: A1

WHEN 输入混合顶层及嵌套安全密文、普通文本和未知类型
THEN 仅清理允许的部分，无适用密文不附带改写。

### Requirement: 必要上下文保持

系统 SHALL 保留 function/tool call_id 及配对。未知载体、仅必要密文的工具结果、清理后非法骨架不得静默删除并重试；不可恢复时保留既有错误与可诊断原因。

#### Scenario: 不安全清理

Acceptance: A2

WHEN 清理将损坏必要上下文或骨架
THEN 不破坏输入、不重试、不伪造工具输出。

### Requirement: 原有界同账号恢复

系统 SHALL 仅在明确 invalid_encrypted_content 的现有恢复点、有效安全改写之后复用当前账号及原预算。不得新增换账号循环或普通请求预清理；原透传顺序保持。

#### Scenario: 恢复停止

Acceptance: A3

WHEN 无有效改写、再次失败或预算耗尽
THEN 停止恢复；安全有效改写只按原上限同账号重试。

### Requirement: 对称隔离的 lineage

系统 SHALL 在同一安全集合上收集摘要、探测命中并剥离。复用原会话/分组键，不保存密文原文；HTTP 和 WS replay 对齐，只删除命中片段。

#### Scenario: 片段和会话隔离

Acceptance: A4

WHEN 一个会话记录某片段失效
THEN 只影响该作用域的对应片段，其他片段、会话和分组不变。

### Requirement: 字节所有权和必要解析

系统 SHALL 不修改原 replay 字节，保留大整数精度，未命中原样返回，损坏 JSON 使用明确错误或原失败路径。在同一处理步骤复用解码结果，不为未命中输入创建改写切片；不引入新扫描框架。

#### Scenario: 未命中与异常输入

Acceptance: A5

WHEN 输入未命中、大整数或损坏 JSON
THEN 原字节不变、数值不丢精度、错误可观察；相同片段不重复完整解码，未命中不分配结果切片。

### Requirement: 保持健康和协议处理基线

系统 SHALL 撤销本 change 为旧健康豁免需求引入的所有行为变化。账号健康、调度上报、handler/result、WS 错误标记、Messages/Chat 与原生/透传 HTTP 200 SSE/JSON 错误归因、通用流终态保持本 change 之前行为。不要求实现内容错误健康豁免。

#### Scenario: 取消健康需求

Acceptance: A6

WHEN 对照基线检查最终差异
THEN 只存在清理、lineage、已有恢复日志及其必要测试改动；已取消的健康/协议归因扩展和额外预热诊断全部撤销。

### Requirement: 脱敏恢复诊断

系统 SHALL 区分识别数量、有效改写、跳过原因、重试与结果，复用请求/账号关联信息，不新增明文、密文、凭证或完整正文日志。

#### Scenario: 可诊断恢复分支

Acceptance: A7

WHEN 成功恢复、无安全载体、无改写或重试失败
THEN 日志区分原因且不泄露新增敏感内容。

### Requirement: 当前候选检查与独立验收

系统 SHALL 提供正反合成回归、必要包检查及独立只读验收。新候选不继承旧候选失效结果；未运行、失败、超时和工具链限制真实记录，不以生产重放为验收条件。

#### Scenario: 裁剪后验收

Acceptance: A8

WHEN 当前候选冻结
THEN Runtime 保存实际检查，独立 Verifier 核验所有当前条目和修改边界，无发布部署。
