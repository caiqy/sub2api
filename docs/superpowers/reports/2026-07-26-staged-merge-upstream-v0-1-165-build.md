# Staged Merge Upstream v0.1.165: Task 1 Build Ledger

## Fixed Baseline

- Branch: `feature/20260726/staged-merge-upstream-v0-1-165`
- HEAD: `075abc07399d6154130d2a2695fb24c785acd69c`
- `backend/cmd/server/VERSION`: `0.1.159.6`

## Release Tags

| Tag | Tag object | Peeled SHA |
| --- | --- | --- |
| `v0.1.160` | `2a519c0f8878aa8d9d75918e3acd734e536cc675` | `8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62` |
| `v0.1.161` | `317df5405c0ff1c67f12dcc0c669a16fc2e21dac` | `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c` |
| `v0.1.162` | `34b7a5ad70b4b9b9bb96955562fe632ad625d783` | `27f094e0960ebd8e52de7ff7e763c6fec2ff4057` |
| `v0.1.163` | `bb752ef7776dc126ffca5df9188087d0d0aed559` | `d0bdd7e771636a8d315f542cafd39484f39bd60c` |
| `v0.1.164` | `38a46fd33795c8946a1e88d0f72597c79ca02a76` | `cd8bb98c44303b2c8f04c0da340447c992f0cb7d` |
| `v0.1.165` | `892c8fa3ab80ada8a624668808c3e575da7c04d5` | `e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29` |

All adjacent peeled-tag ancestry checks exited `0`:
`v0.1.160 -> v0.1.161 -> v0.1.162 -> v0.1.163 -> v0.1.164 -> v0.1.165`.

## Release Boundary

- Latest formal tag merged into `upstream/main`: `v0.1.165`
- Exclusion command: `git log --oneline 'v0.1.165^{}..upstream/main'`
- Sole exclusion commit: `2730c1c43b29be003925b033f3f9e645e726bb8c chore: sync VERSION to 0.1.165 [skip ci]`

## Excluded User File

`git status --short` reports `?? paseo.json`; `git ls-files --error-unmatch
paseo.json` has no output and exits nonzero. It remains excluded from this
task and was not staged or modified.
