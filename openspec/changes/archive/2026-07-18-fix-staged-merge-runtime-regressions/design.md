## Context

上游 `v0.1.159` 将 `ReportOpenAIAccountScheduleResult` 改为强类型四参数调用，并通过 `SucceededForScheduling` 区分 WS 成功与失败终态。合并时加入的可变参数桥让本地旧调用继续编译，却同时掩盖了迁移遗漏。上游 `v0.1.157` 还为图片对象存储注册了六个 Viper 默认键，但这些注册没有进入最终合并结果。

## Goals / Non-Goals

**Goals:**

- 完成调度结果调用迁移，让编译器强制所有调用方提供 canonical model。
- 对 WS 结果按实际终态上报成功或失败。
- 恢复上游已有的六个 `image_storage.*` 默认键及其环境变量覆盖入口。
- 用修复前失败的测试覆盖两项回归。

**Non-Goals:**

- 不调整 first-token / first-output timeout 行为。
- 不修复审计中确认早于 staged merge 存在的调度器和账号统计问题。
- 不扩展上游原本未完整实现的全部 `IMAGE_STORAGE_*` 凭据环境变量支持。

## Decisions

- 删除 `ReportOpenAIAccountScheduleResult(...any)` 兼容桥，恢复强类型签名。相比继续保留桥并逐个补调用，强类型签名能让遗漏在编译期失败，且删除的代码更多、长期维护面更小。
- 调用方统一传 `account.GetMappedModel(requestedModel)`；Messages 使用当前路由模型，保持渠道分派后的模型语义。
- Responses 与 Responses WebSocket 复用 `OpenAIForwardResult.SucceededForScheduling()`，不依据 handler 是否返回 Go error 猜测上游终态。
- 配置侧只恢复上游已有六项 defaults，不顺带扩大环境变量契约。

## Risks / Trade-offs

- [旧调用遗漏导致编译失败] → 搜索全部调用并以 Go 编译和 handler/service 全量测试校验。
- [模型被重复映射] → 仅在 handler 边界调用一次 `GetMappedModel`，与上游 `v0.1.159` 保持一致。
- [默认配置恢复影响现有 YAML] → Viper 中显式配置优先于 defaults，已有显式值不受影响。
