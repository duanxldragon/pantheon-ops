import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { fileURLToPath, pathToFileURL } from 'node:url';
const consumer = await import(pathToFileURL(path.resolve('scripts/business-overlay/foundation-candidate.mjs')));
const run = path.dirname(fileURLToPath(import.meta.url));
const ops = process.cwd();
const base = path.resolve('../pantheon-base');
const before = JSON.parse(fs.readFileSync(path.join(run, 'before.json')));
const save = (name, value) => fs.writeFileSync(path.join(run, name), JSON.stringify(value, null, 2) + '\n', { flag: 'wx' });
let patch = '';
const whitespace = [];
for (const relative of ['scripts/business-overlay/foundation-candidate.mjs', 'scripts/business-overlay/foundation-candidate.test.mjs']) {
  const old = before.sources.find(item => item.owner === 'Ops' && item.relative === relative).captured;
  const current = path.join(ops, relative);
  const delta = spawnSync('git', ['diff', '--no-index', '--', old, current], { encoding: 'utf8' });
  assert.equal(delta.status, 1, delta.stderr);
  patch += delta.stdout;
  const check = spawnSync('git', ['diff', '--no-index', '--check', '--', old, current], { encoding: 'utf8' });
  assert.ok(check.status === 0 || (check.status === 1 && !check.stdout && !check.stderr), check.stdout + check.stderr);
  whitespace.push({ args: ['git', 'diff', '--no-index', '--check', '--', old, current], exitCode: check.status, output: check.stdout + check.stderr });
}
fs.writeFileSync(path.join(run, 'metadata-bound-delta.patch'), patch, { flag: 'wx' });
const original = fs.readFileSync(before.sources.find(item => item.owner === 'Ops' && item.relative.endsWith('/foundation-candidate.mjs')).captured, 'utf8');
const current = fs.readFileSync(path.join(ops, 'scripts/business-overlay/foundation-candidate.mjs'), 'utf8');
const expected = original.replace('});\nexport const sha256', '});\n// Declared Ops metadata also includes retained official cache/evidence. Its\n// bounded capture is separate from the schema1 candidate archive inventory.\nexport const METADATA_LIMITS = Object.freeze({ files: 100000 });\nexport const sha256')
  .replace('if (files.size > LIMITS.files)\n        throw new Error("Candidate input count exceeds limit");',
    'if (files.size > METADATA_LIMITS.files)\n        throw new Error("Declared metadata input count exceeds limit");');
assert.equal(current, expected);
const block = source => source.slice(source.indexOf('// Curated PUBLIC review leaves'), source.indexOf('export function assertCandidatePath'));
const liveProtocol = fs.readFileSync(path.join(base, 'scripts/foundation-release/candidate-descriptor.mjs'), 'utf8');
assert.equal(block(current), block(original));
assert.equal(block(current), block(liveProtocol));
const tail = source => source.slice(source.indexOf('function hashStableInput'));
assert.equal(tail(current), tail(original));
const oldTests = fs.readFileSync(before.sources.find(item => item.owner === 'Ops' && item.relative.endsWith('/foundation-candidate.test.mjs')).captured, 'utf8');
const newTests = fs.readFileSync(path.join(ops, 'scripts/business-overlay/foundation-candidate.test.mjs'), 'utf8');
const start = 'test("captured retained archives stream within explicit metadata bounds"';
assert.equal(newTests.slice(newTests.indexOf(start)), oldTests.slice(oldTests.indexOf(start)));
const oldReceipt = JSON.parse(fs.readFileSync(path.resolve('.harness/evidence/2026-11-07-ops-planned-task-closeout/o4-real-packaging-1794279171056/implementation-receipt.json')));
const applied = oldReceipt.sources.filter(item => item.owner === 'Base').map(item => ({
  path: item.path, expectedSha256: item.implementationSha256,
  actualSha256: consumer.sha256(fs.readFileSync(path.join(base, item.path))),
}));
assert.ok(applied.every(item => item.expectedSha256 === item.actualSha256));
const sourceInventory = JSON.parse(fs.readFileSync(path.resolve('.harness/evidence/2026-11-07-ops-planned-task-closeout/o4-adoption-1794278076834/source-inventory.json')));
const request = { ...sourceInventory, root: base,
  sourcePaths: [...new Set([...sourceInventory.sourcePaths, ...consumer.PUBLIC_REVIEW_ARCHIVE_PATHS])].sort(),
  evidenceIds: ['2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/' + path.basename(run)],
  reviewIds: [] };
save('root-review-request.template.json', request);
save('delta-contract-checks.json', { timestamp: new Date().toISOString(), exactThreeReplacementDelta: true,
  publicContractUnchangedAndEqualToLiveBase: true, streamingHashAndRemainingSourceTailUnchanged: true,
  original74TestsUnchanged: true, sourcePaths: request.sourcePaths.length,
  requiredPublicLeaves: consumer.PUBLIC_REVIEW_ARCHIVE_PATHS.length,
  sourceInventoryOmissions: 0, reviewIdsIntentionallyEmptyUntilRootReview: true, applied, whitespace });
console.log(JSON.stringify({ exactDelta: true, appliedBaseMatchesProposal: true, originalTestsPreserved: true,
  publicContractUnchanged: true, originalSources: sourceInventory.sourcePaths.length, recommendedSources: request.sourcePaths.length }));
