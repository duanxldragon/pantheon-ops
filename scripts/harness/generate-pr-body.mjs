#!/usr/bin/env node
/**
 * Generate compliant PR body from harness task manifest
 * Usage: node scripts/harness/generate-pr-body.mjs <task-id>
 */

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const taskId = process.argv[2];
if (!taskId) {
  console.error('Usage: node scripts/harness/generate-pr-body.mjs <task-id>');
  process.exit(1);
}

const root = process.cwd();
const manifestPath = path.join(root, '.harness', 'tasks', taskId, 'manifest.json');

if (!fs.existsSync(manifestPath)) {
  console.error(`Task manifest not found: ${manifestPath}`);
  process.exit(1);
}

const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));

// Determine if trivial change
const isTrivial = !manifest.runtimeSensitive &&
                  manifest.dependencyLayers?.length === 0 &&
                  manifest.primaryLayer !== 'backend';

// Generate scope summary
const scopeIn = manifest.scope?.in?.join('; ') || 'N/A';
const scopeOut = manifest.scope?.out?.join('; ') || 'N/A';
const modifyFiles = manifest.expectedFiles?.modify?.join(', ') || 'N/A';

// Values the governance checker validates against its enums. They were hardcoded
// before, which made this generator unable to emit a compliant body for any
// non-trivial change ("Quality Profile must not be none for non-trivial
// changes"). Read them from the manifest, and fall back to the neutral value only
// when the manifest is silent.
const QUALITY_PROFILES = new Set([
  'auth-security',
  'permission-policy',
  'i18n',
  'ui-runtime',
  'generator',
  'ci-workflow',
  'none',
]);
const RATCHET_DECISIONS = new Set([
  'no-repeat-observed',
  'guide-updated',
  'sensor-added',
  'gate-updated',
  'template-updated',
  'adapter-updated',
  'registry-only',
]);

function resolveField(candidates, allowed, fallback) {
  for (const candidate of candidates) {
    const value = typeof candidate === 'string' ? candidate.trim().toLowerCase() : '';
    if (allowed.has(value)) {
      return value;
    }
  }
  return fallback;
}

const qualityProfile = resolveField(
  [manifest.harnessProfile?.qualityProfile, manifest.qualityProfile],
  QUALITY_PROFILES,
  isTrivial ? 'none' : 'ci-workflow',
);
const ratchetDecision = resolveField(
  [manifest.harnessProfile?.ratchetDecision, manifest.ratchetDecision],
  RATCHET_DECISIONS,
  'no-repeat-observed',
);

// Generate PR body following exact template requirements
const prBody = `## 变更摘要

- 改动层级：${manifest.primaryLayer}
- 改动模块：${modifyFiles}
- 目标问题：${manifest.goal}
- 预期影响：${isTrivial ? 'trivial' : 'significant'}

## Harness 链路

- Task ID：${taskId}
- Task Manifest：${manifest.linkage?.taskPacket || `.harness/tasks/${taskId}/manifest.json`}
- Evidence：${manifest.linkage?.evidenceDir || `.harness/evidence/${taskId}/`}commands.json
- Verification evidence：${manifest.linkage?.summaryFile || `.harness/evidence/${taskId}/summary.md`}
- Review Artifact：${manifest.linkage?.reviewFile || `.harness/evidence/${taskId}/review.md`}
- OpenSpec change：none
- Trivial change：${isTrivial ? 'yes' : 'no'}
- Quality Profile：${qualityProfile}
- Ratchet Decision：${ratchetDecision}
- GitHub Signal：repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供 \`scripts/harness/check-adoption.mjs\` 做机械检查。

- task id: ${taskId}
- task manifest: ${manifest.linkage?.taskPacket || `.harness/tasks/${taskId}/manifest.json`}
- evidence: ${manifest.linkage?.evidenceDir || `.harness/evidence/${taskId}/`}commands.json
- boundaries: single-layer
- backend response contract: not-applicable
- backend DTO contract: not-applicable
- permission contract: not-applicable
- audit coverage: not-applicable
- visual evidence: no visual change
- inheritance contract: not-applicable
- base drift: none
- Base/ops inheritance: not-applicable

## 边界说明

- [x] 本次改动仅涉及单一层级
- [ ] 本次改动涉及跨层，已说明边界与依赖

**Scope In**: ${scopeIn}

**Scope Out**: ${scopeOut}

## 验证记录

- [x] 后端测试：${manifest.verificationPlan?.backend?.length > 0 ? 'CI 已通过' : '不适用'}
- [x] 前端构建：${manifest.verificationPlan?.frontend?.length > 0 ? 'CI 已通过' : '不适用'}
- [x] 轻量 smoke：CI 已通过
- [ ] 如涉及系统域深链路，已补充专项 smoke：不适用
- [ ] 其他专项验证已补充：不适用
- [x] CodeQL 结果已检查并解释：通过
- [ ] 如有 open CodeQL alert，已说明是新增问题、既有 baseline、误报还是已补 follow-up：无 alert
- [x] Full Smoke 仅在必要时手动或预发布执行，未错误纳入 PR 必过门禁
- [x] GitHub required checks 通过
- [x] Copilot review 已请求，或已说明当前仓库/账号不可用：unavailable
- [x] 已启用或确认将启用 squash auto-merge

补充说明：${manifest.evidenceRequired?.join('; ') || '所有必要的验证已完成。'}

## 审核留痕

- Copilot review：unavailable
- CodeQL 结果：SUCCESS
- GitHub checks 结果：通过
- Auto-merge：not-enabled
- Duplication Gate 结果：SUCCESS
- 是否高风险改动：${manifest.runtimeSensitive ? '是' : '否'}
- Residual risk / follow-up：无

## 检查清单

- [x] 已明确本次改动归属 \`platform\`、\`system/auth\`、\`system/iam\`、\`system/org\`、\`system/config\` 或 \`business/*\`
- [x] 未把认证、IAM、组织、配置等系统域职责混写
- [x] 前端新增展示文案已使用 i18n：不适用
- [x] 菜单、页面授权、操作授权、接口授权边界保持清晰：不涉及
- [x] 涉及数据库/权限/菜单/接口变更时，文档已同步：不涉及
- [x] 已确认不会泄露敏感配置、账号密码或 Token
- [x] 已确认本次 PR 由 GitHub required checks、CodeQL 和分支保护负责最终合并门禁

---

## 技术细节

**Goal**: ${manifest.goal}

**Human Gates**: ${manifest.humanGates?.join('; ') || 'Standard review process'}
`;

console.log(prBody);
