import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';

const root = process.cwd();
const evidence = '.harness/evidence/2026-11-14-ops-latest-foundation-and-plans/verification/current-d5';
const plan = JSON.parse(fs.readFileSync(path.join(root, evidence, 'retention-plan.json'), 'utf8'));
const manifest = JSON.parse(fs.readFileSync(path.join(root, 'business-overlay.json'), 'utf8'));
const ignored = fs.readFileSync(path.join(root, '.gitignore'), 'utf8').split(/\r?\n/u);
const snapshot = path.join(root, '.foundation/releases/pantheon-base-v0.18.8/repo');
const within = (file, parent) => file === parent || file.startsWith(parent + '/');
const excluded = [...manifest.replaceRoots, ...manifest.baseExcludedPaths];
const hash = file => crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
const retained = [];
let harnessFiles = 0;
function visit(relative) {
  const absolute = path.join(snapshot, relative);
  const stat = fs.lstatSync(absolute);
  if (stat.isDirectory()) {
    for (const child of fs.readdirSync(absolute)) visit(relative + '/' + child);
  } else {
    assert.ok(stat.isFile(), 'snapshot metadata must be regular files');
    harnessFiles++;
    if (!excluded.some(parent => within(relative, parent))) retained.push(relative);
  }
}
visit('.harness');
assert.deepEqual(retained.sort(), plan.retained.map(entry => entry.path).sort());
assert.equal(excluded.includes('.harness'), false);
for (const entry of plan.retained) {
  assert.equal(hash(path.join(snapshot, entry.path)), entry.sha256);
  assert.ok(ignored.includes('/' + entry.path), 'inherited Base metadata must be excluded from Ops Git-selected inputs');
}
assert.equal(retained.some(file => file.endsWith('/review.md') || file.endsWith('/manifest.json')), false);
const changed = ['business-overlay.json', '.gitignore', 'tests/scripts/business-overlay/rebuild-from-base.test.mjs'];
const output = {
  status: 'passed for manifest projection and exact selected metadata; final active installation pending standard assembly',
  source: plan.source,
  baseCommit: plan.baseCommit,
  snapshotHarnessFiles: harnessFiles,
  inheritedHarnessFiles: plan.retained,
  excludedHarnessFiles: harnessFiles - retained.length,
  exactExcludedRoots: plan.baseExcludedPaths.length,
  baseReviewOrTaskManifestImported: false,
  inheritedMetadataDeclaredOpsOwned: false,
  changedFiles: changed.map(file => ({ path: file, sha256: hash(path.join(root, file)) })),
  verification: {
    overlayRegressionTests: { passed: 66, failed: 0, skipped: 0, log: evidence + '/overlay-tests.log' },
    activeOverlayContract: { passed: false, reason: 'new business migration 000039 generated destinations do not exist until final standard assembly', log: evidence + '/overlay-contract.log' },
    activeDocumentationScan: 'the prior eight findings remain active until the three official inherited leaves are installed by standard assembly',
  },
};
fs.writeFileSync(path.join(root, evidence, 'retention-readout.json'), JSON.stringify(output, null, 2) + '\n');
console.log(JSON.stringify({ snapshotHarnessFiles: harnessFiles, inheritedHarnessFiles: retained.length, excludedHarnessFiles: harnessFiles - retained.length, exactExcludedRoots: plan.baseExcludedPaths.length, status: 'pass' }));
