# Harness PR Governance Generator

单一来源：本文件是 PR body 生成与 manifest 结构的唯一指南。
`docs/HARNESS_GOVERNANCE_GUIDE.md` 已改为指向本文件的指针（2026-11-03 合并，DOC-02/GOV-07）。

## 为什么需要它

PR governance 要求 PR body 与 harness 产物（task packet、manifest、evidence）双向对得上。
手写必然漏字段，而漏字段的失败发生在 CI 上，不在本地。生成器把这段重复劳动变成一条命令。

## 结构权威在哪

不要照抄本文档里的字段清单当规范，以下三处才是权威：

| 内容 | 权威 |
|---|---|
| PR body 必须包含的 section | `.github/pull_request_template.md`（live 模板，`check-pr-governance.mjs` 逐条校验） |
| manifest 结构合法性 | `scripts/task-manifest.mjs` 的 `validateTaskManifest` |
| task packet 文档 section | `scripts/check-task-packet-template.mjs` 与 `scripts/harness/check-task-packet.mjs` |

`.harness/tasks/sonarcloud-s4666/` 已归档，不要再把它当模板引用。

## 使用流程

### Step 1: 建 task packet 与 manifest

先写 `docs/harness/tasks/<task-id>.task.md`（按 `## Goal` / `## Scope` 等 section 模板），
再写 `.harness/tasks/<task-id>/manifest.json`。最小可用 manifest：

```json
{
  "taskId": "<task-id>",
  "taskDoc": "docs/harness/tasks/<task-id>.task.md",
  "title": "一句话标题",
  "goal": "为什么做，做到什么算完成",
  "primaryLayer": "platform",
  "status": "in-progress",
  "priority": "P2",
  "type": "<remediation|feature|governance>",
  "scope": { "in": ["..."], "out": ["..."] },
  "dependencyLayers": ["platform"],
  "contractAnchors": ["docs/contracts/PLATFORM_CONTRACT.md"],
  "expectedFiles": { "create": [], "modify": [], "doNotTouch": [] },
  "verificationPlan": { "commands": ["..."], "runtimeEvidence": ["..."] },
  "runtimeSensitive": false,
  "evidenceRequired": [".harness/evidence/<task-id>/commands.json"],
  "humanGates": [],
  "linkage": {
    "taskManifest": ".harness/tasks/<task-id>/manifest.json",
    "taskDoc": "docs/harness/tasks/<task-id>.task.md",
    "evidenceDir": ".harness/evidence/<task-id>/",
    "summaryFile": ".harness/evidence/<task-id>/summary.md",
    "reviewFile": ".harness/evidence/<task-id>/review.md",
    "changeRef": "none",
    "planRefs": []
  }
}
```

`status` 与证据绑定：有完整 evidence（含 `summary.md`）的包才是 `completed`，
对应 task 文档才可标 `Archived`；两者不一致会被 `check-task-packet.mjs` 判红。

### Step 2: 写 evidence 三件套

- `.harness/evidence/<task-id>/commands.json` — 执行了什么、读数是几（含 `knownGaps`）
- `.harness/evidence/<task-id>/summary.md` — 结论与逐条着落
- `.harness/evidence/<task-id>/review.md` — reviewer 逐条核对（需含 `## Machine Readable` 块）

### Step 3: 生成并提交 PR body

```bash
# 生成 PR body
node scripts/harness/generate-pr-body.mjs <task-id> > pr-body.txt

# 新建 PR
gh pr create --body-file pr-body.txt

# 更新已有 PR
node scripts/harness/generate-pr-body.mjs <task-id> | gh pr edit <pr-number> --body-file -
```

## 本地预检

推送前跑这三条，比等 CI 便宜得多：

```bash
node scripts/harness/check-task-packet.mjs --root .
node scripts/harness/check-evidence.mjs --root . --strict
node scripts/harness/check-review.mjs --root . --strict
```

## 常见失败

- manifest 里 `linkage.taskManifest` / `taskDoc` 路径写错 → `check-task-packet` 报缺失
- evidence 缺 `knownGaps` 或把 `runtimeSensitive` 写成数组 → `check-evidence --strict` 报结构错误
- review 缺 `## Machine Readable` 块 → `check-review --strict` 报错
- PR body 少了模板里的 section → `check-pr-governance.mjs` 在 CI 判红
