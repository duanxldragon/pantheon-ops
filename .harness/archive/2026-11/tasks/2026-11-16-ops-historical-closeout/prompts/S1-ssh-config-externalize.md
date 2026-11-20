# Prompt S1 — CMDB SSH 采集配置外部化（真正缺实现）

> 派发模型：由执行 Claude 通过 `codex exec ... -C pantheon-ops` 实施；本仓库 Claude 为 planner/reviewer。
> 这是全量对账中少数几个**真正缺失实现**的记录之一：`2026-07-17-cmdb-ssh-config-externalize`。

## 当前事实（对账已核实）

`backend/modules/business/cmdb/host/host_service.go` 仍硬编码：`host.SSHPort = 22`（≈line 185）、`Timeout: 10 * time.Second`（≈line 381）。全仓 grep `collect_timeout` / `default_port` / `cmdb.ssh` 零命中。`docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md` §11.2 仍标"尚未接入配置读取…配置化属于待办"。

## 范围（In / Out）

- In：为 `business.cmdb` 增加配置种子 `cmdb.ssh.collect_timeout`（默认 10）与 `cmdb.ssh.default_port`（默认 22）；host collect 读取这两项，经 **Base 提供的业务可消费 config 扩展点**（平台/系统域提供，不得直接 import `system/config` 内部）；读取缺省回退 10s/22；删除 §11.2 "待办"标注。
- Out：直接改写 Base 配置机制、绕过权限/审计、新增非最小代码、重构采集流程。

## 精确文件所有权

- `backend/modules/business/cmdb/host/host_service.go`（读配置处，回退 10/22）
- `backend/modules/business/cmdb/module.go` 或 cmdb seed（新增配置 key 种子 + i18n/默认值）
- `backend/modules/business/cmdb/host/*_test.go`（新增/锁定回退与生效测试）
- `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`（§11.2 消除待办并注明实现与 Base 扩展点）
- `.harness/tasks/2026-07-17-cmdb-ssh-config-externalize/{manifest.json,plan.md}`、`.harness/evidence/2026-07-17-cmdb-ssh-config-externalize/{commands.json,summary.md,review.md}`（关闭链）

## 硬约束

- **Base-first 阻塞检查（第一步）**：确认 pantheon-base 是否暴露 business 模块可消费的 config 读取扩展点。若不存在：**STOP，不本地绕过**，回 Base 提扩展点，并在本记录 manifest 标 `blocked-on-upstream`，说明真实前置。
- 最小复杂度：仅把两个魔法数改为配置驱动，不以"简化"削弱鉴权/审计/i18n/业务验收。
- 不回写虚构历史命令；commands.json 只记录本批真实运行。
- 不触碰 `deploy`/`bizscope` 其它 SSH 逻辑、不改 `foundation-release.lock.json`、不提交/不 push/不发布。
- UI 无关；但若设计文档渲染涉及 UI 章节，遵守 `../pantheon-base/DESIGN.md`。

## 真实验收（Codex 会核）

- grep 确认 `collect_timeout`/`default_port` 在 cmdb seed + host 读取中存在，且默认 10/22。
- 改配置值为非默认后，host collect 超时/端口实际改变（有测试证明；不满足则以 `go test` 回退断言锁定）。
- §11.2 待办标注去除；文档与实现一致。
- `go test ./backend/modules/business/cmdb/host` 通过；未 import `system/config` 内部。
- 若 blocked-on-upstream：记录真实前置 + Base 侧 ticket link，不谎报完成。
