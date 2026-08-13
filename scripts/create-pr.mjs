#!/usr/bin/env node
/**
 * Create or update a pull request only after its body passes local governance
 * validation. Branch, commit, and push operations remain explicit git steps.
 *
 * Usage:
 *   node scripts/create-pr.mjs --title "PR title" --body-file .harness/evidence/<task-id>/pr-body.md
 *   node scripts/create-pr.mjs --pr 123 --body-file .harness/evidence/<task-id>/pr-body.md
 */

import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { parseArgs } from 'node:util';
import { fileURLToPath } from 'node:url';
import { validatePrBody } from './check-pr-governance.mjs';

const root = process.cwd();

function printUsage() {
  console.log(`Usage:
  node scripts/create-pr.mjs --title "PR title" --body-file <path> [--base main] [--draft]
  node scripts/create-pr.mjs --pr <number> --body-file <path>

The body file must pass scripts/check-pr-governance.mjs before GitHub is called.`);
}

function requireNonEmptyString(value, label) {
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error(`${label} is required`);
  }
  return value.trim();
}

export function resolveBodyFile(rootDir, bodyFile) {
  const requestedPath = requireNonEmptyString(bodyFile, '--body-file');
  const absolutePath = path.resolve(rootDir, requestedPath);
  const relativePath = path.relative(rootDir, absolutePath);
  if (relativePath.startsWith('..') || path.isAbsolute(relativePath)) {
    throw new Error('--body-file must stay inside the repository');
  }
  if (!fs.existsSync(absolutePath)) {
    throw new Error(`PR body file does not exist: ${relativePath}`);
  }
  return absolutePath;
}

export function validateBodyFile(rootDir, bodyFile) {
  const absolutePath = resolveBodyFile(rootDir, bodyFile);
  const body = fs.readFileSync(absolutePath, 'utf8');
  const findings = validatePrBody(body, { rootDir });
  if (findings.length > 0) {
    throw new Error(`PR body governance validation failed:\n- ${findings.join('\n- ')}`);
  }
  return absolutePath;
}

export function buildGhArguments({ title, bodyFile, base, draft, prNumber }) {
  if (prNumber !== undefined) {
    const normalizedPrNumber = String(prNumber);
    if (!/^[1-9]\d*$/.test(normalizedPrNumber)) {
      throw new Error('--pr must be a positive pull request number');
    }
    return ['pr', 'edit', normalizedPrNumber, '--body-file', bodyFile];
  }

  const normalizedTitle = requireNonEmptyString(title, '--title');
  const normalizedBase = requireNonEmptyString(base, '--base');
  const args = ['pr', 'create', '--title', normalizedTitle, '--body-file', bodyFile, '--base', normalizedBase];
  if (draft) {
    args.push('--draft');
  }
  return args;
}

export function main(argv) {
  const { values } = parseArgs({
    args: argv,
    options: {
      title: { type: 'string', short: 't' },
      'body-file': { type: 'string', short: 'f' },
      base: { type: 'string', default: 'main' },
      pr: { type: 'string' },
      draft: { type: 'boolean', default: false },
      help: { type: 'boolean', short: 'h', default: false },
    },
    strict: true,
  });

  if (values.help) {
    printUsage();
    return;
  }

  const bodyFile = validateBodyFile(root, values['body-file']);
  const ghArgs = buildGhArguments({
    title: values.title,
    bodyFile,
    base: values.base,
    draft: values.draft,
    prNumber: values.pr,
  });
  execFileSync('gh', ghArgs, { stdio: 'inherit', shell: false });
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  try {
    main(process.argv.slice(2));
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 1;
  }
}
