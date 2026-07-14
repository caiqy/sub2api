## Context

合并前语言包为 `locales/zh.ts` 和 `locales/en.ts`，合并后改为按域拆分。Git 对 delete/modify 与新增拆分文件无法自动迁移对象属性，`nav.aiImages` 和 `admin.usage` 的详情文案因此丢失。

## Goals / Non-Goals

**Goals:**

- 将遗漏键迁入现有 `common.ts` 与 `admin/resources.ts`。
- 自动检查源码静态引用的 i18n key 在中英文语言包中都存在。

**Non-Goals:**

- 不调整文案风格、i18n 加载机制或组件结构。
- 不处理运行时拼接的动态 key。

## Decisions

- 沿用当前域模块：导航键放入 `common.ts`，用量详情键放入 `admin/resources.ts`。
- 使用 Node stdlib 扫描 `src` 下 `.vue/.ts` 文件中的静态 `t('key')` / `$t('key')` 调用，避免增加依赖。
- 同时校验 `zh` 与 `en`，防止 fallback 掩盖单语言遗漏。

## Risks / Trade-offs

- 静态扫描无法覆盖运行时拼接 key；本次丢失均为静态调用，且现有动态 key 继续由专项测试覆盖。
