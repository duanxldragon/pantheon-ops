import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { after, test } from 'node:test';
import { loadResourceModule } from './lib/load-resource-module.mjs';

const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-load-resource-'));

after(() => {
  fs.rmSync(tmpRoot, { recursive: true, force: true });
});

test('loadResourceModule resolves imported and spread resources without executing code', () => {
  const nestedPath = path.join(tmpRoot, 'nested.ts');
  const basePath = path.join(tmpRoot, 'base.ts');
  const entryPath = path.join(tmpRoot, 'entry.ts');

  fs.writeFileSync(nestedPath, "const nested = { gamma: 'three' };\nexport default nested;\n", 'utf8');
  fs.writeFileSync(
    basePath,
    "import nested from './nested';\nconst base = { alpha: 'one', ...nested };\nexport default base;\n",
    'utf8',
  );
  fs.writeFileSync(
    entryPath,
    "import base from './base';\nconst resource = { ...base, beta: 'two' };\nexport default resource;\n",
    'utf8',
  );

  assert.deepEqual(loadResourceModule(entryPath), {
    alpha: 'one',
    beta: 'two',
    gamma: 'three',
  });
});

test('loadResourceModule rejects unsupported expressions', () => {
  const entryPath = path.join(tmpRoot, 'bad.ts');
  fs.writeFileSync(entryPath, "export default (() => ({ alpha: 'one' }))();\n", 'utf8');

  assert.throws(() => loadResourceModule(entryPath), /Unsupported expression/);
});
