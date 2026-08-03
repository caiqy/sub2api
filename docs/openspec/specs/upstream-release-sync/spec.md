# upstream-release-sync Specification

## Purpose
约束一次上游 release 合并从目标确认、隔离分支、冲突处理到验证和语义 review 的完整维护流程。
## Requirements
### Requirement: 确认上游合并目标
维护流程 SHALL 在合并前确认本地当前版本、upstream 最新 release tag、以及目标分支或 tag 的选择理由。

#### Scenario: 选择 upstream release tag
- **WHEN** upstream 存在比本地当前定制版本更新的 release tag
- **THEN** 维护流程记录目标 tag，并说明为何不默认使用 `upstream/main`

### Requirement: 在隔离分支执行合并
维护流程 SHALL 从干净的本地主线创建临时分支执行上游合并，除非用户明确选择其他隔离方式。

#### Scenario: 创建临时合并分支
- **WHEN** 本地 `main` 干净且已确认目标 upstream tag
- **THEN** 维护流程在临时分支中执行合并，不直接改写 `main`

### Requirement: 合并前建立本地能力保护门禁
维护流程 MUST 在首次上游 merge 前验证当前本地质量门禁，将本地能力映射到现有行为测试，并为上游目标触及且缺少保护的高风险本地能力补充最小回归测试。该门禁未通过时 MUST NOT 开始上游 merge。

#### Scenario: 当前本地基线稳定
- **WHEN** 维护流程尚未合入首个目标 tag
- **THEN** 后端与前端既定本地质量门禁 MUST 在当前本地 `HEAD` 上通过，或将既有失败明确标记为阻塞

#### Scenario: 高风险本地能力缺少行为断言
- **WHEN** 本地独有能力所在路径被目标 release 修改，且现有测试不能断言该能力的关键行为
- **THEN** 维护流程 MUST 在首次 merge 前添加可复现的最小回归测试

### Requirement: 按正式 release tag 分段集成
维护流程 SHALL 允许将一个最终上游 release 目标拆为具有严格祖先顺序的多个正式 tag 阶段。每个阶段 MUST 完成冲突处理、能力审查和阶段验证后，才能进入下一阶段。

#### Scenario: 顺序合入多个 tag
- **WHEN** 用户选择按 `v0.1.166`、`v0.1.168`、`v0.1.169` 分段集成，且版本序列中不存在正式 `v0.1.167` tag
- **THEN** 维护流程 MUST 按三个实际正式 tag 的顺序建立独立 `--no-ff` merge 节点，不得为无正式 tag 的版本建立虚构阶段，也不得跳过尚未完成验证的前置阶段

#### Scenario: 从已验证但未归档的中间 release 继续扩展
- **WHEN** 一个分段合并 change 已通过中间 release 的最终验证但尚未归档，且用户将目标扩展到后续正式 tag
- **THEN** 维护流程 MUST 保留已完成任务和验证报告作为历史证据，使旧验证结果对新增范围失效，并在追加 merge 前重新运行基线与能力映射门禁

#### Scenario: 某阶段首次出现本地能力回归
- **WHEN** 阶段验证发现阶段 0 已保护的本地能力不再成立
- **THEN** 维护流程 MUST 在当前 release 区间内保留失败证据并完成最小修复，不得继续合入下一 tag

### Requirement: 保留上游更新和本地定制
维护流程 MUST 在冲突处理和无文本冲突的语义审查中优先保留上游修复和本地定制能力。仅当用户在了解行为差异后明确批准某项本地能力移除时，维护流程 MAY 将其登记为例外；其他无法共存的语义 MUST 暂停等待用户确认。

#### Scenario: 冲突能力可以共存
- **WHEN** upstream 更新和本地定制修改同一文件或调用链但行为可以同时成立
- **THEN** 合并结果 MUST 同时保留上游更新和本地定制语义

#### Scenario: 用户明确批准能力移除
- **WHEN** upstream 仅部分覆盖本地能力，且用户在获知缺失范围和行为差异后仍明确选择完全采用上游
- **THEN** 维护流程 MAY 删除该本地能力，但 MUST 在 proposal、delta spec、任务和验证报告中记录例外范围

#### Scenario: 未批准的能力不能共存
- **WHEN** upstream 更新和本地定制存在不可共存语义，且该能力不在已批准例外中
- **THEN** 维护流程 MUST 停止自动处理并请求用户选择保留策略

### Requirement: 合并后验证本地关键能力
维护流程 SHALL 在每个分段 merge 后运行所选验证配置要求的自动门禁及该阶段受影响能力的能力级审查，并在最终阶段执行完整本机自动验证和本地能力专项 review。测试通过 MUST NOT 替代能力级审查结论；用户明确选择本机验证配置时，未执行的 Docker-backed integration MUST 作为残余风险保留，不得伪装为通过。

#### Scenario: 分段自动验证通过
- **WHEN** 一个目标 tag 的 merge、冲突处理和兼容修复完成，且用户已选择本机验证配置
- **THEN** 维护流程 MUST 运行根目录 `make test` 与 `make build`、受影响能力聚焦测试、Ent/Wire 两次生成稳定性检查、冲突标记检查和能力映射审查，全部通过后才能进入下一阶段

#### Scenario: 本机 Docker-backed integration 可用
- **WHEN** 本机 Docker/Testcontainers 运行环境可用
- **THEN** 维护流程 MUST 在本机运行 integration，并验证 migration 新库与已有本地记录升级路径；要求的 migration/repository integration test 未实际通过时 MUST 阻塞当前阶段

#### Scenario: Integration 运行环境不可用或目标测试被跳过
- **WHEN** 用户已明确选择仅本机验证，且本机 Docker/Testcontainers 不可用或目标 integration 无法执行
- **THEN** 维护流程 MUST 记录未执行命令、环境原因和受影响契约，将其列入阶段及最终报告的残余风险，并 MAY 在其他本机门禁通过后继续下一 release tag；维护流程 MUST NOT 使用远程服务器补跑，也 MUST NOT 将 integration 记录为通过

#### Scenario: 最终自动验证通过
- **WHEN** 最终目标 tag 合并完成且无冲突残留
- **THEN** 维护流程 MUST 运行后端默认与 unit 测试、后端 lint、前端 ESLint、前端单测、前端类型检查和构建验证

#### Scenario: 本地关键能力专项 review
- **WHEN** 自动验证完成
- **THEN** 维护流程 MUST 逐项复核 scheduler、各平台 sticky、fallback/WaitPlan、DB recheck、privacy、image capability、异步图片任务与对象存储、图片输入计费、上游计费倍率、会话绑定与 step-up、runtime setting 热更新、网关透传字段、请求体重放与清理、用户资源控制、分组复制、用户批量限额、订阅额度周期提前重置、前端本地功能、版本依赖、生成代码和 migrations，并记录每项证据

#### Scenario: 新增上游能力与本地定制交互专项审查
- **WHEN** 目标 release 区间引入触及调度、路由、网关转发、安全审计或 repository 更新语义的新上游能力
- **THEN** 维护流程 MUST 审查该能力入口调用链与本地 advanced/layered scheduler、Grok/platform sticky、prompt cache reuse、body replay、统一 prompt audit 和 subscription quota cycle reset 的交互，并记录不被绕过或改写的证据

#### Scenario: 最终审查发现 Images 审计入口重复与关闭态大 payload 构造
- **WHEN** 生产依赖图中的 unified security-audit coordinator 已包含 legacy content moderation，而 OpenAI Images handler 仍直接调用 legacy moderation，或仅按依赖指针决定是否序列化 prompt/image payload
- **THEN** Images 请求 MUST 只经统一审计入口执行 legacy/prompt 审核，legacy moderation 对每个请求最多执行一次；审核 payload MUST 在线程安全 provider 中最多冻结一次，并只在有效 prompt audit 或完成运行态与范围判定后的 legacy moderation 确实需要时求值，同时保持在请求文本释放前可用

#### Scenario: 同号不同名 migration 兼容
- **WHEN** 上游 `191_passkey_credentials.sql` 与本地已发布 `191_subscription_quota_advance_receipts.sql` 使用相同数字前缀但文件名和用途不同
- **THEN** 维护流程 MUST 保留双方完整文件名与既有校验和，并在可用的 integration 环境中验证迁移执行器在空库和已应用本地 migration 的升级库上正确执行全部文件，不得仅因数字前缀重复而重命名历史 migration

#### Scenario: 从未合入默认分支的已验证分支发布测试版本
- **WHEN** 用户明确授权从已通过 full verify 的隔离分支发布本地四段式 tag，并将 CI 镜像更新到指定测试服务器验收
- **THEN** Release workflow MUST 正常产出精确版本二进制、checksum 与镜像，但 MUST 在同步默认分支 VERSION 前验证 tag commit 是默认分支 HEAD 的祖先；不满足时 MUST 跳过同步，禁止产生 VERSION-only 主线提交
- **AND** 测试服务器 MUST 只拉取 CI 发布的镜像，更新前 MUST 记录旧 image digest 并备份测试数据库，更新后 MUST 验证 health、revision/migration 与关键页面；不得在服务器构建 Sub2API 镜像或把失败烟测记为通过
