# Brainstorm Summary

- Change: staged-merge-upstream-v0-1-165
- Date: 2026-07-26

## 确认的技术方案

- 采用单 change、单隔离分支、六个 `--no-ff --no-commit` merge 节点；每段按冲突融合、能力审查、失败测试/最小修复、full 门禁、阶段封闭推进。
- 基线固定为 `main@075abc073`；实施前重新 fetch 并记录六个 peeled tag SHA。新增正式 tag 时暂停更新 change 范围。
- merge commit 只承载上游树与冲突解决，门禁发现的回归使用后续聚焦修复提交；阶段未封闭不得进入下一 tag。
- 六段期间保持 `backend/cmd/server/VERSION=0.1.159.6`，最终门禁后一次改为 `0.1.165.1`。
- 保留双方完整 migration 文件名与 checksum；不因 172/181 同号或两个上游 186 而重命名历史文件。
- 本地执行常规门禁；Docker-backed integration 在 `local-serv-ai` 临时目录运行。每段用 `git archive HEAD` 传输已提交源码，所有远程操作通过 `ssh-skill`，不构建 Sub2API 镜像、不部署、不触碰服务目录。
- 证据收敛为一份持续更新的六段合并台账/能力矩阵和一份最终验证报告。

## 关键取舍与风险

- 每段 full 门禁耗时高，但可把首次回归定位到单一 release 区间。
- Git 无文本冲突不代表本地语义安全，必须按能力矩阵检查调用链与边界路径。
- migration 172/181 同号及上游两个 186 由完整文件名区分，但仍需验证实际执行顺序和 `_notx` 路径。
- 当前工作站没有 Docker；远程 integration 依赖 `local-serv-ai` 的 Go 工具链和 Docker。每段须先显式检查，使用 `CI=true` 令整套 Docker 缺失路径失败，并从 verbose 日志确认目标 migration/repository test 真实 PASS；无关环境型 skip 单独记录。
- 远程包必须来自已提交 HEAD；未提交修复不会进入 `git archive`，因此远程 integration 只能在 merge/修复提交完成后运行。

## 测试策略

- 阶段 0 与每个 tag 均执行 `make test`、`make build`、`CI=true make -C backend test-integration`、Ent/Wire 两次生成稳定性、静态冲突扫描和 migration 新库/升级库验证。
- migration integration 先用过滤后的 embedded FS 模拟 `0.1.159.6` 本地数据库，再使用完整 FS 升级并重复执行，验证同号文件、checksum、幂等性和 190 notx 路径。
- 复杂 scheduler、sticky、fallback、runtime config 回归先保留失败测试，再做最小修复。
- 最终重新执行全部门禁、Git 拓扑检查、前端浏览器烟测和 thorough 能力级 review。

## Spec Patch

- 已在“分段 full 门禁通过”中加入 `make -C backend test-integration` 和 Docker 可用性前置检查。
- 已增加场景：Docker/Testcontainers 不可用或目标 migration/repository integration test 未真实执行并通过时，当前阶段 MUST 阻塞且不得记为通过。
