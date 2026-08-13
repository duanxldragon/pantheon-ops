import test from 'node:test';
import assert from 'node:assert/strict';
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

test('parseContractBodyReferences reads contract sections', () => {
  const source = `# Contract\n\n关联设计：\n- \`FOO.md\`\n- \`BAR.md\`\n\n关联验收：\n- \`BAZ.md\`\n`;
  const parsed = parseContractBodyReferences(source);
  assert.deepEqual(parsed.all, ['FOO.md', 'BAR.md', 'BAZ.md']);
  assert.deepEqual(parsed.byField.related_designs, ['FOO.md', 'BAR.md']);
  assert.deepEqual(parsed.byField.related_acceptances, ['BAZ.md']);
});

test('extractReadmeDocLinks ignores external and anchor links', () => {
  const source = `[Local](./designs/FOO.md)\n[Web](https://example.com)\n[Anchor](#section)\n[Parent](../DESIGN.md)`;
  assert.deepEqual(extractReadmeDocLinks(source), ['./designs/FOO.md', '../DESIGN.md']);
});

test('extractReadmeMainEntryLinks limits checks to main entry sections', () => {
  const source = `## 2. Main\n[Active](./designs/FOO.md)\n## 5. Archive\n[Archived](./archive/examples/BAR.md)`;
  assert.deepEqual(extractReadmeMainEntryLinks(source), ['./designs/FOO.md']);
});
