# local-serv-ai 服务器 sub2api 更新流程

当用户要求“更新本地服务器上的 sub2api”“更新 local-serv-ai”“更新 192.168.2.110 上的 sub2api”或类似表达时，默认按以下流程执行。

## 目标环境

- SSH 别名：`local-serv-ai`
- 主机地址：`192.168.2.110`
- 用户：`root`
- 服务路径：`/data/sub2api`
- 部署方式：`docker-compose.yml`
- 应用容器：`sub2api`
- 镜像：`ghcr.io/caiqy/sub2api:latest`
- 依赖容器：`sub2api-postgres`、`sub2api-redis`

## 操作原则

- 所有远程操作必须使用 `ssh-skill`，不要直接调用 `ssh` 或 `scp`。
- 更新前先检查当前部署方式、当前镜像/版本、容器状态。
- 保持现有 `.env`、数据目录和 Compose 配置不变，只拉取并重建 `sub2api` 服务。
- 更新后必须做新鲜验证，再报告结果。

## 标准步骤

1. 确认服务器别名：`local-serv-ai`。
2. 在远端检查部署现状：
   - `ls -la /data/sub2api`
   - 查找 `docker-compose.yml`、`.env`
   - `docker compose ps`
   - `docker inspect sub2api` 查看当前镜像 ID 与健康状态
   - `docker compose images sub2api` 查看镜像创建时间与 ID
3. 更新应用服务：
   - `cd /data/sub2api`
   - `docker compose pull sub2api`
   - `docker compose up -d sub2api`
4. 验证：
   - 等待 `docker inspect sub2api` 健康状态为 `healthy`
   - 确认 `docker compose images sub2api` 已显示最新镜像
   - `curl -fsS http://127.0.0.1:8080/health` 应返回 `{"status":"ok"}`
   - 如需补充，再查看最近日志确认没有启动错误

## 2026-05-08 实际验证样例

- 更新前镜像 ID：`962ac496c403`
- 更新后镜像 ID：`599d4246fb68`
- 更新后镜像创建时间：`About an hour ago`
- 容器状态：`Up 3 minutes (healthy)`
- 健康接口：`{"status":"ok"}`
- 主机名：`ai-proxy-serv`

## 注意事项

- 当前连接使用用户目录下密钥 `C:/Users/caiqy/.ssh/id_ed25519`。
- 该密钥带有 passphrase；虽然当前工具可连接，但本机启用 `ssh-agent` 后体验会更稳定。
- PowerShell 本地调用 `ssh_execute.py` 时，包含 `||`、`{{...}}`、多层引号或 heredoc 的远端脚本容易被本地 shell 误解析；复杂验证脚本优先拆分为多个简单命令。
