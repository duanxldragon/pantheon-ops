# Pantheon Base 性能测试套件

本目录包含基于 k6 的性能测试脚本。

## 测试类型

1. **load-test.js** - 负载测试
   - 模拟正常负载下的系统表现
   - 测试场景：登录、获取用户信息、菜单加载
   - 目标：95% 请求 < 500ms

2. **stress-test.js** - 压力测试
   - 逐步增加负载直到系统达到极限
   - 目标：找到系统瓶颈

3. **spike-test.js** - 峰值测试
   - 短时间内产生大量请求
   - 测试系统应对突发流量的能力

## 安装 k6

### macOS
```bash
brew install k6
```

### Linux
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

### Windows
```powershell
choco install k6
```

## 运行测试

### 基础运行

```bash
# 负载测试
k6 run backend/tests/performance/load-test.js

# 压力测试
k6 run backend/tests/performance/stress-test.js

# 峰值测试
k6 run backend/tests/performance/spike-test.js
```

### 自定义配置

```bash
# 指定目标服务器
k6 run --env BASE_URL=http://your-server:8080 backend/tests/performance/load-test.js

# 指定虚拟用户数
k6 run --vus 50 --duration 30s backend/tests/performance/load-test.js

# 输出结果到文件
k6 run --out json=results.json backend/tests/performance/load-test.js

# 输出到 InfluxDB
k6 run --out influxdb=http://localhost:8086/k6 backend/tests/performance/load-test.js
```

## 性能基线

基于测试建立的性能基线：

| 指标 | 目标值 | 说明 |
|------|--------|------|
| P50 响应时间 | < 100ms | 50% 的请求应在 100ms 内完成 |
| P95 响应时间 | < 500ms | 95% 的请求应在 500ms 内完成 |
| P99 响应时间 | < 1000ms | 99% 的请求应在 1s 内完成 |
| 错误率 | < 5% | 错误率应低于 5% |
| 并发用户 | 500+ | 支持 500 并发用户 |
| QPS | 1000+ | 每秒处理 1000+ 请求 |

## 结果分析

测试完成后，k6 会输出摘要：

```
     ✓ health check status is 200
     ✓ login status is 200 or 401
     
     checks.........................: 95.00% ✓ 19000 ✗ 1000
     data_received..................: 15 MB  250 kB/s
     data_sent......................: 2.5 MB 42 kB/s
     http_req_duration..............: avg=250ms min=10ms med=200ms max=2s p(95)=450ms p(99)=800ms
     http_req_failed................: 5.00%  ✓ 1000  ✗ 19000
     http_reqs......................: 20000  333.33/s
     vus............................: 100    min=0 max=500
     vus_max........................: 500    min=500 max=500
```

### 关键指标说明

- **checks**: 断言通过率
- **http_req_duration**: 请求响应时间分布
- **http_req_failed**: 请求失败率
- **http_reqs**: 总请求数和 QPS
- **vus**: 虚拟用户数

## CI 集成

在 GitHub Actions 中运行性能测试：

```yaml
# .github/workflows/performance.yml
name: Performance Tests

on:
  push:
    branches: [main]
  schedule:
    - cron: '0 2 * * *'  # 每天凌晨 2 点

jobs:
  performance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install k6
        run: |
          sudo gpg -k
          sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6
      
      - name: Run load test
        run: k6 run backend/tests/performance/load-test.js
      
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: performance-results
          path: results.json
```

## 监控集成

k6 可以将结果发送到多个监控系统：

### Grafana Cloud
```bash
k6 run --out cloud backend/tests/performance/load-test.js
```

### InfluxDB + Grafana
```bash
k6 run --out influxdb=http://localhost:8086/k6 backend/tests/performance/load-test.js
```

### Prometheus
使用 k6 Prometheus Remote Write exporter。

## 故障排查

### 常见问题

**1. 连接被拒绝**
```
WARN[0001] Request Failed error="Get \"http://localhost:8080\": dial tcp [::1]:8080: connect: connection refused"
```
- 检查服务是否启动
- 检查 BASE_URL 是否正确

**2. 请求超时**
```
WARN[0010] Request Failed error="context deadline exceeded"
```
- 增加超时时间
- 检查服务器性能

**3. 内存不足**
```
fatal error: out of memory
```
- 减少虚拟用户数
- 增加测试机器内存

## 最佳实践

1. **逐步增加负载** - 不要一次性施加最大负载
2. **监控服务器指标** - 同时观察 CPU、内存、网络使用
3. **多次运行** - 至少运行 3 次取平均值
4. **隔离测试环境** - 不要在生产环境测试
5. **记录基线** - 每次测试后记录结果，追踪性能变化

---

**最后更新**: 2026-06-22  
**维护者**: QA Team
