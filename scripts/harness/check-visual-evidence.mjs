#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import {
  ensureTrailingSlash,
  listTaskManifestPaths,
  normalizeRepoRelativePath,
  readTaskManifest,
  resolveRepoPath,
} from '../task-manifest.mjs';

const DEFAULT_ROOT = process.cwd();

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-visual-evidence.mjs [--json] [--strict] [--root <path>]

Default behavior:
  Report visual evidence warnings and exit 0. Use --strict to exit 1 when warnings exist.

Examples:
  node scripts/harness/check-visual-evidence.mjs
  node scripts/harness/check-visual-evidence.mjs --json
  node scripts/harness/check-visual-evidence.mjs --strict
  node scripts/harness/check-visual-evidence.mjs --root /tmp/fixture --strict`);
}

function parseArgs(argv) {
  const options = { json: false, strict: false, help: false, root: DEFAULT_ROOT };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--json') {
      options.json = true;
    } else if (arg === '--strict') {
      options.strict = true;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else if (arg === '--root') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--root requires a path');
      }
      options.root = path.resolve(value);
      index += 1;
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }
  return options;
}

function readEvidence(manifestPayload, root) {
  const evidenceDirRepoPath = ensureTrailingSlash(
    normalizeRepoRelativePath(manifestPayload.linkage.evidenceDir),
  );
  const evidenceDir = resolveRepoPath(root, evidenceDirRepoPath);
  const commandsPath = resolveRepoPath(
    root,
    `${evidenceDirRepoPath}commands.json`,
  );
  const screenshotsDir = evidenceDir ? path.join(evidenceDir, 'screenshots') : null;
  const evidence = {
    dir: evidenceDir,
    exists: evidenceDir ? fs.existsSync(evidenceDir) : false,
    commandsPath,
    commandsExists: commandsPath ? fs.existsSync(commandsPath) : false,
    commandsError: null,
    hasScreenshots: false,
    hasBrowserEvidence: false,
    hasScreenshotGap: false,
    browserEvidence: [],
  };

  if (screenshotsDir && fs.existsSync(screenshotsDir)) {
    evidence.hasScreenshots = fs
      .readdirSync(screenshotsDir)
      .some((name) => /\.(png|jpe?g|webp)$/i.test(name));
  }

  if (commandsPath && fs.existsSync(commandsPath)) {
    try {
      const data = JSON.parse(fs.readFileSync(commandsPath, 'utf8'));
      evidence.browserEvidence = Array.isArray(data.browserEvidence)
        ? data.browserEvidence.filter((entry) => entry && typeof entry === 'object')
        : [];
      evidence.hasBrowserEvidence = evidence.browserEvidence.length > 0;
      evidence.hasScreenshotGap =
        Array.isArray(data.knownGaps) &&
        data.knownGaps.some((gap) => /screenshot|visual|browser|viewport/i.test(String(gap)));
      evidence.hasScreenshotGap ||=
        evidence.browserEvidence.some(
          (entry) => typeof entry.visualGap === 'string' && entry.visualGap.trim() !== '',
        );
    } catch (error) {
      evidence.commandsError = error.message;
    }
  }

  return evidence;
}

function getVisualPlan(manifestPayload) {
  const plan = manifestPayload.verificationPlan?.visualEvidence;
  if (!plan || typeof plan !== 'object' || Array.isArray(plan)) {
    return null;
  }
  return {
    viewports: Array.isArray(plan.viewports) ? plan.viewports.map((entry) => String(entry).trim()).filter(Boolean) : [],
    states: Array.isArray(plan.states) ? plan.states.map((entry) => String(entry).trim()).filter(Boolean) : [],
    routes: Array.isArray(plan.routes) ? plan.routes.map((entry) => String(entry).trim()).filter(Boolean) : [],
  };
}

function normalizeRouteToken(value) {
  const raw = String(value ?? '').trim();
  if (!raw) {
    return '';
  }

  let candidate = raw;
  try {
    if (raw.includes('://')) {
      candidate = new URL(raw).pathname || raw;
    } else if (raw.startsWith('/')) {
      candidate = new URL(raw, 'https://example.invalid').pathname || raw;
    }
  } catch {
    candidate = raw;
  }

  candidate = candidate.replace(/[?#].*$/, '').replace(/\/+$/, '');
  return (candidate || '/').toLowerCase();
}

function normalizeEvidenceTokens(entries, field) {
  const values = new Set();
  for (const entry of entries) {
    if (!entry || typeof entry !== 'object') {
      continue;
    }
    if (field === 'checkedStates') {
      if (!Array.isArray(entry.checkedStates)) {
        continue;
      }
      for (const state of entry.checkedStates) {
        const normalized = String(state).trim().toLowerCase();
        if (normalized) {
          values.add(normalized);
        }
      }
      continue;
    }
    const normalized =
      field === 'url'
        ? normalizeRouteToken(entry[field])
        : String(entry[field] ?? '').trim().toLowerCase();
    if (normalized) {
      values.add(normalized);
    }
  }
  return values;
}

function scanTasks(root) {
  const warnings = [];
  const uiTasks = [];

  for (const manifestPath of listTaskManifestPaths(root)) {
    let manifest;
    try {
      manifest = readTaskManifest(root, manifestPath);
    } catch (error) {
      warnings.push({
        file: manifestPath,
        taskId: normalizeRepoRelativePath(manifestPath),
        reason: `task manifest is unreadable: ${error.message}`,
      });
      continue;
    }

    const visualPlan = getVisualPlan(manifest.payload);
    if (!visualPlan) {
      continue;
    }

    const taskId = manifest.payload.taskId;
    uiTasks.push(taskId);

    const evidence = readEvidence(manifest.payload, root);
    if (!evidence.exists) {
      warnings.push({
        file: manifest.path,
        taskId,
        reason: 'UI task has no matching .harness/evidence directory',
      });
      continue;
    }

    if (evidence.commandsError) {
      warnings.push({
        file: normalizeRepoRelativePath(manifest.payload.linkage.evidenceDir),
        taskId,
        reason: `UI task evidence commands.json is unreadable: ${evidence.commandsError}`,
      });
      continue;
    }

    if (!evidence.hasScreenshots && !evidence.hasBrowserEvidence && !evidence.hasScreenshotGap) {
      warnings.push({
        file: normalizeRepoRelativePath(manifest.payload.linkage.evidenceDir),
        taskId,
        reason: 'UI task evidence has no screenshots/browser evidence and no recorded visual evidence gap',
      });
      continue;
    }

    if (!evidence.hasBrowserEvidence) {
      warnings.push({
        file: normalizeRepoRelativePath(manifest.payload.linkage.evidenceDir),
        taskId,
        reason: 'UI task evidence is missing browserEvidence entries for the manifest visual plan',
      });
      continue;
    }

    const coveredViewports = normalizeEvidenceTokens(evidence.browserEvidence, 'viewport');
    const coveredStates = normalizeEvidenceTokens(evidence.browserEvidence, 'checkedStates');
    const coveredRoutes = normalizeEvidenceTokens(evidence.browserEvidence, 'url');
    const plannedRoutes = visualPlan.routes
      .map((route) => normalizeRouteToken(route))
      .filter(Boolean);

    for (const viewport of visualPlan.viewports) {
      if (!coveredViewports.has(viewport.toLowerCase())) {
        warnings.push({
          file: normalizeRepoRelativePath(manifest.payload.linkage.evidenceDir),
          taskId,
          reason: `UI task evidence is missing browserEvidence coverage for viewport "${viewport}"`,
        });
      }
    }

    for (const state of visualPlan.states) {
      if (!coveredStates.has(state.toLowerCase())) {
        warnings.push({
          file: normalizeRepoRelativePath(manifest.payload.linkage.evidenceDir),
          taskId,
          reason: `UI task evidence is missing browserEvidence coverage for state "${state}"`,
        });
      }
    }

    for (const route of plannedRoutes) {
      if (!coveredRoutes.has(route)) {
        warnings.push({
          file: normalizeRepoRelativePath(manifest.payload.linkage.evidenceDir),
          taskId,
          reason: `UI task evidence is missing browserEvidence coverage for route "${route}"`,
        });
      }
    }
  }

  return { uiTasks, warnings };
}

function printTextReport(result, strict) {
  const mode = strict ? 'strict' : 'report-only';
  console.log(
    `Visual evidence check (${mode}): ${result.uiTasks.length} UI task(s), ${result.warnings.length} warning(s)`,
  );
  if (result.warnings.length === 0) {
    console.log('\nno findings');
  }
  for (const warning of result.warnings) {
    console.log(`\nwarning: ${warning.file}`);
    console.log(`  task: ${warning.taskId}`);
    console.log(`  reason: ${warning.reason}`);
  }
}

function main() {
  let options;
  try {
    options = parseArgs(process.argv.slice(2));
  } catch (error) {
    console.error(error.message);
    return 1;
  }
  if (options.help) {
    printHelp();
    return 0;
  }

  const result = scanTasks(options.root);
  if (options.json) {
    console.log(
      JSON.stringify(
        {
          mode: options.strict ? 'strict' : 'report-only',
          uiTaskCount: result.uiTasks.length,
          warningCount: result.warnings.length,
          uiTasks: result.uiTasks,
          warnings: result.warnings,
        },
        null,
        2,
      ),
    );
  } else {
    printTextReport(result, options.strict);
  }

  return options.strict && result.warnings.length > 0 ? 1 : 0;
}

process.exitCode = main();
