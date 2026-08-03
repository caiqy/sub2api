# dmit 服务器内容占用堵点分析

> 原文：2026-08-03 会话分析输出（未做任何变更；只读分析）

## 一、内存形态结论（结合交接文档 + 本轮代码/DB 分析）

**不是泄漏，是"瞬时持有 × 并发"的乘积问题**。GC 后谷底 392MB 是真实常驻，峰值 980MB 贴边是业务瞬时分配。日均 1.5 万请求（usage_logs 56.5 万行/38 天），每个请求在生命周期内持有以下大对象：

| 持有对象 | 大小量级 | 持有时长 | 证据 |
|---|---|---|---|
| 请求体 `[]byte` 全量驻留（v2 前状态） | ≤10MB/请求（DB 实测最大 1.3MB） | 整个请求生命周期 | v2 前 `request_body_handle.go` spool 阈值为 `10<<20`，**≤10MB 全部留在内存** |
| 内容审核同步调用期间持有 body | body + 提取文本 | 外部 API 往返 300ms~3s（超时上限 30s） | pre_block 模式走 `checkSync` **同步**审核；`input.Body = body()` 后不释放 |
| 审核日志 excerpt | 12000 runes 全量截断（avg 5802 字符/条） | 构建 log 对象期间 | DB 实测 max=12000 |
| usage_log_details 快照 | 响应体最大 3.3MB/条 | 写入 DB 期间 | DB 实测；retention=20 说明持续小量写入 |
| 流式响应 | 逐块转发 + 64KB scanner 池 + 首输出阶段 tempFile 兜底 | — | **这块代码已优化，不是问题** |

**峰值贴边 217KB 的直接原因**：并发高峰时（图片/长文请求叠加），分配速率超过 GC 回收速率，峰值穿透 GOMEMLIMIT 860MiB 软限制撞向 cgroup 980MiB。

## 二、配置可解（不动代码，改设置/环境变量即可）

**1. 内容审核 sample_rate 100% → 降（如 30~50%）**（收益最大、成本最低）
- 现状：`sample_rate:100`、`mode:pre_block`（同步阻塞请求）、`record_non_hits:true`、8 个组、8/2 单日审核 8874 次
- 每个被采样请求 = 一次同步外部 API 往返（内存持有 body 期间 + 请求延迟）
- 降采样直接砍掉这部分瞬时内存和延迟；代价是审核覆盖率

**2. GOMEMLIMIT 860MiB → 512MiB**（安全阀）
- 压低峰值距 cgroup 限制的距离，代价是 GC 更频繁、CPU/延迟略增
- 只治标，不解决瞬时分配根源

**3. 容器内存限制上调**（host 还有 ~394MB 余量）
- 给峰值留缓冲空间；代价是挤占整机余量（wireguard/其他容器用）

**4. Nginx `client_max_body_size` 限制**（如 8MB）
- 挡住超大请求体进入应用，从源头砍掉大 body 驻留；代价是大文件/图片请求被 413

**5. 顺带：`ops_system_logs` 开启采样**（当前 300MB/15 万行，配置项 enable_sampling=false）
- 对进程内存影响小，但 DB 膨胀和写放大明显

## 三、代码可解（需要改代码、走 GitHub Actions 发布）

**1. 调低 RequestBodyHandle spool 阈值 `10<<20` → 1MB**（已落地）
- `backend/internal/service/request_body_handle.go` 的默认 spool 阈值现为 1MB，preview 上限为 256KB。
- 图片/长文请求（实测 1.3MB）将自动落盘，内存只保留小 body 和有界 preview。

**2. 审核提取文本后尽早释放全量 body**
- `CheckLazy` 在 `ExtractContentModerationInput` + `Normalize()`（截断到 12000 runes）后，全量 `input.Body` 已无用途，但持有到 `checkSync` 外部调用结束
- 改成审核请求只携带截断后的 content（≤48KB），全量 body 在提取后即可释放

**3. 请求体全链路 handle 化（本 change 的落地方案）**
- 默认 spool 阈值为 1MB，preview 上限为 256KB。
- canonical body 在请求级改写完成后进入 final handle；failover/retry 每轮按需物化并重建 attempt handle。
- session hash 使用 normalize 前 raw body，usage hash 使用 channel mapping 前 body，cyber key 在 body 尚存时预计算。
- WSv2 reconnect map 不在本 change 范围，后续独立治理。

**4. pre_block 审核加并发上限（独立候选，未实施）**
- 超限时直接 allow（降级）而非堆积，防止突发并发把所有 body 压在审核等待队列里
- 或支持把 pre_block 的审核调用也异步化（observe 已有异步队列 + drop 机制，pre_block 没有）

**5. 开启 pprof**（诊断用，非优化本身）
- 目前 `/debug/pprof/` 被 SPA 路由吞掉，无法函数级定位。精确数据（哪个调用占峰值）需要它

## 四、数据库侧发现（与进程内存无直接关系，但值得知道）

- **总库约 2.2GB**；usage_logs 605MB 但**索引 368MB > heap 237MB**（13 个索引，单索引最大 101MB）
- **统计信息严重失真**：`n_live_tup` 显示 40264 行，实际 `count(*)` = **564,755 行**——autovacuum/analyze 没跟上，可能导致查询计划劣化
- ops_error_logs 316MB 仅 1705 行（每行 ~185KB TOAST 大字段，存了错误详情/响应体）
- content_moderation_logs 167MB：excerpt 存 12000 runes 全量文本而非 240 runes 摘要，可考虑降
- usage_log_details retention=20/5 已在收着（每写一条带响应体快照，最大 3.3MB）

## 建议优先级

| 优先级 | 动作 | 类型 | 预期效果 |
|---|---|---|---|
| 1 | 审核 sample_rate 100%→30~50% | 配置 | 砍掉最大同步持有源 |
| 2 | GOMEMLIMIT→512MiB 或容器限制上调 | 配置 | 立即远离 OOM 线 |
| 3 | 请求体全链路 handle 化（spool 1MB、preview 256KB） | 代码（已落地） | 大请求体落盘，transport 等待期不再按 body 大小线性驻留 |
| 4 | pre_block 审核并发上限 | 独立候选 | 限制同步审核等待队列的峰值持有 |
| 5 | pprof 开启 | 代码 | 拿到函数级证据再决定进一步动作 |

请求体治理 v2 已落地：2MB 与 8.9MB body 在 transport 读完并阻塞时，驻留 heap 增长保持在 3MB 预算内。生产配置与数据库未改，原生产数据和历史分析保留；pre_block semaphore、WSv2 reconnect map 与 pprof 仍是后续独立候选。
