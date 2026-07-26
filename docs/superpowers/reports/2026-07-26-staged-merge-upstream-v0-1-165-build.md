# 分段合入上游 v0.1.165：Task 1 构建台账

## 固定基线

- 分支：`feature/20260726/staged-merge-upstream-v0-1-165`
- HEAD: `075abc07399d6154130d2a2695fb24c785acd69c`
- `backend/cmd/server/VERSION`: `0.1.159.6`

## 正式 release tag

| Tag | Tag 对象 | Peeled SHA |
| --- | --- | --- |
| `v0.1.160` | `2a519c0f8878aa8d9d75918e3acd734e536cc675` | `8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62` |
| `v0.1.161` | `317df5405c0ff1c67f12dcc0c669a16fc2e21dac` | `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c` |
| `v0.1.162` | `34b7a5ad70b4b9b9bb96955562fe632ad625d783` | `27f094e0960ebd8e52de7ff7e763c6fec2ff4057` |
| `v0.1.163` | `bb752ef7776dc126ffca5df9188087d0d0aed559` | `d0bdd7e771636a8d315f542cafd39484f39bd60c` |
| `v0.1.164` | `38a46fd33795c8946a1e88d0f72597c79ca02a76` | `cd8bb98c44303b2c8f04c0da340447c992f0cb7d` |
| `v0.1.165` | `892c8fa3ab80ada8a624668808c3e575da7c04d5` | `e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29` |

所有相邻 peeled tag 的祖先关系检查均以退出码 `0` 通过：
`v0.1.160 -> v0.1.161 -> v0.1.162 -> v0.1.163 -> v0.1.164 -> v0.1.165`.

## release 边界

- 已合入 `upstream/main` 的最新正式 tag：`v0.1.165`
- 排除命令：`git log --oneline 'v0.1.165^{}..upstream/main'`
- 唯一排除提交：`2730c1c43b29be003925b033f3f9e645e726bb8c chore: sync VERSION to 0.1.165 [skip ci]`

## 排除的用户文件

`git status --short` 显示 `?? paseo.json`；`git ls-files --error-unmatch
paseo.json` 没有输出且以非零退出码结束。该文件继续排除在本任务之外，
未被暂存或修改。
