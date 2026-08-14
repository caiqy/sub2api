# Task 11 第二阶段簇 C 审查证据

审查日期：2026-08-14（UTC）

## 范围与结论

- 审查对象：upstream `bba6a55e0`（已由 merge `d92137692` 进入当前 `HEAD=5f9256ed5`）的定时备份 leader 锁，与本地大文件分卷/S3 part/恢复和管理员操作契约的交互。
- Round 1 结论：主窗口将该能力收窄为 best-effort 单实例化，双锁域非对称故障的 split-brain 为已知残余风险。锁机制未改；本簇仅修正边界注释并补充兼容测试，纳入独立兼容提交。
- 工作树中既有的 `15-cluster-A.md` 修改属于其它簇，本审查未触碰。

## Leader 锁审查

- 锁介质：`tryAcquireSingletonLeaderLock` 优先调用 Redis `LeaderLockCache.TryAcquireLeaderLock`；实现使用 `SET NX`，完整 Redis key 为 `leader:lock:backup:scheduled:leader`。Redis 错误时尝试同一数据库的 PostgreSQL session advisory lock；正常 wire 注入同时提供 cache 和 DB。
- 锁键与所有者：备份专用键为 `backup:scheduled:leader`，服务实例启动时生成 UUID `instanceID` 作为 owner。Redis release 是按 owner compare-and-delete，PostgreSQL release 在保留连接上执行 advisory unlock 后关闭连接，旧 owner 不会删除新 owner 的锁。
- 超时与释放：定时任务 context 为 30 分钟，锁 TTL 为 35 分钟；分卷上传失败的 detached cleanup 另受 2 分钟限制，仍落在 TTL 之内。获取锁失败或被 peer 占有时，`runScheduledBackup` 在读取 schedule、创建记录、dump 和 S3 上传前直接返回。成功获取后 `defer release()` 覆盖 `CreateBackup`、清理和错误返回路径；Redis/DB release 各使用 2 秒 detached context。TTL 仅作 crash recovery 边界。
- 降级语义：cache 与 DB 都不可用时保留无锁单实例/单测行为；这不是集群默认路径。Redis 与 PostgreSQL 是互不相交的 best-effort 锁域，只有所有实例一致进入同一后端时才避免并发；局部非对称故障仍可能 split-brain 并并发执行。

## Round 1 兼容修复

- 注释：`leader_lock.go` 的 cache-error 分支明确上述双锁域边界，未改变 acquire、fallback、TTL 或 release 机制。
- 纯 GREEN 测试：现有 `go-sqlmock` 覆盖 DB advisory query failure 返回未获取、scheduled caller 不创建记录/不上传、cache error 后 PostgreSQL fallback acquire/release；既有 leader test 新增 backup dump failure 后的 defer release 断言。最初零值 `sql.DB` 的测试夹具触发 `database/sql` 内部 panic，已丢弃并换为有效 mock，不是生产行为 RED。

## 本地备份契约核对

- 分卷/S3：scheduled leader 最终仍进入同一 `CreateBackup` 路径。超过阈值时先生成并持久化 `running` 分卷计划，再逐卷上传；任一上传错误都会以 detached cleanup 删除计划中的全部对象，记录保持/更新为 `failed`，不会标记为 `completed`。重启恢复也会将遗留 `running` 记录标记失败并清理关联对象。
- 恢复：仅 `completed` 记录可恢复。旧单对象记录仍走流式 gzip 恢复；分卷记录先校验连续 index、非空 key、size 和 32-byte SHA-256，再下载、重组和恢复。缺卷、损坏或 metadata 不匹配均在调用 dumper 前失败。
- 管理员权限：所有 `/v1/admin/backups` 路由处于 `AdminAuthMiddleware` 管理组；手动创建、下载和恢复继续要求 step-up 2FA，恢复 handler 还要求当前管理员重新验证密码。leader 锁只包裹 scheduled 路径，不改变管理员手动操作语义。

## 验证

- `go -C backend test -tags=unit ./internal/service -run '^TestBackupService_' -count=1 -v`：退出码 `0`。新增 DB query failure 跳过和 backup failure release，保留分卷上传/恢复/清理覆盖。
- `go -C backend test ./internal/service -run '^TestTryAcquireSingletonLeaderLock_' -count=1 -v`：退出码 `0`。新增 DB query failure skip，以及 cache error 时 PostgreSQL fallback acquire/release。
- `go -C backend test ./internal/handler/admin -run 'Backup' -count=1 -v`：通过（2 项）。
- `go -C backend build ./...`：退出码 `0`。
- 初次未加 `-tags=unit` 的 service 测试未纳入 `backup_service_test.go`，原因是该文件声明 `//go:build unit`；已使用正确标签重跑，不是测试或实现失败。
- 原始命令、退出码与 `--- PASS:` 行：`16-cluster-C.log`。

## 残余风险

- Redis 与 PostgreSQL 是独立锁域，网络/依赖局部非对称故障可能导致 split-brain；这是主窗口已接受的 best-effort 残余风险。真实 Redis/PostgreSQL 多节点集成未运行，但 PostgreSQL fallback 的 query-error 与 acquire/release 已由 `go-sqlmock` 覆盖。
