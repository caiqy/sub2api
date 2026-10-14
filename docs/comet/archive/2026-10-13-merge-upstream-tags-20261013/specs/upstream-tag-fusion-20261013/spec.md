# 上游 v0.2.16 融合完整目标规格

## R1：冻结范围与版本历史

在 comet/merge-upstream-tags-20261013，从 main 基线 5fa3d16a5c7453df25096e6c0faa9c3b47ff526a / VERSION 0.2.15.5 开始。远端稳定 tag 调查确认唯一未包含版本 v0.2.16，tag 对象 c3cb8588ce6a5aef4dc29fa4f71ac91f3c0bdf3d，commit 01842576b3d5c1e2545b5a99ad555d36e28ed306。系统 SHALL 以独立 no-ff merge 保留两侧历史，第二父精确对应冻结 commit；后续 tag 或 upstream/main 未冻结提交不进入范围。

### Scenario: 冻结发布成为可核对祖先

Acceptance: A1

WHEN 检查稳定 tag 清单、基线与最终拓扑
THEN v0.2.16 是候选祖先，独立 merge 第二父为冻结 commit，已包含稳定版本无遗漏，主线基线可追溯。

## R2：Gemini 显式缓存与供给能力

完整融合 cachedContents 网关入口、AI Studio/Vertex/自建上游支持、缓存身份和授权边界、到期失效、账号删除与暂不支持缓存的区分。缓存修改使用并发租约、串行化及失败回滚，确定金额费用按 API Key/订阅预留；缓存存储内置官方价并按 Pro/Flash 回退，延长使用创建时计费口径；上游缓存不存在的 403 不导致错误停用或换号。新入口同时遵守 fork 用户平台额度、预占释放、mandatory billing、日志和请求体生命周期。融合 Vertex API Key/服务账号模型同步/详情、AQ. 格式 Key、第三方与 Vertex 不套用 AI Studio 模拟额度、国际 Coding 地址额度识别及工具结果图片。

### Scenario: 缓存管理与 fork 结算共存

Acceptance: A2

WHEN 创建/更新/延长/删除缓存、遇到并发/失败/过期/账号失效，或通过 Vertex 和 Gemini 兼容入口发送请求
THEN 缓存生命周期、授权和计费一致，失败不泄漏预占或错误停用账号，fork 额度守卫仍有效，模型同步和工具图片行为正确。

## R3：模型、限流和协议回退

融合临时不可调度规则 scope 与上游模型传递，使模型级规则在故障转移、计数、Bedrock、重试耗尽等路径生效；保留 fork scheduler/sticky/freshDB 与 runtime block。Antigravity 图片模型限流按模型隔离。支持 Claude Haiku 5.5 的模型/采样/effort/协议桥，Sol 生成上下文默认 500k 且订阅取消长上下文附加费，其他计费口径不误改。Responses 结构化输入被拒时仅在安全条件下 Chat 回退，保留工具/历史/推理/流语义和重试边界；Codex 图片桥仅注入 OpenAI 账号。最终 public/route/channel/account 模型身份、请求校验、目录隔离、privacy/图片过滤及缓存/handle/重放/释放守卫继续成立。

### Scenario: 新能力不绕过既有网关保护

Acceptance: A3

WHEN 选择账号和映射模型、模型级失败切换、协议回退或图片桥转发
THEN 上游模型规则和修复正确生效，fork 保护与实际调用路径保持，安全回退不扩大重试或丢失请求语义。

## R4：前端和运维交互

分组名称与描述支持多语言、语言代码规范和 Vary/Origin，相关 DTO、展示、表单及复制一致。融合账号表单/用量/完整分组名、模型白名单、分组 RPM 过期请求/费率校验、监控新增平台/自动刷新、运维账号名/本地时区、倒计时 elapsed time/标签页恢复、IME 键盘及 Gemini 授权错误提示。保留 GPT Image 多图/蒙版/参数/历史工作台、流 keepalive 后 JSON 结果和 Base64 本地解码，保留官方输出 TPS、Codex 点数整数显示、用量详情/样式化 Excel、额度周期/自助开关和 ModelTrace 设置。冲突测试覆盖双方实际有效行为，不删除断言掩盖不兼容。

### Scenario: 上游交互修复和 fork 展示保持

Acceptance: A4

WHEN 编辑分组/账号、查看运维和用量、支付倒计时或生成/编辑图片
THEN 新交互与多语言有效，fork 近期修复、表格/导出和额度设置保留，相关测试覆盖冲突区域。

## R5：fork 不变量和安全

保留最终模型校验、调度/sticky/privacy/图片能力、预占释放/同步计费回退、ModelTrace、配置热更新、请求体生命周期、支付签名/回调/return_url/履约/返利/幂等、新安装随机凭据和 Unicode 缺省。部署资源、四段式派生版本解析、单 linux/amd64 发布矩阵及唯一 workflow_dispatch Release 入口不变。内容审核删除分组后的范围和其他上游安全修复融合。只修本次融合引入的回归；实际业务不可共存时提出明确选项并等待决定。

### Scenario: 语义专项复核保护核心能力

Acceptance: A5

WHEN 按入口、条件、回退与生命周期审查上游及 fork 交互
THEN 核心安全和本地能力有效，非平凡回归以可运行测试复现并修共同根因，未经确认不选择互斥业务行为。

## R6：候选检查和独立验收

开发期做定向检查；最终由 Runtime 冻结候选并执行后端 default/unit 全量、golangci-lint、embed 测试及构建，前端 lint:check/typecheck/test:run/build（含 i18n）。重检查串行并使用 Windows scripts/test.ps1。新只读 Verifier 独立核对 A1-A8、正式规格、实现与候选绑定证据。相关隔离 PostgreSQL/Redis 与浏览器条件具备时实际验证；未执行/skip/超时/历史成功不代替本次通过。临时 PostgreSQL 结束只清理测试实例并确认端口关闭；不发送真实第三方凭据。

### Scenario: 最终候选取得真实证据

Acceptance: A6

WHEN Builder 提交完整候选后 Runtime 和新的 Verifier 执行验收
THEN 必需检查实际绑定当前实现，全部验收项得到独立结论，服务/浏览器/第三方未覆盖范围明确。

## R7：版本、迁移和生成

融合 VERSION 为 0.2.16.1，不创建 tag 或发布。保留旧 SQL 全名、内容和 checksum；新增 242_group_i18n.sql、243_gemini_cached_contents.sql、244_gemini_cached_contents_channel_usage.sql、245_gemini_cached_contents_mutation_lease.sql 与 fork 同前缀迁移按完整文件名识别、排序，核对实际依赖和升级路径，不改名旧文件。Ent/Wire 使用官方命令从合并后的 schema/provider 生成并检查稳定性。Go 1.27.2、fork 安全依赖、CI/Docker 和发布工具一致；SDK 统一使用 vfox，不降级项目工具链。

### Scenario: 新旧迁移与生成输出一致

Acceptance: A7

WHEN 检查 VERSION、迁移输入/排序/checksum、Ent/Wire 生成和构建配置
THEN VERSION 为 0.2.16.1，旧迁移不变，新迁移安全共存，生成稳定且符合输入，Go 和 fork 安全/发布矩阵保留。

## R8：工作流和交付

仅 Agent 编辑正式 brief/spec；状态、检查、报告和事务由公开 Native Runtime 管理。实现留在绑定的独立分支并提交可追溯候选；完整独立验收后等待用户明确接受和选择分支收尾。推送/PR/Release/部署和清理无关工作区不在范围。任务真正结束记录经验证的可复用项目经验并完成学习检查，不将任务日志写成个人偏好。

### Scenario: 可审查成果后选择交付

Acceptance: A8

WHEN 全部验收已有独立结论
THEN 用户可核对候选与报告并选择接受及收尾，未经授权不推送/发版/部署，真正完成时记录学习检查。
