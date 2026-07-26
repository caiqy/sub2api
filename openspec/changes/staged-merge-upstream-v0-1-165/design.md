## Context

- 本地 `main` 已包含上游 `v0.1.159`，`backend/cmd/server/VERSION` 为 `0.1.159.6`，并叠加长期本地定制；不能用 `v0.1.159..HEAD` 的提交数代表本地修复数量，因为其中包含长期分支与归档历史。
- 上游新增 6 个 release tag：`v0.1.160`(24c/133f) → `v0.1.161`(62c/257f) → `v0.1.162`(114c/190f) → `v0.1.163`(69c/171f) → `v0.1.164`(43c/202f) → `v0.1.165`(54c/168f)。
- 六个 tag 构成严格祖先链；当前 `upstream/main` 比 `v0.1.165` 多 1 个 `chore: sync VERSION to 0.1.165` 提交，因此目标锁定正式 tag，而不是 `upstream/main`。
- 既有经验（`2026-07-17-staged-merge-upstream-v0-1-159`）：逐 tag `--no-ff` + 分段门禁 + 能力矩阵审查是可行范式；"测试通过 ≠ 本地能力未被改写"，必须做能力级专项 review。
- 高风险交互区：v0.1.164 的 composite group routing 触及调度/路由核心（与本地 advanced/layered scheduler、Grok sticky、platform sticky 定制交叠）；v0.1.165 的 OpenAI Live gateway 触及 OpenAI 网关路径（与本地 prompt cache reuse、body replay 交叠）；12 个新增 migration 中，上游 172/181 与本地同号不同名，且上游自身存在两个 `186_*`。

## Goals / Non-Goals

**Goals:**

- 6 个上游 tag 全部按顺序合入，Git 祖先关系正确（`v0.1.165` 成为 HEAD 祖先）。
- 每段合并独立通过 full 门禁：根目录 `make test`、`make build`、Docker-backed integration、Ent/Wire 两次生成稳定、migration 新库/升级库兼容、无冲突标记。
- 本地保护清单内的每项能力在最终 HEAD 上仍然成立（能力级验收，不止文件级 diff）。
- `backend/cmd/server/VERSION` 最终规范为 `0.1.165.1`，全量 full verify 通过。

**Non-Goals:**

- 不推送远端、不打 tag、不触发 Release workflow、不部署（归档后另行发版）。
- 不恢复已移除的 `openai-first-token-timeout` 契约。
- 不在本 change 内新增本地功能或重构上游代码。

## Decisions

1. **临时分支承接合并**：全部 6 段在 build 阶段确认的隔离分支完成，final verify 通过后再决定合回方式；分支名与工作方式在 build 决策点联合确认。理由：隔离语义风险，沿用上次成功范式。
2. **逐 tag `--no-ff` 合并而非一次合 `v0.1.165`**：保留每段独立 merge 节点，冲突分摊、回溯粒度小；与用户"逐版本更新"要求一致。
3. **每段 full 门禁（升级自上次的轻门禁）**：每段执行根目录 `make test`（后端默认/unit/lint + 前端 lint/typecheck/Vitest）、`make build`（后端与前端构建），并执行 Ent/Wire 两次生成稳定性检查；失败即停在当段修复，不带病进入下一段。
4. **冲突处理默认"两边共存"**：机械 ours/theirs 禁止；先分类（上游修复/本地定制/接口演进/生成文件），生成文件（Ent/Wire/pnpm-lock）冲突以重新生成为准；无法共存的语义冲突暂停交用户决定。
5. **能力矩阵沿用并扩展**：以主 spec 的专项 review 清单和本地保护能力为行、6 个 tag 为列，每段合并后勾验受影响单元格；composite routing 与 Live gateway 两个新能力列入"与本地定制交互"专项审查。
6. **复杂行为问题走失败测试驱动**：涉及 scheduler、sticky、fallback、runtime config 的回归先补失败测试再最小修复，不直接猜改。
7. **保留同号不同名 migration**：迁移执行器按完整文件名记录并按文件名排序，因此默认同时保留本地和上游 172/181 以及两个上游 186 文件，不擅自重命名已发布 migration；通过新库和已有本地 migration 记录的升级库测试验证依赖顺序，失败时再做最小兼容修复。
8. **保持单 change**：六个 tag 是严格线性依赖，任一阶段失败都必须阻塞后续阶段，且共享同一能力矩阵与最终版本；拆成多个 change 只会复制门禁和上下文，不能独立交付或归档。
9. **Integration 使用 local-serv-ai**：本机没有 Docker；每段把已提交 HEAD 通过 `git archive` 打包，并由 `ssh-skill` 上传到 `local-serv-ai` 临时目录运行 `CI=true GOFLAGS='-v' make -C backend test-integration`。全套命令必须成功且目标 migration/repository test 必须出现真实 PASS；无关环境型 skip 单独记录。只拉取 PostgreSQL/Redis Testcontainers 镜像，不构建 Sub2API 镜像、不部署、不触碰服务运行目录。

## Risks / Trade-offs

- [composite group routing 改写调度入口，静默绕过本地 advanced/layered scheduler 定制] → 每段审查调度入口调用链；针对 Grok sticky + advanced scheduler 保留既有本地测试并要求持续绿灯。
- [OpenAI Live gateway 重构 OpenAI 转发路径，破坏本地 prompt cache reuse / body replay] → v0.1.165 段专项 diff 审查 + 定向测试。
- [同号不同名 migration 因排序或依赖关系在升级库失败] → 保留完整文件名，分别验证空库与已应用本地 172/181 的升级库；确认 `190_*_notx.sql` 非事务执行路径。
- [客户端 IP 体系与本地安全/审计定制交叠] → v0.1.162 段核对 settings JSON backfill 与配置热更新路径。
- [每段 full 门禁耗时高] → 接受；失败停段策略避免时间浪费在带病推进上。
- [release tag 内 `VERSION` 比 tag 名低一个三段式版本，可能误降本地版本] → 每段把版本元数据作为独立冲突决策记录，不用它推断 release 内容；最终统一设为 `0.1.165.1`。
- [远程 integration 环境缺少正确 Go 工具链、Docker 或目标 migration/repository test 未运行] → 每段远程执行前检查 Go 版本与 `docker info`，设置 `CI=true` 令整套 Docker 缺失路径失败，并在 verbose 日志中确认目标测试 PASS；任一前置不满足即阻塞。

## Migration Plan

1. 本 change 只集成源码，不执行生产部署或生产数据库 migration。
2. 每段在隔离分支验证 migration：保留所有已发布文件及校验和，通过 `ssh-skill` 在 `local-serv-ai` 临时目录运行空库和已有本地 migration 记录的升级路径；不得为消除编号重复而重命名历史文件。
3. 任一阶段失败即停在当前 merge 节点修复；尚未合入主线时可直接放弃隔离分支，不需要生产回滚。

## Open Questions

- 文本冲突和语义回归的准确数量只能在逐 tag merge 时确定；无法共存的业务语义仍须暂停交用户决定。
- 172/181 同号 migration 当前由完整文件名隔离，是否还需顺序兼容修复以升级库测试结果为准，不提前改写 migration runner。
