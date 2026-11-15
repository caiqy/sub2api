# Upstream Tag Fusion 20261114

## Purpose

融合 upstream 正式版 v0.2.21 与当前 fork 的全部可共存能力；固定输入、明确语义优先级、保留不可变历史迁移，在可追踪的 Native 单一 change 内完成隔离实施、真实检查、独立验收和本地 main 交付。

## R1：输入、授权与提交边界

基线为 `1cf5be61d9864a73428fdb8407030cc31bed2a1a`（VERSION `0.2.20.8`），共同祖先 v0.2.20 为 `dab829a4e9b7d1cc307189aea8b0319eb29e29d0`。唯一目标为 `v0.2.21^{}` = `10ba8f9b4dc2e6023d4a9879ab77b21841babd10`，tag object `43f15294c477bec427d586ddfc3f918ce1284c98` 不作为 merge commit。当前 upstream/main 的 tag 后提交 `6dd5a045b`、`530c5e3a4` 不吸收。

在 `comet/merge-upstream-v0-2-21` 融合；产生可追溯 no-ff merge，双方固定提交均为最终祖先，文档准备提交可以位于 merge 第一父之前。最终 VERSION 仍为 `0.2.20.8`。本地 main 合并是本轮目标；组长验收和 Runtime 边界满足后完成，禁止 push/tag/release/deploy。

用户已授权：没有需要其选择的冲突，文档完成且组长复核后直接实施。新发现无法兼容的用户可见语义时先记录具体差异并询问，普通文本冲突不触发额外开工审批。正式状态、候选冻结、检查、Verifier 报告和归档由 Runtime 维护，普通实现仅在 Build 写入。

### Scenario：固定 tag 融合并本地交付

Acceptance: A1, A12

给定干净基线和固定 peeled commit，当完成融合与独立验收，则 Git 祖先、VERSION、分支及差异证明边界成立；main 合并后无未解决冲突、无远端写操作。

## R2：冲突与无冲突交集都按能力融合

初步只读 merge-tree 返回 25 个冲突路径，上游差异 200 个文件、6699 行增加/541 行删除，fork 与上游代码交集 87 个文件。Build 应在实际 merge 后复核清单，文本自动合并的同文件及共享调用者仍需审查。禁止整文件 ours/theirs 覆盖业务定制，禁止删弱测试使检查通过。

主要冲突分组：VERSION；Ent generated usage schema/mutation/runtime；usage insert/query 与 request type/session 测试；Responses handler；Gateway scheduling；Messages/Chat fallback/body policy；orphan/tool schema；WS ingress/relay；Claude CLI version；AccountUsageCell 测试。

先合并 `backend/ent/schema/usage_log.go` 中 fork 流量字段与上游 cache TTL 字段，再复用 `backend/ent/generate.go` 入口生成；核对 insert 参数、CopyFrom/批量形状、query columns/scanner、DTO、聚合/导出与前端类型同时包含双方字段。生成物差异可重复，禁止冲突标记进入生成入口。

### Scenario：上游 cache TTL 字段与 fork 流量字段同时持久化

Acceptance: A2, A8

给定同一 usage 同时包含新 TTL 桶和流量计费字段，当写入、批量写入、按 request/session 查询和 DTO 展示，则字段对应准确，无 SQL 列数/占位符/扫描错位，既有用量详情和导出继续可用。

## R3：调度、sticky 与资格

保留 fork 现行 default/weighted 及 upstream 高级调度、TopK/权重、错误率/TTFT 和人工恢复。退休的是 fork `gateway.openai_ws.scheduler_mode=layered`、`scheduler_layered.*` 及其后台健康探测链；旧键保持惰性，不删除存量数据。上游当前 `decision.Layer`、候选 `openAISelectionProbeBudget` 等不属于退休功能，不因名称批量清理。

统一资格约束覆盖候选、snapshot、DB fallback/recheck、sticky/session_hash、previous_response_id、guardian、WaitPlan、抢槽后检查、同账号 replay 和 failover；保留 privacy、公开模型、能力/尺寸、quota、平台 sticky 开关、利润保护和 ModelTrace。CountTokens 虽不计费用，仅豁免相应利润门，仍执行业务客户端限制。

OpenAI OAuth 的客户端路由限制继续以原始请求识别 verdict 为准，只有符合账号 `codex_cli_only` 和 `extra.codex_cli_only_route_only` 现行类型语义时施加对应策略；管理创建/编辑/批量/重授权字段往返完整。网关补头、身份或 page context 改写不能制造客户端资格。

Gateway 与 OpenAI 的 hydration 失败释放规则保留 fork `acquired && release != nil`；取得槽位则只释放一次，成功结果回调仍交调用方。Responses session seed 新 `systemPrefixOpen` 与 fork prompt seed 增强并存，首 user/tool 历史后的提醒不改变初始会话绑定。

融合 OpenAI capacity shed/model cooldown：请求级容量不足不污染账号健康，规则仅匹配真实 error 字段，明确模型 cooldown 优先于受限同账号重放，保持 OpenAI 平台范围。Antigravity Pro Argon cooldown 独立于其他模型/套餐。

### Scenario：sticky 账号不符合原客户端限制

Acceptance: A3, A4

给定 sticky/previous-response 指向不符合原始客户端 verdict 的账号，当新请求或 token-count 到达，则跳过该账号并依现行路径选择合法账号，不依赖后续网关补出的 Codex 标志，不惩罚该账号。

### Scenario：hydrate 失败和 Responses 工具循环

Acceptance: A3

给定已获得槽位后 hydrate 失败，则释放恰一次；未获得不误释放。给定同一原始 system/user 开头、后续工具或 system reminder 变化，则保持种子稳定；不同 fork prompt 仍可区分。

## R4：输入准入与原始请求边界

Responses 原始 body 先作 `utf8.Valid` 检查，非法字节在任何替换/JSON 归一化前返回 400，错误消息沿用上游 `Request body must be valid UTF-8 JSON. Send UTF-8 encoded bytes.`。有效 Unicode 不受影响。

原始客户端 verdict、session seed 在 page/compact/bootstrap/身份改写前冻结。输入准入使用本轮实际有效输入；不能因兼容改写绕过安全检查。原始分组在 `ApplyEffectiveGatewayRoute` 替换 APIKey/Group 前保存，和最终组限额取非零最小值。阈值 0 不限，仅输入估算 > 阈值拒绝；配置默认关闭，缓存/原子保存/last-known-good 语义不变。

可见文本、历史、instructions、tools/schema/args/results 全部按现有 typed/gjson 字段语义统计；重复键及大小写/Unicode 折叠不得通过 map 重解析改变边界；`tools:null` 合法。opaque 媒体或引用仅标注覆盖未知，仍累计可见候选；不下载或解密来估算，不减 output/cache tokens。

HTTP Messages/Chat/Gemini 返回各自协议 400；Responses 非流 400，stream 用 200 `response.failed`/`context_length_exceeded`。检查发生在 keepalive/状态提交前。WS 首轮与每个后续 `response.create` 都执行，失败事件后按现行 policy close；真 compact/count 豁免不能靠伪造 header 获得。拒绝不调用上游、不重试、不扣费、不改变账号健康。

### Scenario：复合分组回退与混合输入

Acceptance: A4

给定原始组较严、回退组较宽及部分 opaque + 可见超限输入，当路由到回退组，则按冻结原始组拒绝；重复键、Unicode、结构化工具和 WS 后续轮使用相同有效字段语义，拒绝在写上游与 keepalive 前发生。

## R5：协议策略、密文载体与请求体生命周期

Messages/CountTokens/Chat fallback 整体融合 effort 策略：显式原始值校验 → 转换允许延迟 synthesized effort 验证 → 应用映射/ceiling/thinking 策略 → 最终验证。原始非法 none/minimal 不被映射隐藏；可合法降级的 synthesized 值不提前拒绝。保留 fork 最终模型、协议选择与计费 reasoning 上下文一致；GPT61 Sol 清除 Temperature/TopP 等上游规则生效。

新增 `encrypted_function_args`、namespace 与 accumulator 的缺省/null/空数组/非空值往返，和 fork encrypted_content lineage/recovery 并存。跳过 `ccswitch-openai-reasoning-v1:` foreign envelope 时保留正常工具历史及原生密文；不恢复已取消的其他健康统计特例。Gemini thought 与正文分离，Antigravity 连续 functionResponse 后追加媒体，兼容错误策略保留账号配置。

有效出站 body handle 在最终 model/协议/effort 与 DeepSeek reasoning_content 归一化后生成；大体积原始 body 提前释放，所需计费/入站字节数、imageSize、inflight estimate 等标量先捕获。1MiB spool、256KiB 预览/二进制脱敏、异步快照、retry/fallback/media 可重放和所有权规则保留。spool/读失败不退回全内存；所有 success/error/cancel/panic 路径释放 handle/预算/临时文件，非空 response+error 关闭 Body；专用 timeout/client-close 分类优先于通用 failover 错误归一化。

自动预算 `max(memoryLimitBytes/4, 1GiB)` 仅在配置和非空环境变量均未指定时应用；显式 0 仍关闭限制，配置解析优先级按上游。physical/cgroup/GOMEMLIMIT 探测使用上游范围限制，不扩大 filesystem 读取面。预算是资源准入，不改变 fork 出站 body 所有权或输入长度限制。

### Scenario：映射、重试与大 body

Acceptance: A5

给定映射至不同模型/协议的大请求并发生重试，当等待上游时，则出站包含最终归一化结果且可回放，原始完整 []byte 不常驻；用于收费/尺寸/估算的标量在释放前保存。成功、timeout、取消、panic 与 response+error 后资源可证明释放。

## R6：WS 每轮行为与图片

WS 保留 fork HTTP bridge、逐轮模型路由、资格/输入限额、密文恢复和流量快照。上游 TTFT 模式回调只对合法 first-token 事件计时，terminal 不提前视为 token。第二轮及以后失败不能将原始首帧交回 handler 重放，应 fail closed，沿用上游 1013 行为。OAuth pooled connection 已初始化后新启用 websearch，仅完整历史可重建时重连，不完整历史不得迁移/重放到其他会话。

携带 `X-Codex-Image-Turn-Id` 的原生图片请求跳过 JSON whitespace keepalive，确保真实错误状态未被预先提交为 200；普通图片 keepalive 行为与 fork client/size/concurrency/billing 路由保留。图片 SSE 含 CR/LF JSON 仍 compact 单行，前端裸续行容错与既有图片/视频工作台、鉴权、远程资产校验不回退。

### Scenario：第二轮 WS 失败与后来启用搜索

Acceptance: A6

给定已完成第一轮，第二轮发生不可恢复错误，则不会重发首轮 payload。后来启用搜索时，完整历史可安全重连，不完整历史停止；所有路径保留逐轮准入、模型映射与失败收费语义。

### Scenario：原生图片返回真实错误

Acceptance: A7

给定原生图片 turn header 与上游 4xx/5xx，当等待和转发结果时，不先发送 whitespace 200；返回真实状态/错误体。并行 tool results 的 functionResponse 连续、媒体追加顺序正确，既有图片尺寸路由和 SSE 消费保持。

## R7：TTL 定价与 fork 收费/订阅

新增 usage `cache_creation_30m_tokens`/`cache_creation_12h_tokens`，渠道模型与时间区间及账号统计两套表的可空 `cache_write_12h_price`，贯穿读取/更新/复制、可用渠道/模型广场/DTO/前端 UI。保持 Anthropic 5m/1h 与 OpenAI 30m/12h 桶分离；12h 显式单价优先，generic 显式配置和上游 GPT5.6/6 默认 2.5 倍规则按上游 resolver 实际契约执行，区间与账号统计同样使用对应价格，null 不当作 0。Sonnet5.5 cache-read 0.10、free-input/Fast multiplier 与 Fable/Mythos5.2 catalog/presets/Bedrock/OpenCode/Fast 计价同步。

fork 流量费保留：实际输入 S>T 才按 `S*2/1e6*P` 收费，S==T 为0，不加模型/账号/分组倍率；仅用户费用与余额/订阅/Key额度纳入合计，不进入账号成本或缺省成本回退。仅完整成功收流量费，partial/client-disconnect/upstream-error 免流量费但保留应有模型账单；每请求/WS turn 冻结配置/同意；异步/batch 排除，Grok realtime 按现行整连接成功规则计一次。以当前 migration253 后的实际新旧用户默认值为准，不重演旧 spec 默认关闭的历史描述。

订阅保留 exactly-one 可预支窗口、`>= max(limit-1,0)`、仅重置所选窗口、剩余期限足够、并发 rowlock/事务/receipt 幂等、缓存 invalidation outbox。多窗口/状态变化错误不变；自动预支需 group allow+auto 和订阅 auto 同时开启，续期保留开关；simple modes 现行语义不改。

### Scenario：混合 TTL + 流量费与部分失败

Acceptance: A8

给定30m/12h缓存写入、显式或默认价格与超过流量阈值请求，当完整成功时，各费用/成本/额度按对应来源分别计算且 UI/导出一致；传输部分失败不收流量费、不把 traffic_actual_cost 计入账号成本回退。

### Scenario：订阅自动预支与并发重试

Acceptance: A8

给定唯一合格窗口和三重开关，达到额度后仅推进该窗口，重复/并发请求只有一个有效变更，缓存一致；多个窗口、期限不足、状态变化和开关关闭保留原拒绝语义与前端提示。

## R8：迁移兼容与不可变历史

现行 runner 的 `schema_migrations.filename TEXT PRIMARY KEY` 按完整文件名标识，trim 内容后算 SHA256，按文件名排序并逐个检查未执行文件，无“只跑最大版本以上”的假设。故原样增加 `249_add_usage_logs_api_key_cost_index_notx.sql`、`250_openai_cache_write_ttl.sql`，与 fork `249_group_exclusive_rate_models.sql`、`250_user_cyber_overflow_exempt.sql` 及已存在251–253共存；不重编号，不改历史文件/checksum/已有记录，不加宽泛豁免。

249 并发创建 `(api_key_id, created_at) INCLUDE(actual_cost)` 覆盖索引，保留 `_notx.sql` 执行和 invalid-index 检测、DROP/重试入口。250 增加两个非空默认0用量列与四张定价表可空 `NUMERIC(20,12)` 12h单价，不破坏旧数据和 null 回退。

实测覆盖：空库全量安装；用 fork 基线全部迁移初始化并保留历史 checksum/样例行，然后升级；第二次执行无重复数据/记录变化；无效索引恢复；新字段、索引有效性和记录完整文件名核对。复用现有 migration FS/isolated DB harness；只在有覆盖缺口时补最小集成测试。

### Scenario：已有253记录的 fork 数据库新增同号上游迁移

Acceptance: A9

给定 fork 旧迁移均已执行且有样例数据，当使用合并后的迁移集启动，则上游249/250按完整名补跑、本地同号迁移保持已执行，历史checksum和数据不变；再启动幂等，索引有效，空库也成功。

## R9：其他上游行为与前端

完整吸收 EasyPay 每实例 refund=0/1 及配置回显、Antigravity forced usage query、原子 refresh token Redis GET+DEL 单次消费、并发 refresh 拒绝重用、审计 publish 后即时本地唤醒且保留多副本/延迟重试轮询、pricing catalog hash retry 与 CLI2.1.321。接口新增同步所有实现/stub，安全错误不能退化为继续签发。

前端 TTL 单价/模型展示、Keys 统计异步独立加载和错误恢复、账号用量强制刷新不卸载既有行、设置浮动保存与 dirty/saving 状态、充值六位精度和既有 fork 模型/限额/流量/订阅入口可用。采用双方现有控件和风格，不借融合重设计界面，不删字段和本地中文/英文翻译。

### Scenario：Token 并发轮换和异步统计

Acceptance: A10, A11

给定相同 refresh token 并发交换，则单次消费只有一个成功。给定 Keys 列表已返回而统计较慢或失败，页面先可用并能恢复统计；账号刷新保留当前行，浮动保存展示真实 dirty/saving，渠道空12h与显式12h价格往返准确。

## R10：验证计划与独立交付

检查选择以现有脚本为准，失败必须定位后在 Build 修复；正式候选冻结后由 Runtime 再执行约定计划。以下是拟执行检查，不代表已通过：

1. 后端 `go test ./...`、`go test -tags unit ./...`、`golangci-lint run ./...`、`go build ./cmd/server`（必要 embed/frontend 标签按现有入口）。Windows 优先现有 `backend/scripts/test.ps1`；共享 `.test-tmp` 的调用串行。新旧定向测试必须出现实际用例，不以 `no tests to run` 通过。
2. `go test -tags integration ./internal/repository/...` 使用可用 WSL Docker 与显式 `SUB2API_TEST_POSTGRES_IMAGE=postgres:16`，包含新 TTL、迁移、成本查询/traffic、订阅与 Refresh Token Redis 场景；其他涉及修改的 integration packages 按实际标签执行。外部真实上游凭证测试如无凭证则明确跳过且不声称端到端通过。
3. 定向分组：调度/sticky/资格与退休配置；输入准入和 body retention；Messages effort/CountTokens/密文；WS later-turn/search/TTFT；native images/媒体；TTL/free-input/Fast/traffic/subscription；payment/audit/refresh。
4. 前端 `pnpm test:run`、`pnpm typecheck`、`pnpm build`（含 i18n 检查）；浏览器本地实测渠道TTL/区间/展示、Keys加载、账号用量、设置保存、充值精度及 fork 关键入口。仅凭构建成功不替代浏览器。
5. 静态边界：Git祖先、VERSION、冲突标记/`git diff --check`、历史 migration diff、Ent 重生成可重复性、保留发布脚本测试与四段版本/单linux-amd64规则。

临时实例开始前记录原有容器/监听端口，给本轮资源明确归属，结束后逐一停止/移除并证明本轮端口关闭；现有 repository TestMain 最后调用 os.Exit，不能依赖之前注册的 defer 自动 Terminate，执行器须做归属核对后的外部清理。不能停止用户原有实例。

新的独立只读 Verifier execution 逐项覆盖 A1–A12，调查代理不替代正式 Verifier。失败/超时/跳过/未运行均保留真实状态；失败先由 Runtime 回 Build，不能在 Verify 改实现。组长验收后按本轮授权、Runtime 接受/归档与本地交付入口完成 main 合并，记录提交、命令和风险，并完成 Comet learning check；只记录实际验证过的可复用知识。

### Scenario：真实检查、独立验收与资源清理

Acceptance: A12

给定已冻结候选与确认文档，当 Runtime 检查和独立 Verifier 完成，则报告包含每项真实结论、日志/证据、候选绑定和剩余限制；测试容器/服务明确停止、端口关闭；组长验收后本地 main 包含固定双方提交且版本保持不变。
