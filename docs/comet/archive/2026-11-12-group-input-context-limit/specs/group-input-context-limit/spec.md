# 分组最大输入上下文限制

状态：Q1—Q4已由用户选择，待完整Shape确认。决策依据见 [brief](../../brief.md)。以下只记录已选目标，不含未选候选方案。

## Requirement: 管理员配置与默认关闭

系统 MUST 在系统设置的网关服务提供总开关和分组最大输入token配置。默认关闭；0/留空表示不限；关闭保留阈值。管理员接口 MUST 校验组存在、阈值为非负int32整数，并以一个JSON设置原子保存。

### Scenario: 关闭时无额外检查

- Given 无配置或总开关关闭
- When 请求进入覆盖入口
- Then 不读取正文进行本策略计数，也不因遗留阈值拒绝
Acceptance: A1

### Scenario: 非法配置与保存失败

- Given 有旧配置
- When 提交无效组/负数/小数/溢出值或保存失败
- Then 返回管理接口错误且不发布部分新配置
Acceptance: A3

## Requirement: 分组阈值与回退

系统 MUST 在路由改写前冻结原绑定组身份。有效阈值为原绑定组与实际有效组非零阈值的最小值，0/缺失不参与；不计中间路由跳板。`count > limit` 拒绝，等值放行。Composite切换平台不虚构新组。进入回退目标组的计费/生成前 MUST 重查目标限制，本地拒绝 MUST NOT 触发宽松组回退。

### Scenario: 分组边界

- Given A=64000，B=128000
- When 同样输入计数80000
- Then A拒绝B放行；64000在A放行，64001拒绝
Acceptance: A2

### Scenario: 合成平台与供给组切换

- Given 原绑定组与最终实际组不同，或触发候选组回退
- When 判定策略
- Then 两组取较小非零阈值；本地拒绝终止；模型/转换变化时重新估算
Acceptance: A4

## Requirement: 可见输入本地估算

系统 MUST 以可见输入本地估算为准入计数，不新增官方计数预检。MUST 计入本次提交的system/developer/user/assistant文本历史、instructions、tools/schema、工具调用参数及工具结果。OpenAI复用模型编码器，Anthropic/Gemini为近似估算。MUST NOT 加输出预算或减缓存优惠token。管理页 MUST 明确“不保证完整上下文硬上限”。

多模态、文件、previous_response_id、conversation、cachedContent、加密compaction项 MUST 标记partial/unknown；可见部分超限仍拒绝，未知部分交上游，不当作已知0。不支持的内容不新增下载/解密/全文历史存储。已知输入损坏沿用参数错误，不将解析失败钳制成0/1。不得直接计原始JSON/base64长度。

### Scenario: 工具与隐藏输入

- Given 请求包含工具参数/结果以及多模态或历史引用
- When 本地估算
- Then 可见工具和文本纳入，记录coverage；未知部分交上游；同样输入改变max_output_tokens不改变本策略结果
Acceptance: A5

## Requirement: 入口范围与压缩豁免

系统 MUST 覆盖Messages、Chat Completions、Responses、Gemini generateContent/streamGenerateContent、Responses WS，包括现有根路径/Codex/Antigravity别名。MUST 豁免真实compact压缩端点和count_tokens/input_tokens/Gemini countTokens；普通生成不能用伪造header/字段取得豁免。

SystemOne、独立搜索、embedding、图像/音视频生成、Realtime/Live及缓存创建等专用非对话入口不在首版范围，UI MUST 明示范围。

### Scenario: 所有声明入口及别名

- Given 任一覆盖入口及别名
- When 同样组策略和可见超限输入
- Then 都判定超限；count与真实compact不因本策略拦截
Acceptance: A6

## Requirement: 按入站协议返回超限

系统 MUST 按客户端入站协议选格式，不按上游账号平台。Messages为HTTP400/type=error/error.type=invalid_request_error；Chat与Responses非流式为HTTP400/error.type=invalid_request_error/error.code=context_length_exceeded，param分别messages/input；Gemini为HTTP400/error.code=400/status=INVALID_ARGUMENT。MUST 带实际阈值和计数，不把网关阈值冒充模型物理窗口。MUST NOT 以413/429表达本策略超限。

Responses stream=true MUST 返回HTTP200 text/event-stream及合法response.failed终态，包含response.error.code=context_length_exceeded，正确序号及必要response字段；采用仓库已有失败事件构造方式。非流式仍400。MUST NOT 承诺所有Codex版本必定自动压缩，需记录实际兼容验证。其他入口在准入拒绝前不提交keepalive。

### Scenario: 三协议与Responses流式契约

- Given 相同超限输入通过不同协议
- When 准入拒绝
- Then 返回相应协议关键字段；Responses stream=true得到200及response.failed，其余HTTP入口得到400；keepalive不破坏所选契约
Acceptance: A7

## Requirement: WebSocket逐轮检查

系统 MUST 检查WS首帧和所有后续response.create，两种ingress均覆盖。超限发送response.failed及response.error.code=context_length_exceeded，带正确序号及适用stream_id，然后按policy close关闭。不允许只在升级HTTP请求或后续BeforeRequest检查。

### Scenario: 首轮与第二轮超限

- Given WS第一轮正常、第二轮超限，或第一轮即超限
- When 请求到达
- Then 对应轮不发送生成请求，客户端收到合法失败终态，连接按约定关闭
Acceptance: A8

## Requirement: 无生成副作用

策略拒绝 MUST 不触发账号故障/冷却/切换/重试/分组回退，也不扣余额、订阅或生成流量费。MUST 标为本地业务限制，日志仅记录协议/组/模型/计数/阈值/coverage/策略版本，不记录正文与工具参数。

### Scenario: 拒绝的账务及调度副作用

- Given 策略拒绝
- When 请求结束
- Then 无生成调用、重试、账号惩罚和生成扣费，日志无输入正文
Acceptance: A9

## Requirement: 快照与热更新

设置 MUST 使用不可变快照，保存节点下一请求读取新配置；其他健康节点在60秒TTL内刷新，DB故障不承诺传播期限。读取失败保留last-known-good，首次无快照按关闭并告警。单请求/turn内冻结策略，下轮刷新。旧refresh不得覆盖新save。删除组孤儿项忽略且管理页提示清理。

### Scenario: 更新与故障回源

- Given 保存和旧refresh并发
- When 旧refresh晚完成
- Then 不覆盖较新save；健康节点按TTL更新；故障保留已知配置
Acceptance: A10
