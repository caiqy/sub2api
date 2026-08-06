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
