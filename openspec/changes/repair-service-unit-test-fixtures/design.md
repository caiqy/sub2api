## Context

三个失败域均已通过聚焦复现和调用链调查确认是测试夹具未跟随生产契约迁移，并非运行时行为缺陷。

## Goals / Non-Goals

**Goals:**
- 让 fixture 使用当前 request body、platform lookup 与 repository missing-value 契约。
- 恢复完整 service unit 门禁的有效信号。

**Non-Goals:**
- 不修改 credits 错误传播、定价解析或 settings 生产逻辑。
- 不新增测试抽象或重构现有 helper。

## Decisions

- 每个失败只在其现有测试文件内修复 fixture；重复 pricing case 直接删除。
- settings 测试显式还原构造期间同步的全局值，避免测试顺序污染。
- scheduler 测试复用生产 stop 入口清理后台 probe；不在 repo stub 添加 no-op 方法掩盖生命周期泄漏。

## Risks / Trade-offs

- fixture 更贴近生产契约，但不扩展产品覆盖；相邻生产行为继续由既有测试负责。
