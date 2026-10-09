# 逐版本上游融合完整目标规格

## R1：冻结范围与版本闭环

在分支 comet/merge-upstream-tags-20261009，从 main 基线 ce10e1e777653310629d651e4b867e3b713781bb / VERSION 0.2.14.2 开始。逐个核查 upstream 三段式稳定 tag 后，唯一待融合版本为 v0.2.15，tag 对象 86a80c13dcba86f52f9ca815b5a471cecc236227，commit f2669c8cf62555cd92389b3f55920e9e6e7c6ff2。独立 no-ff merge 的第二父精确对应该 commit，保留上游与 fork 历史。完成冲突融合、语义审查、实际检查和独立验收形成可恢复版本；调查后新 tag 不自动纳入。

### Scenario: 冻结发布版本成为可核对的祖先

Acceptance: A1

WHEN 检查基线、upstream tag 清单与最终 merge 拓扑
THEN 无遗漏稳定版本，基线保留，v0.2.15 为最终 HEAD 的祖先，merge 第二父精确对应冻结 commit。

## R2：供应商平台与校验边界

完整接入 provider profile、平台清单、Cline、Command Code，统一平台选择、上游端点、配额、账号探测、用量刷新、钱包冷却、限流和模型能力。前端账号/分组/配额/设置枚举、模型候选及同步入口与后端登记和支持能力一致。新增 242 SQL 移除 user_platform_quotas.platform 与 composite_model_routes.target_platform 的旧 CHECK 后，API、service、repository 与 Ent 继续使用权威平台清单拒绝未知平台；要求具体平台的写入不接受 composite 或未登记值。渠道监控 provider CHECK 表示实际探测能力，保持原约束。保留全部现有平台及 fork 配额语义。

### Scenario: 新平台可用且无效平台仍被拒绝

Acceptance: A2

WHEN 创建或修改账号/分组/配额/模型路由，检查模型候选与同步能力
THEN Cline/Command Code 按各自 profile 正确工作，既有平台兼容，未知或非具体平台在适用入口被拒绝，UI 与后端校验不分叉。

## R3：网关协议、目录和计费

完整融合三入站入口 resolveUpstreamProtocol、profile 端点、模型协议目录与账号映射隔离；以最终上游模型和账号选择协议，保留 fork public/route/channel/account 模型身份、请求校验、sticky/freshDB 和回退。保留 Claude 计费与钱包冷却、OpenCode Zen 路由/不支持模型/403/Retry-After、Grok 空完成故障转移和 SSO 修复、OAuth web_search 历史声明与 Responses Lite additional_tools、thinking signature、tool-change beta、cache TTL 顺序、Chat developer/legacy function/tool_choice/refusal 转换、心跳、WS 图片 usage 与当前分组定价。与请求体 replay/cache/handle/释放及 mandatory billing 同时成立。非平凡融合回归使用实际调用链最小测试复现再修共同路径。

### Scenario: 新协议路由不绕过 fork 守卫或结算

Acceptance: A3

WHEN 三类入站选择账号/映射模型、切换协议、故障转移或进入 WS 后续轮次
THEN 上游修复有效，目录隔离、最终模型校验、请求体重放/释放、调度及当前分组计费保持，有对应回归与源码证据。

## R4：前端与运维行为

融合平台驱动表单、紧凑筛选、每请求输出 TPS、监控请求样本健康评分以及异步请求过期/卸载/并发修复。保留 fork 用量表格布局与日志字段、样式化 Excel、平台额度表单及显示、额度周期锚点/按钮/分组自助开关、ModelTrace 设置、Codex 配置生成和点数去小数末尾冗余零。测试冲突保留双方有效断言；新增/替换断言必须对应明确业务行为，不靠删除测试隐去不兼容。

### Scenario: 上游 UI 修复与 fork 展示同时成立

Acceptance: A4

WHEN 编辑账号/设置/额度、浏览用量或运行异步交互与导出
THEN 新平台能力及上游生命周期修复有效，fork 展示、导出与额度周期语义继续有效，自动测试覆盖相应冲突区域。

## R5：fork 不变量与安全保护

保留 privacy、图片能力过滤、最终请求校验、额度预占释放、同步计费回退、ModelTrace、配置热更新、请求体生命周期及部署资源边界。保留 EasyPay 严格回调字段/签名复用防护、return_url 清理、支付履约/充值赠送/返利/退款/幂等及 QueryOrder 回退。保留 Web/CLI/环境变量新安装随机管理员凭据、既有部署跳过及 Go/浏览器 Unicode 缺省一致性。按入口、条件、回退和生命周期专项审查，未变动区域可结合基线 diff 与既有回归证明保留。仅修本次融合回归；双方实际产品语义无法共存时请求用户决定。

### Scenario: 新平台及网关重构保留既有安全语义

Acceptance: A5

WHEN 审查上游变动与 fork 核心交互及安全路径
THEN 原保护和本地增强仍生效，非平凡问题有可运行复现与最小根因修复，真正业务冲突未未经确认偏向一侧。

## R6：真实检查与独立验收

开发期运行定向检查。候选冻结后，由 Runtime 执行并绑定后端 default 全量、unit-tag 串行全量、golangci-lint、embed 测试与构建，以及前端 lint:check、typecheck、test:run、build（含 i18n）。后端沿用 Windows scripts/test.ps1，避免并发清理 .test-tmp。新的只读 Verifier 独立覆盖 A1-A8、真实源码和候选绑定证据，不把历史通过或 Builder 自述当验收。隔离 PostgreSQL/Redis 或目标浏览器环境具备时执行相关实测，缺失/跳过/超时明确记录；临时 PostgreSQL 仅清理本次实例并确认端口关闭，不停止原有服务；测试不发送真实第三方凭据。

### Scenario: 完成候选后真实验收

Acceptance: A6

WHEN Builder 完成全部范围并提交候选
THEN Runtime 执行实际完整门禁，新独立 Verifier 核对所有验收项，明确真实服务、浏览器与第三方覆盖边界，不将未执行记为通过。

## R7：版本、迁移、生成与工具链

融合 VERSION 为 0.2.15.1，仅作本地版本，不创建 tag 或触发发布。已有 SQL 全名、内容和 checksum 保持，新增 242 SQL 与 fork SQL 依赖顺序安全。Ent/Wire 使用官方命令核对与合并后的 schema/provider 一致及生成稳定，不能手改生成产物隐藏不一致。Go 1.27.2 与 x/net 等安全升级在 go.mod、CI、Docker、构建及检查一致；SDK 通过 vfox 安装和执行，不用无 hook 的 session 切换影响全局环境。核对 lint 工具兼容。四段式解析、单 linux/amd64 发布矩阵及 tag 后唯一 workflow_dispatch 入口保留。

### Scenario: 平台迁移及生成代码与 fork 兼容

Acceptance: A7

WHEN 核查版本、旧新 SQL、生成输入输出、Go 及发布构建配置
THEN VERSION 为 0.2.15.1，旧 SQL 不变，新迁移安全，Ent/Wire 稳定且与输入一致，Go 1.27.2 检查成立，fork 发布规则保留且未触发发布。

## R8：Comet 工作流与交付

正式 brief/spec 由 Agent 编辑，状态、检查、报告和事务通过公开 Runtime 管理。工作仅在 comet/merge-upstream-tags-20261009，另一 active worktree 保持原归属。最终独立验收通过后等待用户明确接受及选择收尾，成果与证据可追溯。推送、PR、发版、部署及清理其他工作区均不在范围。任务真正结束时记录经过验证的可复用项目经验及学习检查，不把任务进展、日志或测试结果写成个人偏好。

### Scenario: 完成独立验收后选择交付

Acceptance: A8

WHEN 全部验收项获得实际结论
THEN 用户可审查本地分支和报告并选择接受/交付，未经授权不执行推送/发布/部署，后续收尾与学习记录按公开 Runtime 完成。
