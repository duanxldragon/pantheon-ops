# Compose 凭据外置与废弃 DDL 挂载移除 — 实施摘要

- Task: `2026-11-24-ops-compose-credentials`
- 日期: 2026-11-24
- 状态: **凭据外置与废弃 DDL 挂载已移除，配置解析级验证通过**
- 交付层级: L2（自审见 `review.md`，无独立评审）

## 来源与判定

来源为 2026-11-24 外部综合审查报告 §4.2（O-01）。核实结果：

| 报告结论                        | 核实结果                                                                                                                           |
| ------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| Compose 硬编码 MySQL/Redis 凭据 | 成立。原第 7 行 `MYSQL_ROOT_PASSWORD` 内联固定值，第 24 行 `--requirepass` 内联固定值（值不在本证据中复述）                        |
| 挂载废弃 `system_init.sql`      | 成立。原第 18 行挂载到 `/docker-entrypoint-initdb.d/01-system_init.sql`                                                            |
| 与数据库说明互相矛盾            | 成立。`database/README.md` 与 `docs/designs/REPOSITORY_LAYOUT.md` 均已声明 Compose 不再挂载任何 DDL、建库走 migrate + runtime seed |

`.env.example` 本就定义了 `MYSQL_ROOT_PASSWORD` 与 `REDIS_PASSWORD`，说明插值化是原设计意图，只是 compose 文件没跟上。

## 交付内容

| 验收项       | 实现                                                                                        |
| ------------ | ------------------------------------------------------------------------------------------- |
| 凭据不再入库 | 两处均改为 `${VAR:?<提示>}`，缺失即失败；YAML 内无任何口令字面量                            |
| 失败语义明确 | 缺变量时 `docker compose config` exit 1，分别提示 mysql/redis 变量名与「复制 .env.example」 |
| 建库单源     | 移除 initdb 挂载；YAML 注记「建库只走 migrate + runtime seed」                              |
| 本机不被打断 | 本地 `.env`（gitignore）补两个键，沿用既有数据卷口令                                        |
| diff 受控    | 端口、镜像、字符集、数据卷名、`MYSQL_DATABASE` 均未改                                       |

## 验证

- 缺变量 → `docker compose config` exit 1，两条必填错误（预期失败，作为无隐式默认口号的证据）。
- 齐备 → exit 0；解析结果 `MYSQL_ROOT_PASSWORD` 已注入、`--requirepass` 非空（长度 12，仅记录长度）。
- 解析结果不含 `docker-entrypoint-initdb`、`system_init`。
- `grep -c "DHCCroot\|DHCCdhcc" docker-compose.yml` → 0。
- CI 与脚本均未引用 compose，仓库内无其他引用该口令的位置。

## 显式 gap

1. 未做 `docker compose up` 运行态验证（本机 Docker daemon 未运行），首建库与既有数据卷兼容未实测。
2. Compose 是否为客户安装入口未确认；若确认应升 P0，并补生产密钥生成/托管/轮换（与 O-06 同源）。
3. `.env.example` 属 Base 跟踪，未标注「compose 必填」，文案改进需回 Base。
