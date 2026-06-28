# dmit 服务器 sub2api 更新流程

当用户要求“更新 dmit 服务器上的 sub2api”或类似表达时，默认按以下流程执行。

## 目标环境

- SSH 别名：`dmit-serv-ai`
- 服务路径：`/data/sub2api`
- 部署方式：`docker-compose.yml`
- 应用容器：`sub2api`
- 镜像：`ghcr.io/caiqy/sub2api:latest`
- 依赖容器：`sub2api-postgres`、`sub2api-redis`

## 操作原则

- 所有远程操作必须使用 `ssh-skill`，不要直接调用 `ssh` 或 `scp`。
- 更新前先检查当前部署方式、当前版本、容器状态。
- 保持现有 `.env`、数据目录和 Compose 配置不变，只拉取并重建 `sub2api` 服务。
- 更新后必须做新鲜验证，再报告完成。

## 标准步骤

1. 查找/确认服务器别名：`dmit-serv-ai`。
2. 在远端检查部署现状：
   - `ls -la /data/sub2api`
   - 查找 `docker-compose.yml`、`.env`
   - `docker ps` 查看 `sub2api`、Postgres、Redis 状态
   - `docker inspect sub2api` 查看当前镜像 ID 与健康状态
   - `docker exec sub2api ... --version` 或读取 `/app/VERSION`，确认当前版本
3. 更新应用服务：
   - `cd /data/sub2api`
   - `docker compose pull sub2api`
   - `docker compose up -d sub2api`
4. 验证：
   - 等待 `docker inspect sub2api` 健康状态为 `healthy`
   - 确认版本输出为最新发布版本（例如 `Sub2API 0.1.126`）
   - `curl -fsS http://127.0.0.1:8080/health` 应返回 `{"status":"ok"}`
   - 查看最近日志，确认没有启动错误

## 2026-05-08 实际验证样例

- 更新前：`Sub2API 0.1.121.1`
- 更新后：`Sub2API 0.1.126`
- Commit：`2a8d23a372d0253b1a3ce74d152a45005e35d09b`
- Build time：`2026-05-08T01:45:31Z`
- 容器状态：`running`
- 健康状态：`healthy`
- 健康接口：`{"status":"ok"}`

## 注意事项

- PowerShell 本地调用 `ssh_execute.py` 时，包含 `||`、`{{...}}`、多层引号的远端脚本容易被本地 shell 误解析；复杂验证脚本可用 Base64 封装后传给远端执行。
- `docker compose pull` 的拉取进度可能输出到 stderr，但只要命令 exit code 为 0 且后续健康检查通过即可。
