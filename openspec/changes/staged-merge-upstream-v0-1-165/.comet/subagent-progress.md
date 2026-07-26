# 子代理进度

- 当前任务：29 项中的第 4 项
- 状态：等待任务 4 实施者
- 简报：`.superpowers/sdd/task-4-brief.md`
- 报告：`.superpowers/sdd/task-4-v0-1-165-report.md`
- 基线 SHA：`075abc07399d6154130d2a2695fb24c785acd69c`
- 最后审查 SHA：`c8e0110a9a2354453753db9c4acae0ed7570458d`
- 已完成任务数：3

## 约束

- 保留用户所有的未跟踪 `paseo.json`。
- 不得 push、tag、release、deploy 或 merge 到 `main`。
- 远程工作必须使用 `ssh-skill`；不得调用原生 SSH 或 SCP。
- 使用 OpenCode 角色路由：实施者使用 `general`，审查者使用 `reviewer`。
- 将每个完整 Git 修订范围作为一个 PowerShell 参数引用。
- 用户在任务 4 阻塞后批准 Linux 原生等价 integration 门禁：重建 `backend/.test-tmp`、设置 `TMP`/`TEMP`，运行 `CI=true GOFLAGS='-v' go test -tags=integration ./...`；远端不再要求 Make 或 PowerShell。
- canonical OpenSpec、Design Doc 与 plan 已同步。Comet design handoff 是 design 阶段快照；`phase: build` 下官方刷新命令拒绝执行，因此不得手改其生成内容或 hash。
