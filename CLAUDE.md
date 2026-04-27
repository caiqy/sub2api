# Memory

## Me

当前项目的简要定位、维护者偏好或协作背景可记录在这里。

## People

| Who | Role |
|-----|------|

→ Full list: `memory/glossary.md`, profiles: `memory/people/`

## Terms

| Term | Meaning |
|------|---------|

→ Full glossary: `memory/glossary.md`

## Projects

| Name | What |
|------|------|

→ Details: `memory/projects/`

## Preferences

- 将高频约定、长期偏好或重要协作规则记录在这里。
- 前端功能开发完成后，统一采用“先本地启动 dev 服务，再做浏览器调试预览”的方式验收；优先按“定位根因 → 先写失败测试 → 最小修复 → 重新跑测试/类型检查 → 浏览器实测”流程执行。
- 前端页面预览时，默认启动 `frontend` 的本地开发服务并提供可访问地址（如 `http://127.0.0.1:5173/...`），便于用户立即手动查看。
- 当用户只需要纯前端预览效果、无需依赖真实后端登录时，默认优先采用“本地 dev 服务 + 浏览器中注入临时管理员会话 + mock 目标接口数据”的方式直接进入目标页面，并尽量构造最贴近验收场景的 mock 数据。
- 发版时默认按“检查当前分支/HEAD/工作区状态 → 确认最新 tag 与版本序列 → 若用户指定版本号则严格按指定值发布 → 给当前目标提交打 tag 并 push → 校验远端 tag”执行；发版后默认不再查询 Release workflow 状态。
- 合并 `sub2api` 上游更新时，默认先核对 `upstream/main` 与最新 upstream tag，不预设主分支一定包含最新发布；优先在临时分支基于目标 release tag 合并，冲突时先保留“上游修复 + 本地定制”两边能力，再做专项回归审查与验证。

→ Workflow details: `memory/context/frontend-debug-preview.md`
→ Workflow details: `memory/context/release-workflow.md`
→ Workflow details: `memory/context/upstream-merge-workflow.md`
