import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const checkerPath = fileURLToPath(new URL('../../scripts/harness/check-failure-registry.mjs', import.meta.url));

const HEADER =
  '| Failure ID | Category | Failure Class | Owner Layer | Occurrences | Example | Impact | GitHub Signal | Current Guide | Current Sensor | Current Gate | Detected By | Missed By | Recommended Harness Change | Promotion Decision | Promotion Deadline | Status |';
const SEPARATOR = `|---|---|---|---|---:|---|---|---|---|---|---|---|---|---|---|---|---|`;

function row(failureId, status = 'implemented') {
  const cells = [
    `\`${failureId}\``,
    'method-health',
    'method-health-gap',
    'consumer-repository',
    '1',
    'Example text.',
    'Impact text.',
    'method-gate',
    'none',
    'none',
    'none',
    'none',
    'none',
    'sensor',
    'sensor-added',
    'none',
    status,
  ];
  return `| ${cells.join(' | ')} |`;
}

function makeRoot(body) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'failure-registry-'));
  fs.mkdirSync(path.join(root, 'docs', 'harness'), { recursive: true });
  fs.writeFileSync(
    path.join(root, 'docs', 'harness', 'failure-registry.md'),
    ['# Failure Registry', '', '## Registry', '', HEADER, SEPARATOR, body, '', '## Review Notes', '', '- Done.', ''].join('\n'),
  );
  return root;
}

function run(root) {
  try {
    const stdout = execFileSync(process.execPath, [checkerPath, '--root', root, '--strict'], { encoding: 'utf8' });
    return { code: 0, stdout };
  } catch (error) {
    return { code: error.status, stdout: `${error.stdout ?? ''}${error.stderr ?? ''}` };
  }
}

test('a valid registry passes', () => {
  const root = makeRoot([row('FR-001'), row('FR-002')].join('\n'));
  const result = run(root);
  assert.equal(result.code, 0, result.stdout);
  assert.match(result.stdout, /0 error\(s\)/);
});

test('rows after a blank separator line are still validated', () => {
  const root = makeRoot([row('FR-001'), '', row('FR-002', 'implemented-bogus')].join('\n'));
  const result = run(root);
  assert.equal(result.code, 1, result.stdout);
  assert.match(result.stdout, /Invalid Status "implemented-bogus"/);
});

test('prose after the table still ends the scan', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'failure-registry-'));
  fs.mkdirSync(path.join(root, 'docs', 'harness'), { recursive: true });
  fs.writeFileSync(
    path.join(root, 'docs', 'harness', 'failure-registry.md'),
    [
      '# Failure Registry',
      '',
      '## Registry',
      '',
      HEADER,
      SEPARATOR,
      row('FR-001'),
      '',
      '## Review Notes',
      '',
      '| not a registry row | another cell |',
      '|---|---|',
      '| `FR-999` | bogus-category |',
      '',
    ].join('\n'),
  );
  const result = run(root);
  assert.equal(result.code, 0, result.stdout);
});
