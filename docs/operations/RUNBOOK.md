# Operations Runbook

**Document Version**: 1.0  
**Last Updated**: 2026-09-04  
**Applies To**: pantheon-base v0.11.1+

---

## Table of Contents

1. [Redis Failure Scenarios](#redis-failure-scenarios)
2. [Database Connection Saturation](#database-connection-saturation)
3. [Operation Log Queue Saturation](#operation-log-queue-saturation)
4. [Authentication Failures](#authentication-failures)
5. [Performance Degradation](#performance-degradation)

---

## Redis Failure Scenarios

### Symptom: All authenticated requests return 401 Unauthorized

**Cause**: Redis connection lost, token sessions unavailable  
**Detection**: Check backend logs for "failed to connect redis" error  
**Impact**: Complete authentication system failure

**Immediate Actions**:

```bash
# 1. Check Redis health
redis-cli ping
# Expected: PONG

# 2. Verify network connectivity
telnet redis-host 6379

# 3. Check Redis logs
docker logs <redis-container>
# or
journalctl -u redis

# 4. Restart Redis if needed
docker restart <redis-container>
# or
systemctl restart redis
```

**Recovery**:
- Backend auto-reconnects on next request
- Existing sessions lost (users must re-login)
- No data loss (sessions stored in Redis only)

**Prevention**:
- Enable Redis persistence (AOF or RDB)
- Deploy Redis Sentinel for high availability
- Monitor `redis_connected` metric
- Set up alerting for Redis downtime

---

### Symptom: Rate limiting doesn't work across instances

**Cause**: Redis disconnected, fallback to in-memory (per-instance) limits  
**Detection**: `rate_limiter_backend=memory` metric  
**Impact**: Attackers can bypass limits by hitting different instances

**Actions**:
1. Restore Redis connectivity (see above)
2. Rate limiter auto-switches back to Redis
3. Review rate limit breach logs during outage

**Long-term Fix**:
- Ensure Redis high availability
- Monitor `rate_limiter_backend` metric
- Alert on failover to memory backend

---

## Database Connection Saturation

### Symptom: "Error 1040: Too many connections"

**Cause**: Total connections > MySQL max_connections  
**Detection**: Check MySQL connection count

**Diagnosis**:

```sql
-- Check current usage
SHOW VARIABLES LIKE 'max_connections';
SHOW STATUS LIKE 'Threads_connected';
SHOW STATUS LIKE 'Max_used_connections';

-- Calculate headroom
-- Headroom = max_connections - Threads_connected
-- Alert if headroom < 20%
```

**Immediate Actions**:

```sql
-- 1. Increase limit temporarily (until restart)
SET GLOBAL max_connections = 1000;

-- 2. Find long-running queries
SHOW PROCESSLIST;

-- 3. Kill stuck queries if needed
KILL <thread_id>;

-- 4. Identify source of connection leak
SELECT user, host, COUNT(*) as connection_count
FROM information_schema.processlist
GROUP BY user, host
ORDER BY connection_count DESC;
```

**Long-term Fix**:

1. **Tune backend connection pool** (see `DATABASE_TUNING.md`):
   ```bash
   export PANTHEON_DB_MAX_OPEN_CONNS=50  # Reduce per-instance
   ```

2. **Increase MySQL max_connections** permanently:
   ```ini
   # /etc/mysql/my.cnf
   [mysqld]
   max_connections = 625
   ```

3. **Add monitoring**:
   ```promql
   # Alert when > 80% utilized
   (mysql_global_status_threads_connected / mysql_global_variables_max_connections) > 0.8
   ```

---

## Operation Log Queue Saturation

### Symptom: Audit logs missing under high load

**Cause**: Operation log queue full (10,000 capacity), drops overflow  
**Detection**: `operation_log_dropped_total` Prometheus metric increasing  
**Impact**: Audit trail gaps (acceptable per design)

**Background**:
This is **expected behavior** under sustained overload, not a bug. The system prioritizes availability over perfect audit completeness.

**Actions**:

1. **Verify it's expected**:
   ```promql
   # Check queue size
   operation_log_queue_size
   
   # Check drop rate
   rate(operation_log_dropped_total[5m])
   ```

2. **If chronic** (drops every hour):
   - Option A: Increase queue size in code (requires rebuild):
     ```go
     // backend/internal/middleware/operation_log_middleware.go
     const operationLogBufferSize = 20000  // Was 10000
     ```
   - Option B: Offload to Redis Streams for durability (future enhancement)

3. **If transient** (during peak load):
   - No action required
   - Log gaps are documented trade-off
   - Focus on preventing the underlying load spike

**Mitigation**:
- Scale backend horizontally (more instances = more queue capacity)
- Optimize slow endpoints causing load spikes
- Add rate limiting on expensive operations

---

## Authentication Failures

### Symptom: Users cannot login, "invalid credentials" errors

**Diagnosis Checklist**:

```bash
# 1. Check backend logs
tail -f /var/log/pantheon-base/backend.log | grep -i "login\|auth"

# 2. Verify database connectivity
mysql -u pantheon_user -p -e "SELECT 1 FROM system_user LIMIT 1;"

# 3. Check Redis connectivity
redis-cli GET "nonexistent_key"  # Should return (nil), not connection error

# 4. Verify JWT secrets configured
echo $PANTHEON_ACCESS_TOKEN_SECRET | wc -c  # Should be >= 32
```

**Common Causes**:

| Error Message | Cause | Fix |
|---------------|-------|-----|
| "invalid credentials" | Wrong password or user not found | Check `system_user` table |
| "token expired" | Clock skew or JWT expiration too short | Sync NTP, check `TOKEN_EXPIRES_AT` |
| "unauthorized" | Redis down (session lost) | Restart Redis |
| "forbidden" | Casbin policy mismatch | Check `casbin_rule` table |

**Emergency Access** (if admin locked out):

```sql
-- Reset admin password (cost 12 takes ~300ms)
UPDATE system_user
SET password_hash = '$2a$12$...'  -- Generate via bcrypt tool
WHERE username = 'admin';
```

---

## Performance Degradation

### Symptom: Slow API responses (> 500ms P95)

**Diagnosis**:

```bash
# 1. Check database slow query log
mysql -e "SHOW VARIABLES LIKE 'slow_query%';"
tail -f /var/log/mysql/slow-query.log

# 2. Check backend metrics
curl http://localhost:9090/metrics | grep -E "http_request_duration|db_query_duration"

# 3. Check connection pool utilization
curl http://localhost:9090/metrics | grep go_sql_open_connections
```

**Common Bottlenecks**:

1. **Missing database indexes** → Run migrations, verify indexes exist
2. **Connection pool exhausted** → Increase `MAX_OPEN_CONNS` or scale horizontally
3. **N+1 queries** → Permission cache should mitigate (check Redis hit rate)
4. **Slow external service** → Check OpenTelemetry traces for dependencies

**Quick Wins**:

```bash
# 1. Enable query cache (if not already)
mysql -e "SET GLOBAL query_cache_type = 1;"

# 2. Increase connection pool
export PANTHEON_DB_MAX_OPEN_CONNS=75

# 3. Verify permission cache enabled
redis-cli KEYS "user_perms:*" | wc -l  # Should be > 0 under load

# 4. Restart backend to clear any goroutine leaks
systemctl restart pantheon-backend
```

---

## Health Checks

### Liveness Probe

**Endpoint**: `GET /health/live`  
**Expected**: `200 OK`  
**Failure**: Container/process should be restarted

**What it checks**:
- HTTP server responsive
- Basic memory/goroutine sanity

**Does NOT check**:
- Database connectivity
- Redis connectivity

### Readiness Probe

**Endpoint**: `GET /health/ready`  
**Expected**: `200 OK` when ready to serve traffic  
**Expected**: `503 Service Unavailable` during startup or shutdown

**What it checks** (v0.12.0, task 2026-09-22-production-redis-and-security-gates):
- `GET /api/v1/health` is the live implementation endpoint; it reports per-dependency status for `database`, `migrations`, and `redis` and returns 503 when any checked dependency is down
- Database connectivity (`database`)
- Migrations applied (`migrations`: `schema_migrations` present and readable; 503 while schema bootstrap is incomplete)
- Redis available (`redis`: `disabled` status means no address configured — non-production compat; in production the process fails fast at startup instead)

### Production startup fail-fast (Redis)

Redis stores token sessions and the revocation blacklist; it is a hard
production dependency. Startup exits non-zero when:

- `PANTHEON_ENV=production` and `PANTHEON_REDIS_ADDR` is missing, or
- `PANTHEON_ENV=production` (or `PANTHEON_REDIS_REQUIRED=true`) and the
  Redis connection/ping fails at init.

Non-production deployments keep the legacy behavior (degrade to
`RDB = nil`, auth endpoints fail per-request) unless
`PANTHEON_REDIS_REQUIRED=true` is set explicitly.

Reproduce the security-scan gates locally:

```bash
# Go vulnerability scan (same version as CI)
cd backend && go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...

# npm audit (repo root and frontend)
npm audit --registry=https://registry.npmjs.org --audit-level=high
cd frontend && npm audit --registry=https://registry.npmjs.org --audit-level=high

# Race detector (CI: quality.yml go-race job). Needs a MinGW-w64 gcc:
# Go refuses the Cygwin gcc with "don't use the cygwin compiler to build
# native Windows programs; use MinGW instead". On Windows, point CC at a
# MinGW-w64 toolchain (e.g. a portable winlibs/niXman build extracted under
# a gitignored path) and enable CGO:
#
#   export CC='<toolchain>\bin\gcc.exe' CGO_ENABLED=1
#   export PATH='<toolchain>/bin':$PATH
#
# Linux/macOS needs nothing extra.
cd backend && go test -race -count=1 ./...

# MySQL/Redis backed unit tests (mirrors ci.yml unit-tests services)
PANTHEON_TEST_DSN='root:<pw>@tcp(127.0.0.1:3306)/pantheon_test?charset=utf8mb4&parseTime=true&loc=Local' \
PANTHEON_TEST_REDIS_ADDR=127.0.0.1:6379 \
go test -short ./...
```

### Redis-backed tests

`pkg/testredis.Open` accepts three address variables, in priority order:
`PANTHEON_TEST_REDIS_ADDR`, then `PANTHEON_REDIS_ADDR` (the runtime variable the
server itself reads), then `REDIS_ADDR`; the password resolves the same way over
`PANTHEON_TEST_REDIS_PASSWORD` / `PANTHEON_REDIS_PASSWORD` / `REDIS_PASSWORD`.

When no address resolves, `Open` **skips** by default, so a bare checkout can
still run `go test ./...`. That default is how a green run can mean "nothing ran":
setting only a mistyped or renamed variable produces a skip, not a failure. Every
job that provisions Redis therefore also sets `PANTHEON_TEST_REDIS_REQUIRED=true`,
which turns the missing-address case into a hard failure naming the variables it
looked for. Keep that pairing when adding a new Redis-backed job:

```yaml
PANTHEON_TEST_REDIS_ADDR: 127.0.0.1:6379
PANTHEON_TEST_REDIS_REQUIRED: "true"
```

A resolved address that refuses the connection always fails (never skips), and
`pkg/testredis/redis_test.go` pins both the accepted variable names and the
fail/skip decision, so dropping a name or the required-flag check breaks a test.
```

## Escalation Matrix

| Issue Severity | Response Time | Escalation Path |
|----------------|---------------|-----------------|
| P0 (Production down) | < 15 minutes | On-call engineer → Tech lead |
| P1 (Degraded service) | < 1 hour | On-call engineer → Resolve async |
| P2 (Minor issue) | < 4 hours | Create ticket → Next sprint |

**On-Call Contact**: [Configure per deployment]  
**Incident Slack Channel**: [Configure per deployment]

---

## References

- Database Tuning: `docs/operations/DATABASE_TUNING.md`
- Architecture: `DESIGN.md`
- Metrics: Prometheus at `:9090/metrics`
- Logs: Check deployment-specific log paths

---

**Author**: duanxiaolong <435000465@qq.com>  
**Feedback**: Report operational issues via GitHub issues
