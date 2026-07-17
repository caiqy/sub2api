# Task 32: v0.1.159 Merge Report

## Status
- 已完成独立 `git merge --no-ff v0.1.159`。
- merge commit：`00517cf860cbb328ecaaae0d56bb59f1848d13ec`。
- 第一父：`436e1f1cd14e650d460f6b1ceb431b055d467d5a`。
- 第二父：`2a75d7d2387587d86ca3c5e5cd8ca96cf3d104c6`。
- 未合并 `upstream/main` 或 `v0.1.159` 之后的提交；未 push、release 或 deploy。

## Conflict Ledger

| 路径 | 决策 |
| --- | --- |
| `backend/internal/server/router.go` | 使用 tag 的 `SessionBindingContext(cfg)`，使会话绑定、审计和 API-key ACL 共享可信反代 IP 开关；保留第一父的 embedded frontend 和 `userService` 装配。 |
| `backend/internal/service/openai_alpha_search.go` | 保留第一父的 passthrough `sourceBody` 契约与 PAT fallback；加入 tag 的 API-key `404/405` 换号和 `401/404/405` 不标记账号错误规则。 |
| `frontend/src/i18n/locales/en/admin/accounts.ts` | 保留第一父的 Agent Identity import 描述/动态签名提示，加入 tag 的 Mobile RT/AT 手动输入标签，并删除重复键。 |
| `frontend/src/i18n/locales/zh/admin/accounts.ts` | 同英文 locale 融合。 |

## Validation
- `git diff --name-only --diff-filter=U` 无输出。
- `git diff --check` 无输出。
- `go -C backend test -tags unit ./internal/pkg/ip ./internal/server/middleware -run 'IP|SessionBinding' -count=1` 通过。
- `go -C backend test ./internal/service -run 'AlphaSearch|Grok.*Cache|WSHTTPBridge' -count=1` 通过。
- `pnpm --dir frontend exec vitest run src/views/user/__tests__/StripePaymentView.spec.ts src/views/user/__tests__/stripeLazyLoading.spec.ts src/views/admin/__tests__/AccountsView.sparkShadow.spec.ts`：3 files / 13 tests 通过。
- `make test`：Vitest 194 files / 1493 tests，全部通过。
- `pnpm --dir frontend run build`：987 modules，成功。

## Boundary Checks
- `git diff --name-only v0.1.156^{}..HEAD -- backend/internal/service/openai_first_output_timeout.go` 无输出；native first-output 保留，旧 first-token watchdog 未恢复。
- 本报告和活动报告将作为单独 `docs: record v0.1.159 merge decisions` 提交；不夹带 Task 33 修复。
