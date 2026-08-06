# Restore Backend Lint Gate 验证报告

## 身份与范围

- Change：`restore-backend-lint-gate`
- 实施分支：`feature/20260806/restore-backend-lint-gate`
- 证据 HEAD（报告提交前）：`a928995ec189b2d010ce933164a59fe201bbfcdd`
- Plan base：`9dafc41f7ca0d7ea334698bf554cf7e0facb6038`
- Manifest source base：`b576f73a22c4bf23d61727fc93950766a7e33929`
- 实现提交：`874d19202`、`aef7b6ec8`、`000f69491`
- Go：`go1.26.5 windows/amd64`
- golangci-lint：`2.9.0`（built with Go 1.26.3）

`planBase..HEAD` 的 committed backend 文件集合精确等于 baseline manifest 的 39 个文件；总 committed 文件数为 45。staged 与 unstaged 集合均为空，唯一未跟踪文件为 Comet 运行时选择文件 `.comet/current-change.json`。

## Gate 0

- Plan base 与 manifest source base 均存在，且 plan base 是实施 HEAD 的祖先。
- 5 个 planning checkpoint 产物已提交且干净；实施前不存在 backend diff。
- uncapped JSON lint RED：exit 1，144 issues，39 files；`ineffassign=140`、`staticcheck=3`、`unused=1`。
- baseline 表中的 144 个 `(path,line,linter)` identity 与实际 JSON 逐项一致。
- baseline 表未保存诊断 text；替代证明为：backend tree 相对 manifest source base 无 diff、golangci-lint 版本相同、lint 配置 blob 相同，且全部 144 个三元组一致。未声称直接比较了 baseline 中不存在的 text 字段。
- handler/routes 与 service package tests 通过；retained-heap、handler replay、service replay 清单分别精确匹配并通过 2、7、9 个测试。

## 分批 Lint 闭包

| 批次 | RED | GREEN | Stable identity diff | 结果 |
| --- | --- | --- | --- | --- |
| Handler/routes/QF1003 | 144/140/3/1/39 | 108/107/0/1/26 | added=0，removed=36 | 33 个 ineffassign 与 3 个 QF1003 关闭 |
| Gateway/Anthropic/Bedrock/Antigravity | 108/107/0/1/26 | 45/44/0/1/13 | added=0，removed=63 | 63 个 baseline identity 关闭 |
| OpenAI/Gemini/Grok/unused | 45/44/0/1/13 | 0/0/0/0/0 | added=0，removed=45 | 44 个 ineffassign 与 1 个 unused 关闭 |

Task 1 在 QF1003 改写前额外验证 `111/107/3/1/28`，相对 Gate 0 added=0、removed=33；tagged switch 后总 removed=36，因此额外且仅关闭 3 个 QF1003。

Task 2 拆分混合赋值时暴露了同一语句中原先被遮蔽的 3 个死局部诊断（`body`、`attemptBody`、`attemptCanonicalBody`）。删除这些死局部并保留 `input.SourceBody/input.Body` 字段清理后，最终 stable identity diff 为 added=0、removed=63，未扩大文件范围。

Task 3 删除 `sendCCUpstreamRequest` 前，CodeGraph 未发现静态、接口、callback 或测试调用方；随后 service package 编译通过。仍有 5 个调用方的 `sendCCUpstreamRequestHandle` 保持不变。

## 最终验证

- 对 39 个 manifest 文件运行 `gofmt -w`：exit 0。
- uncapped golangci-lint JSON：exit 0，0 issues。
- `go test -count=1 ./internal/handler ./internal/server/routes`：exit 0。
- `go test -count=1 ./internal/service`：exit 0。
- retained-heap 清单：精确 2 个，`-run` exit 0。
- handler spool/replay 清单：精确 7 个，`-run` exit 0。
- service spool/retry/failover 清单：精确 9 个，`-run` exit 0。
- 根级 `make test`：exit 0。backend 默认测试与 `golangci-lint run ./...` 通过，unit-tagged backend 测试通过；frontend ESLint、Vue typecheck 与 Vitest 通过，汇总为 225 files、1698 tests。
- `git diff --check 9dafc41f7..HEAD`、working tree 与 index whitespace checks：全部通过。

## Protected Inputs

以下 blob 与 manifest source base 一致，且相对该 base 无 diff：

| Path | Blob |
| --- | --- |
| `backend/.golangci.yml` | `92ba3916948b4b859737c3c4831c7416dcd7f01e` |
| `backend/go.mod` | `7d5150f4a969df8a578e5bce8e6f5a01ec856823` |
| `backend/go.sum` | `72146c2305a91a48f92ac8fe2f9d888a2a1a2886` |
| `.github/workflows/backend-ci.yml` | `ee84c994ca2f1e27ae32eb02f25c3d094581b1ff` |
| `Makefile` | `da7c0c59fe67dfc8219ecfb2fbab1238fd0bbb55` |
| `backend/Makefile` | `0327160ff0959575ed6a8f950d7d257a96ae3ab0` |

本 change 未修改 lint/CI/Go/Make 配置，未使用 suppression、假读取或清零 helper，未恢复或修改 `staged-merge-upstream-v0-1-171`。
