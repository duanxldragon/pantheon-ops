# Pantheon-Ops v1.2 用户手册

**版本**: v1.2.0  
**更新日期**: 2026-08-28

---

## 目录

1. [系统概述](#系统概述)
2. [快速开始](#快速开始)
3. [功能模块](#功能模块)
4. [API 文档](#api-文档)
5. [部署指南](#部署指南)
6. [故障排查](#故障排查)

---

## 系统概述

Pantheon-Ops 是一个企业级 Kubernetes 运维管理平台，提供以下核心功能：

- **ArgoCD GitOps 集成**: 自动化应用部署和配置管理
- **Velero 备份恢复**: 集群数据备份和灾难恢复
- **监控告警系统**: 实时监控和多渠道告警通知
- **文件分发系统**: 批量文件传输和分发
- **服务实例管理**: 应用实例状态管理和健康检查

### 系统架构

```
┌─────────────────────────────────────────┐
│           前端 (React + TypeScript)      │
├─────────────────────────────────────────┤
│           API 网关 (Gin)                 │
├─────────────────────────────────────────┤
│  业务层                                  │
│  ├─ ArgoCD 集成                          │
│  ├─ Velero 备份                          │
│  ├─ 监控告警                             │
│  ├─ 文件分发                             │
│  └─ 服务实例管理                         │
├─────────────────────────────────────────┤
│  数据层 (PostgreSQL)                     │
└─────────────────────────────────────────┘
```

---

## 快速开始

### 环境要求

- Go 1.19+
- Node.js 16+
- PostgreSQL 13+
- Kubernetes 1.20+

### 安装步骤

1. **克隆代码**
```bash
git clone https://github.com/your-org/pantheon-ops.git
cd pantheon-ops
```

2. **配置环境变量**
```bash
# 数据库配置
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=pantheon_ops
export DB_USER=postgres
export DB_PASSWORD=your-password

# 加密密钥 (必需，32字节)
export PANTHEON_ENCRYPTION_KEY="your-32-byte-encryption-key-here"

# API 服务端口
export API_PORT=8080
```

3. **启动后端**
```bash
cd backend
go mod download
go run main.go
```

4. **启动前端**
```bash
cd frontend
npm install
npm start
```

5. **访问系统**
```
http://localhost:3000
```

---

## 功能模块

### 1. ArgoCD GitOps 集成

#### 功能特性
- ArgoCD 服务器管理
- Git 仓库集成
- 应用自动部署
- 状态实时同步
- 部署历史记录

#### 使用步骤

**1.1 添加 ArgoCD 服务器**
```bash
POST /api/v1/argocd/servers
{
  "name": "Production ArgoCD",
  "clusterId": 1,
  "endpoint": "https://argocd.example.com",
  "authToken": "argocd-token-xxx"
}
```

**1.2 添加 Git 仓库**
```bash
POST /api/v1/argocd/repositories
{
  "serverId": 1,
  "repoURL": "https://github.com/org/repo",
  "name": "Main Repo",
  "username": "git-user",
  "password": "git-token"
}
```

**1.3 创建应用**
```bash
POST /api/v1/argocd/applications
{
  "name": "my-app",
  "serverId": 1,
  "repoURL": "https://github.com/org/repo",
  "path": "k8s/",
  "targetRevision": "main",
  "project": "default",
  "namespace": "production"
}
```

#### 注意事项
- AuthToken 将自动加密存储
- 状态同步每 30 秒自动执行
- 支持多 ArgoCD 服务器管理

---

### 2. Velero 备份恢复

#### 功能特性
- 集群数据备份
- 定时备份计划
- 快速恢复
- 备份存储管理

#### 使用步骤

**2.1 配置存储位置**
```bash
POST /api/v1/k8s/storage-locations
{
  "name": "S3 Backup",
  "provider": "aws",
  "bucket": "backups",
  "region": "us-east-1",
  "accessKeyId": "AWS_ACCESS_KEY",
  "secretAccessKey": "AWS_SECRET_KEY"
}
```

**2.2 创建备份**
```bash
POST /api/v1/k8s/backups
{
  "name": "daily-backup",
  "clusterId": 1,
  "includedNamespaces": ["default", "production"],
  "ttl": 604800
}
```

**2.3 创建定时备份**
```bash
POST /api/v1/k8s/schedules
{
  "name": "daily",
  "clusterId": 1,
  "cronExpression": "0 1 * * *",
  "enabled": true
}
```

**2.4 恢复备份**
```bash
POST /api/v1/k8s/restores
{
  "name": "restore-20260828",
  "backupName": "daily-backup",
  "clusterId": 1
}
```

---

### 3. 监控告警系统

#### 功能特性
- Prometheus 数据源管理
- PromQL 告警规则
- 自动告警评估
- 多渠道通知 (Email, Webhook, Slack, 钉钉, 飞书)

#### 使用步骤

**3.1 添加 Prometheus 数据源**
```bash
POST /api/v1/observability/datasources
{
  "name": "Production Prometheus",
  "type": "prometheus",
  "url": "http://prometheus:9090"
}
```

**3.2 创建告警规则**
```bash
POST /api/v1/observability/alert-rules
{
  "name": "High CPU Usage",
  "datasourceId": 1,
  "enabled": true,
  "query": "avg(rate(cpu_usage[5m])) > 0.8",
  "interval": 60,
  "forDuration": 300,
  "severity": "warning",
  "labels": {
    "team": "ops"
  }
}
```

**3.3 配置通知渠道**
```bash
POST /api/v1/observability/notification-channels
{
  "name": "Slack Alerts",
  "type": "slack",
  "enabled": true,
  "config": {
    "webhookUrl": "https://hooks.slack.com/services/xxx"
  }
}
```

#### 支持的通知渠道
- **Email**: SMTP 邮件通知
- **Webhook**: HTTP POST 通知
- **Slack**: Slack 消息推送
- **钉钉**: 钉钉群机器人
- **飞书**: 飞书群机器人

---

### 4. 文件分发系统

#### 功能特性
- 文件存储管理
- SFTP 批量传输
- 分发任务调度
- 传输进度监控

#### 使用步骤

**4.1 创建存储**
```bash
POST /api/v1/filedist/storages
{
  "name": "Local Storage",
  "type": "local",
  "basePath": "/data/files",
  "maxSize": 10737418240
}
```

**4.2 上传文件**
```bash
POST /api/v1/filedist/files/upload
Content-Type: multipart/form-data

storageId: 1
file: [binary data]
```

**4.3 创建分发任务**
```bash
POST /api/v1/filedist/tasks
{
  "name": "Deploy Config",
  "fileId": 1,
  "targets": [
    {
      "host": "server1.example.com",
      "port": 22,
      "targetPath": "/etc/app/config.yaml",
      "username": "deploy",
      "password": "password"
    }
  ]
}
```

---

### 5. 服务实例管理

#### 功能特性
- 实例状态监控
- 健康检查
- 状态同步
- 事件追踪

#### 使用步骤

**5.1 查看实例列表**
```bash
GET /api/v1/service-instances?clusterId=1
```

**5.2 同步实例状态**
```bash
POST /api/v1/service-instances/{id}/sync
```

**5.3 重启实例**
```bash
POST /api/v1/service-instances/{id}/restart
```

---

## API 文档

### 认证

所有 API 请求需要在 Header 中携带认证 Token：

```
Authorization: Bearer <token>
```

### 响应格式

成功响应：
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

错误响应：
```json
{
  "code": 1001,
  "message": "validation_failed",
  "data": null
}
```

### 错误码

| 错误码 | 说明 |
|-------|------|
| 0 | 成功 |
| 1001 | 参数验证失败 |
| 1002 | 资源不存在 |
| 1003 | 权限不足 |
| 2001 | 数据库错误 |
| 3001 | 外部服务错误 |

### API 端点总览

完整 API 文档请访问：`http://localhost:8080/swagger/index.html`

---

## 部署指南

### Docker 部署

**1. 构建镜像**
```bash
docker build -t pantheon-ops:v1.2.0 .
```

**2. 运行容器**
```bash
docker run -d \
  --name pantheon-ops \
  -p 8080:8080 \
  -e DB_HOST=postgres \
  -e DB_PASSWORD=password \
  -e PANTHEON_ENCRYPTION_KEY=your-key \
  pantheon-ops:v1.2.0
```

### Kubernetes 部署

**1. 创建 Secret**
```bash
kubectl create secret generic pantheon-ops-secrets \
  --from-literal=db-password=password \
  --from-literal=encryption-key=your-key
```

**2. 部署应用**
```bash
kubectl apply -f deploy/k8s/deployment.yaml
```

### 高可用部署

推荐配置：
- 后端: 3 个副本
- 数据库: PostgreSQL HA 集群
- 负载均衡: Nginx/ALB

---

## 故障排查

### 常见问题

**1. 连接 ArgoCD 失败**
- 检查 Endpoint 是否可访问
- 验证 AuthToken 是否有效
- 查看防火墙规则

**2. Velero 备份失败**
- 确认 Velero 已安装在集群
- 检查存储位置配置
- 验证 S3 凭证

**3. 告警未触发**
- 确认 Prometheus 数据源连接正常
- 检查告警规则 PromQL 语法
- 验证告警评估器是否运行

**4. 文件分发失败**
- 检查 SFTP 连接
- 验证目标路径权限
- 查看传输引擎日志

### 日志查看

```bash
# 查看应用日志
tail -f logs/pantheon-ops.log

# 查看特定模块日志
grep "argocd" logs/pantheon-ops.log

# 查看错误日志
grep "ERROR" logs/pantheon-ops.log
```

### 性能优化

- 数据库连接池: 建议 50-100
- API 超时设置: 30 秒
- 定期清理历史数据
- 启用 Redis 缓存

---

## 最佳实践

1. **定期备份**: 每天至少备份一次
2. **监控告警**: 配置关键指标告警
3. **权限管理**: 最小权限原则
4. **密钥轮换**: 定期更换加密密钥
5. **日志审计**: 保留操作日志至少 90 天

---

## 支持与反馈

- 文档: https://docs.pantheon-ops.com
- 问题反馈: https://github.com/your-org/pantheon-ops/issues
- 技术支持: support@example.com

---

**版权所有 © 2026 Pantheon Team**
