# Evidence Summary

按 2026-11-17 反向角色回退指令（用户明确 Claude 直接实现、Codex 验收）继续推进本任务，
覆盖面为 `backend/modules/business/cmdb/host/`（host_config.go / host_config_test.go /
host_service.go/）、`docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md` §11.2 与本任务 manifest/
evidence。未触碰导航 seed/i18n、P0 历史物料、G1 workflow、全局 STATUS/OPEN_TASKS/INVENTORY。

## 1. 窄范围核对锁定 Base v0.18.9 公共扩展点：结论

对 `runtime_settings.go`、`tenant.SettingFlagReader`（`backend/pkg/tenant/membership.go`）、
`setting.SettingService` 公共 API（`NewSettingService`/`GetByKey`/`WithTenantContext`）、
`setting_seed.go` 逐一核对：

- 读取侧：`tenant.SettingFlagReader`（`GetByKey` 接口形状）与 `pkg/contracts
RegisterRuntimeSettingReloader`（变更刷新回调）都不是**取值供应端**——前者只是接口、
  后者只是通知，业务模块无法借此在不 import 内部 `system/config/setting` Service 的前提
  下获得一个可用 reader 实现。
- seed 注册侧：Base `setting_seed.go` 的 `defaultSettingSeeds` 为包私有硬编码清单，无
  `RegisterSettingSeed` 消费者 API，`cmdb.ssh.*` 两 key 无法由业务注册进系统配置面。

故锁定 Base 无合适的公开注册/读取供应能力，生产接线与 seed 注册确属 **Base-first 缺口**。
未在 Ops 直接读 `system_setting` 表、未 import 内部 `SettingService`、未用环境变量绕过、
未写 Base 源码、未复制共享扩展点假关闭。最小所需公共 contract 与 Base canonical 落点已
记录于 §11.2（取值 seam 沿既有 `RegisterBusinessAuthorizationReader`/`RegisteredBusinessTenantModeReader`
单例模式放在 Base `pkg/contracts`；seed 注册在 Base `system/config/setting/setting_seed.go`）。

## 2. 实际修复（业务边界内）

- 解析溢出：`resolveCollectTimeout` 先在整数秒内与 `maxSSHCollectSeconds`(600) 比较再转
  `time.Duration`，避免 MaxInt64 数量秒乘 `time.Second` 溢出为负数绕过上限。新增回归：
  MaxInt64 秒、巨大正数（可解析且远超上限）、超上限 1s、上限内有效边界（599/600）、
  配置读取错误回落默认。保证返回 timeout 恒正数且 <=10min。
- default_port 业务路径：`Create` 在主机未显式端口时持久化 `defaultSSHPort()`（回落 22），
  使配置 `default_port` 运行时真正生效，而非只测 `effectiveSSHPort(0)` 的死分支。新增
  create 落库回归（配置 2222 生效 + 无 reader 回落 22 + 显式端口优先）。
- 配置变更影响 collect：新增测试证明 `collectTimeout()` 与拨号端口随注入 reader 的
  value 变化而下次即时生效（Collect 每次调用重新解析，无缓存/无重启）。只验证业务注入
  seam，不宣称生产装配。未强化未经批准的 CMDB 多租户语义——两张 stub reader 不是租户
  隔离证明，本批不新增该主张。

## 3. 验证与状态

- Codex 已复核纯配置回归 PASS（见 codex-config-race.log：`ok .../cmdb/host 1.111s`）。
- 本会话无批准面，`go` 命令（gofmt/vet/test/build）均被环境拒绝，未执行，如实记 not-run
  于 commands.json，不将 not-run 写为 passed；新增测试交由 Codex 在可编译环境
  （真实 MySQL 隔离测试，`pkg/testmysql`）统一复核。
- 生产注入（opsinit 装配）与 seed 注册仍未完成，属 Base 前置缺口；任务保持 in-progress /
  awaiting-review，本执行者不自批准。未修改 evidence 内原任务卡字节，未虚构 hosted/liveSSH
  证据（本任务无 hosted/liveSSH 运行，commands.json 不含此类条目）。

## 2026-11-17 Codex verification addendum

Actual Windows CGO race passed the current parsing, overflow, port precedence, dynamic reader change, and isolated MySQL Create persistence regressions; package `ok .../cmdb/host 1.291s`. Log: `../2026-11-16-ops-navigation-repair/go-race-ssh-config-current.log`. Earlier not-run entries remain historical. Stub reader tests do not establish real tenant isolation. Production supplier/seed wiring, independent review, and live SSH remain open; task stays in-progress.
