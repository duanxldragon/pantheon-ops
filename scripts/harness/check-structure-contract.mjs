#!/usr/bin/env node

/**
 * Mechanical gate for repository structure and naming conventions.
 * Contract source: docs/designs/REPOSITORY_LAYOUT.md §2 (placement rules),
 * §5.1 (Go naming), §5.2 (frontend naming + barrels) and §11 (frontend/backend
 * domain correspondence), plus the module-domain taxonomy in DESIGN.md §2.
 *
 * Scans git-TRACKED files only — local noise dirs (REPOSITORY_LAYOUT §3)
 * are invisible to it. All rules were at zero findings when the gate
 * landed: it is a ratchet against future misplacement, not a cleanup TODO.
 * Every rule added later must land at zero findings too, otherwise it is a
 * cleanup backlog wearing a gate's clothes.
 *
 * check-boundaries.mjs owns cross-layer IMPORT rules; this gate owns
 * WHERE files live and WHAT they are named. They are complementary.
 */
import fs from 'node:fs';
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
const FRONTEND_SRC_FILES = new Set(['App.tsx', 'index.css', 'index.ts', 'main.tsx', 'vite-env.d.ts']);
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

// §5.2 naming: components are PascalCase *.tsx, everything else is camelCase.
// Only frontend/src/modules and frontend/src/core are in scope — the same rule is
// not yet enforced for the shared components/hooks layers, which have their own
// narrower rules above.
const NAMED_FRONTEND_SCOPE = /^frontend\/src\/(modules|core)\//;
const PASCAL_CASE = /^[A-Z][A-Za-z0-9]*$/;
const CAMEL_CASE = /^[a-z][A-Za-z0-9]*$/;
const HOOK_NAME = /^use[A-Z][A-Za-z0-9]*$/;

// A .tsx file may only use a camelCase name when it does not declare a component.
// Two signals, because neither alone is sufficient:
//   * an explicit React component type / factory (React.FC, React.memo, …) — this is
//     what catches a component whose local identifier is itself camelCase;
//   * a PascalCase const/function/class declaration — the component naming convention.
// UPPER_SNAKE constants (MENU_ICON_OPTIONS) are not components, so the trailing
// lookahead keeps `const MENU_ICON_OPTIONS` from reading as a component `MENU`.
const COMPONENT_TYPE_MARKER = /React\.FC\b|:\s*FC<|\bReact\.Component\b|\bextends\s+Component\b|React\.memo\(|React\.forwardRef\(|ComponentType</;
const PASCAL_DECLARATION = /(?:^|\s)(?:const|function|class)\s+([A-Z][A-Za-z0-9]*)(?![A-Za-z0-9_])/;

function declaresComponent(content) {
  return COMPONENT_TYPE_MARKER.test(content) || PASCAL_DECLARATION.test(content);
}

// Barrel files re-export only; §5.2 forbids implementation in an index module.
// Scoped to the layers whose convention is a barrel (core/components/hooks);
// modules/**/index.ts are module descriptors and i18n/index.ts holds the locale
// table, so neither is a barrel by design.
const BARREL_FILE = /^frontend\/src\/(components|core|hooks)\/.*index\.tsx?$/;

// §11 frontend↔backend domain correspondence for the system foundation layer.
// A subdomain present on one side only is legal only as a registered §9 exception;
// otherwise the two trees drift and "correspondence" goes back to guesswork.
const CORRESPONDENCE_EXCEPTIONS = new Set([
  // §9: the profile page has no backend counterpart yet (its logic is split across
  // auth/security profile_dto.go and system/iam/user/user_preferences.go).
  'system/profile',
  // §9: tenant switching lives in auth/login; there is no frontend tenant module.
  'system/iam/tenant',
]);
// Structural subdirectories that exist per side and are not domain counterparts.
const STRUCTURAL_DIRS = new Set(['components', 'hooks', 'utils', 'pages', 'styles', 'types', 'tests']);

function readSource(root, file) {
  try {
    return fs.readFileSync(path.join(root, file), 'utf8');
  } catch {
    return null;
  }
}

function goPackageName(content) {
  const match = content.match(/^\s*package\s+([A-Za-z0-9_]+)/m);
  return match ? match[1] : null;
}

// Strips comments, then requires every remaining statement to be an `export … from`.
function isBarrelOnly(content) {
  const stripped = content
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/^\s*\/\/.*$/gm, '')
    .trim();
  if (stripped === '') return true;
  return stripped
    .split(';')
    .map((statement) => statement.trim())
    .filter(Boolean)
    .every((statement) => /^export\s+(?:type\s+)?(?:\*|\{[\s\S]*\})\s+from\s+['"][^'"]+['"]$/.test(statement));
}

function listTrackedFiles(root) {
  return execFileSync('git', ['-C', root, 'ls-files', '-z'], { encoding: 'utf8', maxBuffer: 64 * 1024 * 1024 })
    .split('\0')
    .filter(Boolean);
}

function checkFile(file, root) {
  const segments = file.split('/');
  const base = path.posix.basename(file);
  const ext = path.posix.extname(file).toLowerCase();
  const baseName = base.slice(0, base.length - ext.length);
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
    // §5.1: the package name must equal its directory name. This is what makes
    // `go doc`, compiler errors and import aliases line up with the directory tree.
    // cmd/* are executables, where `package main` is required rather than a choice.
    if (segments[0] === 'backend' && !file.startsWith('backend/cmd/')) {
      const content = readSource(root, file);
      const pkg = content === null ? null : goPackageName(content);
      const dirName = path.posix.basename(path.posix.dirname(file));
      // An external test package (`foo_test` in foo/) is idiomatic Go, not drift.
      if (pkg !== null && pkg !== dirName && pkg !== `${dirName}_test`) {
        issues.push({
          rule: 'go-package-dir-match',
          reason: `Go package "${pkg}" must equal its directory name "${dirName}" (REPOSITORY_LAYOUT §5.1)`,
        });
      }
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

    // §5.2 naming for hand-written module and core code. These scopes had no rule at
    // all before, which is how camelCase .tsx / PascalCase .ts drift accumulated.
    // Test files are excluded: §2.2 already forbids them in src, so reporting their
    // naming on top of that would be duplicate noise.
    const isTestFile = /\.(test|spec)\./.test(base);
    if (second === 'src' && !isTestFile && NAMED_FRONTEND_SCOPE.test(file)) {
      // A .ts file never holds a component, so camelCase is unconditional here.
      if (ext === '.ts' && baseName !== 'index' && !CAMEL_CASE.test(baseName)) {
        issues.push({
          rule: 'frontend-file-naming',
          reason: 'non-component frontend modules are camelCase *.ts (REPOSITORY_LAYOUT §5.2)',
        });
      }
      if (ext === '.tsx' && !PASCAL_CASE.test(baseName) && !HOOK_NAME.test(baseName)) {
        const content = readSource(root, file);
        if (content !== null && declaresComponent(content)) {
          issues.push({
            rule: 'frontend-file-naming',
            reason: 'a *.tsx declaring a component is PascalCase; camelCase *.tsx is only for hooks and non-component JSX modules (REPOSITORY_LAYOUT §5.2)',
          });
        }
      }
    }

    // §5.2: an index module is a barrel — re-exports only, no implementation.
    if (second === 'src' && !isTestFile && BARREL_FILE.test(file)) {
      const content = readSource(root, file);
      if (content !== null && !isBarrelOnly(content)) {
        issues.push({
          rule: 'barrel-only',
          reason: 'index modules are barrels: re-export only, no implementation (REPOSITORY_LAYOUT §5.2)',
        });
      }
    }
  }

  return issues;
}

// §11 correspondence between the backend system subdomains and their frontend
// counterparts. Compared at the subdomain and leaf level only: deeper directories
// are per-side structure (components/, hooks/) rather than domain counterparts.
function checkCorrespondence(files) {
  const issues = [];
  const sides = {
    backend: new Map(),
    frontend: new Map(),
  };

  for (const file of files) {
    if (/\.(test|spec)\./.test(path.posix.basename(file))) continue;
    let side = null;
    let relative = null;
    if (file.startsWith('backend/modules/system/') && file.endsWith('.go')) {
      side = 'backend';
      relative = file.slice('backend/modules/system/'.length);
    } else if (file.startsWith('frontend/src/modules/system/') && /\.tsx?$/.test(file)) {
      side = 'frontend';
      relative = file.slice('frontend/src/modules/system/'.length);
    }
    if (side === null) continue;

    const segments = relative.split('/');
    segments.pop(); // the file itself
    for (let depth = 1; depth <= Math.min(2, segments.length); depth += 1) {
      const candidate = segments.slice(0, depth);
      if (STRUCTURAL_DIRS.has(candidate[depth - 1])) break;
      sides[side].set(`system/${candidate.join('/')}`, true);
    }
  }

  const keys = new Set([...sides.backend.keys(), ...sides.frontend.keys()]);
  for (const key of [...keys].sort()) {
    const inBackend = sides.backend.has(key);
    const inFrontend = sides.frontend.has(key);
    if (inBackend === inFrontend) continue;
    if (CORRESPONDENCE_EXCEPTIONS.has(key)) continue;
    const missing = inBackend ? 'frontend' : 'backend';
    issues.push({
      file: `modules/${key}`,
      rule: 'domain-correspondence',
      reason: `subdomain "${key}" exists only on the ${inBackend ? 'backend' : 'frontend'} side; add the ${missing} counterpart or register a controlled exception in REPOSITORY_LAYOUT §9 / §11`,
    });
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
    for (const issue of checkFile(file, options.root)) {
      findings.push({ file, ...issue });
    }
  }
  findings.push(...checkCorrespondence(files));

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
