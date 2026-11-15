---
generated_from_state_version: 10
---

# 验证

## 当前结果

- 结果: **已归档**
- 验证情况: **已完成检查，验证结果已确认**
- 目标周期: 1
- 迭代: 2
- 验证器尝试次数: 1
- 完成时间: 2026-11-15T12:05:10.341Z
- 摘要: 全新独立只读Verifier完成A1–A12全量验收，本阶段12/12 passed，无新增Blocker。完整逐项源码/原始日志/浏览器证据、实际命令、87路径和限制见output/merge-upstream-v0-2-21/stage3-independent-verifier-report.md；关联执行引用不代表认证身份。请求Runtime进入用户接受提案，后置main与learning尚未完成。

## 验收

| 编号 | 结果 | 来源 | 验收项 | 原因 |
| --- | --- | --- | --- | --- |
| A1 | passed | brief.md | [ ] A1 固定目标与边界：最终历史包含上述 fork 起点和 peeled commit，排除 tag 后提交；最终 VERSION 为 `0.2.20.8`，无远端写操作。 | 固定HEAD、main、peeled upstream和两父merge已独立核对；VERSION0.2.20.8，排除两个tag后提交，clean；本执行无远端写入，历史全局否定命题无法完全验证。报告A1。 |
| A2 | passed | brief.md | [ ] A2 完整融合：25 个冲突逐一按行为解决，87 个重叠文件完成交叉复核；Ent 同时保留 fork 和上游字段，SQL 插入/批量/查询/DTO 无列或参数错位，双方测试不因冲突被删弱。 | 25冲突行为及87重叠路径完成独立交叉复核，52实现增量2036行/测试1085行/Ent字段和生成回执；SQL71参数、dualTTL+traffic、single/batch/best_effort实测保持。报告A2及87路径表。 |
| A3 | passed | brief.md | [ ] A3 调度与客户端资格：default/weighted、候选/快照/DB 回退、sticky、previous-response、WaitPlan、故障重试及 token-count 均服从现行资格/利润/客户端规则；hydrate 失败已取得槽位只释放一次；上游 session seed 与 fork 增强并存，不恢复 layered/probe。 | 候选/快照/DB、default/weighted、sticky/previousresponse/WaitPlan均在取得或等待slot前后保持资格/隐私/配额/利润；hydrate失败一次release和token-count/seed回归通过。报告A3及源码精确行。 |
| A4 | passed | brief.md | [ ] A4 输入准入与安全：原始分组与最终分组限额取非零较小值，原客户端识别在重写前完成；HTTP/Responses streaming/WS 首轮和后续轮错误正确且先于 keepalive；重复键/Unicode/opaque 可见混合输入不绕过，拒绝不扣费、不惩罚账号。无效 UTF-8 返回 400。 | 原始与有效组取非零最小值，raw客户端verdict先于改写；重复/Unicode/opaque可见输入按原始次数校验，UTF8错误400；HTTP及WS逐turn在keepalive/上游前拒绝且无扣费/health惩罚。报告A4。 |
| A5 | passed | brief.md | [ ] A5 协议和请求体：最终映射/effort/DeepSeek 归一化后生成有效出站 body handle；跨重试可回放，上游等待不持有完整原始大 body；读取失败不退回全内存；timeout、cancel、panic、非空 response+error 均正确关闭/释放。budget 缺省自适应、显式 0 保持关闭限制。 | 最终passthrough/DeepSeek/指纹处理后创建handle；借用canonical不被账号污染、retry可回放、spool错误不fallback；关闭与timeout/cancel/panic/response+error专项随正式套件通过；缺省budget与显式0正确。报告A5。 |
| A6 | passed | brief.md | [ ] A6 WebSocket：TTFT 事件口径配置生效；后续轮失败不重发首轮 payload，按上游 1013 行为退出；websearch 仅完整历史允许重连；每轮保留 fork 输入限额、资格、映射、流量快照与失败语义。 | 共享配置TTFT排除codex和terminal；后续turn失败1013且不把首帧交替代账号；websearch重连严格safeRetry+完整replay历史；输入/资格/映射/traffic/pricing逐turn保留。报告A6。 |
| A7 | passed | brief.md | [ ] A7 图片与工具：原生图片成功/错误状态及 keepalive 正确，媒体引用和 tool 结果顺序、encrypted_function_args/namespace、Gemini thought 与正文分离完整；复合分组映射、image 工作流/鉴权/计费保留。 | native image turn header跳过预写whitespace保真实错误；普通keepalive保持；tool media顺序、encrypted_function_args/namespace和Gemini thought专项及fork image/鉴权/计费相关套件通过。报告A7；外部真实模型端到端未运行。 |
| A8 | passed | brief.md | [ ] A8 TTL 与收费：30m/12h 字段贯穿用量/渠道/区间/账号统计/DTO/UI；12h 缺省定价和显式 generic/12h 配置按上游优先级，free-input/Fast/cache-read 修复生效；fork 流量费只计完整成功且不进入账号成本回退，订阅窗口、事务/幂等/缓存失效保持。 | 双TTL贯穿parse/WS/RecordUsage/Ent/SQL/DTO/四表/UI；显式12h含0、generic/default/Fast/free-input/interval价格和traffic用户/账号统计分离由真实RecordUsage/集成检查证明；订阅锁/幂等/outbox保持。报告A8。 |
| A9 | passed | brief.md | [ ] A9 迁移安全：完整文件名为主键、trim 后 SHA256 保持；同号不同名的新迁移独立执行。空库安装、已有 fork 至253升级、重复执行、非事务索引无效状态恢复均有真实检查，历史迁移字节不变，数据不被回填破坏。 | 真实PG空库、fork至253既有数据升级、重复执行和人为制造invalid concurrent index恢复PASS；完整filename主键和trimSHA保持，两新迁移原样，历史blob不变。报告A9、Runtime integration1962–1967。 |
| A10 | passed | brief.md | [ ] A10 其余上游行为：EasyPay 实例 refund=0/1、Refresh Token 并发单次消费、安全审计即时唤醒、Antigravity 用量/限流、模型目录和 effort policy 有对应回归结果；原有鉴权安全与 fork 业务测试继续通过。 | EasyPay仅refund实例0/1；RefreshToken Lua原子消费/并发一次；安全审计9项真实PASS零SKIP及worker即时wake；Antigravity/目录/effort兼容回归保持。报告A10。 |
| A11 | passed | brief.md | [ ] A11 前端验收：测试、typecheck、i18n/build 成功；浏览器实际核验渠道 TTL/价格、Keys 统计、账号用量、设置保存与充值精度，同时检查 fork 输入限制/流量费/订阅入口没有回退；不能用类型检查替代浏览器。 | 独立复核历史原API/DOM副本及PNG，四表null/0/12h真实往返；Keys失败/真实retry、账号保留、设置两原生submit仅一PUT、充值6位精度和fork入口；最终Chromium iPhone仿真两PUT200，非真机Safari。正式432files/3791tests/typecheck/i18n/build通过。所有mock/drop/旧失败边界见报告A11。 |
| A12 | passed | brief.md | [ ] A12 检查与交付：后端全套测试、必要集成、lint 和前端检查记录真实结果；新的独立 Verifier 覆盖全部验收项，失败/跳过不记通过；组长验收后按 Runtime 完成本地 main 合并及学习检查。测试数据库/服务停止后检查本轮端口关闭，交付提交、证据及剩余风险。 | 本阶段独立全12项、Runtime14日志SHA/身份绑定和真实失败/复用/SKIP记录齐全；本轮精确清理回执及本人端口观察无监听/HEAD main clean。通过当前可验条件：组长接受后的main合并与最终learning尚未执行，不能宣称完整交付完成。全量lint历史30既有/新增0和SQL250 EOF例外完整保留，未改标失败。报告A12。 |

## 检查

| 检查 | 命令 | 工作目录 | 状态 | 退出码 | 耗时 |
| --- | --- | --- | --- | ---: | ---: |
| prior-results-binding | -NoProfile -File D:\Caiqy\Projects\Github\sub2api\output\merge-upstream-v0-2-21\stage3-check-round2.ps1 -Check prior-results-binding | . | passed | 0 | 1351 ms |
| go-lint-new | -NoProfile -File D:\Caiqy\Projects\Github\sub2api\output\merge-upstream-v0-2-21\stage3-check-round2.ps1 -Check go-lint-new | . | passed | 0 | 13198 ms |
| audit-real | -NoProfile -File D:\Caiqy\Projects\Github\sub2api\output\merge-upstream-v0-2-21\stage3-check-round2.ps1 -Check audit-real | . | passed | 0 | 15182 ms |

### Builder 报告的证据

以下为 Builder 报告，不等同于 Runtime 检查凭据或独立验收结果。

- Runtime round1 boundaries: passed — action=merge-upstream-v0-2-21:5; executionBinding=7189b094a1af9e049af90530e7f62c01a448bdb328637dbe4a5f705d8dbbda1a; log=native/sdk-checks/ad1584a73c9eeab1086efb791b3b16a772f522056d82d0b0650f7c951e73946e/logs/checks/c369d3b71bd36517c0099bd0c5db028ca64941019be691fa6a9d810d229dd4a0-boundaries.log; sha256=a0fc8b17a01a38f51017ea3ef69b53672daa92efcd0215d6839435807d5c4300; exit=0; durationMs=44970. Retained original result, not a new execution. go-integration command passed but 9 audit SKIPs are excluded until separate audit-real check.
- Runtime round1 go-default: passed — action=merge-upstream-v0-2-21:5; executionBinding=7189b094a1af9e049af90530e7f62c01a448bdb328637dbe4a5f705d8dbbda1a; log=native/sdk-checks/ad1584a73c9eeab1086efb791b3b16a772f522056d82d0b0650f7c951e73946e/logs/checks/c369d3b71bd36517c0099bd0c5db028ca64941019be691fa6a9d810d229dd4a0-go-default.log; sha256=75676019156d23264d5399e0abb7da60b4bb5699cc33b630c307b7c93c397470; exit=0; durationMs=617057. Retained original result, not a new execution. go-integration command passed but 9 audit SKIPs are excluded until separate audit-real check.
- Runtime round1 go-unit: passed — action=merge-upstream-v0-2-21:5; executionBinding=7189b094a1af9e049af90530e7f62c01a448bdb328637dbe4a5f705d8dbbda1a; log=native/sdk-checks/ad1584a73c9eeab1086efb791b3b16a772f522056d82d0b0650f7c951e73946e/logs/checks/c369d3b71bd36517c0099bd0c5db028ca64941019be691fa6a9d810d229dd4a0-go-unit.log; sha256=e4f9f3f6e65c790b81d1a1c421fd854c08ca6887a769f789a71305208aa9e41a; exit=0; durationMs=642798. Retained original result, not a new execution. go-integration command passed but 9 audit SKIPs are excluded until separate audit-real check.
- Runtime round1 go-integration: passed — action=merge-upstream-v0-2-21:5; executionBinding=7189b094a1af9e049af90530e7f62c01a448bdb328637dbe4a5f705d8dbbda1a; log=native/sdk-checks/ad1584a73c9eeab1086efb791b3b16a772f522056d82d0b0650f7c951e73946e/logs/checks/c369d3b71bd36517c0099bd0c5db028ca64941019be691fa6a9d810d229dd4a0-go-integration.log; sha256=11f1bd6dec6f2fd19a06b861bd6b57b007443bff107431b1eedc36e97b0ea7d9; exit=0; durationMs=130573. Retained original result, not a new execution. go-integration command passed but 9 audit SKIPs are excluded until separate audit-real check.
- Runtime round1 go-vet: passed — action=merge-upstream-v0-2-21:5; executionBinding=7189b094a1af9e049af90530e7f62c01a448bdb328637dbe4a5f705d8dbbda1a; log=native/sdk-checks/ad1584a73c9eeab1086efb791b3b16a772f522056d82d0b0650f7c951e73946e/logs/checks/c369d3b71bd36517c0099bd0c5db028ca64941019be691fa6a9d810d229dd4a0-go-vet.log; sha256=fba63cb4e47d53d37e214da7dfdf6ec79ecb464799a5554fcd2cc72c740fb571; exit=0; durationMs=23752. Retained original result, not a new execution. go-integration command passed but 9 audit SKIPs are excluded until separate audit-real check.
- Runtime round1 go-build: passed — action=merge-upstream-v0-2-21:5; executionBinding=7189b094a1af9e049af90530e7f62c01a448bdb328637dbe4a5f705d8dbbda1a; log=native/sdk-checks/ad1584a73c9eeab1086efb791b3b16a772f522056d82d0b0650f7c951e73946e/logs/checks/c369d3b71bd36517c0099bd0c5db028ca64941019be691fa6a9d810d229dd4a0-go-build.log; sha256=2e4c93d0a23b9eb84fbd6e77c2514a5e9b37b4318656f381cba1d0b8769078ab; exit=0; durationMs=9254. Retained original result, not a new execution. go-integration command passed but 9 audit SKIPs are excluded until separate audit-real check.
- Runtime round1 go-lint-new: failed — action=merge-upstream-v0-2-21:5; executionBinding=7189b094a1af9e049af90530e7f62c01a448bdb328637dbe4a5f705d8dbbda1a; log=native/sdk-checks/ad1584a73c9eeab1086efb791b3b16a772f522056d82d0b0650f7c951e73946e/logs/checks/c369d3b71bd36517c0099bd0c5db028ca64941019be691fa6a9d810d229dd4a0-go-lint-new.log; sha256=494638d050691e129211836d63f2c5dbde4dc76e09c5a621ebe208b91966d385; exit=7; durationMs=48983. Retained original result, not a new execution. go-integration command passed but 9 audit SKIPs are excluded until separate audit-real check.
- Runtime round1 frontend-test: passed — action=merge-upstream-v0-2-21:5; executionBinding=7189b094a1af9e049af90530e7f62c01a448bdb328637dbe4a5f705d8dbbda1a; log=native/sdk-checks/ad1584a73c9eeab1086efb791b3b16a772f522056d82d0b0650f7c951e73946e/logs/checks/c369d3b71bd36517c0099bd0c5db028ca64941019be691fa6a9d810d229dd4a0-frontend-test.log; sha256=e81c16c63087334bc8310fd1ed4ec30050b25d5e5a1324ef2f169b2219ebfcc7; exit=0; durationMs=116204. Retained original result, not a new execution. go-integration command passed but 9 audit SKIPs are excluded until separate audit-real check.
- Runtime round1 frontend-typecheck: passed — action=merge-upstream-v0-2-21:5; executionBinding=7189b094a1af9e049af90530e7f62c01a448bdb328637dbe4a5f705d8dbbda1a; log=native/sdk-checks/ad1584a73c9eeab1086efb791b3b16a772f522056d82d0b0650f7c951e73946e/logs/checks/c369d3b71bd36517c0099bd0c5db028ca64941019be691fa6a9d810d229dd4a0-frontend-typecheck.log; sha256=16b4228d1c57ac309173bcc09842df39f1fb79fb760684c665e732ec47983459; exit=0; durationMs=27688. Retained original result, not a new execution. go-integration command passed but 9 audit SKIPs are excluded until separate audit-real check.
- Runtime round1 frontend-build: passed — action=merge-upstream-v0-2-21:5; executionBinding=7189b094a1af9e049af90530e7f62c01a448bdb328637dbe4a5f705d8dbbda1a; log=native/sdk-checks/ad1584a73c9eeab1086efb791b3b16a772f522056d82d0b0650f7c951e73946e/logs/checks/c369d3b71bd36517c0099bd0c5db028ca64941019be691fa6a9d810d229dd4a0-frontend-build.log; sha256=09650670a0899366730d0c5144086fa6370f327b1bc9115151c53b3cc526de98; exit=0; durationMs=91710. Retained original result, not a new execution. go-integration command passed but 9 audit SKIPs are excluded until separate audit-real check.
- Runtime round1 frontend-lint: passed — action=merge-upstream-v0-2-21:5; executionBinding=7189b094a1af9e049af90530e7f62c01a448bdb328637dbe4a5f705d8dbbda1a; log=native/sdk-checks/ad1584a73c9eeab1086efb791b3b16a772f522056d82d0b0650f7c951e73946e/logs/checks/c369d3b71bd36517c0099bd0c5db028ca64941019be691fa6a9d810d229dd4a0-frontend-lint.log; sha256=cccb740c56bfbc9026dbe700305eaf5325555811cf5b32c13bf3098366001d32; exit=0; durationMs=3956. Retained original result, not a new execution. go-integration command passed but 9 audit SKIPs are excluded until separate audit-real check.
- Pinned compatible lint development preflight: passed — stage3-lint-fixed-preflight.log and completed process exit0; golangci-lint2.14.0 built Go1.27.3; 0 issues; no SDK or product dependencies changed.
- Real audit integration development preflight: passed — stage3-audit-real-preflight.log and completed process exit0; all 9 explicitly required cases PASS, no SKIP; owned PostgreSQL16/Redis8.4 fixtures. Runtime repetition planned.
- Historical full uncapped lint: failed — lint-uncapped/lint-baseline-comparison.json at c77071a5 versus fixed main: baseline30/candidate30/existing30/new0. Remains old execution, not relabeled current all-green.
- Raw git diff --check upstream SQL EOF: failed — stage3-runtime-round1/boundaries.log retains raw exit2 for upstream unchanged 250 SQL EOF; exact upstream blob match and excluding only that inherited file passes.
- 已知限制: 全量lint历史30项基线告警仍exit1；Runtime仅将增量lint作为成功门禁，原失败不隐藏。
- 已知限制: 上游SQL250 EOF空行原始diff-check exit2；原样blob保持，边界检查仅验证例外确为唯一且继承。
- 已知限制: 旧Go/前端检查保留原提交身份；下面计划产生新的最终候选Runtime执行记录。
- 已知限制: 无真实外部付费网关/支付E2E；quota及checkout展示mock明确标识，不证明外部服务。
- 已知限制: 若干历史浏览器capture已达存储上限、骨架/DOM几何显式tool结果复制、部分截图是最终态，报告保留缺失边界。
- 已知限制: 手机验证为Chromium iPhone14仿真，不是Safari或物理真机。
- 已知限制: A12正式Verifier/用户接受/归档及本地main交付尚未执行；verify.confirm交组长/用户，不代accept-result。
- 已知限制: First Runtime checks round has a failed lint check due old binary export-data incompatibility; never labeled passed. Current compatible tool uses pinned v2.14.0 and Go1.27.3, isolated output directory.
- 已知限制: First integration command has nine explicitly skipped securityaudit cases; separate real audit check now covers them with fresh isolated fixtures and rejects any remaining SKIP.
- 已知限制: Round2 prior-results-binding reuses 10 first-round current-HEAD Runtime passes with original raw log SHA validation; it is not ten new executions.

## 阻塞项

_无。_

## 风险与跳过的工作

- 全量lint历史baseline30/c770candidate30既有/新增0仍为失败；Action5旧工具exit7保留，Action7最终兼容工具仅增量0issues，不声称全量lint绿。
- raw git diff --check exit2只因backend/migrations/250_openai_cache_write_ttl.sql:25上游原样EOF空行；保留原样而列明例外。
- 浏览器历史局部mock、旧capture丢包/截断和Vite中断均有边界；最终iPhone是Chromium仿真，真实Safari/外部上游/真实支付/真实重置无法验证。
- A12接受后main合并和最终learning check尚待组长和Runtime；本Verifier不执行accept-result/archive/main或远端写入。

## 之前的迭代

| 目标周期 | 迭代 | 尝试 | 结果 | 未解决项 | 摘要 | 完成时间 |
| ---: | ---: | ---: | --- | --- | --- | --- |
| 1 | 1 | 0 | recovery | — | Runtime checks failed: go-lint-new. Preserve their receipts and submit a new Builder candidate after repair. | 2026-11-15T11:24:16.527Z |
| 1 | 2 | 1 | pass | — | 全新独立只读Verifier完成A1–A12全量验收，本阶段12/12 passed，无新增Blocker。完整逐项源码/原始日志/浏览器证据、实际命令、87路径和限制见output/merge-upstream-v0-2-21/stage3-independent-verifier-report.md；关联执行引用不代表认证身份。请求Runtime进入用户接受提案，后置main与learning尚未完成。 | 2026-11-15T12:05:10.341Z |



## 结论

全新独立只读Verifier完成A1–A12全量验收，本阶段12/12 passed，无新增Blocker。完整逐项源码/原始日志/浏览器证据、实际命令、87路径和限制见output/merge-upstream-v0-2-21/stage3-independent-verifier-report.md；关联执行引用不代表认证身份。请求Runtime进入用户接受提案，后置main与learning尚未完成。
