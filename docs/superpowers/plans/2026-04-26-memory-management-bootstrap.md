# 项目级 memory-management 初始化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为当前仓库创建 `memory-management` 所需的最小项目级记忆骨架，并保证目录结构可被 Git 持久保存。

**Architecture:** 在仓库根目录新增一个轻量 `CLAUDE.md` 作为热缓存，并新增 `memory/glossary.md` 作为完整术语表入口。`memory/people/`、`memory/projects/`、`memory/context/` 通过 `.gitkeep` 占位文件保持目录被版本库跟踪，同时不预填任何虚构项目内容。

**Tech Stack:** Markdown、Git、仓库现有 `docs/superpowers` 设计/计划流程。

---

## File Structure

- Create: `CLAUDE.md`
  - 作为项目级热缓存，提供 `Me`、`People`、`Terms`、`Projects`、`Preferences` 五个主要区块。
- Create: `memory/glossary.md`
  - 作为完整术语、别名和项目代号的集中入口。
- Create: `memory/people/.gitkeep`
  - 保留人物详细资料目录，无示例人物文件。
- Create: `memory/projects/.gitkeep`
  - 保留项目详细资料目录，无示例项目文件。
- Create: `memory/context/.gitkeep`
  - 保留团队/流程/工具上下文目录，无示例上下文文件。

---

### Task 1: 创建 `CLAUDE.md` 热缓存骨架

**Files:**
- Create: `CLAUDE.md`

- [ ] **Step 1: 运行缺失校验，确认当前仓库还没有 `CLAUDE.md`**

Run:

```bash
python - <<'PY'
from pathlib import Path
path = Path('CLAUDE.md')
assert path.exists(), 'CLAUDE.md is missing'
PY
```

Expected: FAIL，提示 `CLAUDE.md is missing`。

- [ ] **Step 2: 写入最小 `CLAUDE.md` 内容**

将 `CLAUDE.md` 创建为：

```md
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
```

- [ ] **Step 3: 运行结构校验，确认 `CLAUDE.md` 具备必须区块**

Run:

```bash
python - <<'PY'
from pathlib import Path

content = Path('CLAUDE.md').read_text(encoding='utf-8')
required = [
    '# Memory',
    '## Me',
    '## People',
    '## Terms',
    '## Projects',
    '## Preferences',
]
missing = [item for item in required if item not in content]
assert not missing, f'missing sections: {missing}'
print('CLAUDE.md structure OK')
PY
```

Expected: PASS，并输出 `CLAUDE.md structure OK`。

- [ ] **Step 4: 检查 diff 只包含热缓存骨架**

Run:

```bash
git diff -- CLAUDE.md
```

Expected: diff 仅包含 `CLAUDE.md` 的 Markdown 骨架，不包含虚构人物、术语或项目示例。

---

### Task 2: 创建 `memory/` 深层记忆结构与术语表入口

**Files:**
- Create: `memory/glossary.md`
- Create: `memory/people/.gitkeep`
- Create: `memory/projects/.gitkeep`
- Create: `memory/context/.gitkeep`

- [ ] **Step 1: 运行缺失校验，确认 `memory/` 结构尚未初始化**

Run:

```bash
python - <<'PY'
from pathlib import Path

required = [
    Path('memory'),
    Path('memory/glossary.md'),
    Path('memory/people'),
    Path('memory/projects'),
    Path('memory/context'),
]
missing = [str(path) for path in required if not path.exists()]
assert not missing, f'missing paths: {missing}'
PY
```

Expected: FAIL，提示缺少 `memory` 相关路径。

- [ ] **Step 2: 创建目录占位文件，保证 Git 可以跟踪空目录**

Run:

```bash
mkdir -p memory/people memory/projects memory/context
touch memory/people/.gitkeep memory/projects/.gitkeep memory/context/.gitkeep
```

Expected: 命令成功，无报错。

- [ ] **Step 3: 写入 `memory/glossary.md` 最小内容**

将 `memory/glossary.md` 创建为：

```md
# Glossary

项目术语、缩写、别名和代号统一记录在这里。

## Acronyms

| Term | Meaning | Context |
|------|---------|---------|

## Internal Terms

| Term | Meaning |
|------|---------|

## Nicknames → Full Names

| Nickname | Person |
|----------|--------|

## Project Codenames

| Codename | Project |
|----------|---------|
```

- [ ] **Step 4: 运行结构校验，确认所有路径与区块存在**

Run:

```bash
python - <<'PY'
from pathlib import Path

required_paths = [
    Path('memory/glossary.md'),
    Path('memory/people/.gitkeep'),
    Path('memory/projects/.gitkeep'),
    Path('memory/context/.gitkeep'),
]
for path in required_paths:
    assert path.exists(), f'missing path: {path}'

content = Path('memory/glossary.md').read_text(encoding='utf-8')
required_sections = [
    '# Glossary',
    '## Acronyms',
    '## Internal Terms',
    '## Nicknames → Full Names',
    '## Project Codenames',
]
missing = [item for item in required_sections if item not in content]
assert not missing, f'missing sections: {missing}'
print('memory structure OK')
PY
```

Expected: PASS，并输出 `memory structure OK`。

- [ ] **Step 5: 检查 diff 只包含术语入口和目录占位文件**

Run:

```bash
git diff -- memory/glossary.md memory/people/.gitkeep memory/projects/.gitkeep memory/context/.gitkeep
```

Expected: diff 仅包含 `glossary.md` 骨架和空的 `.gitkeep` 新文件。

---

### Task 3: 验证初始化结果无示例污染且范围正确

**Files:**
- Verify: `CLAUDE.md`
- Verify: `memory/glossary.md`
- Verify: `memory/people/.gitkeep`
- Verify: `memory/projects/.gitkeep`
- Verify: `memory/context/.gitkeep`

- [ ] **Step 1: 运行内容检查，确认没有示例术语污染初始化结果**

Run:

```bash
python - <<'PY'
from pathlib import Path

forbidden = ['Todd', 'PSR', 'Phoenix', 'Horizon', 'Sarah', 'Greg']
files = [Path('CLAUDE.md'), Path('memory/glossary.md')]
hits = []
for file in files:
    content = file.read_text(encoding='utf-8')
    for token in forbidden:
        if token in content:
            hits.append(f'{file}: {token}')
assert not hits, 'unexpected demo content: ' + ', '.join(hits)
print('no demo content found')
PY
```

Expected: PASS，并输出 `no demo content found`。

- [ ] **Step 2: 运行路径总览校验，确认骨架完整**

Run:

```bash
python - <<'PY'
from pathlib import Path

paths = [
    Path('CLAUDE.md'),
    Path('memory'),
    Path('memory/glossary.md'),
    Path('memory/people'),
    Path('memory/projects'),
    Path('memory/context'),
]
for path in paths:
    assert path.exists(), f'missing path: {path}'
print('bootstrap complete')
PY
```

Expected: PASS，并输出 `bootstrap complete`。

- [ ] **Step 3: 查看最终工作区变更范围**

Run:

```bash
git status --short CLAUDE.md memory
```

Expected:

```text
A  CLAUDE.md
A  memory/context/.gitkeep
A  memory/glossary.md
A  memory/people/.gitkeep
A  memory/projects/.gitkeep
```

- [ ] **Step 4: 提交门禁**

Run:

```bash
git diff -- CLAUDE.md memory/glossary.md memory/people/.gitkeep memory/projects/.gitkeep memory/context/.gitkeep
```

Expected: 仅包含本次初始化骨架。**只有在用户明确要求提交时**，才执行 `git add` / `git commit`。
