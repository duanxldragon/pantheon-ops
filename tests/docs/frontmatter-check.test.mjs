import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const moduleUrl = pathToFileURL(
  path.resolve(testDir, '../../scripts/frontmatter-check.mjs'),
).href;

const {
  parseFrontmatter,
  validateDoc,
  hasLegacyMetadata,
  parseContractBodyReferences,
  extractReadmeDocLinks,
  extractReadmeMainEntryLinks,
  summarizeCoverage,
  runCheck,
} = await import(moduleUrl);

test('parseFrontmatter reads yaml-like scalar and array fields', () => {
  const source = `---
title: Example Doc
doc_type: Design
layer: platform
depends_on_layers:
  - system/config
  - system/iam
status: Approved
index_group: superpowers-specs
retention_reason: keep it
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-18
---

# Example Doc
`;

  const parsed = parseFrontmatter(source);

  assert.equal(parsed.data.title, 'Example Doc');
  assert.equal(parsed.data.doc_type, 'Design');
  assert.deepEqual(parsed.data.depends_on_layers, ['system/config', 'system/iam']);
  assert.deepEqual(parsed.data.linked_contracts, ['docs/contracts/PLATFORM_CONTRACT.md']);
});

test('validateDoc reports missing required fields for specs docs', () => {
  const result = validateDoc({
    filePath: 'docs/superpowers/specs/example.md',
    data: {
      title: 'Broken Spec',
      doc_type: 'Design',
      layer: 'platform',
      status: 'Approved',
      updated_at: '2026-05-18',
    },
    repoRoot: process.cwd(),
  });

  assert.equal(result.ok, false);
  assert.match(result.errors.join('\n'), /index_group/);
  assert.match(result.errors.join('\n'), /retention_reason/);
  assert.match(result.errors.join('\n'), /linked_contracts/);
});

test('validateDoc reports index_group mismatch for archive directory', () => {
  const result = validateDoc({
    filePath: 'docs/archive/examples/example.md',
    data: {
      title: 'Archive Example',
      doc_type: 'Acceptance',
      layer: 'platform',
      status: 'Archived',
      index_group: 'archive/baselines',
      retention_reason: 'keep it',
      linked_contracts: ['docs/contracts/PLATFORM_CONTRACT.md'],
      updated_at: '2026-05-18',
    },
    repoRoot: process.cwd(),
  });

  assert.equal(result.ok, false);
  assert.match(result.errors.join('\n'), /index_group/);
});

test('validateDoc reports missing linked contract files', () => {
  const result = validateDoc({
    filePath: 'docs/archive/upgrade/example.md',
    data: {
      title: 'Upgrade Example',
      doc_type: 'Design',
      layer: 'platform',
      status: 'Archived',
      index_group: 'archive/upgrade',
      retention_reason: 'keep it',
      linked_contracts: ['docs/contracts/DOES_NOT_EXIST.md'],
      updated_at: '2026-05-18',
    },
    repoRoot: process.cwd(),
  });

  assert.equal(result.ok, false);
  assert.match(result.errors.join('\n'), /does not exist/);
});

test('validateDoc requires superseded_by when status is Superseded', () => {
  const result = validateDoc({
    filePath: 'docs/archive/baselines/example.md',
    data: {
      title: 'Old Baseline',
      doc_type: 'Acceptance',
      layer: 'platform',
      status: 'Superseded',
      index_group: 'archive/baselines',
      retention_reason: 'kept as baseline',
      linked_contracts: ['docs/contracts/PLATFORM_CONTRACT.md'],
      updated_at: '2026-05-18',
    },
    repoRoot: process.cwd(),
  });

  assert.equal(result.ok, false);
  assert.match(result.errors.join('\n'), /superseded_by/);
});

test('validateDoc also validates non-retained frontmatter docs', () => {
  const result = validateDoc({
    filePath: 'docs/designs/example.md',
    data: {
      title: 'Design Example',
      doc_type: 'Design',
      layer: 'platform',
      status: 'Active',
    },
    repoRoot: process.cwd(),
  });

  assert.equal(result.ok, false);
  assert.match(result.errors.join('\n'), /updated_at/);
});

test('hasLegacyMetadata detects old header style docs', () => {
  const source = `# Example\n\n更新时间：2026-05-18\n\n类型：Design\n归属层：platform\n状态：Active\n`;
  assert.equal(hasLegacyMetadata(source), true);
  assert.equal(hasLegacyMetadata('# Example\n\nNo metadata here'), false);
});

test('hasLegacyMetadata detects bullet and English header styles', () => {
  const bullet = `# Example\n\n- 更新时间：2026-08-24\n- 类型：Design\n- 归属层：\`business/k8s\`\n- 状态：Active\n`;
  assert.equal(hasLegacyMetadata(bullet), true);

  const english = `# Example\n\nUpdated: 2026-08-18  \nType: Review  \nStatus: Active\n`;
  assert.equal(hasLegacyMetadata(english), true);

  // A heading that merely starts with a metadata key is not metadata.
  assert.equal(hasLegacyMetadata('# Example\n\n## Status: done\n'), false);
});

test('hasLegacyMetadata ignores fenced examples and body occurrences', () => {
  const fenced = `# Task Packet Template\n\nCopy and fill:\n\n\`\`\`text\nTarget repo: pantheon-ops\nLayer: business/cmdb\n\`\`\`\n\nBody prose follows.\n`;
  assert.equal(hasLegacyMetadata(fenced), false);

  const bodyOnly = `# Report\n\n${'Filler line.\n'.repeat(25)}Status: waiting for CI\n`;
  assert.equal(hasLegacyMetadata(bodyOnly), false);
});

test('parseContractBodyReferences reads contract sections', () => {
  const source = `# Contract\n\n关联设计：\n- \`FOO.md\`\n- \`BAR.md\`\n\n关联验收：\n- \`BAZ.md\`\n`;
  const parsed = parseContractBodyReferences(source);
  assert.deepEqual(parsed.all, ['FOO.md', 'BAR.md', 'BAZ.md']);
  assert.deepEqual(parsed.byField.related_designs, ['FOO.md', 'BAR.md']);
  assert.deepEqual(parsed.byField.related_acceptances, ['BAZ.md']);
});

test('parseContractBodyReferences tolerates the prettier blank line after a relation heading', () => {
  const source = `# Contract\n\n关联设计：\n\n- \`FOO.md\`\n- \`BAR.md\`\n\n关联验收：\n- \`BAZ.md\`\n`;
  const parsed = parseContractBodyReferences(source);
  assert.deepEqual(parsed.byField.related_designs, ['FOO.md', 'BAR.md']);
  assert.deepEqual(parsed.byField.related_acceptances, ['BAZ.md']);

  const trailing = `# Contract\n\n关联设计：\n- \`FOO.md\`\n\n段落文字\n\n- \`NOTARELATION.md\`\n`;
  assert.deepEqual(parseContractBodyReferences(trailing).byField.related_designs, ['FOO.md']);
});

test('extractReadmeDocLinks ignores external and anchor links', () => {
  const source = `[Local](./designs/FOO.md)\n[Web](https://example.com)\n[Anchor](#section)\n[Parent](../DESIGN.md)`;
  assert.deepEqual(extractReadmeDocLinks(source), ['./designs/FOO.md', '../DESIGN.md']);
});

test('extractReadmeMainEntryLinks limits checks to main entry sections', () => {
  const source = `## 2. Main\n[Active](./designs/FOO.md)\n## 5. Archive\n[Archived](./archive/examples/BAR.md)`;
  assert.deepEqual(extractReadmeMainEntryLinks(source), ['./designs/FOO.md']);
});

test('summarizeCoverage locks scanned vs with-frontmatter vs not-enforced counts', () => {
  const scanned = Array.from({ length: 415 }, (_, index) => `docs/example-${index}.md`);
  const withFrontmatter = scanned.slice(0, 302);
  assert.deepEqual(summarizeCoverage(scanned, withFrontmatter), {
    scanned: 415,
    withFrontmatter: 302,
    withoutFrontmatter: 113,
  });
  assert.deepEqual(summarizeCoverage([], []), {
    scanned: 0,
    withFrontmatter: 0,
    withoutFrontmatter: 0,
  });
});

// AUD-025 lock #14: the 90-day Active review window is reported, never enforced.
// Negative-verified: folding activeReviewOverdue into the ok verdict turns this red
// ('overdue docs must not fail the gate / false !== true').
test('Active docs past the 90-day window are reported as overdue without failing the gate', () => {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-frontmatter-overdue-'));
  try {
    fs.mkdirSync(path.join(repoRoot, 'docs'), { recursive: true });
    const daysAgo = (n) => new Date(Date.now() - n * 86_400_000).toISOString().slice(0, 10);
    const fm = (title, docType, updatedAt, extra = []) => [
      '---',
      `title: ${title}`,
      `doc_type: ${docType}`,
      'layer: platform',
      'status: Active',
      ...extra,
      `updated_at: ${updatedAt}`,
      '---',
      '',
      `# ${title}`,
      '',
    ].join('\n');
    // Design is in DOC_TYPES_REQUIRING_CONTRACTS, so the fixture must carry a
    // resolvable Contract doc — otherwise the docs go red for missing
    // linked_contracts and the window assertion proves nothing.
    const contractRef = ['linked_contracts:', '  - docs/scope-contract.md'];
    fs.writeFileSync(path.join(repoRoot, 'docs', 'scope-contract.md'), fm('Scope Contract', 'Contract', daysAgo(5)), 'utf8');
    fs.writeFileSync(path.join(repoRoot, 'docs', 'old-active.md'), fm('Old Active Doc', 'Design', daysAgo(200), contractRef), 'utf8');
    fs.writeFileSync(path.join(repoRoot, 'docs', 'fresh-active.md'), fm('Fresh Active Doc', 'Design', daysAgo(5), contractRef), 'utf8');

    const result = runCheck(repoRoot);

    assert.equal(result.ok, true, `overdue docs must not fail the gate / ${result.errors.join("; ")}`);
    assert.equal(result.activeReviewOverdue.length, 1);
    assert.equal(result.activeReviewOverdue[0].path, 'docs/old-active.md');
    assert.ok(result.activeReviewOverdue[0].days >= 200);
  } finally {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  }
});
