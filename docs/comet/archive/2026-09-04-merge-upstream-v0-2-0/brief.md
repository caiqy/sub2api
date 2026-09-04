# Outcome

将 upstream 正式 tag `v0.1.184`、`v0.1.185`、`v0.2.0` 按版本顺序安全合并到当前 fork 基线，保留 upstream 行为与既有 fork 定制，并将本地版本闭合为 `0.2.0.1`。

# Scope

- 依次以 `v0.1.184`、`v0.1.185`、`v0.2.0` 的 peeled commit 执行三个独立 `--no-ff` merge，不跳过 tag，不直接合并滚动的 `upstream/main`。
- 每个 tag 都独立完成冲突与无文本语义审查、受影响测试、完整本机质量门禁、构建和独立只读回归审计；发现的问题先以失败测试和最小修复闭合，前一版未闭合不得进入下一 tag。
- 覆盖本轮 upstream 引入的网关与 WebSocket 稳定性、模型目录、计费与价格覆盖、分组 Fast/reasoning policy、Kimi Responses、数据库重试、管理端与前端行为。
- 最终闭合后，将本轮新增且可复用的合并经验更新到 `memory/context/upstream-merge-workflow.md`，不重复已有原则。

# Non-goals

- 不合并 `v0.2.0` 之后的 `upstream/main` 提交或后续 tag。
- 不重构与本次合并无关的 fork 代码，不主动改变既有 scheduler、sticky/fallback、请求体生命周期、审计、计费和插件边界。
- 不创建或推送 release tag，不触发 release workflow，不部署。

# Acceptance examples

- A1: Git 历史按顺序包含三个独立 `--no-ff` merge；其第二父分别为 `e98ef32eb29aecd30d1def615912ec4dc93173f3`、`2ac784c51a5d0925b324efef2ba6b3446c364781`、`aa236488351eb71e120fc2b6fb32e36b0374c918`，且最终结果不含 `v0.2.0` 之后的 upstream 提交。
- A2: 每版合并后、下一版合并前，均完成冲突标记检查、受影响能力测试、后端默认与 unit 测试、后端 lint、前端 ESLint、前端单测、类型检查、前后端构建和独立只读回归审计；任一失败都停留在当前 tag 修复并复验。
- A3: `v0.1.184` 的网关、WebSocket、用量、模型目录、调度、计费、管理端和前端修复在合并结果中成立。
- A4: `v0.1.185` 的价格目录覆盖、长上下文数据驱动计费、数据库启动重试、Codex 能力与 WebSocket 容量处理在合并结果中成立。
- A5: `v0.2.0` 的分组 Fast/reasoning policy、Kimi Responses、fallback 清理、模型与定价 UI、调度快照及相关持久化迁移在合并结果中成立。
- A6: fork 的 scheduler、sticky/fallback、DB recheck、请求体重放与清理、审计、每请求计费、运行时设置、插件边界、网关透传及前端定制语义保持成立。
- A7: `backend/cmd/server/VERSION` 为 `0.2.0.1`，且没有发布动作。
- A8: Ent/Wire 两次生成结果稳定，环境不支持的检查明确记录未覆盖范围，最终新的只读 Verifier 对全部验收项给出独立结论。
- A9: `memory/context/upstream-merge-workflow.md` 保留既有有效原则，并补充本轮实际验证出的、后续逐 tag 合并可复用的新经验；没有新经验时明确保持文件不变。

# Constraints and invariants

- 只接受已获取并核对 peeled commit 的正式 tag，不使用滚动的 `upstream/main`。
- 每个 tag 必须形成独立 merge 边界，并在进入下一 tag 前完成冲突与重叠语义审查。
- 测试通过不替代能力级语义 review；审查围绕 fork 关键能力清单和边界路径进行，不只按冲突文件检查。
- 无法同时保留 upstream 行为和 fork 核心能力时暂停并请求用户决定。
- 任何生产代码修复必须先有针对该回归的失败测试。

# Decisions

- 使用当前 `main` 工作区；当前目录创建 change 时干净且无其他 active Native change。
- 本次按 `v0.1.184`、`v0.1.185`、`v0.2.0` 顺序合并，最终版本为 `0.2.0.1`。
- 沿用此前 multi-tag change 的“每版先审计闭合、再推进下一版”门禁，不把验证推迟到最终 tag。
- 保留 fork 核心能力，完成质量门禁和独立审计，不发布或部署；最终按实际经验增量维护项目记忆。
- 用户已确认上述目标、范围、关键决定、验收项和非目标。

# Open questions

- 无。

# Verification expectations

- 每个 merge 前后检查冲突标记、merge 父提交、tag 祖先关系和下一 tag 之外提交的排除。
- 按 tag 审查 upstream 改动与 fork 定制的重叠调用链和边界路径，先执行受影响测试，再执行完整质量门禁、构建和该版独立审计；前一版验收闭合后才继续。
- 记录 Docker/Testcontainers、race detector 等环境限制，不将未运行项目记为通过。
