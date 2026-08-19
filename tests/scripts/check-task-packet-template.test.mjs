import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath, pathToFileURL } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const moduleUrl = pathToFileURL(
  path.resolve(testDir, '../../scripts/check-task-packet-template.mjs'),
).href;

const { validateTaskPacketTemplate } = await import(moduleUrl);

function withTemplateRepo(templateContent, callback) {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-task-template-'));
  try {
    fs.mkdirSync(path.join(repoRoot, 'docs', 'acceptances'), { recursive: true });
    fs.mkdirSync(path.join(repoRoot, 'docs'), { recursive: true });
    fs.writeFileSync(
      path.join(repoRoot, 'docs', 'README.md'),
      '[Task Packet](./acceptances/TASK_PACKET_BASE_TEMPLATE.md)\n',
      'utf8',
    );
    fs.writeFileSync(
      path.join(repoRoot, 'docs', 'README.en.md'),
      '[Task Packet](./acceptances/TASK_PACKET_BASE_TEMPLATE.md)\n',
      'utf8',
    );
    fs.writeFileSync(
      path.join(repoRoot, 'docs', 'acceptances', 'TASK_PACKET_BASE_TEMPLATE.md'),
      templateContent,
      'utf8',
    );
    fs.writeFileSync(
      path.join(repoRoot, 'docs', 'acceptances', 'TASK_PACKET_BASE_TEMPLATE.en.md'),
      templateContent,
      'utf8',
    );

    callback(repoRoot);
  } finally {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  }
}

test('validateTaskPacketTemplate requires quality profile and ratchet decision markers', () => {
  const template = `目标仓库：pantheon-base
同步要求：
如果共享能力会影响 pantheon-ops，只记录
Target repo: pantheon-base
Sync expectation:
\`base -> ops\` sync is required
`;

  withTemplateRepo(template, (repoRoot) => {
    const findings = validateTaskPacketTemplate(repoRoot);

    assert.match(findings.join('\n'), /Quality Profile/);
    assert.match(findings.join('\n'), /Ratchet Decision/);
  });
});
