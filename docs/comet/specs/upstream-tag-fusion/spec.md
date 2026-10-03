# 完整目标规格：冻结上游版本与 fork 行为融合

本规格覆盖 v0.2.9、v0.2.10、v0.2.11 全部已批准目标，不因当前轮仅推进 v0.2.9 而省略后续行为。用户已批准完整Shape、隔离B和A1-A8。需求来源逐项完整覆盖保存在 brief.md 的 Source coverage。

## R1：执行环境与 Native 契约

2026-10-03补充修复的有效执行边界：用户明确选择当前DSH会话，允许宿主原生edit工具，继续使用公开Comet Runtime与新的独立只读Verifier。不把原CLI、apply_patch或原自动hook证据当作当前宿主证据；下列原CLI环境和审批条款仅保留为逐tag历史要求，本次补充修复由此段取代其宿主/编辑方式/自动hook前置条件。不修改持久安全、信任、隐私或hook配置，不申请权限升级；SDK使用vfox exec显式版本执行。失败、未执行和环境差异必须如实披露。本轮仅修合并/上下游交互回归，已确认的三版目标及A1-A8不缩减；修复后重新独立验收，用户另行接受前不归档，不合并main、不推送、不发布或部署。

开发与验证必须在指定官方普通本地 Codex CLI 同一会话中执行，使用本机 Comet Native skill、公开CLI/runtime和已审核全局hook。指定CLI为 C:/Users/caiqy/AppData/Local/OpenAI/Codex/bin/a51e250fa15c740a/codex.exe，模型 gpt-6.1-sol/high，已配置上下文450000（宿主已核验effective427500），Ponytail full。保留Comet/Ponytail和disable_response_storage配置；CLI忽略配置项等实际警告如实记录，不能宣称隐私选项已生效。用户亲自完成项目信任，不改变信任、安全、隐私、hook、插件或持久配置。

显式Native优先，不运行ambient resume-probe，不切换workflow。先读AGENTS与Native Skill，按阶段读引用。状态、事务、locks、检查证据和报告由Runtime管理；Agent只能编辑 Runtime new 注册返回路径内的 brief、完整Spec、需要时delta及children。Runtime模板与continuation中的版本/动作/candidateId/executionRef原样保留。需求完整正式落盘后才能记录已明确批准Shape。最终用户接受前不能accept-result/archive，当前阶段不能task --complete。

本轮官方 --approve-for-me/workspace-write，仅为必要单条命令申请require_escalated，说明已批准vfox/Comet/Git本地操作；拒绝必须给精确理由，禁止bypass、danger-full-access、手改PATH、SDK目录重定向、脚本包装或安全设置修改规避。正式源码修改前必须取得当前CLI自动调用已审核本地hook的实际证据；配置存在性、手工router调用、历史验证不代替。可以合法正式产物apply_patch写入核实，不恶意探测、不读凭证/受限日志。证据不确定时停止源码开发，明确阻塞。

## R2：基线、隔离和冻结版本

重新核验main干净、HEAD bd70137ac903031f8bf24efb29476aed8ff31665、backend/cmd/server/VERSION 0.2.8.5，保护用户修改。origin为https://github.com/caiqy/sub2api.git，upstream为https://github.com/Wei-Shaw/sub2api.git。先确认无分支冲突，再由 native new --isolation branch --change-branch comet/merge-upstream-tags-20261002 --target-branch main 创建绑定，使用同目录独立分支，保留main。

仅允许三个正式上游tag，必须核验远端真实tag对象并commit peel：

| 上游tag | 精确peeled commit | fork VERSION |
| --- | --- | --- |
| v0.2.9 | 4c00df2e0183e2c70b7fa8ba45914205e36aad0c | 0.2.9.1 |
| v0.2.10 | 2f3fed2fdb0787141294cec81487a5df30426f7f | 0.2.10.1 |
| v0.2.11 | 96f4c115c9749078f90cbf210a01d39baf3f53b6 | 0.2.11.1 |

fetch使用 --no-tags 和独立refs/upstream-tags命名空间，不覆盖本地tags，不扩展后来tag。三版按顺序各自独立--no-ff merge，第二父精确对应冻结commit。每版全门禁、实际独立审查、commit及bundle/checkpoint闭环之后才开始下一tag。v.11上游VERSION漏升也必须明确fork0.2.11.1。真实未经批准业务冲突停报，不机械ours/theirs。

不push/release/deploy/mainmerge、强制reset、改写历史、接受或归档。初始化成功立即写真实checkpoint。v.9现已由协调者实际独立只读报告关闭，本续轮推进v.10；其实现、候选门禁与最终完整bundle完成后交接协调者实际只读审查，只有该版实际关闭才能接续v.11。未完成版本不算通过。如果等待独立审查/Verifier，只交接实际任务包，不虚构闭环。上述逐版续轮边界沿用用户完整已批准Shape，不缩减三版目标或A1-A8。

## R3：SDK与本地图

SDK安装、选择、升级统一vfox。backend/go.mod最低1.27.0且无toolchain；没有额外本地精确锁定时直接已有1.27.1。backend-ci.yml的go version grep go1.27.0精确断言属于其setup-go CI步骤，不构成本地toolchain锁定；保留CI现状。默认1.26.3不能构建本项目。必须官方scope选择并应用官方env，核验当前及后续真实子进程go version，不用手改PATH冒充。Node现有vfox22.23.1，corepack可用，前端pnpm按真实项目锁定/CI要求提供，不无端升级依赖。

已有CodeGraph1.6.0、官方ScoopWindows bundle、全局command=codegraph args=[serve,--mcp]；serve隐藏但有效。不重装或扩大agent集成。约539MB本地图由1.5.0生成，按授权使用公开 codegraph index 仅重建.codegraph。先CodeGraph后grep定位源码。所有相关进程DO_NOT_TRACK=1，遥测和后台更新检查关闭，保持全局配置不变。图只读本地源码写本地图，不外传源码或备份。

## R4：保留 fork 能力与历史参考

保留调度、sticky、freshDB、ModelTrace、OpenCode、请求体可重放与生命周期、审计、计费、设置及前端定制，不因上游融合删除任何fork能力。参考历史Git安全融合及main first-parent 058207f04、b4629e894、13e3b408f对应分支内部正式上游merge。历史云端v.9/v.10工作已丢失，v.11未做，不假设旧SHA恢复或现存历史实现等于当前通过。所有本次门禁/安全测试/独立审查必须实际执行。

## R5：WS安全切换与模型身份映射

WS允许同平台且当前account支持的安全模型切换，吸收upstream公共模型准入、精确account-model ownership，以及response/usage身份。不得照搬所有模型变更强制重连。public->route->channel->account映射链保留最初public身份和逐turn不可变快照，group allowlist与ownership不能被任一映射绕过。接受生成、usage/billing、调度与安全校验都使用对应有效身份，不能把后续session变化写回已冻结turn。

## R6：Moderation与Sonnet beta

Legacy allowlist接入fork真正CheckLazy入口。命中allowlist的Legacy检查保持log-only审计，不产生ban/hash/email/counter副作用；独立PromptGuard仍拦截。Sonnet5.5 beta过滤在fork passthrough overrides全部应用之后重新执行，覆盖多值header和spooled body，不因请求体生命周期/映射绕过。

## R7：生成式依赖接线

Wire同时保留upstream ClaudeResetCreditService与fork ModelTrace/OpenCode/setting/effective+composite resolver。通过官方生成命令生成，不手改生成文件。最终Ent/Wire生成两轮内容稳定，完整构建通过。

## R8：v.11预留额度、计价和异步生命周期

在接受的实际生成/模型变化时，预留额度原子增长为max(previous,newEstimate)；session.update单独不占额；降价或重试不累计。过期额度不可复活，必须重新reserve；异步任务持旧句柄，不得释放新预留额度。必须真实测试nil/unpriced->priced、cancel/close/race/refcounts、queue失败/panic及mandatory billing回退。

保留fork小型scalar body快照，不重新持有已释放原body。异步usage exactly-once release发生在实际扣费及缓存同步之后，确保异常、并发及取消路径无漏放、重放或提前释放。估算使用effective route/group/subscription/platform/account/channel，以及冻结PricingAt和reasoning倍数，保持每turn上下文一致。

## R9：v.11其他完整发布能力

APIKey creation计数删除不重置，默认200/60及Redis失败策略按真实上游行为吸收并测试。GPT6.1Sol校验在最终mapping之后执行，保留转换前的客户端意图校验，并在fork账号字段规则、兼容转换及最终模型/请求体覆写全部完成后、出站前复用共享校验。通过公共HTTP入口和本地fake upstream验证：最终Sol请求的none/minimal推理选择必须本地拒绝且不出站，不创建账号失败样本；合法推理请求与其他模型继续按既有行为转发。AstraUltrafast为6倍且独立于Fast。Claude reset保留幂等租约/审计，remote catalog身份与fallback完整保留。与fork融合不能丢失这些上游行为或任何fork能力。

## R10：允许的最小测试修复与第三方边界

本轮只修由合并、冲突处理或上下游交互新增的回归；冻结upstream原样缺陷和未受合并影响的fork存量问题不修。API Key并发存量上限、余额缓存不确定扣减重试、原队列关闭竞态与存量前端问题均排除。额度引用仅在证明合并新增不安全影响后才修，缓存同步失败时停止handler续期并保留至TTL的既有保守策略不改变。

实际复现时允许独立commit以下最小清理：现存8个ModelTrace文件静态检查101项的行为保持修复；TestOpenAIWSConnPool_TargetConnCountAndPrewarmBranches使用现有capture fake dialer替代example.com真实连接，断言DialCount=1和closed；GeminiOAuthService_RefreshAccountToken_CodeAssist_NoProjectID_FailsEmpty用内存RoundTripper返回projects[]并保留错误断言；TokenRefreshService_RefreshWithRetry_Antigravity缓存fixture privacy_mode=AntigravityPrivacySet，避免真实Google设置请求；usage detail unit stub适配新增metric参数。

未知凭证或合成token不能发送真实Google等第三方。遇拒绝不换路径重试。不能为绿灯跳过测试、弱化断言或把未跑/失败/超时标通过。风险回归真实red->最小共享根因修复->green，保留完整调用路径理解；不增加无请求抽象或依赖。Windows限制实际复现，旧Linux socket限制不自动套用。

## R11：检查、独立审查、日志和恢复

按仓库根/backend Makefile官方Windows scripts/test.ps1执行。每版backend lint/default/unit/build，default含golangci-lint；每版frontend lint:check/typecheck/test:run/build，build含check:i18n。限制重Go任务并行，开发期定向检查。最终完整计划填builder-handoff.verification_checks，Runtime冻结候选后执行，不提前重复完整最终计划。每版明确用户要求的全门禁仍逐版执行。

最终还要release-tools unittest、migration静态、Ent/Wire两轮稳定、VERSION/拓扑/用户文件检查和fresh只读独立Comet Verifier。Verifier首Runtime动作verifier-started绑定原样candidateId/executionRef，独立判断全部A1-A8，复用只限匹配当前候选的Runtime正式检查。Builder不能自行声称独立review或验收通过。逐版非正式审查也需实际独立审查者。

可以在Runtime任务包到达后，将原样任务包、continuation、候选及检查路径写audit/verifier-dispatch-needed.json交协调者启动独立只读CLI或原生subagent；该文件不作state/report，不算已启动、已通过或用户接受。需求只有runtime确实需要且符合Skill条件才拆Supervisor，不擅自改隔离。

所有新审计日志、checkpoint、每tag实际测试日志、commit记录及可恢复bundle仅存C:/Users/caiqy/Documents/Codex/2026-10-02/task-2/sub2api-merge-audit，不向父目录写新文件、不外传。main已有完整历史bundle及历史验证只作背景：sub2api-main-bd70137ac-before-merge.bundle，SHA256 38D57AAB349F141B2CA7384B9546B0E06141669D158DC17E9789A9FF68577A8E。每tag另存当前commit、实际日志和bundle。关键节点主动短报冲突、门禁通过/失败/未测、备份路径、下一步；阻塞提供准确命令与证据。

无Docker/DB/Redis则明确integration未测范围，不能宣称全通过。所有环境/服务排除真实透明。项目知识只保存当前验证后可复用经验；任务摘要、进度、测试日志不写个人记忆。当前轮未完成整体任务，不task --complete。

## R12：最终验收

### A1：精确三版拓扑
只有按R2三个精确冻结commit依序形成三个独立no-ff merge，每个第二父经核验匹配，才通过。仅v.9完成时其余待执行。

### A2：每版独立闭环
只有每版真实全门禁与独立只读审查、可恢复commit/bundle/checkpoint完成后进入下一tag，才通过。任务包、登记或Builder自述不算独立审查已完成。

### A3：完整吸收上游
R2三版正式发布功能完整吸收，包含R5-R9全部应保留行为，不以fork融合为由遗漏，才通过。

### A4：完整保留fork
R4原有安全/调度/sticky/freshDB/body/replay/lifecycle/billing/ModelTrace/前端能力，以及R5-R10已批准融合语义均有当前候选证据，才通过。

### A5：真实风险回归
安全/WS/mapping/ownership/body/moderation/billing风险留真实red->minimal root fix->green；未知第三方token不外发；未批准真实业务冲突停报，已决事项不重复问，才通过。

### A6：正式检查与fresh验收
R11每版全部门禁及最终工具/migration/Ent/Wire两轮/版本拓扑用户文件检查真实执行，Runtime冻结候选绑定正式证据，新的只读独立Verifier先started后独立覆盖全部A1-A8，才通过。

### A7：本地交付边界
最终VERSION0.2.11.1留在独立本地分支，main基线保留，未push/release/deploy/mainmerge/reset/history rewrite/accept-result/archive，才通过。

### A8：来源、环境和执行真实性
R1正式普通本地CLI、真实自动hooks和vfox选择证据可靠；Runtime独占state/locks/report/checks；完整来源及Shape批准落盘；未运行或失败如实标识，服务/平台限制透明；R11安全本地备份与项目知识边界符合要求，才通过。
