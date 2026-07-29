# 验证报告：unify-effective-gateway-route-state

## 摘要

| 维度 | 结果 |
|---|---|
| 完整性 | 16/16 项任务已记录完成 |
| 正确性 | 规格场景已有实现与测试证据，审查无 Critical/Important |
| 一致性 | 实现符合 proposal、OpenSpec delta 与技术 Design Doc |
| 本地门禁 | 在用户明确接受上游基线例外的前提下通过 |

## 验证证据

- Follow-up 范围：`babe29e00f18df9a0011d8464446654148d5eb53..4778e32dc879f682fd5774c1fb0c5a63867802c6`。
- 四组聚焦测试全部通过。
- 在干净的 detached worktree 中连续运行两次 `make -C backend generate`，两次均未产生 diff，hash 为 `e69de29bb2d1d6434b8b29ae775ad8c2e48c5391`。
- `git diff --check` 通过。
- `openspec validate "unify-effective-gateway-route-state" --strict` 通过。
- `make SHELL=D:/scoop/shims/bash.exe build` 通过；显式指定 shell 是因为 backend Makefile 使用 POSIX 环境变量赋值和 `.sh` 脚本。
- `make test-frontend` 通过：lint、typecheck、213 个测试文件及 1613 个测试全部通过。
- 完整门禁运行期间，后端普通测试、例外之外的 unit 测试及 `golangci-lint` 均通过。

## 用户接受的上游基线例外

`make test` 未取得 exit 0。剩余失败为 `TestPassthroughLifecycle_LeaseLossSendsRetryClose`：客户端偶发收到 EOF，而不是 WebSocket 1013 close code。

- 该测试来自上游提交 `f0e0b7e6d84f05d0936fb281d0a50263db5eefad`，存在于 `v0.1.161` 至 `v0.1.165` 以及 `upstream/main`。
- 本 follow-up 范围没有修改 lease-loss 相关实现和测试文件。
- 在未含本地改动的原始上游 `v0.1.165` tag（`e9a58c1cb`）detached worktree 中，以下命令复现了相同 EOF 失败：

```text
go test ./internal/service -run '^TestPassthroughLifecycle_LeaseLossSendsRetryClose$' -count=20
```

用户明确要求跳过该上游基线问题并归档，不在本 change 中修复。本报告不声称 `make test` 已通过。

## Fresh Review

Paseo agent `74b8c51b-f4ae-4a56-b3a2-d8bae2d640e7` 使用 `openai/gpt-5.6-sol` 与 `ultra` reasoning；runtime session 为 `ses_055058ab6ffeZf8yxzq33JSWqw`，内部最终 reviewer session 为 `ses_0550342a5ffeiJAD61Nle4BPcg`。

- Important 1: final group authority across middleware subscription, protocol dispatch, Gin API key, scheduler/billing and Ops — PASS。
- Important 2: prompt-too-long resolves final group before validation and never bills a subscription group with nil subscription — PASS。
- Important 4: HTTP bridge later frame performs account mapping after route rewrite and retains provider affinity — PASS。
- Minor: no channel mapping preserves concrete routing model in requested/channel-mapped/upstream audit stages — PASS。
- New Critical/Important findings: none — PASS。

仍有 3 个不阻断的 Minor：

- `TestEffectiveRouteConsumersAssignSubscriptionAuthoritatively` 是脆弱的源码字符串护栏，不能替代运行时证明。
- Gateway/OpenAI count_tokens 缺少“旧 subscription + balance route + nil effective subscription”的运行时回归。
- Ollama singleflight 测试使用固定 50ms 调度 barrier，在极端负载下仍可能受时序影响。

## 结论

实现不存在未解决的 Critical 或 Important。根据用户明确接受的上游基线例外，本 change 可以归档；未通过的完整测试命令已在上文如实披露。
