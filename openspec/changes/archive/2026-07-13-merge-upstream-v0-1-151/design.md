## Context

当前本地 `main` 已合入大输入内存优化并同步到 `origin/main`，版本为 `v0.1.146.4`。upstream 最新正式 release tag 为 `v0.1.151`；`upstream/main` 比该 tag 额外前进 40 个提交。本次需要吸收正式版更新，同时保护本地网关、调度、隐私和请求体生命周期定制。

## Goals / Non-Goals

**Goals:**
- 以 upstream `v0.1.151` 作为唯一上游合并目标。
- 在隔离分支完成合并、冲突协调、修复和验证。
- 同时保留可共存的上游修复与本地定制，不可共存时交由用户决策。
- 通过自动检查和能力级 review 证明本地关键行为未回退。

**Non-Goals:**
- 不合入 `v0.1.151` 之后的 `upstream/main` 提交。
- 不在本 change 内发布版本或部署服务器。
- 不新增与上游合并无关的业务能力、public API、schema 或重构。

## Decisions

1. 合并目标固定为 release tag `v0.1.151`，不使用 `upstream/main`。
   - 原因：release tag 范围稳定、可追踪，避免额外引入 40 个尚未发布提交。
   - 替代方案：直接合并 `upstream/main`。放弃原因是范围会随远端变化且包含未发布内容。

2. 从同步后的 `main` 创建独立 merge 分支。
   - 原因：本地与 upstream 分叉较大，隔离分支为冲突处理、专项修复和回退保留清晰边界。
   - 替代方案：直接在 `main` 合并。放弃原因是失败或半完成状态会污染主线。

3. 冲突按业务语义协调，不机械选择 ours 或 theirs。
   - 可共存时同时保留上游更新与本地能力；不可共存时暂停并列出影响供用户选择。
   - `VERSION`、配置、生成文件和 migration 单独复核，避免文本无冲突但运行语义错误。

4. 验证分为自动检查与能力级 review。
   - 自动检查运行后端全量测试、前端单测、typecheck 和 build。
   - 能力级 review 聚焦 scheduler、sticky/fallback、privacy、image capability、runtime setting 热更新、网关透传及大输入请求体保留、落盘和释放路径。

## Risks / Trade-offs

- 上游与本地在同一路径存在不可共存语义 → 停止自动决策并请求用户选择。
- 文本合并成功但本地行为被静默覆盖 → 使用关键能力清单逐项 review，并为发现的回归补失败测试后最小修复。
- 上游生成文件或依赖变化导致构建不一致 → 单独核对依赖、生成代码、migration 和构建产物。
- 全量验证耗时较长 → 保留完整验证，因为跨版本合并的影响面无法用少量定向测试可靠覆盖。
