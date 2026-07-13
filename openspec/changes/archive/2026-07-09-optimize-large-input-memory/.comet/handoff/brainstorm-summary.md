# Brainstorm Summary

- Change: optimize-large-input-memory
- Date: 2026-07-09

## 已确认事实

- Nginx 已承担 80MiB 请求体入口上限，程序端本 change 不新增业务层大 input 拦截策略。
- 程序侧目标是减少合法大请求进入后在内容审计和 usage 记录链路中的额外内存副本。
- 内容审计遇到超大文本时保留最新内容，符合现有 `trimLatestRunes` 倾向。

## 确认的技术方案

采用候选 A：局部收集器。

- 新增内容审计 collector，替代 `parts []string` / `images []string` 的无界收集。
- 文本按“最新内容”保留，达到上限后丢弃旧内容；最终输出仍走现有 normalize/hash/ModerationInput。
- 图片在收集阶段限量，达到 `maxContentModerationInputImages` 后不再读取/拼接额外 base64 data URL。
- usage 记录先聚焦 OpenAI Responses 路径：提交任务前构造轻量快照，避免闭包保留 body、gin context 或完整大对象。
- 不做流式 JSON 抽取，不新增对象池，不改变 Nginx/`gateway.max_body_size` 入口限制职责。

## 关键取舍与风险

- 放弃流式 JSON 抽取，换取小 diff 和低协议风险；基础请求体仍会完整读入内存。
- 内容审计只覆盖保留后的最新文本和有限图片；旧历史上下文可能不再进入审计。
- usage 快照化需要确保计费、分组、渠道、图片计费字段不丢失。

## 测试策略

- 内容审计单测覆盖超大 Responses input，断言只保留最新文本且输出长度受限。
- 图片抽取单测覆盖多 inline/base64 图片，断言超过数量后不构造额外 data URL。
- OpenAI usage 记录回归测试覆盖计费字段、图片计费字段、渠道字段与现有行为一致。
- 合成大 input 跑一次局部验证，比较请求结果与临时分配趋势。

## 候选方案记录

- 候选 A：局部收集器。将 `parts []string` / `images []string` 替换为小型 collector，在 `add` 时截断文本、限量图片。
- 候选 B：流式 JSON 抽取。边读 body 边解析 JSON 并提取审计内容。
- 候选 C：只调小配置/队列。依赖入口和 worker 上限降低峰值。

## Spec Patch

无。
