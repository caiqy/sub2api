# 账号透传字段规则设计

- 日期：2026-03-20
- 状态：已在对话中确认
- 范围：账号新建/编辑界面中的 API Key 账号透传字段规则

## 1. 背景与目标

当前账号配置中已经存在若干“自动透传（仅替换认证）”能力，但这些能力按平台和链路内置，无法让管理员按账号精细控制哪些请求字段可以放行到上游，或为特定账号固定注入某些字段。

本设计新增一个与现有自动透传能力完全独立的“透传字段规则”能力，用于所有 API Key 类型账号。管理员可以在账号维度开启该能力，并按规则配置：

- 允许客户端已有字段继续透传到上游（forward）
- 为请求固定注入字段（inject）

该能力同时支持：

- Header 字段
- Body 对象层级字段（仅点号路径）

## 2. 范围

### 2.1 本次纳入范围

- `CreateAccountModal.vue`
- `EditAccountModal.vue`
- 所有 API Key 类型账号
- Header / Body 两类规则
- `forward` / `inject` 两种模式
- 前后端校验、保存、回显、运行时生效

### 2.2 明确不在本次范围

- BulkEditAccountModal 批量编辑
- OAuth / setup-token / upstream / bedrock 等非 API Key 类型账号
- 与现有自动透传能力做联动或互相隐式影响
- 数组下标路径，如 `messages.0.role`
- 通配符、JSONPath、表达式语法

## 3. 设计原则

1. **与自动透传解耦**：不复用现有自动透传开关，不改变其语义。
2. **统一规则模型**：Header/Body、forward/inject 走同一套规则结构。
3. **显式优于隐式**：冲突在保存前直接拦截，不依赖运行时覆盖顺序。
4. **增量兼容**：旧账号无配置时行为完全不变。
5. **后端硬校验**：关键安全规则不能只依赖前端。

## 4. 用户可见行为

### 4.1 展示条件

仅当当前账号类型为 API Key 时，在账号新建/编辑表单中展示“透传字段规则” section。其他账号类型不展示该 section，后端也不应让此能力生效。

### 4.2 总开关行为

Section 顶部提供总开关，例如：`透传字段规则`。

- 开启：展示规则列表与新增入口
- 关闭：规则配置保留但不生效

UI 需明确提示：关闭只是不生效，不会清空已有规则。

### 4.3 规则能力

每条规则支持：

- `target = header | body`
- `mode = forward | inject`
- `key = header 名或 body 路径`
- `value = inject 时必填`

语义如下：

- `forward`：仅当客户端请求里本来就存在该字段时，允许它继续透传到上游
- `inject`：无论客户端是否携带，都由系统在转发前固定写入

## 5. 方案选择

在方案讨论中考虑过三种形式：

1. 单列表规则配置
2. 按目标拆为 Header / Body 两个列表
3. 按能力拆为 forward / inject 两个列表

最终选择 **单列表规则配置**。

选择理由：

- 结构最统一，前后端建模最简单
- 用户可以在一个区域中集中管理 Header / Body 与 forward / inject
- 后续若扩展新的 target 类型，仍可复用同一规则模型

## 6. 数据模型设计

### 6.1 存储位置

继续沿用现有账号扩展配置模式，存入 `accounts.extra`。

新增键为：

- `passthrough_fields_enabled: boolean`
- `passthrough_field_rules: PassthroughFieldRule[]`

### 6.2 规则结构

示例：

```json
{
  "passthrough_fields_enabled": true,
  "passthrough_field_rules": [
    {
      "target": "header",
      "mode": "forward",
      "key": "OpenAI-Beta"
    },
    {
      "target": "body",
      "mode": "forward",
      "key": "reasoning.effort"
    },
    {
      "target": "header",
      "mode": "inject",
      "key": "X-Env",
      "value": "prod"
    },
    {
      "target": "body",
      "mode": "inject",
      "key": "metadata.user_id",
      "value": "fixed-user"
    }
  ]
}
```

逻辑模型：

```ts
type PassthroughFieldTarget = 'header' | 'body'
type PassthroughFieldMode = 'forward' | 'inject'

interface PassthroughFieldRule {
  target: PassthroughFieldTarget
  mode: PassthroughFieldMode
  key: string
  value?: string
}
```

其中 `value` 在 v1 中**有意只支持字符串**：

- Header inject：写入字符串 header value
- Body inject：在目标 JSON 路径写入字符串值

本版本不支持 number / boolean / object / array 注入，以保持 UI、校验和序列化逻辑简单明确。

### 6.3 前端 DTO 扩展

前端 `Account`、`CreateAccountRequest`、`UpdateAccountRequest` 继续通过 `extra` 承载新增字段，无需新开独立顶层字段。若需要增强类型安全，可在前端类型中补充可选字段定义，便于表单读写和测试。

### 6.4 后端 DTO 扩展

后端账号 DTO 保持以 `Extra map[string]any` 为主，但需要在 admin handler/service 层增加对如下键的读写、校验和回填：

- `passthrough_fields_enabled`
- `passthrough_field_rules`

## 7. UI 设计

### 7.1 Section 位置

在 `CreateAccountModal.vue` 与 `EditAccountModal.vue` 中增加独立 section，标题为：

- 标题：`透传字段规则`
- 辅助说明：`仅对 API Key 类型账号生效；与自动透传能力独立`

### 7.2 单条规则表单

每条规则使用一行或一卡片，包含：

1. 目标选择：`Header` / `Body`
2. 模式选择：`放行透传` / `固定注入`
3. 字段输入：header 名或 body 路径
4. 值输入：仅 `inject` 时展示
5. 删除按钮

### 7.3 动态提示

- 当 `target = header`：提示“Header 比较时不区分大小写”
- 当 `target = body`：提示“仅支持 `xx.xx` 形式的对象层级路径”
- 当 `mode = inject`：提示“固定注入将在转发前写入上游请求”

### 7.4 开关关闭状态

当总开关关闭时：

- 保留规则列表数据
- UI 将规则区置灰但仍保持可见，便于管理员确认已有配置
- 展示说明：`已配置规则会保留，但当前不会生效`

## 8. 校验规则

### 8.1 通用校验

- `target` 仅允许 `header` / `body`
- `mode` 仅允许 `forward` / `inject`
- `key` 必填
- `key` 在保存前会做首尾空白裁剪；裁剪后为空则非法
- `inject` 模式下 `value` 必填，且全空白字符串非法
- `inject.value` 保留用户原始内容，不做自动 trim，仅用于“是否全空白”的校验时做空白判断
- 非 API Key 账号请求中如果显式携带 `passthrough_fields_enabled` 或 `passthrough_field_rules`，后端直接拒绝保存并返回 400

### 8.2 Header 校验

- 存储时保留用户原始输入
- 比较和去重时按大小写不敏感处理
- 因此 `X-Test` 与 `x-test` 视为重复，禁止同时保存
- 参与比较前先做首尾空白裁剪

### 8.3 Body 路径校验

- 仅支持点号路径，如 `metadata.user_id`
- 每一段表示对象 key
- 不支持数组下标、通配符、空段、前后点号
- 保存前先做首尾空白裁剪

应拒绝的例子：

- `messages.0.role`
- `.metadata.user_id`
- `metadata..user_id`

### 8.4 冲突校验

下列情况保存前直接拦截：

- 同一 `target` 下同一 `key` 重复
- 同一 `target` 下同一 `key` 同时配置 `forward` 和 `inject`
- 命中系统保留字段

这里不采用“后者覆盖前者”策略，避免运行时行为不透明。

## 9. 系统保留字段策略

需要在后端维护一份不可配置的保留字段清单，用于保护认证、路由、计费、审计和安全逻辑。

### 9.1 Source of truth

保留字段规则的唯一真相源放在**后端单一校验模块**中，由该模块同时服务于：

- 创建/更新账号时的保存校验
- 请求转发时的运行时保护
- 单元测试与集成测试

前端只做体验型预校验和提示文案，不能成为最终判定来源。

### 9.2 最小必含项

后端保留字段清单至少要覆盖以下类别：

1. **认证类 header**
   - 例如：`Authorization`、`x-api-key`、`api-key`
2. **传输层/连接管理 header**
   - 例如：`Host`、`Content-Length`、`Transfer-Encoding`、`Connection`
3. **网关内部控制 header**
   - 例如当前系统会写入、转发或依赖的 request-id、trace、审计、兼容模式相关 header
4. **核心路由/计费/审计相关 body 字段**
   - 例如会影响模型路由、账号调度、计费、审计归因的关键字段
5. **现有系统已主动改写或维护的字段**
   - 包括当前 gateway 在不同平台链路中已经有专门逻辑处理的 header/body 字段

### 9.3 判定要求

- Header 保留字段判定按大小写不敏感处理
- Body 保留字段判定按裁剪后的完整路径字符串处理
- 校验错误需返回可读信息，指出冲突字段和原因

## 10. 运行时应用规则

### 10.1 生效前提

仅当以下条件全部满足时，规则才在请求转发阶段生效：

1. 当前账号为 API Key 类型
2. `passthrough_fields_enabled = true`
3. 规则列表存在且校验通过

否则按现有逻辑处理，不应用任何透传字段规则。

### 10.1.1 与账号类型切换的关系

- 前端只在表单当前账号类型为 API Key 时展示并提交这两个配置项
- 如果用户在编辑表单中把账号类型从 API Key 切换为非 API Key，前端应提示“保存后将移除透传字段规则配置”
- 任何**最终保存结果为非 API Key 类型**的账号，后端都必须在持久化前强制移除 `passthrough_fields_enabled` 和 `passthrough_field_rules`
- 如果后端收到非 API Key 账号显式提交这两个字段，直接返回 400，而不是静默接受

这样可以保证“只有 API Key 账号能持久化该能力配置”这一规则唯一明确，并避免 update/merge 语义下旧规则残留在非 API Key 账号中。

### 10.1.2 运行时集成点

规则应用在**账号已选定、基础请求已解析、上游请求尚未最终构造**的阶段。

具体来说：

- `forward` 的数据来源是客户端进入网关时的原始可用 header / 已解析请求体字段
- `inject` 和 `forward` 的写入目标都是同一个“待发送上游请求”的可变表示
- 所有规则都在最终发出上游请求之前完成应用

这样可以避免有人在“原始请求”“平台兼容转换后的请求”“最终 HTTP 请求”三个不同层次各自实现一套逻辑。

### 10.2 Forward 行为

#### Header

- 若客户端请求中存在该 header，则允许其传给上游
- 若客户端请求中不存在，则不自动补充

#### Body

- 若客户端请求 body 中存在该路径，则允许该字段原样进入上游请求
- 若客户端请求 body 中不存在，则不自动补充

### 10.3 Inject 行为

#### Header

- 在向上游构造请求时写入对应 header

#### Body

- 在构造上游请求体时，按点号路径写入对象层级
- 如果中间对象不存在，则自动创建缺失对象节点
- 如果中间节点已存在但不是对象，则视为**运行时结构冲突**：拒绝该次请求，返回明确 4xx 错误，并记录日志

### 10.4 应用顺序

采用以下原则：

1. 先通过配置校验消除重复与冲突
2. 运行时先执行系统保留字段保护
3. 再应用 `inject`
4. 再应用 `forward`

由于保存阶段已经禁止同一路径重复配置，运行时不再需要做“谁覆盖谁”的模糊判定。

### 10.5 规则标准化与判定职责

规则的标准化和最终判定集中在后端单一模块中：

- 对 `key` 做首尾空白裁剪
- Header 比较时生成小写比较键
- Body 路径按裁剪后的原值进行语法校验
- 校验保留字段冲突
- 输出给网关可直接消费的已验证规则

前端只复刻同样的校验体验，不拥有最终判定权。

## 11. 组件与职责边界

### 11.1 前端

`CreateAccountModal.vue` / `EditAccountModal.vue`

- 负责显示和编辑总开关与规则列表
- 负责基础表单校验
- 将规则写入 `extra`
- 在编辑场景正确回填

要求抽离一个共享子组件，例如 `PassthroughFieldRulesEditor`，专门承载规则编辑区，减少两个 modal 的重复逻辑和校验漂移。

### 11.2 后端 admin 层

- 接收账号创建/更新请求
- 校验该能力仅限 API Key 账号
- 调用统一规则校验模块，完成结构校验、标准化和保留字段判定
- 将配置写入 `accounts.extra`
- 将配置读回并返回前端

### 11.3 网关/转发层

- 读取已经过后端验证并可直接消费的规则
- 在“待发送上游请求”的可变表示上应用 `inject` / `forward`
- 遇到运行时结构冲突时终止该次请求并记录日志
- 保证不突破系统保留字段保护

## 12. 错误处理

### 12.1 保存阶段错误

需要能明确定位到规则：

- header 重复（含大小写不同的重复）
- body 路径非法
- inject 缺少 value
- 命中保留字段
- 非 API Key 账号使用该能力

前端应尽量把错误挂到对应规则项附近，而不是只给全局 toast。

### 12.2 运行阶段错误

主要关注 body inject 的结构冲突：例如路径中间节点不是对象，导致无法继续向下写入。

此类问题的处理方式明确为：

- 拒绝该次请求，返回 `400 invalid_request_error`
- 在日志中记录账号 ID、规则 target/key、冲突节点

保存阶段只解决静态配置合法性；运行阶段只处理请求实际结构带来的动态冲突。

## 13. 测试设计

### 13.1 前端测试

覆盖点：

- API Key 账号展示 section，非 API Key 不展示
- 新建和编辑场景都能正确回填与提交
- 开关关闭时规则保留且提示正确
- 新增、删除规则行为正常
- `inject` 模式下 `value` 必填
- Header 重复校验大小写不敏感
- Body 仅接受点号路径

### 13.2 后端测试

覆盖点：

- `extra` 中新字段的序列化/反序列化
- 非 API Key 账号提交该能力时被拒绝
- Header 重复与大小写冲突拦截
- Body 路径非法拦截
- 保留字段冲突拦截
- `forward` 不自动补字段
- `inject` 能正确写入 header/body
- 开关关闭时配置保留但不生效

### 13.3 集成测试

至少覆盖：

- 带 `forward` header 的请求透传成功
- 带 `forward` body 路径的请求透传成功
- `inject` header 固定写入成功
- `inject` body 对象路径写入成功
- `inject` body 在运行时遇到非对象中间节点时返回明确 4xx 错误
- 配置关闭时上述规则均不生效

## 14. 风险与缓解

### 14.1 保留字段被误覆盖

缓解：后端硬校验 + 单元测试覆盖。

### 14.2 Header 大小写导致隐式冲突

缓解：比较时统一转小写，展示与存储保留原样。

### 14.3 Body 路径写入破坏请求结构

缓解：仅支持对象层级路径；中间节点非对象时拒绝应用，并记录日志。

### 14.4 与自动透传能力混淆

缓解：UI 标题、说明文案和帮助提示中明确“该功能与自动透传独立”。

## 15. 兼容性与迁移

- 旧账号默认无这两个新键，行为不变
- 不需要数据迁移脚本
- 新增配置仅存放在 `extra` 中，符合当前账号扩展字段模式

## 16. 最终决策摘要

- 功能入口：新建账号 + 编辑账号
- 适用范围：所有 API Key 类型账号
- 与自动透传能力：完全独立
- 配置关闭：保留规则但不生效
- Header 大小写：保存原样，比较不区分大小写
- Body 路径：仅支持点号对象路径
- 规则结构：单列表，每条规则自带 `target` 和 `mode`
- 冲突处理：保存前直接拦截，不允许重复或覆盖
