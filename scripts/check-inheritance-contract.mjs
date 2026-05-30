import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const root = process.cwd();

const requiredFiles = [
  'AGENTS.md',
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

for (const relativePath of requiredFiles) {
  const absolutePath = path.join(root, relativePath);
  if (!fs.existsSync(absolutePath)) {
    findings.push(`${relativePath}: required inheritance file is missing`);
    continue;
  }

  const content = fs.readFileSync(absolutePath, 'utf8');
  for (const marker of requiredMarkers[relativePath] ?? []) {
    if (!content.includes(marker)) {
      findings.push(`${relativePath}: missing required marker: ${marker}`);
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
