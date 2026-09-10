# 目标

将 upstream 正式 tag `v0.2.1`、`v0.2.2`、`v0.2.3`、`v0.2.4` 按版本顺序安全合并到当前 fork 基线，保留 upstream 行为与既有 fork 定制，并将本地版本闭合为 `0.2.4.1`。

# 范围

- 依次以四个 tag 的 peeled commit 执行四个独立 `--no-ff` merge，不跳过 tag，不直接合并滚动的 `upstream/main`。
- 每个 tag 都独立完成冲突与无文本冲突语义审查、受影响测试、完整本机质量门禁、构建和独立只读回归审计；发现的问题先以失败测试和最小修复闭合，前一版未闭合不得进入下一 tag。
- 覆盖模型 allowlist 与调度、OpenAI/Codex/Claude/Gemini/Ollama/MiniMax/Grok 协议能力、WebSocket 生命周期、计费与价格、迁移、代理与运行时设置、管理端与前端行为。
- 最终复核 fork 的关键定制，并按本轮实际经验决定是否增量更新 `memory/context/upstream-merge-workflow.md`。

# 非目标

- 不合并 `v0.2.4` 之后的 `upstream/main` 提交或后续 tag。
- 不重构与本次合并无关的 fork 代码，不主动改变既有 scheduler、sticky/fallback、DB recheck、请求体生命周期、审计、计费、运行时设置和插件边界。
- 不创建或推送 release tag，不触发 release workflow，不部署。

# 验收示例

- A1: Git 历史按顺序包含四个独立 `--no-ff` merge；其第二父分别为 `578785ee7fb35030b094b69624efe25670a36f5f`、`5485f368b29d05adb95a00f71801c7c23d8f48af`、`8fa67d477d6651a744754392a8982ea589c26ae6`、`5de5e2bed035d43591a2e10e51f420ef6a84eb98`，且最终结果不含 `v0.2.4` 之后的 upstream 提交。
- A2: 每版合并后、下一版合并前，均完成冲突标记检查、受影响能力测试、后端默认与 unit 测试、后端 lint、前端 ESLint、i18n、单测、类型检查、前后端构建和独立只读回归审计；任一失败都停留在当前 tag 修复并复验。
- A3: `v0.2.1` 的 channel/model 限制调度、GPT-6 Astra 与 Codex 能力、WebSocket 重放与状态、图片回填、计费价格、上游请求 ID 和相关管理端/前端行为成立。
- A4: `v0.2.2` 的 group model allowlist/simple mode、模型列表与 failover、请求体预读、WebSocket turn 生命周期、计费/支付/兑换/备份和相关前端行为成立。
- A5: `v0.2.3` 的 allowlist 修复迁移、Ollama Cloud DeepSeek token 限制、Anthropic Bearer 鉴权和连接测试模型展示成立。
- A6: `v0.2.4` 的 MiniMax 平台、HTTP/2 长流、OpenAI OAuth 429 冷却、跨实例 channel cache 失效、代理共享备份、系统日志保留、Image 2.5、Grok media eligibility 和相关前端行为成立。
- A7: fork 的 scheduler、sticky/fallback、DB recheck、请求体重放与清理、审计、每请求计费、运行时设置、插件边界、网关透传及前端定制语义保持成立。
- A8: `backend/cmd/server/VERSION` 为 `0.2.4.1`，且没有创建或推送 tag、触发 release workflow 或部署。
- A9: migrations、Ent/Wire 两次生成结果稳定；环境不支持的检查明确记录未覆盖范围；最终新的只读 Verifier 对全部验收项给出独立结论。
- A10: `memory/context/upstream-merge-workflow.md` 保留既有有效原则，并仅补充本轮实际验证出的可复用新经验；没有新经验时明确保持文件不变。

# 约束与不变量

- 只接受已获取并核对 peeled commit 的正式 tag，不使用滚动的 `upstream/main`。
- 每个 tag 必须形成独立 merge 边界，并在进入下一 tag 前完成冲突与重叠语义审查。
- 测试通过不替代能力级语义 review；审查围绕 fork 关键能力清单和边界路径进行，不只按冲突文件检查。
- 无法同时保留 upstream 行为和 fork 核心能力时暂停并请求用户决定。
- 任何生产代码回归修复必须先有针对该回归的失败测试。

# 决策

- 使用当前 `main` 工作区；创建 change 时目录干净且无其他 active Native change。
- 按 `v0.2.1`、`v0.2.2`、`v0.2.3`、`v0.2.4` 顺序合并，最终版本为 `0.2.4.1`。
- 沿用既有 multi-tag 同步流程的“每版先审计闭合、再推进下一版”门禁。
- 保留 fork 核心能力，完成质量门禁和独立审计，不发布或部署；最终按实际经验增量维护项目记忆。

# 待解决问题

- 无。

# 验证预期

- 每个 merge 前后检查冲突标记、merge 父提交、tag 祖先关系和下一 tag 之外提交的排除。
- 按 tag 审查 upstream 改动与 fork 定制的重叠调用链和边界路径，先执行受影响测试，再执行完整质量门禁、构建和该版独立审计；前一版验收闭合后才继续。
- 对 migrations、Ent 和 Wire 执行生成与稳定性检查；记录 Docker/Testcontainers、race detector 等环境限制，不将未运行项目记为通过。
