# Outcome

修复 Codex 经其他中转切换到本渠道后遇到 invalid_encrypted_content 时，现有恢复只识别顶层 reasoning 的覆盖缺口。复用原有恢复链路，安全可恢复的请求在同一账号有界重试；必要上下文无法恢复时明确失败，不承诺任意外来密文都可恢复。

依据：公开 issue、本地调用链及 dmit 请求 73093b5c-b8ff-4e93-a212-02102a216852 的只读日志。日志只证实未找到 encrypted reasoning 而跳过恢复，随后错误透传；原生产请求体未知，不认定具体 amsg 载体，不声称生产验证通过。

# Scope

仅覆盖 OpenAI Responses 现有 HTTP/WS 共用载体清理、replay/lineage、已有重试决策及脱敏恢复日志。根据类型、位置和保留明文判断安全性；摘要收集、命中探测、剥离使用一致边界。

## 目录结构

### 修改

- backend/internal/service/openai_gateway_request_body.go — 共用协议感知清理，保留普通文本及工具配对，拒绝不安全改写。
- backend/internal/service/openai_encrypted_content_lineage.go — 对齐摘要和 HTTP/WS replay 剥离，保持会话隔离、字节所有权、数值精度；避免不必要的重复解码和未命中切片分配。
- backend/internal/service/openai_gateway_forward.go — 仅保留既有恢复点的数量、跳过原因、重试及结果日志，不新增重试通道或预算。
- backend/internal/service/openai_encrypted_carriers.go — 仅保留上述清理、校验及摘要共用函数，移除健康错误分类类型和函数。
- 上述模块对应测试 — 保留或合并清理、工具配对、边界、lineage、真实 Forward 有界重试及日志回归，不保留健康归因或预热诊断专项测试。

### 明确不建

- 不增加数据库、依赖、全局黑名单、恢复服务、调度策略或健康豁免。

# Non-goals

不实现加密内容错误的账号健康/调度统计豁免。撤销上一候选为该需求引入的 handler/result、健康分类、WS 错误标记及 Messages/Chat/原生/透传 HTTP 200 SSE/JSON 归因改动，包括通用 bare error 终态语义调整；这些模块恢复本 change 开始前行为。

不修改下游渠道粘性，不实现 HTTP 200 SSE 恢复，不全局预删历史，不按 amsg_ 删除，不解密，不保留额外预热取消诊断，不改版本、不推送、不发布部署。

# Acceptance examples

- A1：已知安全顶层 reasoning/compaction 和嵌套 content[] 可清理；普通文本、元数据、未知类型及无密文输入不误改，保留既有顶层 reasoning content:null 兼容。
- A2：保留工具 call_id 和配对；仅有必要密文或清理后非法骨架不能被静默删除并重试；无安全修改时给出现有错误及可诊断原因。
- A3：明确 invalid_encrypted_content 仅在有效安全改写后按原上限同账号重试；无改写、再次失败或预算耗尽终止，保留透传顺序，不新增换账号循环。
- A4：摘要收集、命中与剥离覆盖同一安全集合；只删命中片段，HTTP/WS replay 一致，其他会话和分组隔离。
- A5：不原地改写 replay 字节，JSON 大整数精度保持，未命中原样返回，损坏 JSON 走明确错误或既有失败路径；避免本修复引入的重复解码和未命中切片分配。
- A6：健康统计需求已取消。账号健康、调度、handler/result 传递、各协议错误归因与通用流终态处理不包含本 change 的行为变更；通过最终差异检查确认相关上一候选修改均已撤销。
- A7：恢复日志区分识别数量、实际改写、跳过原因、重试和结果，并复用请求/账号关联信息；不新增明文、密文、凭证或完整正文泄露。
- A8：裁剪后定向回归、受影响 Go 包必要检查和独立只读验收覆盖全部当前条目；旧候选结果不冒充新候选结果，失败和环境限制保留。

# Constraints and invariants

- 不扩大错误识别到所有 400，不因消息 ID 前缀推断载体类型。
- 原有同账号重试预算与错误透传顺序保持，只有有效改写才重试。
- lineage 复用原会话/分组范围，不记录密文原文或扩大为全局失效。
- 撤销仅限本 change 的已识别修改，保留任何无关用户修改；不整仓重置。
- 不创建扫描框架或新抽象；在现有共用函数内消除重复工作。

# Decisions

2026-11-06 用户明确取消“加密内容错误不污染健康统计”并要求继续。因此撤销旧 A6 及其调用链扩展，新的 A6 仅验证这些行为保持基线。保留载体恢复、lineage、日志和必要测试；重新形成候选并执行独立验收，旧候选的全项通过不直接沿用。

# Verification expectations

先定向检查，再由 Runtime 检查受影响 service 包、vet、格式与最终文件边界，并由独立 Verifier 验收。golangci-lint 与 Go 导出格式不兼容的旧失败继续披露，不声称 lint 通过。旧预热时限失败原因未知，既有测试断言不放宽；不为本次裁剪新增预热逻辑。若启动临时数据库，仅关闭本任务实例；本任务不需要生产重放。
