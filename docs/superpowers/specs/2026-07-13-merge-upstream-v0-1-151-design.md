---
comet_change: merge-upstream-v0-1-151
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-13-merge-upstream-v0-1-151
status: final
---

# 合并上游 v0.1.151 技术设计

## 1. 背景与边界

本地 `main@46d92f1d7` 已同步到 `origin/main`，包含本地 `v0.1.146.4`、网关与调度定制，以及大输入请求体内存优化。upstream 最新正式 release tag `v0.1.151` 指向 `deff3123ded1d14e51df1fd1286e3d43ed9ec9bd`；`upstream/main` 在该 tag 之后还有 40 个提交。

本 change 只合并 `v0.1.151`。不引入 tag 后提交，不执行发布或部署，不借机做无关重构。

## 2. Git 集成方案

1. 从干净且与 `origin/main` 同步的 `main` 创建独立 merge 分支。
2. 再次 fetch upstream，并校验 `v0.1.151^{}` 仍指向已记录提交。
3. 使用一次 `--no-ff` merge 合入 `v0.1.151`，保留完整上游祖先关系。
4. 上游 merge commit 与后续本地兼容修复分开提交。
5. 验证通过后由用户决定是否合回 `main`、推送或保留分支。

不按 tag 逐版合并，因为这会重复解决相邻版本冲突并制造多余 merge 节点；不使用 cherry-pick，因为容易遗漏依赖，也无法证明已完整同步到正式版。

## 3. 冲突处理

每个冲突先按以下类别归因：

- 上游修复或功能演进；
- 本地业务定制；
- 接口、配置或数据模型演进；
- 版本、依赖、生成文件或 migration 差异。

两边语义可共存时进行最小融合，不机械采用 ours/theirs。无法共存时停止处理，给出两种行为、调用链影响、数据与兼容性风险及推荐方案，等待用户选择。

文本合并完成后，单独复核 `VERSION`、`go.mod/go.sum`、前端依赖锁、配置默认值、Wire/Ent 等生成代码和 migrations，确认运行时与发布元数据一致。

## 4. 能力级审查

自动测试之外，按能力矩阵检查无文本冲突的语义覆盖：

| 能力 | 重点路径 |
|---|---|
| 调度 | scheduler、session/previous-response sticky、fallback/WaitPlan、DB recheck |
| 网关 | Messages/Responses/Chat 转换、stream/non-stream、透传字段、终止 usage |
| 安全与能力 | privacy、内容审计、image capability 和模型过滤 |
| 运行时 | setting 热更新、缓存失效、服务重建或重载 |
| 大输入内存 | body 读取、内存到磁盘切换、重放、所有权转移、成功/失败清理 |

审查发现回归时先添加能稳定复现的失败测试，再实施最小兼容修复。测试必须覆盖主路径和 previous response、fallback、重试、终止错误等边界路径。

## 5. 验证策略

冲突处理和兼容修复期间先运行受影响包或前端文件的定向测试。最终验收必须包括：

```text
backend:  go test ./... -count=1
frontend: pnpm test:run
frontend: pnpm typecheck
frontend: pnpm build
```

同时执行 `git diff --check`、冲突标记扫描、工作树检查，并验证 `v0.1.151` 是 merge 结果的祖先。已知的非阻塞警告必须记录，新增失败不得忽略。

## 6. 完成条件与回退

完成条件：无未解决冲突；版本、依赖、配置、生成文件和 migration 已复核；能力矩阵逐项有结论；完整验证通过；验证报告记录冲突决策、修复、测试结果和残余风险。

合并尚未进入 `main` 时，回退方式是丢弃独立 merge 分支。合回 `main` 后如发现问题，优先 revert merge 或对应兼容修复提交，不改写已推送历史。
