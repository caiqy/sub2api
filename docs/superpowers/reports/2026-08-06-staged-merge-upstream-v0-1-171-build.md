# v0.1.171 Build Ledger

## 阶段 0：基线

- 执行位置：`D:/Caiqy/Projects/Github/sub2api`
- 执行分支：`feature/20260806/staged-merge-upstream-v0-1-171`
- immutable source base：`b576f73a22c4bf23d61727fc93950766a7e33929`
- execution base：`b576f73a22c4bf23d61727fc93950766a7e33929`
- source `VERSION`：`0.1.169.3`
- execution `VERSION`：`0.1.169.3`
- source-to-execution 路径：无
- runtime selection 状态：`git status --short --untracked-files=all` 输出为空；未出现 `?? .comet/current-change.json`，且无其他 dirty path。

### 禁止操作

- 不切换分支、merge、push、tag、release、GitHub Actions、镜像构建或发布。
- 不部署，不操作服务器、数据库、Redis 或 Nginx。
- 不修改应用源码、plan、OpenSpec tasks、`.comet/**` 或 `.comet/current-change.json`。

### Task 1 命令与退出码

```text
comet classic root show                                      exit 0
layout.schema=comet.classic-layout.v1

git rev-parse --show-toplevel                               exit 0
D:/Caiqy/Projects/Github/sub2api

git branch --show-current                                   exit 0
feature/20260806/staged-merge-upstream-v0-1-171

git merge-base --is-ancestor b576f73a22c4bf23d61727fc93950766a7e33929 HEAD
exit 0

git rev-parse HEAD                                          exit 0
b576f73a22c4bf23d61727fc93950766a7e33929

git show b576f73a22c4bf23d61727fc93950766a7e33929:backend/cmd/server/VERSION
exit 0
0.1.169.3

git show b576f73a22c4bf23d61727fc93950766a7e33929:backend/cmd/server/VERSION
exit 0
0.1.169.3

git log -m --format= --name-only b576f73a22c4bf23d61727fc93950766a7e33929..b576f73a22c4bf23d61727fc93950766a7e33929
exit 0
(no paths)

Assert-CleanGate                                            exit 0
staged paths: (none)
status: (empty)
unexpected ignored change artifacts: (none)
```

### TDD

不适用。本任务只创建基线证据文档，不包含生产代码或行为变更；未伪造 RED/GREEN。

### 阶段结论

基线身份和隔离状态均通过；可由协调者决定是否推进后续 Task。

## Task 2：上游 tag manifest

- refs 刷新：`git fetch upstream --tags --prune` 成功；`upstream/main` 从 `00b859617` 前进到 `c123caddd`。
- `v0.1.170^{}`：`c043c24774228ba891ddf90d783aa6dc7d0855b5`，与固定 peeled SHA 一致。
- `v0.1.171^{}`：`f0e7a9c7a23a7d02fb159b62fa809621eb0475a6`，与固定 peeled SHA 一致。
- 严格祖先链：`git merge-base --is-ancestor v0.1.170 v0.1.171` exit 0。
- merged `upstream/main` 的正式 tag 筛选为 `^v0\.1\.\d+$`；降序首项仍为 `v0.1.171`，未发现更高正式 tag。
- 冻结范围预期规模：`v0.1.169..v0.1.170` 为 `62/242`；`v0.1.170..v0.1.171` 为 `49/206`。

### Task 2 命令与退出码

```text
comet classic root show                                      exit 0
layout.schema=comet.classic-layout.v1

git rev-parse --show-toplevel                               exit 0
D:/Caiqy/Projects/Github/sub2api

git branch --show-current                                   exit 0
feature/20260806/staged-merge-upstream-v0-1-171

Assert-CleanGate                                            exit 0
git diff --cached --name-only                               exit 0
git status --short --untracked-files=all                    exit 0
git ls-files --others --ignored --exclude-standard -- docs/openspec/changes/staged-merge-upstream-v0-1-171 docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md docs/superpowers/specs/2026-08-06-staged-merge-upstream-v0-1-171-design.md docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-build.md docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-verify.md
exit 0

git fetch upstream --tags --prune                           exit 0
upstream/main: 00b859617 -> c123caddd

git rev-parse v0.1.170^{}                                   exit 0
c043c24774228ba891ddf90d783aa6dc7d0855b5

git rev-parse v0.1.171^{}                                   exit 0
f0e7a9c7a23a7d02fb159b62fa809621eb0475a6

git merge-base --is-ancestor v0.1.170 v0.1.171              exit 0

git for-each-ref refs/tags --merged=upstream/main --format='%(refname:short)' | Where-Object { $_ -match '^v0\.1\.\d+$' } | Sort-Object { [version]$_.Substring(1) } -Descending
exit 0
highest formal v0.1.* tag: v0.1.171

git log --oneline v0.1.171..upstream/main                   exit 0
```

### Task 2 范围外提交

以下提交均在 `v0.1.171..upstream/main`，只记录为范围外，不在当前变更中 merge：

```text
c123caddd chore: update sponsors
e08aee49e Merge pull request #5266 from shentry/fix/transient-streak-rate-dependence
c9e60d1f2 Merge pull request #5031 from keaipiao/fix/easypay-error-utf8
47c03c75d Merge pull request #5232 from fengshao1227/fix/billing-quantize-monetary-scale
00b859617 chore: update sponsors
c5e046b7d chore: update sponsors
aac53afe0 chore: sync VERSION to 0.1.171 [skip ci]
7d38e6712 fix(openai): keep transient failure streak from resetting on sparse traffic
e2652eb85 fix(billing): quantize usage billing amounts to the NUMERIC(20,8) scale
e3e033bb3 fix(payment): preserve UTF-8 in EasyPay errors
```

### Task 2 TDD 与风险

- TDD 不适用。本任务只刷新 Git refs 并更新证据文档，不包含生产代码或行为变更；未伪造 RED/GREEN，也未运行应用测试。
- 风险信号：`upstream/main` 已有 10 个 tag 后提交；它们已完整列为范围外，且没有更高正式 tag 改变当前范围。
- 顾虑：正式 tag 与祖先链在本次 fetch 后稳定；后续 Task 仍必须只通过固定的两个 tag merge 入口引入上游内容。
