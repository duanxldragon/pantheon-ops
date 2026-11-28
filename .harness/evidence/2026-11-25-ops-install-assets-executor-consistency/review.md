# Review — 安装包、模板与执行器能力一致性

## 结论

`approved with documented P2 follow-up`。固定模板与预置包现在有机械门禁，agent 执行器在 API 与 UI 同步关闭且存量行可读不可启动；Helm 页面不存在这一点按实际仓库状态记录，而不是伪造一个页面。

## Findings

1. **P1 fixed — 固定模板声明了不存在于预置目录的安装包。**
   以前只有注释层面的约定。新增 `FixedTemplateAssets` 目录：nginx/redis/jdk/bisheng-jdk/prometheus 声明文件名前缀，mysql/minio/harbor 显式 `NeedsOwnPackage`；`TestFixedTemplateAssetCoverage` 对照真实 `files/amd64`，`npm run check:deploy-assets` 已接入。

2. **P1 fixed — 新增模板可以绕过资产决策。**
   `fixedTemplateCodes()` 是模板白名单唯一来源，覆盖测试要求目录描述每个模板，并且每个条目要么有前缀要么有“需自备包”标记，否则直接失败。

3. **P1 fixed — agent 执行器可创建但永远不执行。**
   `validExecutorType` 去掉 agent；创建与更新对 agent 返回 `business.deploy.task.executorTypeUnsupported`，start 对存量 agent 行同样拒绝（此前会启动后静默不派发）。前端创建表单移除 Agent 选项，smoke 在真实弹窗里断言只剩手动/模拟/SSH。

4. **P1 fixed — 存量 agent 行会被误伤或无法读取。**
   无迁移、无改写：库存行详情可读、只改名称仍保留 agent 类型；启动被明确拒绝而不是卡在 running。用例 `TestLegacyAgentTaskStaysReadableButCannotStart` 覆盖。

5. **P2 fixed — 文档与代码不一致。**
   `BUSINESS_DEPLOY_MODULE_DESIGN.md` 原先写“注册 manual/simulated/ssh”，但代码接受 agent。已补：agent 在三条路径显式拒绝、存量只读、资产检查机制。

6. **P2 accepted-gap — Helm 没有 UI 页面。**
   仓库里没有 Helm 前端页面；`k8s/helm` 包注释与 `BUSINESS_K8S_MODULE_DESIGN.md` 已声明仅登记快照。S5 的“页面明示”没有可改面，登记为 gap，不新造页面。

7. **P2 accepted-gap — 资产检查只看文件名前缀。**
   同名但内容错误的归档仍能通过；哈希校验需要预置清单，属发布流程。

8. **P2 accepted-gap — 需自备包的模板没有选包时预检。**
   MySQL/MinIO/Harbor 仍可选，操作员上传归档名不匹配时仍会到执行阶段才失败；本包只把“需自备”从隐含变显式。

## 验证要点

- `npm run check:deploy-assets` 两个资产用例 PASS；5 个 deploy 一致性/执行器用例 PASS；helm 包与既有 create/import/executor 用例 PASS。
- `tsc -b`、ESLint、`npm run build`（含全部 prebuild 门禁）exit 0。
- `deploy-api` 5 passed；新 UI 用例断言创建弹窗无 Agent 选项，截图 `t22-executor-options-1440.png`。
- `deploy-pages` 9 passed / 1 failed（既有 route-content-fallback 竞态，单跑通过），与本包无关。
- 未触碰 `backend/modules/system/**`、`backend/internal/**`、`frontend/src/core/**`、`frontend/src/api/request.ts`、`foundation-release.lock.json`。

## Machine Readable

```json
{
  "taskId": "2026-11-25-ops-install-assets-executor-consistency",
  "reviewer": "primary agent self-review (no independent review on this candidate)",
  "verdict": "approved with documented P2 follow-up",
  "decision": "approved with documented P2 follow-up",
  "scope": "deploy: fixed-template asset catalog and npm gate, agent executor rejection on create/update/start with legacy read compatibility, UI option removal and legacy marker, design doc alignment, i18n key",
  "linkage": {
    "taskManifest": ".harness/tasks/2026-11-25-ops-install-assets-executor-consistency/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-11-25-ops-install-assets-executor-consistency.task.md",
    "evidence": ".harness/evidence/2026-11-25-ops-install-assets-executor-consistency/commands.json",
    "reviewFile": ".harness/evidence/2026-11-25-ops-install-assets-executor-consistency/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  },
  "findings": [
    {
      "id": "T22-R1",
      "severity": "P1",
      "status": "fixed",
      "summary": "fixed templates declared preset packages absent from files/amd64",
      "verification": "FixedTemplateAssets + TestFixedTemplateAssetCoverage + npm run check:deploy-assets"
    },
    {
      "id": "T22-R2",
      "severity": "P1",
      "status": "fixed",
      "summary": "a new template could ship without an install-asset decision",
      "verification": "fixedTemplateCodes whitelist + coverage test"
    },
    {
      "id": "T22-R3",
      "severity": "P1",
      "status": "fixed",
      "summary": "agent executor was creatable but never dispatched",
      "verification": "Create/Update/Start rejection tests + UI option smoke"
    },
    {
      "id": "T22-R4",
      "severity": "P1",
      "status": "fixed",
      "summary": "legacy agent rows could not be validated for read/update compatibility",
      "verification": "TestLegacyAgentTaskStaysReadableButCannotStart"
    },
    {
      "id": "T22-R5",
      "severity": "P2",
      "status": "fixed",
      "summary": "design doc contradicted the accepted executor set",
      "verification": "BUSINESS_DEPLOY_MODULE_DESIGN.md updated"
    },
    {
      "id": "T22-R6",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "no Helm UI page exists, so the snapshot-only wording stays in the package comment and K8s design doc",
      "verification": "helm.go comment + BUSINESS_K8S_MODULE_DESIGN.md review"
    },
    {
      "id": "T22-R7",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "asset check matches name prefixes, not archive hashes",
      "verification": "UnbackedFixedTemplates implementation review"
    },
    {
      "id": "T22-R8",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "needs-own-package templates are not pre-flighted at upload time",
      "verification": "deploy create/template UI review"
    }
  ]
}
```
