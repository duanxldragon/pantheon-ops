# Database Connection Pool Tuning Guide

**Document Version**: 1.0  
**Last Updated**: 2026-09-04  
**Applies To**: pantheon-base v0.11.1+

---

## Overview

Proper database connection pool tuning prevents connection exhaustion, reduces wait times, and ensures stable performance under load. This guide provides production-tested formulas and recommendations.

---

## Production Calculation Formula

```
Total Connections = Backend Instances × Max Open Connections Per Instance
MySQL max_connections ≥ Total Connections × 1.25 (safety margin)
```

### Example Scenario

**Setup**:
- 10 backend instances
- 50 max_open_conns per instance
- Total: 500 active connections

**MySQL Configuration**:
```sql
SET GLOBAL max_connections = 625;  -- 500 × 1.25
```

---

## Recommended Environment Variables

### Per-Instance Configuration (.env)

```bash
# Connection Pool Sizing
PANTHEON_DB_MAX_OPEN_CONNS=50              # Lower than default 100 for multi-instance
PANTHEON_DB_MAX_IDLE_CONNS=10              # Keep idle connections warm
PANTHEON_DB_CONN_MAX_LIFETIME_MINUTES=30   # Rotate connections every 30 min
PANTHEON_DB_CONN_MAX_IDLE_TIME_MINUTES=5   # Close idle connections faster

# Connection String (example)
PANTHEON_DB_DSN=pantheon_user:password@tcp(mysql:3306)/pantheon_base?charset=utf8mb4&parseTime=True&loc=Local
```

### Configuration Explanation

| Variable | Default | Production Recommendation | Rationale |
|----------|---------|---------------------------|-----------|
| `MAX_OPEN_CONNS` | 100 | 50 (multi-instance) | Prevents MySQL saturation |
| `MAX_IDLE_CONNS` | 10 | 10 | Keeps warm connections available |
| `CONN_MAX_LIFETIME` | 60 min | 30 min | Forces reconnection, prevents stale connections |
| `CONN_MAX_IDLE_TIME` | 30 min | 5 min | Aggressively closes idle connections |

---

## MySQL Server Configuration

### Check Current Status

```sql
-- Connection limits
SHOW VARIABLES LIKE 'max_connections';

-- Current usage
SHOW STATUS LIKE 'Threads_connected';
SHOW STATUS LIKE 'Max_used_connections';

-- Connection waits (should be low)
SHOW STATUS LIKE 'Connection_errors_max_connections';
```

### Recommended Settings (my.cnf or RDS parameter group)

```ini
[mysqld]
# Connection limits
max_connections = 625                    # Adjust based on calculation above
max_user_connections = 0                 # No per-user limit

# Connection timeouts
wait_timeout = 600                       # 10 minutes (idle connection timeout)
interactive_timeout = 600                # 10 minutes
connect_timeout = 10                     # Connection handshake timeout

# Thread cache (reduces thread creation overhead)
thread_cache_size = 100                  # Reuse threads for new connections

# Connection backlog
back_log = 512                           # Queue for pending connections
```

### Apply Changes

```sql
-- Temporary (until restart)
SET GLOBAL max_connections = 625;
SET GLOBAL wait_timeout = 600;
SET GLOBAL interactive_timeout = 600;

-- Permanent: Edit my.cnf and restart MySQL
-- Or update RDS parameter group and reboot instance
```

---

## Monitoring and Alerting

### Prometheus Metrics

pantheon-base exports the following Prometheus metrics:

```
# Current open connections
go_sql_open_connections{db="pantheon_base"}

# Current idle connections
go_sql_idle_connections{db="pantheon_base"}

# Total connection wait events (should be low)
go_sql_wait_count_total{db="pantheon_base"}

# Wait duration in seconds
go_sql_wait_duration_seconds{db="pantheon_base"}
```

### Recommended Alerts

```yaml
# Prometheus alerting rules
groups:
  - name: database_connection_pool
    rules:
      - alert: DatabaseConnectionPoolNearExhaustion
        expr: go_sql_open_connections / 50 > 0.8
        for: 5m
        annotations:
          summary: "Database connection pool >80% utilized"
          description: "Instance {{ $labels.instance }} using {{ $value | humanizePercentage }} of connection pool"

      - alert: DatabaseConnectionWaits
        expr: rate(go_sql_wait_count_total[5m]) > 10
        for: 2m
        annotations:
          summary: "High database connection wait events"
          description: "Instance {{ $labels.instance }} experiencing {{ $value }} connection waits/sec"
```

### Grafana Dashboard Queries

```promql
# Connection pool utilization (%)
(go_sql_open_connections / 50) * 100

# Idle connection ratio (%)
(go_sql_idle_connections / go_sql_open_connections) * 100

# Connection wait rate (events/sec)
rate(go_sql_wait_count_total[5m])
```

---

## Troubleshooting

### Issue: "Error 1040: Too many connections"

**Symptom**: MySQL rejects new connections  
**Diagnosis**:
```sql
SHOW STATUS LIKE 'Threads_connected';
SHOW VARIABLES LIKE 'max_connections';
-- If Threads_connected ≈ max_connections, pool exhausted
```

**Short-term Fix**:
```sql
-- Increase limit temporarily
SET GLOBAL max_connections = 1000;

-- Kill long-running idle connections
SELECT CONCAT('KILL ', id, ';') AS kill_command
FROM information_schema.processlist
WHERE command = 'Sleep' AND time > 600
ORDER BY time DESC;
```

**Long-term Fix**:
1. Tune `PANTHEON_DB_MAX_OPEN_CONNS` per instance
2. Increase MySQL `max_connections` permanently
3. Add connection pool monitoring

---

### Issue: High connection wait events

**Symptom**: `go_sql_wait_count_total` increasing rapidly  
**Diagnosis**: Connection pool undersized for traffic

**Fix**:
```bash
# Increase max_open_conns per instance
export PANTHEON_DB_MAX_OPEN_CONNS=75  # Was 50

# Restart backend instances
```

**Validation**:
```promql
# Wait rate should drop to near-zero
rate(go_sql_wait_count_total[5m])
```

---

### Issue: Stale connection errors ("bad connection", "connection reset")

**Symptom**: Random DB errors under low load  
**Diagnosis**: Connections held past MySQL `wait_timeout`

**Fix**:
```bash
# Rotate connections more aggressively
export PANTHEON_DB_CONN_MAX_LIFETIME_MINUTES=15  # Was 30
export PANTHEON_DB_CONN_MAX_IDLE_TIME_MINUTES=3  # Was 5
```

---

## Capacity Planning

### Traffic-Based Sizing

| Daily Active Users | Concurrent Requests | Backend Instances | Max Open Conns/Instance | Total MySQL Conns | Recommended max_connections |
|--------------------|---------------------|-------------------|-------------------------|-------------------|----------------------------|
| < 1,000 | < 50 | 2 | 50 | 100 | 150 |
| 1,000 - 10,000 | 50 - 500 | 5 | 50 | 250 | 350 |
| 10,000 - 100,000 | 500 - 2,000 | 10 | 50 | 500 | 625 |
| > 100,000 | > 2,000 | 20+ | 50 | 1,000+ | 1,250+ |

### Load Testing Recommendations

Before production deployment:

1. **Baseline Test** (current settings):
   ```bash
   # Example: Apache Bench
   ab -n 10000 -c 100 http://backend:8080/api/v1/auth/current-user
   ```

2. **Monitor Metrics**:
   - `go_sql_open_connections` peak value
   - `go_sql_wait_count_total` increment rate
   - MySQL `Threads_connected` peak

3. **Adjust Settings** based on observed utilization

4. **Re-test** until connection waits drop to acceptable level

---

## References

- GORM Connection Pool: https://gorm.io/docs/generic_interface.html#Connection-Pool
- MySQL Connection Limits: https://dev.mysql.com/doc/refman/8.0/en/too-many-connections.html
- Prometheus MySQL Exporter: https://github.com/prometheus/mysqld_exporter

---

**Author**: duanxiaolong <435000465@qq.com>  
**Feedback**: Report tuning issues via GitHub issues
