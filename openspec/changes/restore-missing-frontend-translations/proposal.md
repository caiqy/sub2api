## Why

上游 v0.1.151 将前端中英文语言包从单文件拆分为域模块，合并时部分本地新增键未迁入新模块，导致界面直接显示 `admin.usage.viewDetail` 等原始 i18n key。

## What Changes

- 补回导航栏 AI 生图及管理端用量详情相关的中英文翻译。
- 增加静态 i18n key 完整性检查，防止语言包拆分或合并再次遗漏界面已引用的键。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

无。本次仅恢复既有界面文案，不改变产品行为或验收要求。

## Impact

仅影响前端语言包和 i18n 回归测试；不涉及 API、数据库或依赖变更。
