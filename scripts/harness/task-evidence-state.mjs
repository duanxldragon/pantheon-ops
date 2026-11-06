#!/usr/bin/env node

// Predicate for "does this task packet's frontmatter status match reality?".
//
// The failure mode this exists to stop (2026-10-31 audit, AUD-006): task packet
// docs stayed `status: Active` months after their evidence was archived, so the
// index read as if 70 tasks were still in flight. Guessing completion from
// wording is what produced that; this module refuses to guess and reports the
// mechanical facts instead:
//
//   evidence dir present (live or archived)  -> `.harness/{evidence,archive/<YYYY-MM>/evidence}/<task-id>`
//   summary readout present                  -> `<evidence>/summary.md` with a result heading
//   manifest status                          -> `.harness/{tasks,archive/<YYYY-MM>/tasks}/<task-id>/manifest.json`
//
// A packet is only an archive candidate when evidence exists AND a summary
// exists AND either the manifest says `completed` or the summary carries a
// readout. Everything else stays `Active` with a reason code, so the counter
// cannot be lowered by relabelling an unfinished task.
//
// Wiring lives in 2026-10-31-audit-p2-ci-checker-wiring; this module only
// exposes the predicate so the checker and the paperwork share one definition.

import fs from 'node:fs';
import path from 'node:path';

import {
  buildTaskManifestPath,
  findArchivedReferencePath,
  findArchivedTaskManifestPath,
} from '../task-manifest.mjs';

const EVIDENCE_ROOT = '.harness/evidence';
const TASK_DOC_ROOT = 'docs/harness/tasks';
const TASK_DOC_SUFFIX = '.task.md';

// A completion readout is a result-shaped heading. Deliberately narrow: it must
// name a verification/result section, not merely contain the word "完成" in
// prose, because that phrase appears in planning text too.
const READOUT_HEADING =
  /^#{2,4}\s*(?:验证|验证读数|验证结果|结果|结论|完成读数|Verification|Verification\b.*|Result|Results|Outcome|Conclusion)\b/im;

function isRepoPath(pathValue) {
  return typeof pathValue === 'string' && pathValue !== '' && !pathValue.includes('..');
}

export function listTaskDocIds(rootDir) {
  const docRoot = path.join(rootDir, TASK_DOC_ROOT);
  if (!fs.existsSync(docRoot)) {
    return [];
  }
  return fs
    .readdirSync(docRoot)
    .filter((name) => name.endsWith(TASK_DOC_SUFFIX))
    .map((name) => name.slice(0, -TASK_DOC_SUFFIX.length))
    .sort((left, right) => left.localeCompare(right));
}

export function readTaskDocStatus(rootDir, taskId) {
  const docPath = path.join(rootDir, TASK_DOC_ROOT, `${taskId}${TASK_DOC_SUFFIX}`);
  if (!fs.existsSync(docPath)) {
    return null;
  }
  const source = fs.readFileSync(docPath, 'utf8');
  const match = source.match(/^---\r?\n([\s\S]*?)\r?\n---/u);
  if (!match) {
    return null;
  }
  const statusLine = match[1].match(/^status:\s*(.+)$/mu);
  return statusLine ? statusLine[1].trim() : null;
}

export function resolveTaskEvidenceDir(rootDir, taskId) {
  const live = `${EVIDENCE_ROOT}/${taskId}`;
  if (fs.existsSync(path.join(rootDir, live))) {
    return { location: 'live', dir: live };
  }
  const archived = findArchivedReferencePath(rootDir, live);
  if (archived) {
    return { location: 'archived', dir: archived };
  }
  return { location: 'missing', dir: null };
}

function readManifestStatus(absolutePath) {
  try {
    const payload = JSON.parse(fs.readFileSync(absolutePath, 'utf8'));
    return typeof payload.status === 'string' ? payload.status.trim() : null;
  } catch {
    return null;
  }
}

export function readTaskManifestState(rootDir, taskId) {
  const live = buildTaskManifestPath(taskId);
  if (fs.existsSync(path.join(rootDir, live))) {
    return { location: 'live', path: live, status: readManifestStatus(path.join(rootDir, live)) };
  }
  const archived = findArchivedTaskManifestPath(rootDir, taskId);
  if (archived) {
    return {
      location: 'archived',
      path: archived,
      status: readManifestStatus(path.join(rootDir, archived)),
    };
  }
  return { location: 'missing', path: null, status: null };
}

export function inspectTaskEvidence(rootDir, taskId) {
  const evidence = resolveTaskEvidenceDir(rootDir, taskId);
  if (!evidence.dir || !isRepoPath(evidence.dir)) {
    return { ...evidence, hasSummary: false, hasReadout: false };
  }
  const summaryAbsolute = path.join(rootDir, evidence.dir, 'summary.md');
  if (!fs.existsSync(summaryAbsolute)) {
    return { ...evidence, hasSummary: false, hasReadout: false };
  }
  const summary = fs.readFileSync(summaryAbsolute, 'utf8');
  return { ...evidence, hasSummary: true, hasReadout: READOUT_HEADING.test(summary) };
}

export function evaluateTaskLifecycleState(rootDir, taskId, options = {}) {
  const docStatus = options.docStatus ?? readTaskDocStatus(rootDir, taskId);
  const manifest = readTaskManifestState(rootDir, taskId);
  const evidence = inspectTaskEvidence(rootDir, taskId);
  const evidenceLoaded = evidence.location !== 'missing';

  let verdict = 'keep-active';
  let reason = 'no-evidence';

  if (docStatus === 'Archived') {
    verdict = 'already-archived';
    reason = 'doc-status-archived';
  } else if (!evidenceLoaded) {
    reason = 'no-evidence';
  } else if (!evidence.hasSummary) {
    reason = 'evidence-without-summary';
  } else if (manifest.status === 'completed' || evidence.hasReadout) {
    verdict = 'archive';
    reason = manifest.status === 'completed' ? 'manifest-completed' : 'summary-readout';
  } else {
    reason = 'evidence-without-completion-readout';
  }

  return {
    taskId,
    docStatus,
    evidenceLocation: evidence.location,
    evidenceDir: evidence.dir,
    hasSummary: evidence.hasSummary,
    hasReadout: evidence.hasReadout,
    manifestLocation: manifest.location,
    manifestStatus: manifest.status,
    verdict,
    reason,
  };
}

function formatTable(rows) {
  const header = 'taskId\tdoc\tmanifest\tevidence\tsummary\treadout\tverdict\treason';
  const lines = rows.map((row) =>
    [
      row.taskId,
      row.docStatus ?? '-',
      row.manifestStatus ?? '-',
      row.evidenceLocation,
      row.hasSummary ? 'yes' : 'no',
      row.hasReadout ? 'yes' : 'no',
      row.verdict,
      row.reason,
    ].join('\t'),
  );
  return [header, ...lines].join('\n');
}

function main(argv) {
  const rootIndex = argv.indexOf('--root');
  const rootDir = rootIndex === -1 ? process.cwd() : path.resolve(argv[rootIndex + 1]);
  const rows = listTaskDocIds(rootDir).map((taskId) => evaluateTaskLifecycleState(rootDir, taskId));
  if (argv.includes('--json')) {
    const counts = rows.reduce((accumulator, row) => {
      accumulator[row.verdict] = (accumulator[row.verdict] ?? 0) + 1;
      return accumulator;
    }, {});
    console.log(JSON.stringify({ total: rows.length, counts, rows }, null, 2));
    return 0;
  }
  console.log(formatTable(rows));
  const counts = rows.reduce((accumulator, row) => {
    accumulator[row.verdict] = (accumulator[row.verdict] ?? 0) + 1;
    return accumulator;
  }, {});
  console.log(`\nTotal: ${rows.length} ${JSON.stringify(counts)}`);
  return 0;
}

if (process.argv[1] && process.argv[1].endsWith('task-evidence-state.mjs')) {
  process.exitCode = main(process.argv.slice(2));
}
