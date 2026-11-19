# O4 迁移恢复回归 — Root Cause & 处置记录

**taskId**: `2026-11-17-ops-o4-recovery-regression`
**status**: awaiting-review（不自行批准）

## Root cause（Codex 窄范围定位，已复现核验代码路径）

`go test -race` 的 `TestO4_KnownPartialRecoveryAndUnknownRefusal` 三个 subtest
（`revision`、`index idx_datasource_tenant_default`、`obs_saved_view`）在
migration_recovery_test.go 中取实时 DB 名：

```go
var name string
db.Raw("SELECT DATABASE()").Scan(&name)
ownership := O4RecoveryOwnership{Database: name, ...}
if _, err := RecoverO4Migration37(ctx, db, ownership); err != nil { t.Fatal(err) }
```

而 production `RecoverO4Migration37`（backend/modules/business/observability/
migration_recovery.go:148）在入口强制：

```go
!disposableO4Database.MatchString(ownership.Database)  // → ErrQuerySchema
```

`disposableO4Database = ^ops_p0_acceptance_testo4_[a-z0-9_]+_[0-9]{19}_[0-9]{4}$`

`testmysql.Open` 用 DSN 的 `DBName` 作前缀种子经 `buildTestDBName` 生成隔离库；
当前 generic 开发/CI DSN 的 DBName 非 `ops_p0_acceptance`，因此生成名不定向命中
白名单 → 入口返回 `observability.schema_unavailable / ErrQuerySchema` → 三 subtest FAIL。
**非慢 DB 导致**（独立重跑 ~2s 仍 FAIL）。

其余两个 O4 测试（`LegacyCiphertextAndIndexUpgrade`、`FreshAndRetainedMigrationRecipe`）
不调用 `RecoverO4Migration37`，不依赖白名单，未改动。

## 修复（最小 diff，仅 migration_recovery_test.go）

新增 `rebindO4DisposableDSN(t)`：

- 仅当 `PANTHEON_TEST_DSN`/`PANTHEON_DSN` 已配置时，用 `mysql.ParseDSN` 解析，
  只把 `cfg.DBName` 替换为 `ops_p0_acceptance`，`t.Setenv("PANTHEON_TEST_DSN", cfg.FormatDSN())`
  隔离测试范围；未配置时返回，交由公共 `testmysql` required/skip 逻辑。
- 三个 subtest 在 `testmysql.Open` 前调用；并显式断言
  `disposableO4Database.MatchString(name)`，确保 fixture 名命中生产白名单。

不改 production 白名单、不向业务库写 fixture、不打印 DSN、不读用户凭证。
60 字节名截断复核：最长 subtest 截断后 `ops_p0_acceptance_testo4_knownpartia_{19}_{4}`
仍命中正则（前缀 25 字 + ≥1 个 `[a-z0-9_]+` 保留）。

## 运行证据

本会话为无 approval surface 的 headless 环境，`go vet` / `go test -race` / 真实 MySQL
执行命令均被环境自动拒绝，**未能产出运行 log**；已记录且未将 not-run 写为 passed。
交由独立 Codex（`check-go.ps1` 可用）复测上述 verification commands 并附原始输出。

## 未验项（交 Codex）

1. `go vet ./modules/business/observability/` 与 `gofmt -l`（编译/格式）。
2. 真实 MySQL `go test -race -run TestO4_KnownPartialRecoveryAndUnknownRefusal` 三 subtest 全绿。
3. known partial 三态恢复、unknown extra 列拒绝并保留 dirty、旧数据保全覆盖仍真实通过。
4. 未配置 DSN 时按公共逻辑 skip / `PANTHEON_TEST_DSN_REQUIRED` 时 fail 的分支行为。

若复测仍失败，附真实证据定位，不盲目改期待/skip。

## Codex independent acceptance — 2026-11-17

已独立验收本次夹具回归：三个 O4 迁移测试及三个 known-partial 恢复子场景在真实隔离 MySQL 下通过 CGO race，无 skip。生产恢复代码未修改；原失败日志保留。证据：codex-o4-migrations-race.log、commands.json、review.md。全仓 race 其他包超时和完整 O4 产品验收仍开放。
