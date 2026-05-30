import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const root = process.cwd();

const requiredFiles = [
  'docs/README.md',
  'docs/README.en.md',
  'docs/TASK_PACKET_OPS_TEMPLATE.md',
  'docs/TASK_PACKET_OPS_TEMPLATE.en.md',
];

const requiredMarkers = {
  'docs/README.md': [
    'TASK_PACKET_OPS_TEMPLATE.md',
    '`base -> ops`',
  ],
  'docs/README.en.md': [
    'TASK_PACKET_OPS_TEMPLATE.md',
    '`base -> ops` sync',
  ],
  'docs/TASK_PACKET_OPS_TEMPLATE.md': [
    '目标仓库：pantheon-ops',
    '或 `base -> ops` 同步',
    '写清 base commit',
  ],
  'docs/TASK_PACKET_OPS_TEMPLATE.en.md': [
    'Target repo: pantheon-ops',
    'or `base -> ops` sync',
    'record the base commit',
  ],
};

const findings = [];

for (const relativePath of requiredFiles) {
  const absolutePath = path.join(root, relativePath);
  if (!fs.existsSync(absolutePath)) {
    findings.push(`${relativePath}: required task-packet template file is missing`);
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
  console.error('Pantheon Ops task-packet template check failed');
  for (const finding of findings) {
    console.error(`- ${finding}`);
  }
  process.exit(1);
}

console.log('OK pantheon-ops task-packet template is present and linked');
