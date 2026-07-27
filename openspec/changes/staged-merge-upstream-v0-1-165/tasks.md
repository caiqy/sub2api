## 1. 阶段 0：基线与能力保护门禁

- [x] 1.1 记录 base ref `075abc073` 与 `backend/cmd/server/VERSION=0.1.159.6`，重新 fetch upstream tags，确认六个目标 tag 的祖先链且 `v0.1.165` 仍为最新正式 release；若出现更新 tag，暂停更新范围
- [x] 1.2 按 build 阶段确认的隔离方式创建工作分支/工作区，归属 Comet 规划产物且不夹带 `paseo.json` 等无关改动
- [x] 1.3 在当前本地 HEAD 上运行本地 full 门禁基线：根目录 `make test` 与 `make build`；失败项记录为阻塞
- [x] 1.4 通过 `ssh-skill` 将已提交 HEAD 的 `git archive` 上传到 `local-serv-ai` 临时目录，检查 Go 工具链与 Docker；在 Linux 上按 `backend/scripts/test.ps1` 等价语义重建 `backend/.test-tmp`、设置 `TMPDIR`/`TMP`/`TEMP`，并运行 `CI=true GOFLAGS='-v' go test -tags=integration ./...`；全套命令和目标 migration/repository test 未真实通过均阻塞
- [x] 1.5 运行两次 `make -C backend generate` 并核对 Ent/Wire 第二次无新增 diff，记录生成稳定性基线
- [x] 1.6 建立能力映射矩阵：以主 spec 专项 review 清单和本地保护能力为行、6 个 tag changed files 为列，标记高风险交叉点
- [x] 1.7 核对被上游触及且缺少行为断言的本地能力，必要时先补最小失败测试

## 2. 合入 v0.1.160

- [x] 2.1 `git merge --no-ff v0.1.160`，按"上游修复+本地定制共存"原则处理冲突（重点：security-audit full prompt 与本地 privacy、Grok media 隔离、image_gen 权限、migration 181/182 及本地同号 181）
- [x] 2.2 运行 full 门禁（`make test`、`make build`、`local-serv-ai` Docker-backed integration、两次 backend generate、migration 新库/升级库、无冲突标记）并修复回归
- [x] 2.3 按能力矩阵完成 v0.1.160 触及能力的映射审查并记录证据

## 3. 合入 v0.1.161

- [x] 3.1 `git merge --no-ff v0.1.161`，处理冲突（重点：step-up 2FA 开关化、模型级临时冷却与本地 scheduler、Grok 视频代理、migration 183/184）
- [x] 3.2 运行 full 门禁并修复回归
- [ ] 3.3 完成 v0.1.161 触及能力的映射审查并记录证据

## 4. 合入 v0.1.162

- [ ] 4.1 `git merge --no-ff v0.1.162`，处理冲突（重点：客户端 IP 请求头与可信代理体系、Grok client tool 缓存与 sticky、S3 备份/image storage）
- [ ] 4.2 运行 full 门禁并修复回归
- [ ] 4.3 完成 v0.1.162 触及能力的映射审查，核对 settings JSON backfill 与配置热更新路径并记录证据

## 5. 合入 v0.1.163

- [ ] 5.1 `git merge --no-ff v0.1.163`，处理冲突（重点：OpenAI reasoning policy、scheduler quota metadata/LastUsedAt、优雅关停 Cleanup、计费修复、axios 安全升级、migration 185）
- [ ] 5.2 运行 full 门禁并修复回归
- [ ] 5.3 完成 v0.1.163 触及能力的映射审查并记录证据

## 6. 合入 v0.1.164

- [ ] 6.1 `git merge --no-ff v0.1.164`，处理冲突（重点：composite group routing、ollama Cloud、Grok 402 冷却、Alipay deep link、migration 172 与本地同号 172、两个上游 186）
- [ ] 6.2 运行 full 门禁并修复回归
- [ ] 6.3 专项审查 composite group routing 入口调用链与本地 advanced/layered scheduler、Grok/platform sticky 的交互，确认本地调度定制未被绕过并记录证据

## 7. 合入 v0.1.165

- [ ] 7.1 `git merge --no-ff v0.1.165`，处理冲突（重点：OpenAI Live gateway、ollama 用量刷新、email alias 注册查重、migration 187-190、postcss 安全升级）
- [ ] 7.2 运行 full 门禁并修复回归
- [ ] 7.3 专项审查 OpenAI Live gateway 与本地 prompt cache reuse、body replay 的交互，确认本地 OpenAI 定制仍生效并记录证据

## 8. 最终验证与收尾

- [ ] 8.1 将 `backend/cmd/server/VERSION` 规范为 `0.1.165.1` 并确认 `openai-first-token-timeout` 未被任何 tag 恢复
- [ ] 8.2 运行最终 full verify：`make test`、`make build`、`local-serv-ai` Docker-backed integration、Ent/Wire 两次生成稳定性检查
- [ ] 8.3 校验 Git 祖先关系（`v0.1.160`~`v0.1.165` 均为 HEAD 祖先）、6 个 `--no-ff` merge 节点、无冲突标记残留；确认 12 个上游 migration 与本地同号 172/181 均保留且新库/升级库验证通过
- [ ] 8.4 按本地保护清单逐项完成能力级专项 review 并输出验证报告（不含推送/发版/部署）
