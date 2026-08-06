# Brainstorm Summary

- Change: staged-merge-upstream-v0-1-171
- Date: 2026-08-06

## 已确认事实与约束

- 使用 Comet Classic full workflow，单 change 分段合入 `v0.1.170`、`v0.1.171`。
- 当前 source base 为 `main@b576f73a2`，`VERSION=0.1.169.3`；最终版本为 `0.1.171.1`。
- 两个 tag 必须形成独立 `--no-ff` merge 节点；merge 只包含上游树和必要冲突融合，语义兼容修复另行提交。
- 每段运行聚焦测试、`make test`、`make build`、两轮 backend generate、静态检查和能力级 review。
- 保留双方 `191_*`、双方 `192_*` 与上游 `193_*` migration 的完整文件名；Docker integration 仅在本机可用时执行。
- 不推送、不打 tag、不发布、不部署、不操作服务器；这些动作需要后续单独授权。

## 候选技术方案

1. **两段 merge + 按能力簇拆分兼容修复（已确认采用）**：每个 release 先形成唯一 merge commit；测试或语义审查发现的修复按 scheduler/usage、gateway/body、audit/auth、subscription/migration、frontend 等能力簇分别提交。
2. **两段 merge + 每段单一兼容修复提交（候选）**：历史较短，但多个独立回归聚合后难以单独审查和回退。
3. **直接一次合入 `v0.1.171`（已由 OpenSpec 排除）**：提交最少，但无法归属 `v0.1.170` 与 `v0.1.171` 的首次回归区间。

## 关键取舍与风险

- 首段预测 28 个文本冲突、两段共 151 个本地重叠路径；修复提交粒度决定后续审查和回退成本。
- 用户已确认按能力簇拆分兼容修复；代价是提交数增加，收益是独立审查、定位和回退。
- 生成代码必须由 schema/provider 源重建，并与对应 merge 或兼容修复提交一起归属。
- 无文本冲突的语义回归仍需通过能力矩阵和聚焦测试发现。

## 测试策略

- 阶段 0 先固定本地保护门禁；每个 tag 阶段先聚焦、再 full；最终版本提交后再执行最终全门禁和拓扑验证。
- Migration 在本机 Docker 可用时覆盖空库及 `main@b576f73a2` 本地 migration 集合升级；不可用时明确记录 unverified。

## Spec Patch

当前无；OpenSpec delta spec 已覆盖已识别验收边界。

## 确认的实施结构

```text
阶段 0：固定 refs / 基线门禁 / 能力矩阵 / Docker 预检
  -> merge v0.1.170（只含上游树与必要冲突融合）
  -> 源驱动生成 Ent/Wire/锁文件
  -> 聚焦 RED -> 按能力簇最小修复提交
  -> v0.1.170 聚焦 + full + 可用的 integration 门禁
  -> merge v0.1.171（相同边界）
  -> 按能力簇最小修复提交
  -> v0.1.171 阶段门禁
  -> VERSION=0.1.171.1
  -> 最终全门禁 / 拓扑 / migration identity / 能力终审
```

- 冲突在 merge commit 前按六类登记并融合；无法共存的用户可见语义阻塞，不创建当前 merge commit。
- merge 后发现的回归必须保留 RED；修复按 scheduler/usage、gateway/body、audit/auth、subscription/migration、frontend 等能力簇提交。
- 每个阶段的能力矩阵 `gap` 必须归零后才能进入下一 tag；`unverified` 只允许用于有环境证据的本机 integration 缺口。
- 发布、推送、tag、部署与服务器操作不属于本设计。

## 待确认

- 无。用户已确认最终技术方案。
