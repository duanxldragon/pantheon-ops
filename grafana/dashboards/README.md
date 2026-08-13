# Pantheon Base Grafana 监控仪表板配置

本目录包含 Pantheon Base 的 Grafana 仪表板配置文件。

## 仪表板列表

1. **pantheon-api-overview.json** - API 监控概览
   - HTTP 请求 QPS
   - 响应时间分布 (P50/P95/P99)
   - 错误率
   - 请求状态码分布

2. **pantheon-database-performance.json** - 数据库性能监控
   - 数据库连接池状态
   - 慢查询监控
   - 连接数趋势

3. **pantheon-business-metrics.json** - 业务指标监控
   - 登录成功/失败率
   - 活跃会话数
   - 用户活跃度

4. **pantheon-infrastructure.json** - 基础设施监控
   - Redis 连接状态
   - 系统资源使用

## 导入方式

### 方式 1: Grafana UI 导入

1. 登录 Grafana (默认: http://localhost:3000)
2. 点击左侧菜单 "+" → "Import"
3. 上传对应的 JSON 文件或粘贴内容
4. 选择 Prometheus 数据源
5. 点击 "Import"

### 方式 2: 通过 API 导入

```bash
# 导入所有仪表板
for dashboard in grafana/dashboards/*.json; do
  curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
    -H "Content-Type: application/json" \
    -d @"$dashboard"
done
```

### 方式 3: Provisioning (推荐)

将此目录配置到 Grafana 的 provisioning 目录：

```yaml
# grafana/provisioning/dashboards/pantheon.yaml
apiVersion: 1

providers:
  - name: 'Pantheon Base'
    orgId: 1
    folder: 'Pantheon'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 30
    options:
      path: /etc/grafana/dashboards/pantheon
```

## 数据源配置

确保已配置 Prometheus 数据源：

```yaml
# grafana/provisioning/datasources/prometheus.yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
```

## 告警规则

部分仪表板包含预配置的告警规则：

- API 错误率 > 5%
- P95 响应时间 > 500ms
- 数据库连接池使用率 > 80%
- 登录失败率 > 10%

## 使用说明

1. **API 监控**：实时查看 API 性能和错误
2. **数据库监控**：诊断数据库性能问题
3. **业务指标**：了解业务健康度
4. **基础设施**：监控系统资源

## 自定义

所有仪表板都支持自定义：
- 调整时间范围
- 修改刷新间隔
- 添加/删除面板
- 调整告警阈值

## 维护

- 仪表板版本：1.0
- 最后更新：2026-06-22
- 兼容 Grafana: 10.x+
- 兼容 Prometheus: 2.x+
