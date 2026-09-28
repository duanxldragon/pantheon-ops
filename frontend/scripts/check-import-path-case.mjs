/**
 * Import path case guard.
 *
 * Why this exists: local Windows/macOS checkouts are case-insensitive, so
 * `import './Dashboard.css'` resolves fine even when the tracked file is
 * `dashboard.css`. Linux CI then fails the build with UNRESOLVED_IMPORT and the
 * failure looks unrelated to the change that introduced it (2026-09-23: a CSS
 * rename was applied to the import but the file itself stayed lowercase, which
 * broke `Frontend Contract` on PR #338).
 *
 * Every relative module specifier in `src/` is resolved against the real
 * on-disk entry names, so a case-only drift fails here on any filesystem.
 */
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const frontendRoot = path.resolve(__dirname, '..');
const srcRoot = path.join(frontendRoot, 'src');

const SOURCE_EXTENSIONS = new Set(['.ts', '.tsx', '.js', '.jsx', '.mjs', '.cjs']);
const RESOLUTION_SUFFIXES = [
  '',
  '.ts',
  '.tsx',
  '.js',
  '.jsx',
  '.mjs',
  '.cjs',
  '.css',
  '.scss',
  '.less',
  '.json',
  '.txt',
  '/index.ts',
  '/index.tsx',
  '/index.js',
  '/index.jsx',
];

const SPECIFIER_PATTERNS = [
  /(?:^|\n)\s*import\s+[^'"]*?from\s*['"]([^'"]+)['"]/g,
  /(?:^|\n)\s*export\s+[^'"]*?from\s*['"]([^'"]+)['"]/g,
  /(?:^|\n)\s*import\s*['"]([^'"]+)['"]/g,
  /\bimport\s*\(\s*['"]([^'"]+)['"]\s*\)/g,
  /\brequire\s*\(\s*['"]([^'"]+)['"]\s*\)/g,
  /@import\s+(?:url\()?['"]([^'"]+)['"]/g,
];

function walk(dir) {
  const files = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === 'node_modules' || entry.name.startsWith('.')) continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walk(full));
      continue;
    }
    files.push(full);
  }
  return files;
}

function isSourceFile(file) {
  const extension = path.extname(file);
  return SOURCE_EXTENSIONS.has(extension) || extension === '.css';
}

/**
 * Blanks out comments and template literals while keeping offsets.
 *
 * The lowcode module ships generated-code *templates* as backtick strings that
 * contain literal `import ... from './api'` lines; those are sample code for
 * generated modules, not module edges of this project, and must not be checked.
 */
function maskNonCode(source) {
  const chars = [...source];
  let i = 0;
  const blank = (index) => {
    if (chars[index] !== '\n') chars[index] = ' ';
  };
  const skipTemplate = (start) => {
    let depth = 0;
    let index = start;
    while (index < source.length) {
      const char = source[index];
      const next = source[index + 1];
      if (char === '\\') {
        blank(index);
        blank(index + 1);
        index += 2;
        continue;
      }
      if (char === '$' && next === '{') {
        depth += 1;
        blank(index);
        blank(index + 1);
        index += 2;
        continue;
      }
      if (char === '}' && depth > 0) {
        depth -= 1;
        blank(index);
        index += 1;
        continue;
      }
      if (char === '`') {
        if (depth === 0) {
          blank(index);
          return index + 1;
        }
        // Nested template inside an interpolation.
        blank(index);
        index = skipTemplate(index + 1);
        continue;
      }
      blank(index);
      index += 1;
    }
    return index;
  };

  while (i < source.length) {
    const char = source[i];
    const next = source[i + 1];
    if (char === '/' && next === '/') {
      while (i < source.length && source[i] !== '\n') {
        blank(i);
        i += 1;
      }
      continue;
    }
    if (char === '/' && next === '*') {
      while (i < source.length && !(source[i] === '*' && source[i + 1] === '/')) {
        blank(i);
        i += 1;
      }
      blank(i);
      blank(i + 1);
      i += 2;
      continue;
    }
    if (char === '`') {
      blank(i);
      i = skipTemplate(i + 1);
      continue;
    }
    i += 1;
  }
  return chars.join('');
}

/**
 * Case-insensitive index of every on-disk source entry: lower path -> real path.
 *
 * Resolution must never touch fs.existsSync: on a case-insensitive filesystem
 * `./Dashboard.css` happily "exists" while the real entry is `dashboard.css`,
 * which is exactly the drift this guard has to see.
 */
function buildCaseIndex(files) {
  const index = new Map();
  for (const file of files) {
    index.set(file.toLowerCase(), file);
  }
  return index;
}

function collectSpecifiers(source) {
  const specifiers = [];
  for (const pattern of SPECIFIER_PATTERNS) {
    for (const match of source.matchAll(pattern)) {
      specifiers.push(match[1]);
    }
  }
  return specifiers;
}

function resolveExactCase(importerFile, specifier, caseIndex) {
  // Vite query suffixes (`?raw`, `?url`) are not part of the on-disk name.
  const specifierPath = specifier.split('?')[0];
  if (!specifierPath) return { status: 'ok' };
  const base = path.resolve(path.dirname(importerFile), specifierPath);
  for (const suffix of RESOLUTION_SUFFIXES) {
    // normalize: the `/index.ts` suffixes must not leave mixed separators
    // behind, or every directory import would read as unresolved.
    const candidate = path.normalize(`${base}${suffix}`);
    const actual = caseIndex.get(candidate.toLowerCase());
    if (!actual) continue;
    if (actual === candidate) return { status: 'ok' };
    return {
      status: 'case-mismatch',
      actual: path.relative(frontendRoot, actual).replaceAll('\\', '/'),
      expected: path.relative(frontendRoot, candidate).replaceAll('\\', '/'),
    };
  }
  return { status: 'unresolved' };
}

const allFiles = walk(srcRoot);
// Resolution targets include non-source assets (`.txt` templates, `.json`).
const caseIndex = buildCaseIndex(allFiles);
const files = allFiles.filter(isSourceFile);
const problems = [];

for (const file of files) {
  const source = maskNonCode(fs.readFileSync(file, 'utf8'));
  const seen = new Set();
  for (const specifier of collectSpecifiers(source)) {
    if (!specifier.startsWith('./') && !specifier.startsWith('../')) continue;
    const key = `${file}|${specifier}`;
    if (seen.has(key)) continue;
    seen.add(key);

    const result = resolveExactCase(file, specifier, caseIndex);
    if (result.status === 'ok') continue;
    problems.push({
      file: path.relative(frontendRoot, file).replaceAll('\\', '/'),
      specifier,
      ...result,
    });
  }
}

if (problems.length > 0) {
  console.error(`import path case guard failed: ${problems.length} problem(s)`);
  for (const problem of problems) {
    if (problem.status === 'case-mismatch') {
      console.error(
        `  ${problem.file}: '${problem.specifier}' resolves to '${problem.expected}', ` +
          `but the file on disk is '${problem.actual}' — fix the specifier case (case-insensitive ` +
          `filesystems hide this until Linux CI builds)`,
      );
      continue;
    }
    console.error(
      `  ${problem.file}: '${problem.specifier}' does not resolve to any file under src/`,
    );
  }
  process.exit(1);
}

console.log(`import path case guard passed for ${files.length} source files`);
