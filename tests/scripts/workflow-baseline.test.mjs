import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const workflowsRoot = path.resolve('.github', 'workflows');
const workflowFiles = fs.readdirSync(workflowsRoot)
  .filter((fileName) => fileName.endsWith('.yml'))
  .sort();
const workflowSource = workflowFiles
  .map((fileName) => fs.readFileSync(path.join(workflowsRoot, fileName), 'utf8'))
  .join('\n');

test('workflow actions and Node runtime match the certified Base baseline', () => {
  assert.doesNotMatch(workflowSource, /NODE_VERSION:\s*["']?22|node-version:\s*["']?22/u);
  for (const stalePin of [
    'df4cb1c069e1874edd31b4311f1884172cec0e10',
    '48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e',
    'fbd0ab8f3e69293af611ebaee6363fc25e6d187d',
    '4a3601121dd01d1626a1e23e37211e3254c1c06c',
    'a309ff8b426b58ec0e2a45f0f869d46889d02405',
    '7211b7c8077ea37d8641b6271f6a365a22a5fbfa',
    '65462800fd760344b1a7b4382951275a0abb4808',
    '08891c257ab1e72ecee86875cb7bd7cd404c9583',
  ]) {
    assert.doesNotMatch(workflowSource, new RegExp(stalePin, 'u'));
  }
  assert.match(workflowSource, /actions\/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1/u);
  assert.match(workflowSource, /actions\/setup-node@820762786026740c76f36085b0efc47a31fe5020/u);
  assert.match(workflowSource, /dorny\/paths-filter@ceb8a2b8f2d89434be7ff52d3de7ec3738c5cc9d/u);
});

test('base compatibility workflow rebuilds Ops on the current Base snapshot', () => {
  const source = fs.readFileSync(
    path.join(workflowsRoot, 'inheritance-drift-detection.yml'),
    'utf8',
  );
  assert.match(source, /path:\s*pantheon-ops/u);
  assert.match(source, /path:\s*pantheon-base/u);
  assert.match(source, /working-directory:\s*pantheon-ops/u);
  assert.match(source, /test:business-overlay/u);
  assert.match(source, /rebuild:from-base/u);
  assert.match(source, /check-business-overlay\.mjs/u);
});
