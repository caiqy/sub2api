# staged-merge-upstream-v0-1-159 验证报告

## 结论

完整验证通过，无 CRITICAL 或 IMPORTANT 问题。change 已满足 OpenSpec、技术设计和实现计划，可进入分支处理；归档仍需用户单独确认。

| 维度 | 结果 |
| --- | --- |
| 完整性 | 37/37 tasks 完成，11/11 requirements 有实现或批准移除证据 |
| 正确性 | 七段 merge、能力保护、唯一批准移除项和 11 个场景均有对应证据 |
| 一致性 | proposal、OpenSpec design、Technical Design、实现与验证记录一致 |

## 新鲜验证证据

- `make test`：通过；后端测试与 lint、前端 lint/typecheck 均通过，Vitest 为 194 files / 1493 tests。
- `pnpm --dir frontend run build`：通过；Vite 构建 987 modules。保留既有 Browserslist、动态导入与 chunk 大小警告。
- `make -C backend generate`：通过；`backend/ent` 与 `backend/cmd/server/wire_gen.go` 无 diff。
- `openspec validate staged-merge-upstream-v0-1-159 --strict`：通过。
- `git diff --check`：通过；tasks 未完成数为 0；runtime `VERSION` 为 `0.1.159.1`。
- `v0.1.152`、`v0.1.153`、`v0.1.155`、`v0.1.156`、`v0.1.157`、`v0.1.158`、`v0.1.159` 的 peeled commit 均为 `HEAD` 祖先；七个 merge 的第二父与对应 tag 一致。
- `upstream/main` 不是 `HEAD` 祖先，`v0.1.159` 后 8 个未发布提交未进入结果。
- 旧 `openai_text_first_token_timeout`、`openai_image_first_token_timeout`、`first_token_timeout` 与专用事件在业务代码中无残留。
- thorough review 结论为 pass；未发现实现正确性、安全或边界条件问题。报告中的历史 Task 24/34 陈旧描述已改为明确的时间点说明。

## 规格与设计核对

- 三个新增正式 tag 按顺序使用独立 `--no-ff` merge，并在每段完成冲突台账、能力审查和阶段门禁。
- 首轮四段 merge 和验证报告保留为历史证据；新增范围重新执行基线、能力矩阵、逐段门禁和最终验证。
- 本地旧首 Token watchdog 是唯一 `approved-removal`；上游 native first-output 与客户端 WebSocket 首消息超时保留。
- scheduler、Sticky、fallback/WaitPlan、DB recheck、协议转换、privacy、图片、审计/session/step-up、计费、分组复制、用户批量限额、前端、生成物与 migrations 均在 capability matrix 中有 protected、manual 或 approved-removal 结论，无 gap。
- 未合入 tag 后 `upstream/main`，未 push、release、deploy 或 archive。

## 残余人工验证

- 未运行 Docker-backed/live PostgreSQL migration、事务回滚和 repository integration tests。
- 真实 S3/图片 worker、上游计费探测、trusted proxy/session、step-up、Grok transport/cache 和 Stripe 失败路径仍需部署环境验收。
- 上述项目保持 `manual`，未计作自动验证通过。

## 问题分级

- CRITICAL：无。
- IMPORTANT：无。
- WARNING：无未处理项。
- SUGGESTION：无。
