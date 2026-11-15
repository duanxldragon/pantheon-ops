import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';

const root = process.cwd();
const snapshot = path.join(root, '.foundation/releases/pantheon-base-v0.18.8/repo');
const retained = [
  '.harness/tasks/2026-11-07-ops-formal-queue-foundation-gaps/o4-tenant-implementation-plan.md',
  '.harness/evidence/2026-11-13-v0188-release-preparation/summary.md',
  '.harness/evidence/2026-11-13-v0188-release-preparation/smoke-empty-readiness/summary.md',
].sort();
const excluded = [];
const within = (file, parent) => file === parent || file.startsWith(parent + '/');
function visit(relative) {
  if (!retained.some(file => within(file, relative))) {
    excluded.push(relative);
    return;
  }
  const absolute = path.join(snapshot, relative);
  if (fs.lstatSync(absolute).isDirectory()) {
    for (const entry of fs.readdirSync(absolute).sort()) visit(relative + '/' + entry);
  }
}
visit('.harness');
for (const file of retained) {
  if (!fs.lstatSync(path.join(snapshot, file)).isFile()) throw new Error('Missing retained leaf: ' + file);
}
const output = {
  source: 'verified official pantheon-base-v0.18.8 repo.tar extraction',
  baseCommit: '34428814a9efc997fd93d4434406a53df6460323',
  retained: retained.map(file => ({ path: file, sha256: crypto.createHash('sha256').update(fs.readFileSync(path.join(snapshot, file))).digest('hex') })),
  baseExcludedPaths: excluded,
};
const outputPath = path.join(root, '.harness/evidence/2026-11-14-ops-latest-foundation-and-plans/verification/current-d5/retention-plan.json');
fs.writeFileSync(outputPath, JSON.stringify(output, null, 2) + '\n');
console.log(JSON.stringify({ retainedLeaves: retained.length, excludedRoots: excluded.length, outputPath }));
