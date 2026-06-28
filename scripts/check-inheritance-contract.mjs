import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const root = process.cwd();

const requiredFiles = [
  'AGENTS.md',
  'foundation-release.lock.json',
  'docs/README.md',
  'docs/README.en.md',
  'docs/PROJECT_INHERITANCE.md',
  'docs/PROJECT_INHERITANCE.en.md',
];

const requiredMarkers = {
  'AGENTS.md': [
    'pantheon-base',
    'business/*',
    '先判断是否应在 `pantheon-base` 修复，再同步到 ops',
  ],
  'docs/README.md': [
    'PROJECT_INHERITANCE.md',
    'TASK_PACKET_OPS_TEMPLATE.md',
  ],
  'docs/README.en.md': [
    'PROJECT_INHERITANCE.md',
    'TASK_PACKET_OPS_TEMPLATE.md',
  ],
  'docs/PROJECT_INHERITANCE.md': [
    'Base repository：当前继承源是 `../pantheon-base`',
    'Base version：当前锁定到',
    'business/cmdb',
    'business/deploy',
    '如果 foundation 规则必须变更，先改 `pantheon-base`，再升级 `pantheon-ops`',
    '这次共享改动对应的 base commit 是什么',
    '共享路径哪些已同步，哪些故意未同步',
    '是否分别验证了 base 和 ops 的最小启动、build 或 smoke',
  ],
  'docs/PROJECT_INHERITANCE.en.md': [
    'Base repository: `../pantheon-base`',
    'Base version:',
    'business/cmdb',
    'business/deploy',
    'update `pantheon-base` first and then upgrade `pantheon-ops`',
    'which base commit introduced the shared change',
    'which shared paths were synced and which were intentionally left out',
    'whether base and ops each received their minimum validation pass',
  ],
};

const findings = [];
let releaseLock = null;

for (const relativePath of requiredFiles) {
  const absolutePath = path.join(root, relativePath);
  if (!fs.existsSync(absolutePath)) {
    findings.push(`${relativePath}: required inheritance file is missing`);
    continue;
  }

  const content = fs.readFileSync(absolutePath, 'utf8');
  if (relativePath === 'foundation-release.lock.json') {
    try {
      releaseLock = JSON.parse(content);
    } catch (error) {
      findings.push(`foundation-release.lock.json: invalid JSON (${error.message})`);
      continue;
    }
  }
  for (const marker of requiredMarkers[relativePath] ?? []) {
    if (!content.includes(marker)) {
      findings.push(`${relativePath}: missing required marker: ${marker}`);
    }
  }
}

if (releaseLock) {
  if (releaseLock.schemaVersion !== 1) {
    findings.push('foundation-release.lock.json: schemaVersion must be 1');
  }
  if (releaseLock.baseRepo !== '../pantheon-base') {
    findings.push('foundation-release.lock.json: baseRepo must be ../pantheon-base');
  }
  if (releaseLock.sourceRepo !== 'pantheon-base') {
    findings.push('foundation-release.lock.json: sourceRepo must be pantheon-base');
  }
  if (releaseLock.consumerMode !== 'foundation-release-consumer') {
    findings.push('foundation-release.lock.json: consumerMode must be foundation-release-consumer');
  }
  if (typeof releaseLock.releaseLine !== 'string' || releaseLock.releaseLine.length === 0) {
    findings.push('foundation-release.lock.json: releaseLine must be a non-empty string');
  }
  if (typeof releaseLock.releaseVersion !== 'string' || releaseLock.releaseVersion.length === 0) {
    findings.push('foundation-release.lock.json: releaseVersion must be a non-empty string');
  }
  if (typeof releaseLock.baseCommit !== 'string' || releaseLock.baseCommit.length === 0) {
    findings.push('foundation-release.lock.json: baseCommit must be a non-empty string');
  }
  if (releaseLock.releaseDisplayName !== releaseLock.releaseVersion.replace(/^.*?(v\d+\.\d+\.\d+)$/, '$1')) {
    findings.push('foundation-release.lock.json: releaseDisplayName must match the short semver suffix of releaseVersion');
  }
  if (releaseLock.releaseArtifact?.tagName !== releaseLock.releaseVersion) {
    findings.push('foundation-release.lock.json: releaseArtifact.tagName must equal releaseVersion');
  }
  if (releaseLock.releaseArtifact?.releaseName !== releaseLock.releaseDisplayName) {
    findings.push('foundation-release.lock.json: releaseArtifact.releaseName must equal releaseDisplayName');
  }

  const expectedZhVersionLine = `- Base version：当前锁定到 \`${releaseLock.releaseVersion}\`（\`${releaseLock.baseCommit}\`）`;
  const expectedZhReleaseLine = `- Base release line：当前跟随 \`${releaseLock.releaseLine}\``;
  const expectedEnVersionLine = `- Base version: \`${releaseLock.releaseVersion}\` (\`${releaseLock.baseCommit}\`)`;
  const expectedEnReleaseLine = `- Base release line: \`${releaseLock.releaseLine}\``;
  const zhDocPath = path.join(root, 'docs', 'PROJECT_INHERITANCE.md');
  const enDocPath = path.join(root, 'docs', 'PROJECT_INHERITANCE.en.md');

  if (fs.existsSync(zhDocPath)) {
    const zhDoc = fs.readFileSync(zhDocPath, 'utf8');
    if (!zhDoc.includes(expectedZhReleaseLine)) {
      findings.push(`docs/PROJECT_INHERITANCE.md: release line must match foundation-release.lock.json (${releaseLock.releaseLine})`);
    }
    if (!zhDoc.includes(expectedZhVersionLine)) {
      findings.push(`docs/PROJECT_INHERITANCE.md: release version/base commit must match foundation-release.lock.json (${releaseLock.releaseVersion})`);
    }
  }

  if (fs.existsSync(enDocPath)) {
    const enDoc = fs.readFileSync(enDocPath, 'utf8');
    if (!enDoc.includes(expectedEnReleaseLine)) {
      findings.push(`docs/PROJECT_INHERITANCE.en.md: release line must match foundation-release.lock.json (${releaseLock.releaseLine})`);
    }
    if (!enDoc.includes(expectedEnVersionLine)) {
      findings.push(`docs/PROJECT_INHERITANCE.en.md: release version/base commit must match foundation-release.lock.json (${releaseLock.releaseVersion})`);
    }
  }
}

if (findings.length > 0) {
  console.error('Pantheon Ops inheritance contract check failed');
  for (const finding of findings) {
    console.error(`- ${finding}`);
  }
  process.exit(1);
}

console.log('OK pantheon-ops inheritance contract markers are present');
