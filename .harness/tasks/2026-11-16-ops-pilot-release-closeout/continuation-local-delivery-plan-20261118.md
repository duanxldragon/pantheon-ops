# C06 本地交付准备计划（2026-11-18）

Parent task: `2026-11-16-ops-pilot-release-closeout`；L2；业务消费者交付。Minimal Complexity Rung: reuse（已锁定 Foundation 重建器、现有 Dockerfile/启动入口与发布身份契约）。本轮只准备计划与清单；package / install / upgrade / failure recovery / rollback / publication 均 `not-run`，不改变原任务状态。

## In / Out

In：绑定 C05 通过的同一 clean candidate，定义 Ops 包内容、SHA-256、业务版本、锁定 Foundation、迁移与扩展开放清单；列出独立安装、升级、故障恢复、回滚所需证据及真实命令来源。

Out：不打包、不启动服务、不访问凭据或生产、不发布、不复制 Base producer、不修改应用/共享脚本/compose/README/lock/现有 manifest/status。依赖未通过不执行后续验收。

## 文件与步骤

1. 新增 `continuation-local-delivery-checklist-20261118.md`，覆盖原 manifest 的全部四项验收（包含开放扩展 E04 或实际裁剪验证）。
2. 新增 `.harness/evidence/2026-11-16-ops-pilot-release-closeout/local-preparation-handoff-20261118.md` 与 `.json`，记录准备边界、来源哈希、真实 gap 与所有 runtime 步骤 `not-run`。
3. 静态核对所有引用路径/脚本存在、JSON 可解析、四项原验收都有落点；由协调者评审后冻结本子任务文件。

## 前置门禁与停止条件

- C01–C05 完成，同一候选 P0 零遗留；本轮 C05 的 CI 回归成功只证明接线，尚未证明 hosted 或真实 provider/role 验收。
- 开放扩展逐项通过 E04；不开放能力在同一候选验证菜单、路由/权限、可见入口与说明裁剪。
- 工作区 clean；最终重建报告的 Foundation commit、锁文件、源码与 migration hash 一致。当前未提交工作区不能用 HEAD 作为交付候选证明。
- P1 延期有维护者决定、owner、截止日期与补偿控制；agent 不降低门禁。
- 实施者提交本地准备证据，协调者负责独立 review/后续执行；最终视觉/功能与风险接受由维护者验收。

准备完成的停止条件：清单内容可追溯，静态验证通过，runtime gaps 明确。C06 关闭的停止条件另见清单；准备完成不代表 C06 完成。

## Linkage

- Task manifest: `.harness/tasks/2026-11-16-ops-pilot-release-closeout/manifest.json`
- Task doc: `docs/harness/tasks/2026-11-16-ops-pilot-release-closeout.task.md`
- Checklist: `.harness/tasks/2026-11-16-ops-pilot-release-closeout/continuation-local-delivery-checklist-20261118.md`
- Evidence: `.harness/evidence/2026-11-16-ops-pilot-release-closeout/local-preparation-handoff-20261118.md`
- Machine evidence: `.harness/evidence/2026-11-16-ops-pilot-release-closeout/local-preparation-handoff-20261118.json`
