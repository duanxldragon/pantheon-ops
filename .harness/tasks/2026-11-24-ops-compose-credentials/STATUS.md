# Compose 凭据外置与废弃 DDL 挂载移除

- Status: completed
- Updated: 2026-11-24
- Baseline: 11f9b004（候选 HEAD）+ 未提交工作树；原始实现基线 697c0771，697c0771..11f9b004 未触及 docker-compose.yml
- Owner: 部署配置维护者
- Priority: P1
- Activation: core
- Dependencies: none

## Implemented

- `MYSQL_ROOT_PASSWORD` 由内联固定口令改为 `${MYSQL_ROOT_PASSWORD:?...}`，缺失时报错退出并给出复制 `.env.example` 的指引。
- Redis `--requirepass` 由内联固定口令改为 `${REDIS_PASSWORD:?...}`，同样必填。
- 移除 `./database/system_init.sql:/docker-entrypoint-initdb.d/01-system_init.sql:ro` 挂载；建库只走应用启动的 migrate + runtime seed，与 `database/README.md`、`docs/designs/REPOSITORY_LAYOUT.md` 的既有结论一致。
- 本地 `.env`（已 gitignore）补 `MYSQL_ROOT_PASSWORD` / `REDIS_PASSWORD`，沿用既有数据卷口令，保证本机 compose 不被本次变更打断。

## Verification

- 缺变量：`docker compose config` exit 1，两条明确错误（mysql 与 redis 各一条），无隐式默认口令。
- 齐备后：`docker compose config` exit 0；解析结果 `MYSQL_ROOT_PASSWORD` 已注入、`--requirepass` 值非空（长度 12）。
- 解析结果不含 `docker-entrypoint-initdb` 与 `system_init`。
- `grep -c "DHCCroot\|DHCCdhcc" docker-compose.yml` → 0。
- 仓库内其他位置未检索到同一口令（CI、脚本均未引用 compose）。
- 11f9b004 复核：`docker compose --env-file .tmp/empty.env config` 缺变量仍 exit 1（两条必填错误），齐备 exit 0，解析结果仍无 initdb/system_init 且无旧口令明文。
- 证据：`../evidence/2026-11-24-ops-compose-credentials/`。

## Known Gaps

1. 容器实际启动、首建库与既有数据卷兼容未验证（本机 Docker daemon 未运行，`config` 为离线解析）。
2. Compose 是否作为客户安装入口未确认；若确认，应提升为 P0 并补生产密钥生成/托管/轮换要求（报告 O-06 同源）。
3. `.env.example`（Base 跟踪）已定义两个变量但未标注「compose 必填」，文案改进需回 Base。

## Next Atomic Action

无（本包闭合）。如需运行态验证，在可用 Docker daemon 环境执行 `docker compose up -d mysql redis` 并核对空库首建。
