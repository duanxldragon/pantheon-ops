# C06 本地准备交接（2026-11-18）

交付计划与独立演练清单已准备；package/install/upgrade/failure recovery/rollback/offline/publication/maintainer acceptance 全部 `not-run`。没有更新 C06 或父任务完成状态。

## 交付内容

- 新增本地准备 plan，明确 C05 与开放扩展 E04 前置、文件 ownership、同候选哈希和真实验收边界。
- 新增 checklist，覆盖 clean candidate、包内/包外 SHA-256、业务版本、Foundation lock/rebuild report、migration 版本及配对字节、镜像/离线依赖、独立安装/重复启动/备份/升级/失败恢复/回滚、开放扩展/实际裁剪及维护者最终验收。
- 原 manifest 有四项验收，清单逐项映射；原 task doc 未包含的扩展验收仍保留，不缩小 acceptance。

## 验证与 review 边界

17 个源文件/plan/checklist 路径用 `Test-Path -LiteralPath` 检查存在，并捕获 SHA-256，缺失 0。具体来源、校验和和 `not-run` 操作矩阵见 JSON。本次读取真实现有脚本和启动/构建入口，没有执行这些后续命令。

Findings-first self-review：文档未虚构 install/rollback CLI，未复制 Base producer 或把其正式资产当 Ops 业务包；所有四项 acceptance 保留，候选版本/哈希缺失时不填写成功值。协调者独立 review 待完成；本记录是实施者自检。

Gate Outcomes: source review caught stale overlay report and unsuitable legacy compose delivery path | none

## 后续必须解决的 gap

1. C05 当前候选 live/role/render/hosted 验收未完成，不能开始 C06 依赖验收。
2. 原 Ops compose 只有 MySQL/Redis，包含固定资源身份/端口、明文开发凭据及 legacy SQL 挂载；不能原样用于隔离/离线业务安装。本子任务未改该文件。
3. Dockerfile 中复制 dist 不能证明 SPA/static/API proxy 能访问；真实安装/升级/回滚 runner 与目标环境入口仍须确定并验证，共享问题 Base-first。
4. 当前 overlay report baseCommit `3203624e55438866edb951bd74e671eab38d2f16` 与 lock `6316f9b7189c86542424145fa49344ad737e952b` 不同；最终冻结候选重建并刷新报告，不将旧报告用作当期继承证据。
5. 业务版本、上一 Ops 回滚包、target OS/arch、最终 immutable candidate/hash、独立执行者证据尚未形成。生产迁移/维护冻结继续独立跟踪。

## Linkage

- Task: `.harness/tasks/2026-11-16-ops-pilot-release-closeout/manifest.json`
- Plan: `.harness/tasks/2026-11-16-ops-pilot-release-closeout/continuation-local-delivery-plan-20261118.md`
- Checklist: `.harness/tasks/2026-11-16-ops-pilot-release-closeout/continuation-local-delivery-checklist-20261118.md`
- Source hashes and operation matrix: `.harness/evidence/2026-11-16-ops-pilot-release-closeout/local-preparation-handoff-20261118.json`
