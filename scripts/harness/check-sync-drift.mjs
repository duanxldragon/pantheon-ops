#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { resolvePantheonHarnessRoot } from './upstream-root.mjs';

const DEFAULT_ROOT = process.cwd();
const DEFAULT_CONFIG = 'config/method.config.json';

function parseArgs(argv) {
  const options = { json: false, strict: false, help: false, root: DEFAULT_ROOT, config: DEFAULT_CONFIG };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--json') options.json = true;
    else if (arg === '--strict') options.strict = true;
    else if (arg === '--help' || arg === '-h') options.help = true;
    else if (arg === '--root') {
      const value = argv[index + 1];
      if (!value) throw new Error('--root requires a path');
      options.root = path.resolve(value);
      index += 1;
    } else if (arg === '--config') {
      const value = argv[index + 1];
      if (!value) throw new Error('--config requires a path');
      options.config = value;
      index += 1;
    } else throw new Error(`Unknown argument: ${arg}`);
  }
  return options;
}

function printHelp() {
  console.log('Usage: node scripts/harness/check-sync-drift.mjs [--json] [--strict] [--root <path>] [--config <path>]');
}

function normalizeContent(content) {
  return content.replace(/\r\n/g, '\n').trim();
}

function loadMirrorPairs(root, configPath) {
  const absolute = path.isAbsolute(configPath) ? configPath : path.join(root, configPath);
  if (!fs.existsSync(absolute)) return [];
  try {
    const config = JSON.parse(fs.readFileSync(absolute, 'utf8'));
    if (!Array.isArray(config.syncMirrors)) return [];
    return config.syncMirrors
      .filter((pair) => Array.isArray(pair) && pair.length === 2)
      .filter(([source, mirror]) => typeof source === 'string' && typeof mirror === 'string');
  } catch {
    return [];
  }
}

function resolveMirrorPath(root, methodRoot, repoPath) {
  const normalized = repoPath.replaceAll('\\', '/');
  for (const prefix of ['../pantheon-harness/', 'pantheon-harness/']) {
    if (normalized.startsWith(prefix)) {
      return path.join(methodRoot, normalized.slice(prefix.length));
    }
  }
  return path.isAbsolute(repoPath) ? repoPath : path.join(root, repoPath);
}

function scan(root, configPath) {
  const findings = [];
  const methodRoot = resolvePantheonHarnessRoot(root, { config: configPath });
  for (const [source, mirror] of loadMirrorPairs(root, configPath)) {
    const sourcePath = resolveMirrorPath(root, methodRoot, source);
    const mirrorPath = resolveMirrorPath(root, methodRoot, mirror);
    if (!fs.existsSync(sourcePath) || !fs.existsSync(mirrorPath)) {
      findings.push({
        file: fs.existsSync(sourcePath) ? mirror : source,
        reason: `configured sync mirror pair is incomplete: ${source} <-> ${mirror}`,
      });
      continue;
    }

    const sourceContent = normalizeContent(fs.readFileSync(sourcePath, 'utf8'));
    const mirrorContent = normalizeContent(fs.readFileSync(mirrorPath, 'utf8'));
    if (sourceContent !== mirrorContent) {
      findings.push({
        file: mirror,
        reason: `mirror drift detected against ${source}`,
      });
    }
  }
  return findings;
}

function main() {
  let options;
  try {
    options = parseArgs(process.argv.slice(2));
  } catch (error) {
    console.error(error.message);
    return 1;
  }
  if (options.help) return printHelp(), 0;
  const findings = scan(options.root, options.config);
  if (options.json) console.log(JSON.stringify({ mode: options.strict ? 'strict' : 'report-only', findingCount: findings.length, findings }, null, 2));
  else {
    console.log(`Sync drift check (${options.strict ? 'strict' : 'report-only'}): ${findings.length} finding(s)`);
    for (const finding of findings) console.log(`finding: ${finding.file}\n  reason: ${finding.reason}`);
  }
  return options.strict && findings.length > 0 ? 1 : 0;
}

process.exitCode = main();
