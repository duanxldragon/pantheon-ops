#!/usr/bin/env node

/**
 * Mechanical gate for repository structure and naming conventions.
 * Contract source: docs/designs/REPOSITORY_LAYOUT.md §2 (placement rules)
 * plus the module-domain taxonomy in DESIGN.md §2.
 *
 * Scans git-TRACKED files only — local noise dirs (REPOSITORY_LAYOUT §3)
 * are invisible to it. All rules were at zero findings when the gate
 * landed: it is a ratchet against future misplacement, not a cleanup TODO.
 *
 * check-boundaries.mjs owns cross-layer IMPORT rules; this gate owns
 * WHERE files live and WHAT they are named. They are complementary.
 */
import path from 'node:path';
import process from 'node:process';
import { execFileSync } from 'node:child_process';

const DEFAULT_ROOT = process.cwd();

function parseArgs(argv) {
  const options = { json: false, strict: false, help: false, root: DEFAULT_ROOT };
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
    } else throw new Error(`Unknown argument: ${arg}`);
  }
  return options;
}

function printHelp() {
  console.log('Usage: node scripts/harness/check-structure-contract.mjs [--json] [--strict] [--root <path>]');
  console.log('Checks git-tracked file placement and naming against REPOSITORY_LAYOUT.md.');
}

// Whitelists are the union of pantheon-base and pantheon-ops reality so the
// same script can be vendored unchanged. Tighten per-repo only when a rule
// actually fires cross-repo noise.
const BACKEND_TOP_DIRS = new Set(['cmd', 'internal', 'modules', 'pkg', 'tests']);
const BACKEND_TOP_FILES = new Set(['go.mod', 'go.sum', 'start-dev.sh', 'start-dev.bat', 'DEV_DB_INIT_GUIDE.md', '.golangci.yml']);
const BACKEND_MODULE_DOMAINS = new Set(['auth', 'business', 'lowcode', 'platform', 'system']);
const FRONTEND_SRC_DIRS = new Set(['api', 'assets', 'components', 'core', 'hooks', 'i18n', 'modules', 'store']);
const FRONTEND_SRC_FILES = new Set(['App.tsx', 'main.tsx', 'index.css', 'vite-env.d.ts']);
const FRONTEND_MODULE_DOMAINS = new Set(['auth', 'business', 'generated', 'lowcode', 'platform', 'system']);
const FRONTEND_TOP_DIRS = new Set(['config', 'public', 'scripts', 'src', 'tests']);
const FRONTEND_TOP_FILE_PATTERNS = [
  /^\.(eslintrc\.json|gitignore|prettierignore|prettierrc)$/,
  /^(AGENTS|README)(\.[a-z]{2})?\.md$/,
  /^eslint\.config\.js$/,
  /^index\.html$/,
  /^package(-lock)?\.json$/,
  /^playwright(\.[a-z0-9-]+)?\.config\.ts$/,
  /^tsconfig(\.[a-z]+)?\.json$/,
  /^vite\.config\.ts$/,
  /^vitest\.config\.ts$/,
];
const BINARY_EXTENSIONS = new Set(['.exe', '.dll', '.so', '.dylib']);
const GO_FILE_NAME = /^[a-z0-9_]+\.go$/;

function listTrackedFiles(root) {
  return execFileSync('git', ['-C', root, 'ls-files', '-z'], { encoding: 'utf8', maxBuffer: 64 * 1024 * 1024 })
    .split('\0')
    .filter(Boolean);
}

function checkFile(file) {
  const segments = file.split('/');
  const base = path.posix.basename(file);
  const ext = path.posix.extname(file).toLowerCase();
  const issues = [];

  if (BINARY_EXTENSIONS.has(ext)) {
    issues.push({ rule: 'no-tracked-binaries', reason: 'compiled binaries must not be committed' });
  }

  if (ext === '.go') {
    if (segments[0] !== 'backend') {
      issues.push({ rule: 'go-placement', reason: 'Go source lives under backend/ only (REPOSITORY_LAYOUT §2.1)' });
    }
    if (!GO_FILE_NAME.test(base)) {
      issues.push({ rule: 'go-file-naming', reason: 'Go files use snake_case: ^[a-z0-9_]+\\.go$' });
    }
  }

  if (segments[0] === 'backend' && segments.length >= 2) {
    const second = segments[1];
    const isDirEntry = segments.length > 2;
    if (isDirEntry ? !BACKEND_TOP_DIRS.has(second) : !(BACKEND_TOP_DIRS.has(second) || BACKEND_TOP_FILES.has(second))) {
      issues.push({ rule: 'backend-root', reason: `backend/${second} is outside the backend layout whitelist (cmd/internal/modules/pkg/tests)` });
    }
    if (second === 'modules' && segments.length > 3 && !BACKEND_MODULE_DOMAINS.has(segments[2])) {
      issues.push({ rule: 'backend-module-domain', reason: `backend/modules/${segments[2]} is not a known domain (${[...BACKEND_MODULE_DOMAINS].join('/')})` });
    }
  }

  if (segments[0] === 'frontend' && segments.length >= 2) {
    const second = segments[1];
    if (segments.length === 2) {
      if (!FRONTEND_TOP_DIRS.has(second) && !FRONTEND_TOP_FILE_PATTERNS.some((p) => p.test(second))) {
        issues.push({ rule: 'frontend-root', reason: `frontend/${second} is outside the frontend layout whitelist` });
      }
    } else if (!FRONTEND_TOP_DIRS.has(second)) {
      issues.push({ rule: 'frontend-root', reason: `frontend/${second}/ is outside the frontend layout whitelist (config/public/scripts/src/tests)` });
    }

    if (second === 'src' && segments.length >= 3) {
      const third = segments[2];
      if (segments.length === 3) {
        if (!FRONTEND_SRC_DIRS.has(third) && !FRONTEND_SRC_FILES.has(third)) {
          issues.push({ rule: 'frontend-src-root', reason: `frontend/src/${third} is outside the src layout whitelist` });
        }
      } else if (!FRONTEND_SRC_DIRS.has(third)) {
        issues.push({ rule: 'frontend-src-root', reason: `frontend/src/${third}/ is outside the src layout whitelist` });
      }

      if (third === 'modules' && segments.length > 4 && !FRONTEND_MODULE_DOMAINS.has(segments[3])) {
        issues.push({ rule: 'frontend-module-domain', reason: `frontend/src/modules/${segments[3]} is not a known domain (${[...FRONTEND_MODULE_DOMAINS].join('/')})` });
      }

      if (/\.(test|spec)\./.test(base)) {
        issues.push({ rule: 'no-tests-in-src', reason: 'tests live under frontend/tests/, not frontend/src/ (REPOSITORY_LAYOUT §2.2)' });
      }

      if (third === 'components' && ext === '.tsx' && !/^[A-Z][A-Za-z0-9]*(\.[a-z]+)?\.tsx$/.test(base)) {
        issues.push({ rule: 'component-naming', reason: 'shared component .tsx files are PascalCase' });
      }

      if (third === 'hooks' && segments.length === 4 && !/^(index\.ts|use[A-Z][A-Za-z0-9]*\.tsx?)$/.test(base)) {
        issues.push({ rule: 'hook-naming', reason: 'hooks are named use<PascalCase>.ts (or index.ts barrel)' });
      }
    }
  }

  return issues;
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

  let files;
  try {
    files = listTrackedFiles(options.root);
  } catch (error) {
    console.error(`structure check failed to enumerate files: ${error.message}`);
    return 1;
  }

  const findings = [];
  for (const file of files) {
    for (const issue of checkFile(file)) {
      findings.push({ file, ...issue });
    }
  }

  if (options.json) {
    console.log(JSON.stringify({ mode: options.strict ? 'strict' : 'report-only', scannedCount: files.length, findingCount: findings.length, findings }, null, 2));
  } else {
    console.log(`Structure contract check (${options.strict ? 'strict' : 'report-only'}): ${findings.length} finding(s) across ${files.length} tracked file(s)`);
    for (const finding of findings) {
      console.log(`finding: ${finding.file} [${finding.rule}]`);
      console.log(`  reason: ${finding.reason}`);
    }
  }
  return options.strict && findings.length > 0 ? 1 : 0;
}

process.exitCode = main();
