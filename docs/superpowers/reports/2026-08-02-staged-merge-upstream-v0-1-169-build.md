# Stage 0 基线身份账本

- 实施时间：2026-08-02T09:23:14+08:00
- Comet 确认的隔离位置：`D:/Caiqy/Projects/Github/sub2api`
- 绑定分支：`feature/20260802/staged-merge-upstream-v0-1-169`
- immutable source base：`e9a0e4aa53b5d9d5f5c84986cfadd8098dc8e4f3`
- execution base：`10ee678a49c389958315bfdb1466796dc715f2e5`

## Source-to-execution 规划路径

`git merge-base --is-ancestor` 已确认 execution base 是 immutable source base 的后代。tree diff 与 commit path 均仅包含以下 planning-only 路径：

- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet.yaml`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/artifacts.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/checkpoint.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/context.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/handoff/brainstorm-summary.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/handoff/spec-context.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/handoff/spec-context.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/run-state.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/skill-snapshots/9bd4ffab011ae18aef91dc0db336ffc12d454b513229178c15b8b75d50930ba1/package.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/skill-snapshots/9bd4ffab011ae18aef91dc0db336ffc12d454b513229178c15b8b75d50930ba1/sha256`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/state-events.jsonl`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/subagent-progress.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/trajectory.jsonl`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.openspec.yaml`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/design.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/proposal.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/specs/upstream-release-sync/spec.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/tasks.md`
- `docs/superpowers/plans/2026-08-02-staged-merge-upstream-v0-1-169.md`
- `docs/superpowers/specs/2026-08-02-staged-merge-upstream-v0-1-169-design.md`

## 已验证身份

- source `VERSION`：`0.1.165.4`
- execution `VERSION`：`0.1.165.4`
- source receipt migration blob：`c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6`
- source outbox migration blob：`502ecec1caf9f76e022c2e83acf3707190539301`
- runtime selection：存在，唯一允许的运行时未跟踪状态为 `?? .comet/current-change.json`
- 目标 tag：`v0.1.166`、`v0.1.168`、`v0.1.169`
- Windows build shell：`D:/scoop/shims/bash.exe` 存在且可由 `Get-Command` 解析。

## TDD 与边界

- RED：N/A（证据型 docs-only 任务）
- GREEN：N/A（证据型 docs-only 任务）
- 不创建生产代码或行为变更，因此不伪造失败测试。
- 禁止 push、tag、release、deploy、镜像、SSH/服务器/数据库/Redis/Nginx 操作。

## Task 2 upstream tag manifest

- 状态：通过
- 实施时间：2026-08-02
- `git fetch upstream --tags --prune` 已完成；`upstream/main` 从 `7ceabb3fd` 更新到 `b74024c78`。

### 正式 tag manifest

| Tag | Peeled SHA | 预期 commit/file 数 |
| --- | --- | --- |
| `v0.1.166` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | `62/142` |
| `v0.1.168` | `99c8e4bf7564823bafbab369acab6539e734c1bb` | `36/170` |
| `v0.1.169` | `26d894ef4f50645a4bf1030e378ac892f17d0223` | `38/72` |

### 命令证据

```powershell
git fetch upstream --tags --prune
```

- exit code：`0`
- 关键输出：`fcb185abf..357215105  cla-signatures -> upstream/cla-signatures`、`7ceabb3fd..b74024c78  main -> upstream/main`；新增 `upstream/feat/moderation-proxy-and-smtp-starttls`、`upstream/fix/issue-5148-stream-partial-usage-billing`、`upstream/fix/issue-5152-classifier-multi-system-entries`、`upstream/fix/openai-ws-passthrough-close-frame-race`。

```powershell
git rev-parse 'v0.1.166^{}'
git rev-parse 'v0.1.168^{}'
git rev-parse 'v0.1.169^{}'
```

- 各命令 exit code：`0`
- 输出依次为：`dc893dd0b8eab41df5be595ae9fcd1aa74a062b8`、`99c8e4bf7564823bafbab369acab6539e734c1bb`、`26d894ef4f50645a4bf1030e378ac892f17d0223`。

```powershell
git merge-base --is-ancestor v0.1.166 v0.1.168
git merge-base --is-ancestor v0.1.168 v0.1.169
```

- 两个命令均无标准输出且 exit code 为 `0`：`v0.1.166` 是 `v0.1.168` 的祖先，`v0.1.168` 是 `v0.1.169` 的祖先。

```powershell
git for-each-ref refs/tags --merged=upstream/main --format='%(refname:short)' --sort=-v:refname
```

- exit code：`0`
- 关键输出（按版本降序）：`v0.1.169`、`v0.1.168`、`v0.1.166`、`v0.1.165`；最新正式 `v0.1.*` tag 是 `v0.1.169`，未发现更高正式 tag。

```powershell
git log --oneline v0.1.169..upstream/main
```

- exit code：`0`
- 以下提交明确在 release 范围外，未合并：

```text
b74024c78 Merge pull request #5167 from Wei-Shaw/fix/openai-ws-passthrough-close-frame-race
21aacde0b fix(openai-ws): keep downstream writes off the relay cancellation context
b22f73e72 Merge pull request #5154 from Wei-Shaw/fix/issue-5148-stream-partial-usage-billing
d6d53052f chore: update sponsors
bd52e5d77 fix(gateway): record observed usage when anthropic stream is interrupted
d4cada3b6 Merge pull request #5089 from feeeei/fix/openai_sse_rate_limit
8f5caef78 Merge pull request #5153 from Wei-Shaw/fix/issue-5152-classifier-multi-system-entries
2ef124629 fix(anthropic): recognize classifier requests with extra system entries
dd9a177a6 chore: update sponsors
85a27fae3 fix(openai): retry SSE rate limits as HTTP 429
eb1c5c7ee Merge pull request #5146 from tudoujunha/codex/fix-responses-tool-output-media
d9fba8fe7 Merge pull request #5101 from Tongzai123/feat/admin-select-all-filtered-results
15b3c0c5a Merge pull request #5145 from zvensmoluya/codex/update-auto-review-pricing
2e338af82 Merge pull request #5085 from feeeei/main
682c4fe0e Merge pull request #5147 from Wei-Shaw/feat/moderation-proxy-and-smtp-starttls
948b63c9c feat(moderation): route content moderation through configurable proxy server
4c80d160d fix(email): unify SMTP connection path between send and test-connection
570ea74d1 Merge pull request #5117 from gaoren002/feat/prompt-audit-blocking-latest-input
2980ff385 Merge pull request #5094 from wucm667/feat/issue-5065-compact-homepage
04c96a201 Merge pull request #4981 from INKCR0W/fix/openai-preserve-codex-namespaces
07f980b99 Merge pull request #5084 from apple-ouyang/codex/fix-openai-compaction-encrypted-retry
fe2172586 fix(openai): recover stale encrypted compaction
698547418 fix(pricing): keep Auto-review rates evidence-based
f54e9827a fix(pricing): update Codex Auto-review rates
d29acc29a Merge pull request #5066 from wucm667/fix/issue-5051-subscription-quota-window
66998918b Merge pull request #5143 from wucm667/fix/issue-5138-codex-instructions
da6194c1c Merge pull request #5112 from chenty2333/fix/openai-stream-capacity-pool-retry
132d446ca Merge pull request #5133 from dawnx/fix/payment-visible-method-wipe
0eac363e6 Merge pull request #5120 from Vibeone/fix/grok-pool-mode-cooldown-bypass
796313e99 Merge pull request #5131 from wucm667/fix/issue-5125-image-data-url-offload
c772d1866 Merge pull request #5130 from moonfunjohn/codex/fix-epay-method-selector-overflow
2bf9c6d56 修复工具输出图片桥接
94df1fffc Merge pull request #5124 from wucm667/fix/issue-5105-filter-grok-billing-ping
30967d5d9 fix(grok): ping 帧统一改写为 SSE 注释并限制过滤缓冲
dfdbc2770 fix(openai): default missing passthrough instructions
beeb2f989 test(settings): include compact home in API contracts
77d4df954 test(grok): check filter body close error
7ceabb3fd chore: sync VERSION to 0.1.169 [skip ci]
d6467f6eb fix(images): decode data URLs during task offload
8ed9f754c fix(payment): prevent method selector overflow
3deb2f17d fix(payment): 保存系统设置时不再清空可见支付方式配置
baaae8e12 fix(grok): filter billing ping response events
5c9629ddb fix(grok): unify pool mode bypass for all default cooldown paths
4d13925c9 fix(grok): skip entitlement 403 cooldown for pool mode accounts
d74e669a2 feat(security-audit): add narrow blocking audit scope
7d3bf86e5 fix(openai): retry streamed capacity errors in pool mode
2a871ec85 feat(admin): 支持按筛选结果全选账号
a35ff9613 feat(admin): 为账号批量删除增加并发限制
739c0ff9c feat(home): add compact home page preset to avoid abuse classification
3d99acb0a 优化模型广场UI:筛选栏换行对齐、模型排序与表格留白
7b6111f2f fix(subscription): align quota windows with subscription term
272735b0a fix(openai): preserve Codex namespace tools on OAuth Responses forwarding
```

### TDD 与自审

- RED：N/A（证据型 docs-only 任务）
- GREEN：N/A（证据型 docs-only 任务）
- 不创建生产代码或行为变更，因此不伪造失败测试。
- 未修改 Plan、OpenSpec `tasks.md`、`.comet/subagent-progress.md`、selection、应用源码或配置。
- 未执行 merge、push、tag、release、deploy、镜像、SSH/服务器/数据库/Redis/Nginx 操作。

### 风险信号与顾虑

- 风险信号：fetch 时 `upstream/main` 从 `7ceabb3fd` 前进到 `b74024c78`，其全部 `v0.1.169..upstream/main` 提交已在本账本中明确排除。
- 顾虑：无阻塞顾虑；release 上界和祖先链均符合范围。
