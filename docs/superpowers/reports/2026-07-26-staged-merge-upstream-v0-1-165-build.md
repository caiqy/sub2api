# 分段合入上游 v0.1.165 构建台账

## 固定对象与范围

- 分支：`feature/20260726/staged-merge-upstream-v0-1-165`。
- 分支创建基线：`075abc07399d6154130d2a2695fb24c785acd69c`。
- `backend/cmd/server/VERSION`：`0.1.159.6`。
- 当前工作区：`D:/Caiqy/Projects/Github/sub2api`；它是主工作树，隔离形式为独立 feature 分支，而非额外 linked worktree。`git worktree list --porcelain` 还列出两个与本 change 无关的 detached 临时工作树：`C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-task27-pre157` 和 `C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-task27-v0157`。
- 当前文档 HEAD：`f1ad4a6da432e005d904f1deb1f1ab9bd339df63`，不是 base ref。文档提交链为：
  - `f5656d5ef6b8dd4d93b10b7779f044e14ca8f43f docs: record staged merge baseline`（父提交：`075abc07399d6154130d2a2695fb24c785acd69c`）
  - `6e18ca4270109b098940223c4a9b317f41aa4292 docs: localize staged merge baseline report`（父提交：`f5656d5ef6b8dd4d93b10b7779f044e14ca8f43f`）
  - `f1ad4a6da432e005d904f1deb1f1ab9bd339df63 docs: translate task 1 ledger title`（父提交：`6e18ca4270109b098940223c4a9b317f41aa4292`）

| Tag | Tag 对象 | Peeled SHA |
| --- | --- | --- |
| `v0.1.160` | `2a519c0f8878aa8d9d75918e3acd734e536cc675` | `8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62` |
| `v0.1.161` | `317df5405c0ff1c67f12dcc0c669a16fc2e21dac` | `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c` |
| `v0.1.162` | `34b7a5ad70b4b9b9bb96955562fe632ad625d783` | `27f094e0960ebd8e52de7ff7e763c6fec2ff4057` |
| `v0.1.163` | `bb752ef7776dc126ffca5df9188087d0d0aed559` | `d0bdd7e771636a8d315f542cafd39484f39bd60c` |
| `v0.1.164` | `38a46fd33795c8946a1e88d0f72597c79ca02a76` | `cd8bb98c44303b2c8f04c0da340447c992f0cb7d` |
| `v0.1.165` | `892c8fa3ab80ada8a624668808c3e575da7c04d5` | `e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29` |

- 已验证的相邻 peeled tag 祖先链：`v0.1.160 -> v0.1.161 -> v0.1.162 -> v0.1.163 -> v0.1.164 -> v0.1.165`；全部 `git merge-base --is-ancestor` 检查以退出码 `0` 通过。
- release 上界：已合入 `upstream/main` 的最新正式 tag 是 `v0.1.165`。
- 排除命令：`git log --oneline 'v0.1.165^{}..upstream/main'`。
- 唯一排除提交：`2730c1c43b29be003925b033f3f9e645e726bb8c chore: sync VERSION to 0.1.165 [skip ci]`。
- `paseo.json` 是未跟踪用户文件，继续排除在本任务之外，绝不暂存、修改、删除或移动。
- 根目录 `.comet/current-change.json` 是本地 Comet selector，继续排除在本任务提交之外，绝不暂存、修改、删除或移动。

## 阶段 0

- Task 1 的固定基线、tag 链、release 上界和排除提交证据已记录在本台账的“固定对象与范围”。
- Task 2 隔离检查结果：当前不在 `main`，当前分支和主工作树状态符合分支级隔离；工作树仅含根 `.comet/current-change.json`、本 change 的 OpenSpec 规划工件和 `paseo.json` 三类未跟踪项。
- 本任务仅初始化规划证据，不执行业务 TDD、本地门禁或远程 integration；这些项由后续 OpenSpec task 真实执行。

## 能力矩阵

| 能力 | 受影响 tag | 当前状态 | 证据 |
| --- | --- | --- | --- |
| changed-files 与本地能力交集 | `v0.1.160` 至 `v0.1.165` | `待执行` | Task 1.6 尚未运行六段实际 diff 与调用链审查。 |

## v0.1.160

- changed-files：待执行。
- 冲突台账：无（尚未合并）。
- 能力矩阵交集：待执行。
- 聚焦测试：待执行。
- 本地门禁：待执行。
- 远程门禁：待执行。
- 放行结论：待执行。

## v0.1.161

- changed-files：待执行。
- 冲突台账：无（尚未合并）。
- 能力矩阵交集：待执行。
- 聚焦测试：待执行。
- 本地门禁：待执行。
- 远程门禁：待执行。
- 放行结论：待执行。

## v0.1.162

- changed-files：待执行。
- 冲突台账：无（尚未合并）。
- 能力矩阵交集：待执行。
- 聚焦测试：待执行。
- 本地门禁：待执行。
- 远程门禁：待执行。
- 放行结论：待执行。

## v0.1.163

- changed-files：待执行。
- 冲突台账：无（尚未合并）。
- 能力矩阵交集：待执行。
- 聚焦测试：待执行。
- 本地门禁：待执行。
- 远程门禁：待执行。
- 放行结论：待执行。

## v0.1.164

- changed-files：待执行。
- 冲突台账：无（尚未合并）。
- 能力矩阵交集：待执行。
- 聚焦测试：待执行。
- 本地门禁：待执行。
- 远程门禁：待执行。
- 放行结论：待执行。

## v0.1.165

- changed-files：待执行。
- 冲突台账：无（尚未合并）。
- 能力矩阵交集：待执行。
- 聚焦测试：待执行。
- 本地门禁：待执行。
- 远程门禁：待执行。
- 放行结论：待执行。

## 远程 integration 记录

- 阶段 0：待执行。
- `v0.1.160`：待执行。
- `v0.1.161`：待执行。
- `v0.1.162`：待执行。
- `v0.1.163`：待执行。
- `v0.1.164`：待执行。
- `v0.1.165`：待执行。

## 阻塞与残余风险

- 当前无隔离或工作树范围阻塞。
- 尚未执行任何 tag merge、changed-files 审查、能力矩阵填充、本地门禁或 `local-serv-ai` integration；在对应证据完成前不得进入下一阶段。
