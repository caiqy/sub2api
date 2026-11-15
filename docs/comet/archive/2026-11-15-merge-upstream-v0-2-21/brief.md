# Outcome

将 upstream 正式 tag `v0.2.21` 的完整能力融合进 fork，保留当前本地定制及既有 layered/probe 移除决策，经独立验证和研发组长验收后合回本地 `main`。本次不发版；版本文件保持 `0.2.20.8`。

固定输入：
- fork/main 起点：`1cf5be61d9864a73428fdb8407030cc31bed2a1a`，初始工作树干净。
- upstream tag peeled commit：`10ba8f9b4dc2e6023d4a9879ab77b21841babd10`；annotated tag object：`43f15294c477bec427d586ddfc3f918ce1284c98`。
- merge-base / 已包含上游：`v0.2.20`，`dab829a4e9b7d1cc307189aea8b0319eb29e29d0`。
- 实施分支：`comet/merge-upstream-v0-2-21`；唯一 Native change：`merge-upstream-v0-2-21`。
- `upstream/main` 在 tag 后的两个提交不纳入；以固定 commit 进行 no-ff 融合。

决策与授权：用户本轮明确授权“如果没有什么冲突需要我做决定的话，你们文档写完就可以开始执行，不用问我”。完整文档经组长复核后进入 Build；文件冲突按双方能力并存解决，仅有无法兼容的用户可见业务语义时升级确认。目前未发现必须二选一的业务冲突。

# Scope

覆盖 tag 的 200 个文件差异（+6699/-541）与 fork 的实际调用链；初步交集 87 个文件、只读 merge-tree 发现 25 个文件冲突。不能将无文本冲突视为无语义冲突。

上游能力包括 Claude Fable/Mythos 5.2、模型与 reasoning effort 兼容、OpenAI 30m/12h cache-write 用量与定价、Antigravity 强制用量查询与工具媒体、EasyPay 实例退款开关、请求体内存预算自适应、API Key 异步统计与索引、账号界面稳定性、设置浮动保存、充值精度、安全审计即时唤醒、Refresh Token 原子轮换、WS TTFT/重连/后续轮故障处理、Responses 会话种子/桌面页面上下文/UTF-8 校验、图片原生错误状态及模型冷却等修复。

fork 保留范围：现行高级调度、统一资格筛选与利润保护、Codex 原始客户端识别、复合分组路由、输入上下文限额、请求体磁盘承载与释放、协议映射、图片/视频工作流、流量计费、订阅额度窗口与自动提前重置、管理界面和四段式发布约定。既有 Spec 只作参考；历史 Shape 中的待确认文字和已被后续迁移取代的默认值不重演。

## 目录结构

### 修改

- `backend/internal/service/`：融合调度/粘性、OpenAI HTTP/WS、协议转换、缓存 TTL/模型定价、计费、Token 轮换与账号用量等上游改动，保留 fork 公共约束。
- `backend/internal/handler/`、`backend/internal/server/`：保持输入协议错误形状、准入与权限顺序，融合 UTF-8/page context、原生图片路由、渠道/模型接口。
- `backend/internal/repository/`：融合 usage 字段、SQL 占位符/扫描顺序、渠道定价、索引恢复与 Refresh Token 原子操作。
- `backend/ent/schema/usage_log.go` 与 `backend/ent/`：合并 fork 流量字段和上游 cache TTL 字段后按现有 Ent 入口重生成，不选择整套 ours/theirs。
- `backend/internal/config/`、`backend/internal/pkg/`、`backend/internal/securityaudit/`、`backend/internal/payment/`、`backend/resources/`：融合配置预算、协议/模型目录、审计和支付变更。
- `frontend/src/`：融合渠道 TTL 价格及公开展示、账号用量稳定性、设置保存提示、Keys 异步加载、充值精度、模型目录和相关测试，保留 fork 管理与计费/订阅/输入限制功能。
- `deploy/config.example.yaml`、`deploy/EDGE_SECURITY.md` 和 tag 中其他文档：保留双方可共存说明；修正引用仅限本次融合需要。
- 与上述路径对应的已有单元/集成/前端测试：合并双方断言；只补充实际交叉风险缺口。

### 新建

- `backend/migrations/249_add_usage_logs_api_key_cost_index_notx.sql`：原样引入上游 API Key actual_cost 覆盖索引，保留非事务并发建索引和无效索引恢复。
- `backend/migrations/250_openai_cache_write_ttl.sql`：原样引入上游 30m/12h 用量、渠道四张定价表的可空 12h 单价。
- 本 change `specs/upstream-tag-fusion-20261114/spec.md`：完整合并行为和本轮验证契约。
- 必要的最小跨能力回归测试；优先复用现有测试/集成 harness，不建新框架。

### 明确不建

- 不新建调度抽象、第二套 eligibility、迁移重编号方案、全局 checksum 豁免或新发布入口。
- 不恢复上轮移除的 fork layered/probe 运行链、前端配置入口和其专属状态；保留上游正常高级调度、TTFT、错误率与人工恢复能力。

# Non-goals

- 不发布、不打发布 tag、不推送远端、不部署、不改生产数据或数据库。
- 不更改 `backend/cmd/server/VERSION` 的最终值 `0.2.20.8`；不纳入 tag 后提交。
- 不修改、删除、重命名任何已执行的历史迁移，不改数据库现有 checksum，不扩大兼容白名单。
- 不把本轮合并扩成业务重设计或无关清理；不根据字符串 probe 批量删除上游功能。
- 不停止用户既有 PostgreSQL/Redis/服务；仅清理本轮可证明归属的测试资源。

# Acceptance examples

- [ ] A1 固定目标与边界：最终历史包含上述 fork 起点和 peeled commit，排除 tag 后提交；最终 VERSION 为 `0.2.20.8`，无远端写操作。
- [ ] A2 完整融合：25 个冲突逐一按行为解决，87 个重叠文件完成交叉复核；Ent 同时保留 fork 和上游字段，SQL 插入/批量/查询/DTO 无列或参数错位，双方测试不因冲突被删弱。
- [ ] A3 调度与客户端资格：default/weighted、候选/快照/DB 回退、sticky、previous-response、WaitPlan、故障重试及 token-count 均服从现行资格/利润/客户端规则；hydrate 失败已取得槽位只释放一次；上游 session seed 与 fork 增强并存，不恢复 layered/probe。
- [ ] A4 输入准入与安全：原始分组与最终分组限额取非零较小值，原客户端识别在重写前完成；HTTP/Responses streaming/WS 首轮和后续轮错误正确且先于 keepalive；重复键/Unicode/opaque 可见混合输入不绕过，拒绝不扣费、不惩罚账号。无效 UTF-8 返回 400。
- [ ] A5 协议和请求体：最终映射/effort/DeepSeek 归一化后生成有效出站 body handle；跨重试可回放，上游等待不持有完整原始大 body；读取失败不退回全内存；timeout、cancel、panic、非空 response+error 均正确关闭/释放。budget 缺省自适应、显式 0 保持关闭限制。
- [ ] A6 WebSocket：TTFT 事件口径配置生效；后续轮失败不重发首轮 payload，按上游 1013 行为退出；websearch 仅完整历史允许重连；每轮保留 fork 输入限额、资格、映射、流量快照与失败语义。
- [ ] A7 图片与工具：原生图片成功/错误状态及 keepalive 正确，媒体引用和 tool 结果顺序、encrypted_function_args/namespace、Gemini thought 与正文分离完整；复合分组映射、image 工作流/鉴权/计费保留。
- [ ] A8 TTL 与收费：30m/12h 字段贯穿用量/渠道/区间/账号统计/DTO/UI；12h 缺省定价和显式 generic/12h 配置按上游优先级，free-input/Fast/cache-read 修复生效；fork 流量费只计完整成功且不进入账号成本回退，订阅窗口、事务/幂等/缓存失效保持。
- [ ] A9 迁移安全：完整文件名为主键、trim 后 SHA256 保持；同号不同名的新迁移独立执行。空库安装、已有 fork 至253升级、重复执行、非事务索引无效状态恢复均有真实检查，历史迁移字节不变，数据不被回填破坏。
- [ ] A10 其余上游行为：EasyPay 实例 refund=0/1、Refresh Token 并发单次消费、安全审计即时唤醒、Antigravity 用量/限流、模型目录和 effort policy 有对应回归结果；原有鉴权安全与 fork 业务测试继续通过。
- [ ] A11 前端验收：测试、typecheck、i18n/build 成功；浏览器实际核验渠道 TTL/价格、Keys 统计、账号用量、设置保存与充值精度，同时检查 fork 输入限制/流量费/订阅入口没有回退；不能用类型检查替代浏览器。
- [ ] A12 检查与交付：后端全套测试、必要集成、lint 和前端检查记录真实结果；新的独立 Verifier 覆盖全部验收项，失败/跳过不记通过；组长验收后按 Runtime 完成本地 main 合并及学习检查。测试数据库/服务停止后检查本轮端口关闭，交付提交、证据及剩余风险。
