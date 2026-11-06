#!/usr/bin/env node

/**
 * Checker wiring gate (GOV-03, the FR-026 recurrence guard).
 *
 * A governance checker that nothing invokes is not coverage: it looks like
 * assurance while the class it was written for accumulates silently. This
 * script asks the only question that matters for every checker file — does
 * anything actually run this file?
 *
 * A checker counts as wired when its filename appears in a `package.json`
 * script or in a workflow under `.github/workflows/`. Anything else must be
 * listed in EXCEPTIONS with a reason, mirroring the wiring table in
 * scripts/harness/README.md.
 *
 * Usage:
 *   node scripts/harness/check-checker-wiring.mjs [--root <path>] [--json]
 */

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const CHECKER_DIRECTORIES = ['scripts', path.join('scripts', 'harness')];
const CHECKER_PATTERN = /^check-.*\.mjs$/;
const WORKFLOW_DIRECTORY = path.join('.github', 'workflows');
const WORKFLOW_PATTERN = /\.ya?ml$/;

// Explicit, reviewable exceptions. Each entry must state why the checker is
// deliberately unreferenced; "not wired yet" is never a valid reason.
const EXCEPTIONS = new Map([
  [
    'check-doc-frontmatter.mjs',
    'mirror-only: scripts/frontmatter-check.mjs is the single implementation run as check:docs-frontmatter (2026-10-31 W5 decision)',
  ],
]);

function parseArgs(argv) {
  const options = { root: process.cwd(), json: false };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--json') {
      options.json = true;
    } else if (arg === '--root') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--root requires a path');
      }
      options.root = path.resolve(value);
      index += 1;
    } else {
      throw new Error(`unknown argument: ${arg}`);
    }
  }

  return options;
}

function listFiles(directory) {
  if (!fs.existsSync(directory)) {
    return [];
  }
  return fs
    .readdirSync(directory, { withFileTypes: true })
    .filter((entry) => entry.isFile())
    .map((entry) => entry.name)
    .sort((left, right) => left.localeCompare(right));
}

function collectCheckers(root) {
  const checkers = [];

  for (const relativeDirectory of CHECKER_DIRECTORIES) {
    const absoluteDirectory = path.join(root, relativeDirectory);
    for (const name of listFiles(absoluteDirectory)) {
      if (!CHECKER_PATTERN.test(name)) {
        continue;
      }
      checkers.push({
        name,
        file: path.posix.join(relativeDirectory.split(path.sep).join('/'), name),
      });
    }
  }

  return checkers;
}

/**
 * readWiring returns the text that can reference a checker: the root
 * package.json scripts and every workflow file. Text search is deliberate —
 * the question is "is this filename invoked anywhere", and a parsed model would
 * still miss a nested `run: |` block that calls it by path.
 */
function readWiring(root) {
  const sources = [];

  const packageJsonPath = path.join(root, 'package.json');
  if (fs.existsSync(packageJsonPath)) {
    sources.push({ file: 'package.json', text: fs.readFileSync(packageJsonPath, 'utf8') });
  }

  const workflowDirectory = path.join(root, WORKFLOW_DIRECTORY);
  for (const name of listFiles(workflowDirectory)) {
    if (!WORKFLOW_PATTERN.test(name)) {
      continue;
    }
    sources.push({
      file: path.posix.join('.github/workflows', name),
      text: fs.readFileSync(path.join(workflowDirectory, name), 'utf8'),
    });
  }

  return sources;
}

function evaluate(root) {
  const sources = readWiring(root);
  const checkers = collectCheckers(root);

  const results = checkers.map((checker) => {
    const wiredIn = sources
      .filter((source) => source.text.includes(checker.name))
      .map((source) => source.file);

    if (wiredIn.length > 0) {
      return { ...checker, status: 'wired', wiredIn };
    }
    if (EXCEPTIONS.has(checker.name)) {
      return { ...checker, status: 'exception', reason: EXCEPTIONS.get(checker.name) };
    }
    return { ...checker, status: 'unwired' };
  });

  return results;
}

function main(argv) {
  const options = parseArgs(argv);
  const results = evaluate(options.root);
  const unwired = results.filter((result) => result.status === 'unwired');
  const wired = results.filter((result) => result.status === 'wired');
  const exceptions = results.filter((result) => result.status === 'exception');

  if (options.json) {
    process.stdout.write(`${JSON.stringify({ total: results.length, wired: wired.length, exceptions: exceptions.length, unwired, results }, null, 2)}\n`);
  } else {
    for (const result of results) {
      if (result.status === 'wired') {
        console.log(`wired      ${result.file} <- ${result.wiredIn.join(', ')}`);
      } else if (result.status === 'exception') {
        console.log(`exception  ${result.file} (${result.reason})`);
      } else {
        console.error(`NOT WIRED  ${result.file} — no package.json script and no .github/workflows reference`);
      }
    }
    console.log(
      `checker-wiring: ${results.length} checkers / ${wired.length} wired / ${exceptions.length} exception / ${unwired.length} unwired`,
    );
  }

  if (unwired.length > 0) {
    console.error(
      '\nA checker must land in the same diff as its wiring: an npm script plus a workflow step, ' +
        'or an explicit exception in check-checker-wiring.mjs and the wiring table in scripts/harness/README.md.',
    );
    return 1;
  }
  return 0;
}

if (process.argv[1] && path.resolve(process.argv[1]) === path.resolve(fileURLToPath(import.meta.url))) {
  try {
    process.exitCode = main(process.argv.slice(2));
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 2;
  }
}

export { evaluate };
