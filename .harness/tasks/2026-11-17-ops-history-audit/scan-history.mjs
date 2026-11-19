import fs from 'node:fs';
import path from 'node:path';
import { createHash } from 'node:crypto';
import { evaluateTaskLifecycleState } from '../../../scripts/harness/task-evidence-state.mjs';
import { resolveArchiveAwarePath } from '../../../scripts/task-manifest.mjs';

const root = process.cwd();
const evidenceDir = '.harness/evidence/2026-11-17-ops-history-audit';
const inventoryPath = '.harness/tasks/2026-11-16-ops-historical-closeout/INVENTORY.json';
const inventory = JSON.parse(fs.readFileSync(inventoryPath, 'utf8'));
const records = inventory.records.map((record) => {
  const state = evaluateTaskLifecycleState(root, record.id);
  const artifactNames = ['summary.md', 'review.md', 'commands.json'];
  const artifacts = artifactNames.map((name) => {
    const canonical = `.harness/evidence/${record.id}/${name}`;
    const resolved = resolveArchiveAwarePath(root, canonical);
    if (!resolved || !fs.existsSync(resolved)) return { name, canonical, exists: false };
    const bytes = fs.readFileSync(resolved);
    return { name, canonical, path: path.relative(root, resolved).replaceAll('\\', '/'), exists: true,
      sha256: createHash('sha256').update(bytes).digest('hex') };
  });
  return { ...record, priorDisposition: record.disposition, lifecycle: state, artifacts };
}).sort((a, b) => a.id.localeCompare(b.id));
const report = { taskId: '2026-11-17-ops-history-audit', scannedAt: new Date().toISOString(),
  baseline: inventoryPath, note: 'Mechanical inspection only. Prior classification is retained as a claim pending current acceptance.',
  records, counts: Object.fromEntries([...new Set(records.map((r) => r.priorDisposition))]
    .map((key) => [key, records.filter((r) => r.priorDisposition === key).length])) };
fs.mkdirSync(evidenceDir, { recursive: true });
fs.writeFileSync(`${evidenceDir}/history-scan.json`, JSON.stringify(report, null, 2) + '\n');
console.log(JSON.stringify({ records: records.length, counts: report.counts,
  completedLive: records.filter((r) => r.lifecycle.manifestLocation === 'live' && r.lifecycle.manifestStatus === 'completed').map((r) => r.id),
  missingSummary: records.filter((r) => r.idType === 'historical-task' && !r.lifecycle.hasSummary).map((r) => r.id) }, null, 2));
