# Review — Compose 凭据外置与废弃 DDL 挂载移除

## 结论

`approved with documented P2 follow-up`。O-01 的两处事实（明文凭据、废弃 DDL 挂载）均已在 Ops 自有 overlay 文件内闭合，失败语义与解析结果都有可复现证据；运行态容器验证与「是否客户安装入口」的判断留作显式 gap。

## Findings

1. **P1 fixed — Compose 内联固定口令。**
   `docker-compose.yml` 曾把 MySQL root 与 Redis 口令写死在文件中（值不在本证据复述）。该文件在 `business-overlay.json` 的 `repositoryOverlayPaths` 内，属 Ops 自有资产，因此本地修复不会被 Base 重建覆盖。
   改为 `${MYSQL_ROOT_PASSWORD:?...}` / `${REDIS_PASSWORD:?...}`：未配置时 compose 立即失败并给出操作指引，不再存在「忘记配置就使用弱口令」的路径。

2. **P1 fixed — 挂载废弃 DDL 建库。**
   原挂载会在空数据目录首启时执行旧 16 表 DDL，建出落后于迁移的 schema，与 `database/README.md`、`docs/designs/REPOSITORY_LAYOUT.md` 的「建库单源」结论直接冲突。已移除挂载并在 YAML 注记建库路径。

3. **P2 fixed — 本地开发连续性。**
   若只改 compose 而不补 `.env`，本机 `docker compose up` 会因必填校验失败。已在 gitignore 的 `.env` 补两个键，沿用既有数据卷口令，避免破坏已初始化卷的登录。

4. **P2 accepted-gap — 无运行态验证。**
   本机 Docker daemon 未运行，只做了 `docker compose config` 离线解析验证。容器启动、首建库、既有卷兼容仍未实测，不能据此宣称「compose 可用」。

5. **P2 accepted-gap — 安装入口定性未定。**
   报告指出若 Compose 属客户安装入口则应为 P0。本包不改变这一判断，仅按 P1 处理；生产密钥生成/托管/轮换仍缺（与 O-06 同源）。

6. **P2 accepted-gap — Base 侧文案。**
   `.env.example` 定义了这两个变量但未标注「compose 必填」，属 Base 跟踪文件，需在 Base 侧补文案后随 foundation release 进入 Ops。

## 边界确认

- 未触碰 `backend/modules/system/**`、`.env.example`、`database/README.md`、`docs/designs/REPOSITORY_LAYOUT.md`、`foundation-release.lock.json`。
- 口令值未写入任何 evidence 文件，只记录长度与解析成功状态。

## Machine Readable

```json
{
  "taskId": "2026-11-24-ops-compose-credentials",
  "reviewer": "primary agent self-review (no independent review on this candidate)",
  "verdict": "approved with documented P2 follow-up",
  "decision": "approved with documented P2 follow-up",
  "scope": "docker-compose credentials interpolation, removal of the deprecated system_init.sql initdb mount",
  "taskManifest": ".harness/tasks/2026-11-24-ops-compose-credentials/manifest.json",
  "evidenceDir": ".harness/evidence/2026-11-24-ops-compose-credentials",
  "summaryFile": ".harness/evidence/2026-11-24-ops-compose-credentials/summary.md",
  "reviewFile": ".harness/evidence/2026-11-24-ops-compose-credentials/review.md",
  "changeRef": "none",
  "findings": [
    {
      "id": "OCC-R1",
      "severity": "P1",
      "status": "fixed",
      "summary": "plaintext MySQL/Redis credentials inlined in docker-compose.yml",
      "verification": "docker compose config exits 1 with required-variable errors when keys are absent; grep count 0 in the file"
    },
    {
      "id": "OCC-R2",
      "severity": "P1",
      "status": "fixed",
      "summary": "compose mounted the deprecated system_init.sql into docker-entrypoint-initdb.d",
      "verification": "resolved config contains no docker-entrypoint-initdb or system_init"
    },
    {
      "id": "OCC-R3",
      "severity": "P2",
      "status": "fixed",
      "summary": "required interpolation broke local runs without .env keys",
      "verification": "local gitignored .env carries both keys; docker compose config exit 0"
    },
    {
      "id": "OCC-R4",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "no container start / first-init / existing-volume verification (daemon not running)"
    },
    {
      "id": "OCC-R5",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "compose-as-customer-install-entry undetermined; production key custody and rotation still missing"
    },
    {
      "id": "OCC-R6",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": ".env.example is base-tracked and does not mark the two variables as compose-required"
    }
  ],
  "verification": {
    "config_fail_fast": "docker compose config -> exit 1, two required-variable errors",
    "config_ok": "docker compose config -> exit 0; MYSQL_ROOT_PASSWORD injected; redis --requirepass non-empty",
    "no_legacy_mount": "resolved config has no docker-entrypoint-initdb / system_init",
    "no_plaintext": "grep -c 'DHCCroot|DHCCdhcc' docker-compose.yml -> 0"
  },
  "linkage": {
    "taskManifest": ".harness/tasks/2026-11-24-ops-compose-credentials/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-11-24-ops-compose-credentials.task.md",
    "evidence": ".harness/evidence/2026-11-24-ops-compose-credentials/commands.json",
    "reviewFile": ".harness/evidence/2026-11-24-ops-compose-credentials/review.md",
    "changeRef": "none",
    "planRefs": []
  },
  "humanGate": "compose-as-install-entry and production credential custody need maintainer confirmation"
}
```
