# database/

## 权威建库路径：golang-migrate（单一来源）

Pantheon Base 的数据库 schema **唯一权威来源**是版本化迁移：

```
backend/pkg/database/migrations/*.sql   ← 权威 schema（29 表）
```

应用启动时默认执行 `migrate up` 建表，并由运行时 seed 填充默认数据：

- `backend/modules/system/seed.go` — 菜单、admin 角色、admin 权限
- `backend/modules/system/i18n/i18n_seed.go` — i18n 词条
- Casbin admin 默认策略、首个 admin 用户 — 应用启动时确保

### 本地起库（docker-compose）

```
docker-compose up -d mysql redis
```

`docker-compose.yml` 的 `MYSQL_DATABASE: pantheon_base` 会创建**空库**，应用启动时自动迁移 + seed。**不再**通过 `docker-entrypoint-initdb.d` 挂载任何 DDL 脚本。

## 目录内容

| 文件                     | 用途                                                                                        |
| ------------------------ | ------------------------------------------------------------------------------------------- |
| `system_init.sql`        | ⚠️ **DEPRECATED**——旧 16 表 DDL，已落后于迁移（缺 13 张表），仅作历史参考，**切勿用于建库** |
| `diagnose_menu_data.sql` | 菜单数据诊断查询（只读排查用）                                                              |

## 为什么不再用 system_init.sql 建库

`system_init.sql` 与权威迁移分歧（16 表 vs 29 表），误用会得到残缺过时 schema。Schema 单源化后，所有建库路径统一到 golang-migrate，消除新人/脚本踩坑。详见 `docs/designs/DATABASE.md §5`。
