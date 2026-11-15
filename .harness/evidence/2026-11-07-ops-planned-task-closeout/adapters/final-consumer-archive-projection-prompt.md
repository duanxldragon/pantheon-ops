Read-only CODE review; use no tools. Supplied source only. Prove P1/P2 defects and minimal fix, otherwise APPROVE this bounded scope. Max 220 words; state omitted scope.

FILE pantheon-base\config\foundation-consumer-artifacts.json
{
"schemaVersion": 1,
"sourceOwner": "pantheon-base",
"reviewLegacyRegistry": {
"path": "config/review-loop-legacy-artifacts.json",
"artifactRoot": ".harness/archive"
}
}

FILE Ops rebuild-from-base.mjs lines551-620

function writeReport(targetRoot, baseCommit, copiedBusiness, copiedRepository, rewritten, generated) {
const owned = [...new Set([...copiedBusiness, ...copiedRepository, ...generated])].sort();
const report = {
schemaVersion: 1,
baseCommit,
ownedFiles: owned.map((file) => ({ path: file, sha256: fileHash(path.join(targetRoot, file)) })),
rewrittenGoImports: rewritten,
generatedFiles: generated.sort(),
};
const output = path.join(targetRoot, '.business-overlay-report.json');
fs.writeFileSync(output, `${JSON.stringify(report, null, 2)}\n`, 'utf8');
return report;
}

// Base publishes this packaging contract. Only excluded Base archive entries
// can leave the consumer allowlist; current consumer reviews are always checked.
export function projectConsumerReviewRegistry(baseRoot, targetRoot, manifest, baseCommit) {
const descriptorPath = path.join(baseRoot, 'config/foundation-consumer-artifacts.json');
if (!fs.existsSync(descriptorPath)) return [];
const descriptor = JSON.parse(fs.readFileSync(descriptorPath, 'utf8'));
const contract = descriptor.reviewLegacyRegistry;
if (descriptor.schemaVersion !== 1 || descriptor.sourceOwner !== 'pantheon-base' ||
contract?.path !== 'config/review-loop-legacy-artifacts.json' ||
contract?.artifactRoot !== '.harness/archive') {
throw new Error('Invalid Base consumer review packaging contract');
}
const sourcePath = path.join(baseRoot, contract.path);
const sourceBytes = fs.readFileSync(sourcePath);
const source = JSON.parse(sourceBytes.toString('utf8'));
if (!Array.isArray(source.entries)) throw new Error('Base legacy review entries must be an array');
const excluded = [...(manifest.replaceRoots ?? []), ...(manifest.baseExcludedPaths ?? [])].map(repoPath);
const retained = [];
const external = [];
for (const entry of source.entries) {
if (typeof entry !== 'string' || !entry.startsWith('.harness/archive/') ||
entry.includes('..') || entry.includes('\\') || !entry.endsWith('/review.md')) {
throw new Error('Base legacy entry must be a canonical archive review');
}
if (!fs.existsSync(path.join(baseRoot, entry))) throw new Error(`Stale Base legacy entry: ${entry}`);
if (excluded.some((root) => isWithin(entry, root))) external.push(entry);
else {
if (!fs.existsSync(path.join(targetRoot, entry))) throw new Error(`Missing retained Base review: ${entry}`);
retained.push(entry);
}
}
if (external.length === 0) return [];
const targetPath = path.join(targetRoot, contract.path);
const projection = {
...source,
description: 'Consumer projection of the Base archive registry. Excluded Base artifacts stay recorded below; all current consumer reviews remain subject to strict review validation.',
entries: retained,
projection: { sourceOwner: descriptor.sourceOwner, baseCommit,
sourceConfigSha256: crypto.createHash('sha256').update(sourceBytes).digest('hex'),
externalBaseArtifacts: external },
};
fs.mkdirSync(path.dirname(targetPath), { recursive: true });
fs.writeFileSync(targetPath, `${JSON.stringify(projection, null, 2)}\n`, 'utf8');
return [contract.path];
}

export function assembleConsumerTree({ opsRoot = defaultOpsRoot, snapshot, targetRoot, manifestPath } = {}) {
const resolvedManifest = manifestPath ?? path.join(opsRoot, 'business-overlay.json');
const manifest = JSON.parse(fs.readFileSync(resolvedManifest, 'utf8'));
assertSafeManifest(manifest);

const resolvedBase = snapshot?.baseRoot;
if (!resolvedBase || !snapshot.baseCommit) throw new Error('Verified release snapshot is required');
const resolvedTarget = path.resolve(opsRoot, targetRoot ?? '.tmp/business-overlay-rebuild');
const targetContainsSource = [path.resolve(opsRoot), path.resolve(resolvedBase)].some((source) => source === resolvedTarget || source.startsWith(resolvedTarget + path.sep));
FILE pantheon-ops\tests\scripts\business-overlay\consumer-review-projection.test.mjs
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { projectConsumerReviewRegistry } from '../../../scripts/business-overlay/rebuild-from-base.mjs';
const checker = path.resolve('scripts/harness/check-review.mjs');
function fixture(t) {
const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ops-review-projection-'));
t.after(() => fs.rmSync(root, {recursive:true,force:true}));
const base = path.join(root,'base'); const consumer = path.join(root,'consumer');
const entry = '.harness/archive/2026-09/evidence/old-base/review.md';
const write = (dir,file,value) => { const p=path.join(dir,file);fs.mkdirSync(path.dirname(p),{recursive:true});fs.writeFileSync(p,typeof value==='string'?value:JSON.stringify(value)); };
write(base,'config/foundation-consumer-artifacts.json',{schemaVersion:1,sourceOwner:'pantheon-base',reviewLegacyRegistry:{path:'config/review-loop-legacy-artifacts.json',artifactRoot:'.harness/archive'}});
write(base,'config/review-loop-legacy-artifacts.json',{schemaVersion:1,entries:[entry]});
write(base,entry,'historical review that cannot be restated');
fs.mkdirSync(consumer,{recursive:true});return {base,consumer,entry,write};
}
test('excluded Base history is recorded, while an invalid active consumer review still fails',t=>{
const {base,consumer,entry,write}=fixture(t);
const before=fs.readFileSync(path.join(base,'config/review-loop-legacy-artifacts.json'),'utf8');
assert.deepEqual(projectConsumerReviewRegistry(base,consumer,{replaceRoots:['.harness']},'abc'),['config/review-loop-legacy-artifacts.json']);
const config=JSON.parse(fs.readFileSync(path.join(consumer,'config/review-loop-legacy-artifacts.json')));
assert.deepEqual(config.entries,[]);assert.deepEqual(config.projection.externalBaseArtifacts,[entry]);
assert.equal(fs.readFileSync(path.join(base,'config/review-loop-legacy-artifacts.json'),'utf8'),before);
write(consumer,'.harness/evidence/current/review.md','current invalid review');
const result=spawnSync(process.execPath,[checker,'--root',consumer,'--strict'],{encoding:'utf8'});
assert.equal(result.status,1);assert.match(result.stdout,/missing ## Machine Readable/);
assert.doesNotMatch(result.stdout,/stale legacy/);
});
test('retained archives remain listed and missing retained artifacts fail',t=>{
const {base,consumer,entry,write}=fixture(t);
assert.throws(()=>projectConsumerReviewRegistry(base,consumer,{},'abc'),/Missing retained/);
write(consumer,entry,'retained history');assert.deepEqual(projectConsumerReviewRegistry(base,consumer,{},'abc'),[]);
fs.rmSync(path.join(base,entry));assert.throws(()=>projectConsumerReviewRegistry(base,consumer,{replaceRoots:['.harness']},'abc'),/Stale Base/);
});
test('current or traversal entries cannot be projected away',t=>{
const {base,consumer,write}=fixture(t);
for(const entry of ['.harness/evidence/current/review.md','.harness/archive/../evidence/current/review.md']) {
write(base,'config/review-loop-legacy-artifacts.json',{entries:[entry]});
assert.throws(()=>projectConsumerReviewRegistry(base,consumer,{replaceRoots:['.harness']},'abc'),/canonical archive/);
}
});
