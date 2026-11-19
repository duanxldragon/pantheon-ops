# Ops 迁移漂移修复与回归 — 完成

父任务：`../ops-test-fixes-2026-11-05/.harness/tasks/2026-11-05-base-v0183-consumption/plan.md`。承接 `ops-architect-retry-report.md` 的架构阻塞，本轮只处理 Ops 业务迁移装配的漂移检查及回归。

## 改动

- `scripts/business-overlay/check-shared-drift.mjs`：复用 `assertSafeManifest` 与 `planBusinessMigrations`；只有精确派生目标且 source/target 均为普通文件、字节完全一致才按业务迁移处理。Base 的工作树与 HEAD 版本冲突继续拒绝，HEAD 中已提交而在工作树删除的迁移也保持受保护。
- `tests/scripts/business-overlay/check-shared-drift.test.mjs`：覆盖合法目标、Up/Down 字节篡改（含 allowlist 绕过尝试）、未声明目标、伪造 generatedFiles、Base 文件保护、源文件缺失或目录、目标缺失、Base 版本碰撞及通用路径禁令。
- `tests/scripts/business-overlay/rebuild-from-base.test.mjs`：在确定性重建后执行真实漂移检查；验证缺少任一源文件或源路径为目录时，在复制任一派生 SQL 前失败。原有配对、版本、来源校验与 Base 碰撞测试保留。
- `backend/modules/business/cmdb/host/host_state_version_migration_test.go`：增加缺表、缺列子场景；首次 Up 仍验证装配后的完整迁移链。之后使用模块 canonical SQL 执行真实 Down/Up，断言 Down 保留已有版本，重新 Up 回填新增零版本行、保留版本 1/5，并继续验证 CAS 及陈旧版本拒绝。
- 装配器已有缺失源文件检查，本轮无需修改 `rebuild-from-base.mjs`；其 SHA-256 与开工快照一致。未编辑 SQL、manifest、其他业务代码或配置。

## 验证

- Red：`node audit-artifacts/base-freeze-2026-11-05/validate.mjs opsAssembly drift-repair-red`。新增重建/合法目标用例在旧检查器下出现 `DRIFT_MISSING_IN_BASE`，`npm run test:business-overlay` exit 1。见 [Red 日志](opsAssembly-drift-repair-red-0.log) 与 [结构化结果](opsAssembly-drift-repair-red-results.json)。
- Green：`node audit-artifacts/base-freeze-2026-11-05/validate.mjs opsAssembly drift-repair-green`。`npm run test:business-overlay` **37/37 通过，0 skipped**；`npm run check:business-overlay` exit 0；`go test -race -count=1 ./modules/business/cmdb/host -run TestHostStateVersionBackfillEnablesCAS$` exit 0（含编译、3 个隔离 MySQL 子场景）。见 [Green 结果](opsAssembly-drift-repair-green-results.json)。
- Go 环境沿用 `validate.mjs` 的 Go 1.26.10、MinGW、CGO_ENABLED=1 与必需的本地测试 MySQL；连接由 `testmysql.Open` 创建的独立随机数据库，测试清理负责删除。未执行生产数据库操作。
- `node --check`：漂移脚本和两个变更测试文件均通过；`gofmt -l` 无输出；定向 `git diff --check` 与 `go vet ./modules/business/cmdb/host` exit 0。见 [诊断与最终源码哈希](ops-drift-repair-diagnostics.json)。
- 当前工作树两个真实 `000032` 目标与模块 canonical SQL 字节、SHA-256 均相等，已记录在诊断 JSON。

## 自检与剩余边界

- 自检未发现剩余范围内阻塞。新增 gate 不信任报告的 generatedFiles，也不允许 allowlist 绕过 canonical SQL 一致性检查；通用共享路径仍走原漂移规则。
- 用户禁止调用子代理，本轮为 Generator 自检；独立架构复审由父任务承接。本轮证据对应 4 个修改文件，已有装配器及其他协作者改动保留。
- 正式 v0.18.3 制品消费、真实完整产品树重建、全量 race、前端及业务 smoke 未在本轮执行，仍属于父任务后续门禁；这里不作发布或完整升级完成声明。未 commit/push。

Gate Outcomes: shared-drift false-positive 合法派生迁移被判 Base 缺失（已修复） | shared-drift caught 字节篡改、伪造派生目标、缺失来源与 Base 版本冲突。
