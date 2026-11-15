import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { assertSafeManifest, planBusinessMigrations, projectConsumerReviewRegistry, verifyBusinessDerivedOutputs } from '../../../../../scripts/business-overlay/rebuild-from-base.mjs';

const evidence = path.dirname(fileURLToPath(import.meta.url));
const root = process.cwd();
const sha = file => crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
const within = (file, entry) => file === entry || file.startsWith(entry + '/');
const read = file => JSON.parse(fs.readFileSync(path.join(root, file), 'utf8'));
const lock = read('foundation-release.lock.json');
const manifest = read('business-overlay.json');
const report = read('.business-overlay-report.json');
assertSafeManifest(manifest);
const releaseDir = path.join(root, lock.releaseArtifact.localPath);
const base = path.join(releaseDir, 'repo');
const latest = JSON.parse(fs.readFileSync(path.join(evidence, 'initial-0.txt'), 'utf8'));
const assetResults = latest.assets.map(asset => ({ name: asset.name, expected: asset.digest.slice(7), actual: sha(path.join(releaseDir, asset.name)) }));
const tarResult = spawnSync('tar', ['-xOf', path.join(releaseDir, lock.releaseArtifact.assetName), 'manifest.json'], { encoding: 'utf8', maxBuffer: 64 * 1024 * 1024 });
if (tarResult.status !== 0) throw new Error('Cannot extract verified official manifest');
const officialManifest = JSON.parse(tarResult.stdout);
const snapshot = { baseRoot: base, baseCommit: lock.baseCommit, manifest: officialManifest };
const canonical = verifyBusinessDerivedOutputs({ root, snapshot, manifest });
const projected = new Map();
projectConsumerReviewRegistry(base, root, manifest, lock.baseCommit, {
  ...fs,
  mkdirSync() {},
  writeFileSync(file, bytes) { projected.set(path.relative(root, file).replaceAll('\\', '/'), Buffer.from(bytes)); },
});
const projectionFailures = [...projected].filter(([file, bytes]) => !fs.existsSync(path.join(root, file)) || !bytes.equals(fs.readFileSync(path.join(root, file)))).map(([file]) => file);
const fixedGenerated = new Set([...canonical, ...planBusinessMigrations(manifest).map(m => m.target), ...projected.keys(), 'backend/go.mod', 'backend/go.sum', 'package.json', 'frontend/package.json']);
const generated = [...new Set(report.generatedFiles)];
const unexpectedGenerated = generated.filter(file => !fixedGenerated.has(file));
const exclusions = [...manifest.replaceRoots, ...manifest.baseExcludedPaths];
const declared = [...manifest.businessPaths, ...manifest.repositoryOverlayPaths];
const files = [];
function walk(directory, prefix = '') {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const file = prefix + entry.name;
    if (entry.isDirectory()) walk(path.join(directory, entry.name), file + '/');
    else if (entry.isFile()) files.push(file);
    else throw new Error('Nonregular official snapshot entry: ' + file);
  }
}
walk(base);
const categories = { inheritedIdentical: [], declaredOverlay: [], excludedBase: [], generatedAssembly: [], missingInherited: [], changedInherited: [] };
for (const file of files) {
  if (exclusions.some(entry => within(file, entry))) categories.excludedBase.push(file);
  else if (fixedGenerated.has(file)) categories.generatedAssembly.push(file);
  else if (declared.some(entry => within(file, entry))) categories.declaredOverlay.push(file);
  else if (!fs.existsSync(path.join(root, file))) categories.missingInherited.push(file);
  else if (sha(path.join(root, file)) !== sha(path.join(base, file))) categories.changedInherited.push(file);
  else categories.inheritedIdentical.push(file);
}
const ownedMissing = [];
const ownedChanged = [];
for (const item of report.ownedFiles) {
  if (!fs.existsSync(path.join(root, item.path))) ownedMissing.push(item.path);
  else if (sha(path.join(root, item.path)) !== item.sha256) ownedChanged.push(item.path);
}
const results = {
  checkedAt: new Date().toISOString(),
  claim: 'Official latest stable foundation consumption, independently verified against full extracted GitHub snapshot',
  releaseVersion: lock.releaseVersion,
  baseCommit: lock.baseCommit,
  latestIdentityMatches: latest.tag_name === lock.releaseVersion && latest.target_commitish === lock.baseCommit && !latest.draft && !latest.prerelease,
  manifestIdentityMatches: officialManifest.baseCommit === lock.baseCommit && officialManifest.releaseVersion === lock.releaseVersion && report.baseCommit === lock.baseCommit,
  assets: assetResults,
  reportSha256: sha(path.join(root, '.business-overlay-report.json')),
  manifestSha256: sha(path.join(root, 'business-overlay.json')),
  snapshotFileCount: files.length,
  counts: Object.fromEntries(Object.entries(categories).map(([key, value]) => [key, value.length])),
  categories,
  canonicalAssemblerOutputs: [...canonical].sort(),
  reviewProjectionOutputs: [...projected.keys()],
  reviewProjectionFailures: projectionFailures,
  unexpectedGenerated,
  packageAndGoDependencyLimit: 'Raw Base equality is not claimed for package.json, frontend/package.json, backend/go.mod and backend/go.sum; declared overlay dependency/script merge must pass final rebuild and readonly dependency/test loading.',
  reportBinding: { totalOwned: report.ownedFiles.length, missing: ownedMissing, changed: ownedChanged, generatedChanged: ownedChanged.filter(file => fixedGenerated.has(file)) },
};
results.foundationVerdict = results.latestIdentityMatches && results.manifestIdentityMatches && assetResults.every(asset => asset.actual === asset.expected) && !categories.missingInherited.length && !categories.changedInherited.length && !projectionFailures.length && !unexpectedGenerated.length ? 'PASS' : 'FAIL';
fs.writeFileSync(path.join(evidence, 'full-tree-readout.json'), JSON.stringify(results, null, 2) + '\n');
console.log(JSON.stringify({ foundationVerdict: results.foundationVerdict, latestIdentityMatches: results.latestIdentityMatches, manifestIdentityMatches: results.manifestIdentityMatches, assetChecks: assetResults.map(asset => ({ name: asset.name, passed: asset.actual === asset.expected })), snapshotFileCount: files.length, counts: results.counts, canonicalOutputs: canonical.size, reviewProjectionFailures: projectionFailures, unexpectedGenerated, reportBinding: { totalOwned: report.ownedFiles.length, missing: ownedMissing.length, changed: ownedChanged.length, generatedChanged: results.reportBinding.generatedChanged } }, null, 2));
process.exitCode = results.foundationVerdict === 'PASS' ? 0 : 1;
