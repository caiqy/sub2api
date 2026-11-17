# Outcome

将 upstream 正式 tag `v0.2.22` 全量融合到 fork，保留当前本地定制，完成验证与审核后合回本地 main。

固定输入：
- fork 起点 `7b6a81d8610d5969d3f0e58293a472960a65fb9a`，main 初始干净。
- upstream tag object `b11573c9d55d39b6780e1e9f6e193ba2e6ca2321`，peeled commit `b0fa9f3e6eeab1968a53e1289b69daa4bb40ca2b`。
- 已合入基准 v0.2.21；仅新 tag v0.2.22，排除 tag 后 VERSION 同步提交及后续发布。
- 实施分支 `comet/merge-upstream-v0-2-22`，以独立 no-ff merge 保留历史；最终 VERSION 保持 `0.2.21.2`。

授权：用户要求“照惯例合并上游新版本”，群内已有明确长期授权：没有需要用户取舍的冲突，文档完成后直接执行，不再询问开工。沿用该授权；仅无法共存的用户可见语义冲突升级确认。组长直接承担全部实施；必要独立只读审核不承担实现。不将历史授权扩展为推送、发布或部署授权。用户已针对 A10 时序循环明确回复“修正并继续收尾”，确认将验收及交付准备作为归档前门槛，归档后本地 main 合入仍为必做交付，并授权复验通过后接受结果、归档和本地合入。

# Scope

上游 v0.2.21..v0.2.22 涉及 340 文件（+20708/-1652），merge-tree 预检 28 个冲突文件。文本自动合并仍需语义复核。一个 change 串行实施：各模块共享网关/模型/计费契约，不拆 Supervisor，避免集成成本。

完整引入上游：Gemini 4 Carbon/Hydrogen、Vertex 多区域端点与默认价格乘数、并行工具调用/结构化输出/原生粘性与缓存用量、Ultra Fast 与 Fast 模型门控、OpenAI 透传模型过滤及快照字段、快照加载合并与冷却恢复、按用户计费通道与批事务、请求体热点零拷贝、Anthropic/Responses/Chat 桥接观察与 effort、keepalive 与首输出失败换号、工具 schema/Codex 上下文与加密工具数据、token 轮换重试、配额快照和用量统计、Bedrock 错误及渠道兼容配置、稳定退款幂等、HTML 禁缓存、连接保活以及其他 tag 内功能与测试。

保留 fork：现行高级调度和统一资格/利润保护、原始 Codex 客户端识别、复合分组路由与输入上下文限制、请求体磁盘承载和显式副本租约/错误传播、流量计费与事务幂等、订阅额度窗口/自动提前重置、图片视频、模型映射、管理界面、四段式版本及单入口发布。既有 layered/probe 移除决定持续有效。新增只读 view 如需 UI 测试沿用已有预览配方。

## 目录结构

### 修改
- `backend/internal/service/`：融合网关、模型、调度快照、计费通道、配额与错误语义；所有重叠调用者保留 fork 约束和租约生命周期。
- `backend/internal/handler/`、`backend/internal/server/`：协议入口、错误形状、keepalive、客户端与输入准入；管理接口与 DTO 同时保留双方字段。
- `backend/internal/repository/`：快照投影/缓存、批量计费与用量聚合、费率事务；保留 fork 流量费/幂等/锁顺序及历史数据行为。
- `backend/ent/schema/group.go`、`backend/ent/`：保留 fork 分组字段并加入 Ultra Fast 字段，按现有入口生成。
- `backend/cmd/server/wire.go`、`backend/cmd/server/wire_gen.go`：接入计费通道，保留 fork 服务及 shutdown 清理，按现有 Wire 入口生成。
- `backend/internal/pkg/`、`backend/internal/payment/`、`backend/internal/config/`、`backend/internal/web/`、`backend/resources/`：完整融合协议/支付/配置/模型价格/HTML 缓存变更。
- `frontend/src/`：账号配额/白名单、Ultra Fast、Bedrock 渠道配置、用量展示和测试，保留 fork 输入限制/流量费/订阅及管理入口。
- Go 工具链及 CI/Docker/Makefile：融合 Go 1.27.4 安全升级，保留 fork 发布入口、四段式与单架构矩阵。
- 上游 tag 其他文档、资源和测试：保留双方可共存内容。

### 新建
- 上游新增的服务、测试与两份不同完整文件名的 251 迁移，保留完整文件名为主键语义。
- 仅针对真实交叉风险补必要回归，复用现有 harness；证据保存 `output/merge-upstream-v0-2-22/`。

### 明确不建
- 不新建工程师 Bot、计费/调度第二套抽象、迁移重编号方案或新发布入口；不恢复已移除 layered/probe。

# Non-goals

不推送、不发布、不部署、不操作生产；不修改历史已执行迁移字节或 checksum；不纳入 tag 后提交；不把本轮扩展成业务重设计；不停止用户已有服务，只清理本轮可证明归属的资源。

# Acceptance examples

- [ ] A1 固定范围与拓扑：候选含 fork 起点和精确上游 peeled commit，以 no-ff merge 保留两侧历史，排除 tag 后提交；最终 VERSION 为0.2.21.2，无远端写操作。
- [ ] A2 完整融合与生成：28个冲突逐项解决，重叠自动合并语义复核；Ent同时保留 fork 和 Ultra Fast 字段，Wire计费通道及 shutdown 正确；双方测试不因冲突被删除或减弱。
- [ ] A3 请求体与协议：上游零拷贝不使租约释放后别名逃逸；80MiB/1GiB真实Handler换号回归保持通过，结束预算归零，真实耗尽仍503/Retry-After；错误不吞掉，归一化后出站handle可回放，取消/超时清理正确。Codex上下文、sealed内容、tool schema、桥接effort及响应模型观察保留。
- [ ] A4 调度与粘性：新快照合并/投影与模型过滤保留统一资格/利润/客户端约束及冷却恢复；sticky/previous-response/WaitPlan/故障切换继续正确，不恢复layered/probe。
- [ ] A5 计费：同用户通道/批事务保持逐条Apply语义、幂等和失败回滚；fork流量费、订阅窗口、额度重置、费率/RPM作用域与用量统计不回退；Ultra Fast门控与收费一致，Vertex仅默认价加10%，显式价格覆盖保持。
- [ ] A6 网关及其余上游：keepalive后错误形状、首输出前换号和输出后副作用、空流释放粘性、Bedrock错误/usage、Gemini并行工具/结构化输出/多区域/粘性、token轮换重试、配额快照/退款幂等/禁缓存及连接保活均有对应检查。
- [ ] A7 数据安全：历史迁移字节不变；两份251按完整文件名独立执行；真实数据库检查空库/已有fork升级/重复执行/分组Ultra Fast字段与usage聚合、批事务幂等回滚；保留fork数据和迁移校验。
- [ ] A8 前端：相关测试、typecheck、i18n/build通过；实际浏览器核验Ultra Fast/Bedrock/账号配额与白名单并检查fork管理能力，类型检查不冒充浏览器。
- [ ] A9 检查与审核：后端全量测试及适用unit/integration、构建与静态检查有真实命令和结果；独立只读审核覆盖关键交叉风险，失败/未执行/跳过如实记录，修复后仅补失效检查。
- [ ] A10 验收与交付准备：Native独立验收覆盖A1–A9；候选提交及本地main基线已核对，候选工作树干净；交付所需提交、改动、有效证据和剩余限制已备齐；本轮资源清理有据可查，测试端口已关闭。归档及main合入按下方强制交付顺序执行，不将尚未执行的交付动作报告为已完成。

# Local delivery

1. A1–A10复验全部通过后，按用户“修正并继续收尾”的明确授权接受结果并由Native Runtime完成归档；复用仍适用的原始检查，不因时序澄清重跑全量测试。
2. 归档完成后，仅提交本change的正式产物，将已验收候选及该归档提交合回本地main。核对main包含候选与精确上游peeled commit、工作树干净、历史迁移字节不变及VERSION仍为0.2.21.2。
3. 汇报实际main提交、改动、有效证据和剩余限制；只有本地Git交付实际完成才宣布本任务完成。不得推送、发布或部署。

# Open questions

无待用户决定事项；若实施发现用户可见语义无法共存则登记并提问。
