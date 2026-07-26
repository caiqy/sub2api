## MODIFIED Requirements

### Requirement: 按正式 release tag 分段集成
维护流程 SHALL 允许将一个最终上游 release 目标拆为具有严格祖先顺序的多个正式 tag 阶段。每个阶段 MUST 完成冲突处理、能力审查和阶段验证后，才能进入下一阶段。

#### Scenario: 顺序合入多个 tag
- **WHEN** 用户选择按 `v0.1.160`、`v0.1.161`、`v0.1.162`、`v0.1.163`、`v0.1.164`、`v0.1.165` 分段集成
- **THEN** 维护流程 MUST 按该顺序建立独立 `--no-ff` merge 节点，且不得跳过尚未完成验证的前置阶段

#### Scenario: 从已验证但未归档的中间 release 继续扩展
- **WHEN** 一个分段合并 change 已通过中间 release 的最终验证但尚未归档，且用户将目标扩展到后续正式 tag
- **THEN** 维护流程 MUST 保留已完成任务和验证报告作为历史证据，使旧验证结果对新增范围失效，并在追加 merge 前重新运行基线与能力映射门禁

#### Scenario: 某阶段首次出现本地能力回归
- **WHEN** 阶段验证发现阶段 0 已保护的本地能力不再成立
- **THEN** 维护流程 MUST 在当前 release 区间内保留失败证据并完成最小修复，不得继续合入下一 tag

### Requirement: 合并后验证本地关键能力
维护流程 SHALL 在每个分段 merge 后运行完整分段门禁（后端默认/unit/lint、前端 lint/typecheck/Vitest、前后端构建、Docker-backed integration、生成代码稳定性、migration 新库与升级库兼容性、冲突标记检查）及该阶段受影响能力的能力级审查，并在最终阶段执行完整自动验证和本地能力专项 review。测试通过 MUST NOT 替代能力级审查结论。

#### Scenario: 分段 full 门禁通过
- **WHEN** 一个目标 tag 的 merge、冲突处理和兼容修复完成
- **THEN** 维护流程 MUST 运行根目录 `make test` 与 `make build`；在 Linux Docker 环境中按 `backend/scripts/test.ps1` 的等价语义重建 `backend/.test-tmp`、设置 `TMPDIR`/`TMP`/`TEMP` 并运行 `CI=true GOFLAGS='-v' go test -tags=integration ./...`；同时完成 Ent/Wire 两次生成稳定性检查、migration 新库与已有本地记录升级路径、冲突标记检查，以及该 tag 触及能力的映射审查，全部通过后才能进入下一阶段

#### Scenario: 最终自动验证通过
- **WHEN** 最终目标 tag 合并完成且无冲突残留
- **THEN** 维护流程 MUST 运行后端默认与 unit 测试、后端 lint、前端 ESLint、前端单测、前端类型检查和构建验证

#### Scenario: 本地关键能力专项 review
- **WHEN** 自动验证完成
- **THEN** 维护流程 MUST 逐项复核 scheduler、各平台 sticky、fallback/WaitPlan、DB recheck、privacy、image capability、异步图片任务与对象存储、图片输入计费、上游计费倍率、会话绑定与 step-up、runtime setting 热更新、网关透传字段、请求体重放与清理、用户资源控制、分组复制、用户批量限额、前端本地功能、版本依赖、生成代码和 migrations，并记录每项证据

#### Scenario: 新增上游能力与本地定制交互专项审查
- **WHEN** 目标 release 区间引入触及调度、路由或网关转发核心路径的新上游能力（如 composite group routing、OpenAI Live gateway）
- **THEN** 维护流程 MUST 审查该能力入口调用链与本地定制（advanced/layered scheduler、Grok/platform sticky、prompt cache reuse、body replay）的交互，并记录不被绕过或改写的证据

#### Scenario: 同号不同名 migration 兼容
- **WHEN** 上游新增 migration 与本地已发布 migration 使用相同数字前缀但文件名和用途不同
- **THEN** 维护流程 MUST 保留双方完整文件名与既有校验和，验证迁移执行器在空库和已应用本地 migration 的升级库上正确执行全部文件，不得仅因数字前缀重复而重命名历史 migration

#### Scenario: Integration 运行环境不可用或目标测试被跳过
- **WHEN** Docker/Testcontainers 运行环境不可用、工具链前置不满足，或本阶段要求的 migration/repository integration test 未实际执行并通过
- **THEN** 当前阶段 MUST 记录为阻塞且 MUST NOT 记为门禁通过，也不得进入下一 release tag
