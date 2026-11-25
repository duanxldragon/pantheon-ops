---
title: Compose 凭据外置与废弃 DDL 挂载移除
doc_type: Remediation
layer: business
status: Archived
updated_at: 2026-11-24
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
---

# Task Packet: 2026-11-24-ops-compose-credentials

Compose 凭据外置与废弃 DDL 挂载移除

## Goal

把 `docker-compose.yml` 中的 MySQL/Redis 明文口令改为仅从环境读取并在缺失时快速失败，同时移除已废弃的 `database/system_init.sql` 初始化挂载，使 Compose 与权威迁移建库路径一致

## Primary Layer

business/*

## Dependency Layers

- business/*

## Dependencies

- blockedBy: none
- Conditional Dependencies: none
- Activation: core

## Harness Profile

- Template: custom
- Overlay: pantheon-ops
- Quality Profile: auth-security
- Delivery Tier: L2
- Priority: P1
- Coverage Dimensions:
  - behaviour
  - architecture-fitness

## Workspace Context

- Target Repository: pantheon-ops
- Repository Role: business-consumer
- Upstream Dependencies: pantheon-base
- Downstream Consumers: none
- Sync Expectation: none
- Release Requirement: none
- Downstream Sync Note: `docker-compose.yml` 属 Ops repository overlay，源文件权威在本仓；`.env.example`、`database/README.md`、`docs/designs/REPOSITORY_LAYOUT.md` 属 Base 跟踪文件，本轮不改，其内容已与本改动一致。

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `docs/TASK_PACKET_OPS_TEMPLATE.md`
- `database/README.md`（Base 跟踪；声明 Compose 不再挂载任何 DDL）
- `docs/designs/REPOSITORY_LAYOUT.md`（Base 跟踪；第 57 行同结论）
- `business-overlay.json`

## Source Findings

- O-01（2026-11-24 外部综合审查报告 §4.2：Compose 硬编码凭据并挂载废弃 `system_init.sql`）

来源判定：可由文件内容直接核实；报告已指出 Compose 与 `database/README.md` 互相矛盾。

## Scope

### In

- MySQL `MYSQL_ROOT_PASSWORD` 与 Redis `--requirepass` 改为 `${VAR:?...}` 必填插值，缺失时 compose 直接失败
- 移除 `./database/system_init.sql:/docker-entrypoint-initdb.d/...` 挂载
- 在 YAML 注记建库单源（migrate + runtime seed）与凭据来源
- 本地 `.env`（已 gitignore）补齐两个密钥以避免本机 compose 失效

### Out

- `platform/system/*`、共享 UI/权限/租户/请求拦截器实现由 pantheon-base 拥有，按正式 foundation release 同步
- 生产口令生成、托管与轮换流程（部署参数校验与密钥管理独立任务）
- `.env.example`（Base 跟踪）与 `database/README.md`、`docs/designs/REPOSITORY_LAYOUT.md` 文案（内容已正确）
- 真实 `docker compose up` 运行态验证（本机 Docker daemon 未运行；本轮为配置解析级验证）

## Assumptions and Open Questions

- Confirmed Facts: 原文件在第 7、24 行内联固定口令，第 18 行挂载废弃 DDL；`.env.example` 已定义 `MYSQL_ROOT_PASSWORD`、`REDIS_PASSWORD`，说明插值化是原设计意图；Base 文档已声明 Compose 不再挂载 DDL。
- Working Assumptions: 已初始化的本地数据卷口令不变，故本地 `.env` 沿用既有值保持兼容；CI 与脚本均未引用 compose（全仓检索为空）。
- Open Questions: 生产 Compose 是否为客户安装入口未确认；若是，则本项应提升为 P0 并补密钥生成与轮换要求。
- Evidence State: 配置解析级验证已完成；容器实际启动与首建库行为未在本轮执行。

## Structural Scope

- Affected Subgraph: compose 安装入口 -> MySQL/Redis 凭据与初始化目录
- Boundary Crossings: Ops overlay -> 部署配置；建库权威 -> Base 迁移
- Risk Nodes: O-01
- Graph Focus: sensitive-input-flow

## Expected Files

### Create

- `docs/harness/tasks/2026-11-24-ops-compose-credentials.task.md`
- `.harness/tasks/2026-11-24-ops-compose-credentials/manifest.json`
- `.harness/tasks/2026-11-24-ops-compose-credentials/STATUS.md`
- `.harness/evidence/2026-11-24-ops-compose-credentials/commands.json`
- `.harness/evidence/2026-11-24-ops-compose-credentials/summary.md`
- `.harness/evidence/2026-11-24-ops-compose-credentials/review.md`

### Modify

- `docker-compose.yml`
- `.env`（本机、已 gitignore，不进入仓库）

### Do Not Touch

- `backend/modules/system/**`
- `backend/internal/**`
- `.env.example`
- `database/README.md`
- `docs/designs/REPOSITORY_LAYOUT.md`
- `foundation-release.lock.json`
- `既有归档原始证据与其他执行者未提交改动`

## Implementation Notes

- Minimal Complexity Rung: 标准能力
- 使用 Compose 内建插值与 `:?` 必填语法，不引入 `env_file`、外部密钥工具或新脚本
- 保留端口、镜像、字符集、数据卷与数据库名，控制 diff 到凭据与挂载两处
- Redis 沿用 `--requirepass` 命令行方式，避免为单个参数替换镜像 entrypoint
- 口令值不写入仓库；evidence 只记录长度与解析成功，不落明文

## Acceptance Criteria

- [x] compose 文件中不再出现任何明文数据库/缓存口令
- [x] 缺省变量时 `docker compose config` 以明确信息失败，不提供隐式默认口令
- [x] 变量齐备时 `docker compose config` 解析成功，Redis 口令非空
- [x] 解析结果不含 `docker-entrypoint-initdb` / `system_init` 挂载
- [x] 仓库内无其他硬编码口令出现位置（检索为空）

## Verification Plan

- `docker compose config`（缺 `.env` 密钥时应失败），工作目录仓库根
- `docker compose config`（密钥齐备时应成功），工作目录仓库根
- `docker compose config | node -e "<assert requirepass 与无 initdb>"`，工作目录仓库根
- `grep -c "DHCCroot\|DHCCdhcc" docker-compose.yml`，工作目录仓库根

- 配置解析级验证；容器启动、首建库、既有数据卷兼容未执行（Docker daemon 未运行）
- 无接口/权限/菜单/i18n/schema/seed/fixture 影响

## Execution Roles

- Implementer Posture: 部署配置维护者，只改 Ops overlay 内的 compose 文件与本地忽略文件。
- Reviewer Posture: 对照凭据泄露面、失败语义、建库单源与既有数据卷兼容审查。

## Stop Points

- 生产口令托管/轮换方案与 Compose 是否为客户安装入口需维护者确认
- 需要修改 Base 跟踪文件（`.env.example`、`database/README.md`、`docs/designs/REPOSITORY_LAYOUT.md`）时不在本包内改

## State Plan

- Checkpoint Expectation: 单文件配置级闭合，含失败语义与解析成功双证据

## Linkage

- Task ID: 2026-11-24-ops-compose-credentials
- Task Manifest: .harness/tasks/2026-11-24-ops-compose-credentials/manifest.json
- Evidence Directory: .harness/evidence/2026-11-24-ops-compose-credentials/
- Review File: .harness/evidence/2026-11-24-ops-compose-credentials/review.md
- OpenSpec Change: none
- Change Ref: none
- Superpowers Plan: none
- Plan References: none

## Evidence Required

- commands.json 含真实执行状态、cwd、退出码，且不记录任何口令明文
- summary.md 记录修复前后事实、失败语义与显式 gap
- review.md 为 findings-first 自审并含机器可读 linkage

## Human Gates

- 需求/范围偏好、门禁策略与豁免、最终视觉/功能验收按维护者三触点契约处理
- 生产凭据、外部环境操作与正式发布需已有授权

## Rollback Plan

回退 `docker-compose.yml` 即可恢复原配置；本地 `.env` 增量键可保留（被 gitignore），无数据库变更

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
