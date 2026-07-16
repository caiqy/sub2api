# 上游 v0.1.156 分段合并验证记录

## 固定对象与工作区

- 隔离方式：当前仓库中的 feature 分支；本任务未创建、切换或合并分支。
- 隔离分支：`feature/20260716/staged-merge-upstream-v0-1-156`。
- 开始 `HEAD`：`d5f8192d32d9840d63477c24d4a567abb8cb4a90`。
- `HEAD` 父提交：`d1cc02502271f54b3b7f0593a18db4f2aaab63ea`。
- `HEAD` 主题：`test: isolate Go test temporary files`。
- `d1cc02502..HEAD`：仅该已确认的测试基础设施提交；差异文件为 `.gitignore`、`backend/Makefile`、`backend/scripts/test.ps1`，没有本次业务合并变更。

| Tag | Annotated tag object | Peel commit |
| --- | --- | --- |
| `v0.1.152` | `553ab6f911247963eb368fcf6ac1dcb65d5495b1` | `b73d8c3efe01a290eaaa9326b6e40ece02c67a0e` |
| `v0.1.153` | `53717a125583e3916b751c2a5340901c4bfa2bb3` | `a2bc1337474b68b62391116835e5698ebb5526bd` |
| `v0.1.155` | `ec4a37da4f023fbaa4d46d2ee46a6e7f22e313d4` | `41cec0db059ffb82d0efdcfcf07a24ab51fbfe97` |
| `v0.1.156` | `9cc1b469a24e6f79aeec9401ad1f9534f9b98aec` | `12f991dde8a58e183d4bd16a87ef6fd0df714757` |

- `git fetch upstream --tags` 后，`upstream/main` 从 `807850769` 更新至 `09c6c6d74`。
- 排除范围：`git log --oneline "v0.1.156^{}"..upstream/main` 仅用于记录 release 后上游历史；输出起点为 `09c6c6d74 Merge pull request #4387 from yardbirds0/feat/upstream-rate-scheduling`，尾部为 `75fb3c41c fix(apicompat): responses->chat ...`。未将该范围或 `upstream/main` merge 到当前分支。

## 初始工作树

`git status --short` 输出仅为：

```text
?? .comet/current-change.json
?? openspec/changes/staged-merge-upstream-v0-1-156/
```

`.comet/current-change.json` 保持未暂存、未提交。`openspec/changes/staged-merge-upstream-v0-1-156/` 是本任务允许提交的协调产物目录。

## 执行命令与结果

| 命令 | 关键输出 | 退出状态 |
| --- | --- | --- |
| `git status --short` | 仅初始工作树章节所列两个未跟踪路径 | 0 |
| `git rev-parse HEAD` | `d5f8192d32d9840d63477c24d4a567abb8cb4a90` | 0 |
| `git merge-base --is-ancestor d1cc02502271f54b3b7f0593a18db4f2aaab63ea HEAD` | 无输出，祖先关系成立 | 0 |
| `git log --oneline d1cc02502271f54b3b7f0593a18db4f2aaab63ea..HEAD` | `d5f8192d3 test: isolate Go test temporary files` | 0 |
| `git diff --name-status d1cc02502271f54b3b7f0593a18db4f2aaab63ea..HEAD` | 仅 `.gitignore`、`backend/Makefile`、`backend/scripts/test.ps1` | 0 |
| `git fetch upstream --tags` | `upstream/main`：`807850769..09c6c6d74` | 0 |
| `git rev-parse v0.1.152 "v0.1.152^{}"` | object/peel 与固定对象表一致 | 0 |
| `git rev-parse v0.1.153 "v0.1.153^{}"` | object/peel 与固定对象表一致 | 0 |
| `git rev-parse v0.1.155 "v0.1.155^{}"` | object/peel 与固定对象表一致 | 0 |
| `git rev-parse v0.1.156 "v0.1.156^{}"` | object/peel 与固定对象表一致 | 0 |
| `git log --oneline "v0.1.156^{}"..upstream/main` | 仅记录 release 后排除范围，未 merge | 0 |
| `git branch --show-current` | `feature/20260716/staged-merge-upstream-v0-1-156` | 0 |
| `git show -s --format='%H%n%P%n%s' HEAD` | `HEAD`、父提交和主题与本报告一致 | 0 |

## 提交与自审

- 首次协调提交 SHA：`3877dc247ea58ef2194051399db3e67974d68473`，message 为 `docs: add staged upstream merge plan`。本报告更正后另行创建普通文档提交，不在本次提交中记录其自身 SHA。
- 变更文件：3 个 `docs/superpowers/{specs,plans,reports}/2026-07-16-staged-merge-upstream-v0-1-156*` 文档，以及 `openspec/changes/staged-merge-upstream-v0-1-156/` 下 19 个协调文件，共 22 个新增文件。
- 暂存自审：`git diff --cached --check` 退出 0；`git diff --cached --name-only -- .comet/current-change.json .superpowers` 无输出。
- 提交自审：首次提交的 `git show --name-status --format=fuller` 仅列出上述 22 个允许路径；根目录 `.comet/current-change.json` 保持未跟踪，未提交 `.superpowers/` 或业务代码。
- 事实自审：分支、开始 `HEAD`、父提交、四个 tag object/peel SHA 与 brief 完全一致；没有执行 `git merge`、分支切换、业务代码修改、测试、push、release、deploy 或 main 合并。未勾选计划或 OpenSpec task。

## 顾虑

- `upstream/main` 在 fetch 时前进至 `09c6c6d74`，其相对 `v0.1.156^{}` 的完整范围只作记录；后续四个 tag 分段 merge 必须继续以本报告固定的 tag peel commit 为目标。
- 本任务依用户裁决只核验 Git/工作树证据，不运行或伪造 RED/GREEN 测试。
- 协调产物为 22 文件、2300 行，超过 200 行风险阈值；均为本次既有设计、计划、OpenSpec/Comet 协调内容。
- 暂存时 Git 提示这些文档的工作副本下次被 Git 触及时可能发生 LF/CRLF 工作树转换；本次 `git diff --cached --check` 通过。
