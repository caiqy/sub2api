## Context

分层调度器先从分组快照选择候选账号，再通过 `recheckSelectedOpenAIAccountFromDB` 读取数据库最新状态。该函数近期增加分组归属校验，但分层调度器仍使用旧的四参数调用形式，导致 `groupID` 默认为 `nil`，所有已分组账号在二次校验时被拒绝。

## Goals / Non-Goals

**Goals:**

- 在分层调度的普通选择、等待回退和粘性恢复路径中保留请求分组 ID。
- 用回归测试覆盖已分组账号的分层调度。

**Non-Goals:**

- 不改变分组归属规则、调度权重、账号测试行为或 API。
- 不重构兼容旧调用签名的桥接函数。

## Decisions

- 在分层调度器现有 3 处二次校验调用中传入 `req.GroupID`。这是最小修复，并与非分层调度路径的当前调用方式一致。
- 测试真实的分层调度选择结果，而不是只断言函数参数或 mock 调用次数，确保回归测试能捕获生产故障。

## Risks / Trade-offs

- [遗漏其他旧签名调用] → 搜索分层调度器内全部 `recheckSelectedOpenAIAccountFromDB` 调用并统一修复。
- [修复影响未分组账号] → `groupID=nil` 的请求仍原样传入，保持未分组池行为。
