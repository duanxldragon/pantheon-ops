# 能力文档、测试资产与治理对账审查

结论：approve（本包内部自查与机械门禁）。Codex 当前不可用，独立复核人未参与；维护者最终验收仍为开放 Human Gate。改动只涉及文档、测试、CI 接线与 `VERSION`，不含产品运行时代码。

## 逐条核对

1. **版本漂移**：`VERSION` 旧值 0.18.7 与 `foundation-release.lock.json`（pantheon-base-v0.18.9）和 Base 快照 `6316f9b7:VERSION`（0.18.9）不一致，属真实继承漂移而非口径差异；改为 0.18.9 后与两者一致，不影响 go.mod 锁定或 release 资产校验。
2. **版本字段角色**：在继承合同里明确六个来源的语义，并指出 `SHELL_VERSION.json`（1.4.0，锁定快照外壳 pin）与 `.harness/method-consumption.json`（1.5.0，直接方法消费）不同层；`frontend/package.json`（@duanxldragon/pantheon-base-ui 0.11.2）与 `CHANGELOG.md`（业务历史 1.2.0）不代表当前产品版本。避免后续把任一字段误当权威基线。
3. **单测腐化根因**：共享组件新增 `buildStandardPagination` 导出，两处 `vi.mock('../../../../src/components')` 工厂未同步，渲染时抛 "No export is defined" 导致 15 个用例失败；这是测试资产未接门禁的直接后果。补 stub 后不再遮蔽被测行为（stub 只用于表格分页参数，不参与断言）。
4. **CI 接线**：把 `npm run test:unit` 放进 Frontend Contract 作业，放在 lint 之后、build 之前；workflow baseline 与 `test:scripts`（242/242）仍然绿。
5. **e2e 退役依据**：`e2e-tests/tests/smoke.spec.ts` 的 19 个端点为 `/api/v1/argocd`、`/api/v1/filedist`、`/api/v1/k8s`、`/api/v1/observability`、`/api/v1/service-instances`；当前注册前缀为 `/api/v1/business/{cicd/argocd,filedist,k8s,observability,service}`。套件日期 2026-08-28、版本 v1.2.0，不在 CI、不在任何 npm 脚本；同模块由五套 CI 业务 smoke 与 430 个单测覆盖。退役是删除重复且已死资产，而不是降低覆盖面。
6. **反例已实际执行**：授权/租户/恢复反例在隔离 MySQL 上真实运行（租户迁移、namespace 可见性、permission route、背压、注入拒绝等），不是静态推断。
7. **既有红灯基线**：`check-task-packet` 657 errors / 118 warnings（115 个历史 packet）与 `check-encoding` 2 处 U+FFFD（2026-11-17 归档 JSON）均为既有基线，本批未新增，也未改动其所属文件。

## 风险与约束

- 未新增依赖；`deadcode` 未运行，避免 go.mod 漂移。
- `.business-overlay-report.json` 仍记录退役文件，属重建时点回执，无门禁读取。
- 本次 CI 接线在 PR 上会实际执行 `npm run test:unit`，是最终验证；本地已完成同命令全绿（430/430）。

## Machine Readable

```json
{
  "taskId": "2026-11-20-ops-review-docs-and-test-governance",
  "reviewer": "Claude (self-review; independent Codex unavailable)",
  "verdict": "approved",
  "decision": "approved",
  "scope": "business docs, test assets, CI wiring, VERSION alignment",
  "taskManifest": ".harness/tasks/2026-11-20-ops-review-docs-and-test-governance/manifest.json",
  "evidenceDir": ".harness/evidence/2026-11-20-ops-review-docs-and-test-governance",
  "summaryFile": ".harness/evidence/2026-11-20-ops-review-docs-and-test-governance/summary.md",
  "changeRef": "none",
  "findings": [
    "fixed: VERSION 0.18.7 did not match foundation-release.lock.json (pantheon-base-v0.18.9) and Base snapshot VERSION 0.18.9",
    "fixed: two business unit-test files failed (15 tests) because their src/components mock lacked the new buildStandardPagination export",
    "fixed: business unit tests were not wired into CI; Frontend Contract now runs npm run test:unit (430 tests green)",
    "retired: e2e-tests suite dated 2026-08-28 with 19 endpoints on pre-v1.2.0 prefixes, never in CI; modules covered by five CI business smoke suites and unit tests",
    "documented: version field roles across foundation lock, VERSION, SHELL_VERSION.json, method-consumption receipt, frontend package identity and CHANGELOG"
  ],
  "baselines": {
    "taskPacket": "115 files / 657 errors / 118 warnings (historical packets, pre-existing)",
    "encoding": "2 U+FFFD findings in .harness/evidence/2026-11-17-ops-history-audit (archived, pre-existing)",
    "docLinks": "0 findings at repository root"
  },
  "gaps": [
    "maintainer final acceptance Human Gate remains open",
    "deadcode not run: not a declared tool dependency and would force go.mod changes",
    ".business-overlay-report.json still lists retired e2e files until the next rebuild"
  ],
  "linkage": {
    "taskPacket": "docs/harness/tasks/2026-11-20-ops-review-docs-and-test-governance.task.md",
    "taskManifest": ".harness/tasks/2026-11-20-ops-review-docs-and-test-governance/manifest.json",
    "evidence": ".harness/evidence/2026-11-20-ops-review-docs-and-test-governance/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-review-docs-and-test-governance/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  }
}
```
