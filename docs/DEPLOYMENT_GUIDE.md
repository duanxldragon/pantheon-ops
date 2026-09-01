# Pantheon Base 生产部署指南

**适用基线**：当前 `main` 与其认证后的 `pantheon-base-vX.Y.Z` foundation release
**最后更新**：2026-08-11

## 1. 支持范围

`pantheon-base` 提供单体后端与前端静态资源镜像。生产运行依赖：

| 组件 | 支持版本 | 用途 |
| --- | --- | --- |
| MySQL | 8.0+ | `pantheon_base` 主数据库 |
| Redis | 7.0+ | 认证会话、令牌吊销、限流与 Casbin watcher |
| OTLP 后端 | 可选 | OpenTelemetry traces |

仓库根目录的 `docker-compose.yml` 只启动 MySQL 与 Redis，供本地开发和验证使用；它不是完整生产编排文件。仓库当前也不提供可直接应用的 Kubernetes manifests。生产平台应基于本指南维护自己的 Secret、Deployment、Service、Ingress、备份和告警配置。

## 2. 构建镜像

从已认证的 tag 构建，并把同一版本注入后端日志和 OpenTelemetry resource：

```bash
git checkout pantheon-base-vX.Y.Z
docker build \
  --build-arg PANTHEON_VERSION=pantheon-base-vX.Y.Z \
  -t registry.example.com/pantheon-base:pantheon-base-vX.Y.Z \
  .
docker push registry.example.com/pantheon-base:pantheon-base-vX.Y.Z
```

Dockerfile 使用 Node.js 24 构建前端、Go 1.26.5 构建后端，最终进程以非 root 用户运行。不要从未通过 `Release Gate Summary` 的 commit 构建生产镜像。

## 3. 数据库初始化

数据库名固定为 `pantheon_base`。先创建空数据库和具有 DDL/DML 权限的应用账号，应用启动时会按顺序执行：

1. `golang-migrate` migrations；
2. system runtime seed；
3. i18n runtime seed。

不要挂载 `database/system_init.sql`。该文件仅保留为 deprecated 历史参考，不是当前 schema 来源。

默认启动路径使用 migrations。仅开发环境可使用 `PANTHEON_AUTO_MIGRATE=true`；生产环境不得启用 GORM AutoMigrate。

## 4. 必需环境变量

生产环境至少配置以下变量：

| 变量 | 要求 |
| --- | --- |
| `PANTHEON_ENV` | 固定为 `production` |
| `PANTHEON_DSN` | 指向独占的 `pantheon_base` 数据库 |
| `PANTHEON_REDIS_ADDR` | Redis 地址；认证链依赖 Redis |
| `PANTHEON_REDIS_PASSWORD` | Redis 密码 |
| `PANTHEON_INITIAL_ADMIN_PASSWORD` | 首次 seed 使用，至少 12 位 |
| `PANTHEON_ACCESS_TOKEN_SECRET` | 至少 32 字节 |
| `PANTHEON_REFRESH_TOKEN_SECRET` | 至少 32 字节 |
| `PANTHEON_OP_TOKEN_SECRET` | 至少 32 字节 |
| `PANTHEON_SETTING_SECRET` | 至少 32 字节 |
| `PANTHEON_MFA_SECRET` | 至少 32 字节 |
| `PANTHEON_ALLOWED_ORIGINS` | 明确列出生产前端 origin |
| `PANTHEON_ENABLE_DYNAMIC_MODULES` | 固定为 `false` |

可从 `.env.example` 复制变量名，但必须通过集群 Secret、Vault 或等价密钥系统注入真实值。不要提交生产 `.env`、DSN、密码或 token。

示例 DSN：

```text
pantheon_app:<password>@tcp(mysql:3306)/pantheon_base?charset=utf8mb4&parseTime=True&loc=UTC
```

## 5. Kubernetes 工作负载要求

生产 Deployment 至少应满足：

- 镜像使用不可变 digest 或认证后的版本 tag；
- Secret 提供第 4 节全部敏感变量；
- `PANTHEON_PORT=8080`；
- readiness 与 liveness 均请求 `GET /api/v1/health`；
- `terminationGracePeriodSeconds` 大于 `PANTHEON_SHUTDOWN_TIMEOUT_SECONDS`；
- 使用滚动发布，并在迁移失败时停止 rollout；
- 挂载持久存储到 `/app/uploads`（如果启用本地文件上传）；
- 不把 `/metrics` 直接暴露到公网。

探针示例：

```yaml
readinessProbe:
  httpGet:
    path: /api/v1/health
    port: 8080
  periodSeconds: 5
  timeoutSeconds: 3
livenessProbe:
  httpGet:
    path: /api/v1/health
    port: 8080
  periodSeconds: 10
  timeoutSeconds: 5
```

应用在 `SIGTERM` 后停止接收新连接、等待在途请求，并排空操作日志队列。默认关闭超时为 15 秒，可通过 `PANTHEON_SHUTDOWN_TIMEOUT_SECONDS` 调整。

## 6. 健康、指标与追踪

健康检查：

```bash
curl --fail http://127.0.0.1:8080/api/v1/health
```

生产环境的 `/metrics` 默认不会无保护暴露。选择其一：

- 设置 `PANTHEON_METRICS_BEARER_TOKEN`，并让 Prometheus 使用 Bearer token；
- 在已受网关和网络策略保护的内网显式设置 `PANTHEON_METRICS_PUBLIC=true`；
- 设置 `PANTHEON_METRICS_ENABLED=false` 完全关闭。

配置 `OTEL_EXPORTER_OTLP_ENDPOINT` 后启用 OTLP HTTP traces；未配置时不创建 exporter。

操作日志使用异步队列。队列满时新记录会被丢弃而不是同步降级写入，并增加：

```text
pantheon_operation_log_dropped_total
```

必须为该指标配置告警，并结合 `pantheon_operation_log_queue_depth` 调整 `PANTHEON_OPERATION_LOG_QUEUE_SIZE`。

## 7. 发布验证

发布后至少执行：

```bash
curl --fail https://<host>/api/v1/health
```

并确认：

1. 应用日志中的 `version` 等于部署 tag；
2. migrations 与 runtime seed 无错误；
3. 管理员可以登录并刷新会话；
4. Redis 中断会被监控捕获；
5. `/metrics` 未匿名暴露；
6. 动态模块注册在 production 被拒绝；
7. 操作日志 dropped counter 未持续增长。

## 8. 备份与恢复

备份 `pantheon_base`，不要使用旧数据库名 `pantheon`：

```bash
mysqldump --single-transaction -h <mysql-host> -u <backup-user> -p pantheon_base \
  | gzip > pantheon_base_$(date +%Y%m%d_%H%M%S).sql.gz
```

恢复前先停止写流量，在隔离环境验证备份，再恢复到空的 `pantheon_base` 数据库并启动同版本应用执行 migrations。上传文件使用本地卷时，数据库和 `/app/uploads` 必须采用一致的恢复点。

## 9. 回滚

1. 停止新版本 rollout；
2. 评估本次 migration 是否向后兼容；
3. 仅在 schema 兼容时回滚到上一认证镜像；
4. schema 不兼容时按 migration runbook 恢复数据库备份；
5. 回滚后重新验证健康、登录、权限、审计与指标。

不要仅回滚镜像而忽略已执行的数据库 migration。
