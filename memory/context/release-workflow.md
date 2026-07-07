# 发版工作流

## 目标

在 `sub2api` 仓库中稳定、连续地发布新版本 tag，并触发 GitHub Release workflow，尽量避免版本号跳错、tag 打错或基于错误提交发版。

## 默认流程

1. 先检查当前分支、`HEAD`、工作区状态，确认当前提交是否适合作为发版目标。
2. 查看最新 tag、当前版本序列，以及当前提交上是否已经存在 tag，避免重复发版或误判下一个版本号。
3. 如果用户明确指定版本号，严格按用户给出的版本号发布，不自行改成别的主版本或序列。
4. 如果用户只说“发布下一个版本”，先确定当前 `HEAD` 已包含的最高上游三段式 tag，再发布该基准下一个本地四段式 tag。
5. 默认给当前目标提交打 tag，并执行 `git push origin <tag>` 推送到远端。
6. 推送后校验远端 tag 已存在。
7. 后续发布下一个版本时，默认采用完整 Release 方式：记录当前 `SIMPLE_RELEASE` 仓库变量，临时执行 `gh variable set SIMPLE_RELEASE --body false --repo caiqy/sub2api`，再用 `gh workflow run release.yml --repo caiqy/sub2api --ref <tag> -f tag=<tag> -f simple_release=false` 触发目标 tag。
8. 使用 `gh run view/watch` 跟进 Release workflow 到最终结果；成功后核验 release assets 至少包含对应平台二进制归档和 `checksums.txt`。
9. 核验完成后恢复 `SIMPLE_RELEASE` 原值；如果原值是 `true`，执行 `gh variable set SIMPLE_RELEASE --body true --repo caiqy/sub2api`。

## 下一个本地版本号算法

1. 从远端上游获取三段式 tag 列表及其 peeled commit（优先用 `git ls-remote --tags upstream 'refs/tags/v*'`，不要依赖可能过期或冲突的本地 tag）。
2. 按版本号从高到低检查每个上游三段式 tag：`git merge-base --is-ancestor <peeled commit> HEAD`。
3. 第一个被当前 `HEAD` 包含的上游三段式 tag，就是本地四段式版本的基准。
4. 查看 `origin` 和本地已存在的同基准四段式 tag，取最大第四段并加 1。

示例：上游最新是 `v0.1.130`，但当前 `HEAD` 未包含它；当前 `HEAD` 已包含的最高上游三段式 tag 是 `v0.1.129`，且已有 `v0.1.129.1`，则下一个本地版本是 `v0.1.129.2`。

## 用户偏好与约束

- 用户经常直接在 `main` 上继续发布版本，先以当前分支实际状态为准，不额外假设需要切换分支。
- 用户对版本号序列非常敏感；一旦明确指定某个版本号，必须严格照做。
- 本地派生版本使用四段式版本号，基准必须是当前 `HEAD` 已包含的最高上游三段式 tag；不能提升前三段，也不能基于尚未合入的上游 tag 发 `.1`。
- 若 `git fetch upstream --tags` 出现 tag clobber/rejected，不能把本地 `upstream/main` 或本地 tag 视为最新事实；必须额外用 `git ls-remote upstream refs/heads/main refs/tags/v*` 或修复 fetch 后再判断。
- 不要擅自为了发版加入 `[skip ci]` 或做无意义改动。
- 如果用户要求“commit & push & 发布下一个版本”或要求刷新 `HEAD`，优先使用真实、最小、合理的改动，而不是空提交或伪改动。
- 发版后要主动使用 `gh` 跟进 Release workflow 状态，并基于最终结果再决定是否继续后续部署动作；不要只依赖 tag push 触发的默认行为，因为 `SIMPLE_RELEASE=true` 会导致 release 缺少页面内更新所需的二进制 assets。

## 仓库行为

- `.github/workflows/release.yml` 会在 `push.tags: 'v*'` 时自动触发 Release workflow。
- workflow 内部可能会执行 VERSION 同步，并产生包含 `[skip ci]` 的自动提交；这是仓库既有行为，不等于手动发版时可以自行加 `[skip ci]`。
- `backend/cmd/server/VERSION` 可作为主线版本判断的重要参考之一，但最终仍需结合最新 tag 与用户指令共同判断。

## 执行原则

- 先核对事实，再发版：分支、提交、tag、版本序列都要先确认。
- 版本序列确认必须包含“当前代码包含关系”：先找当前 `HEAD` 已包含的最高上游三段式 tag，再递增对应四段式 tag。
- 用户指定版本号时，用户指令优先，不自行“顺延”或“修正”。
- 若当前提交已经有目标 tag，先说明现状，再决定是否需要修正。
- 发版完成后，最好补一次远端 tag 校验，避免只看本地结果就误报成功。
