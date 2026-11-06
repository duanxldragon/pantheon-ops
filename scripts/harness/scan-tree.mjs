/**
 * Recursive text scanner for the pattern checkers.
 *
 * Why this exists: `check-security-patterns.mjs` and `check-ui-tokens.mjs` used
 * to shell out to ripgrep. That made two blocking gates depend on a tool that
 * (a) is not installed on every developer machine, and (b) silently swallowed
 * its own failures until 2026-11-04, so "no violations" could mean "rg never
 * ran". It also forced a CI `apt-get install ripgrep` step, and two of the rules
 * used look-around, which ripgrep's default engine rejects outright.
 *
 * The scan surface is small (a few hundred files, single-digit MB), so a plain
 * Node walk is fast enough and removes the external dependency, the install
 * step, and the regex-engine mismatch in one move.
 *
 * Contract:
 *   - `pattern` is a JS RegExp; look-ahead is supported natively.
 *   - Matching is line-oriented; one entry per matching line.
 *   - Directory names in SKIP_DIRECTORIES are never descended into.
 *   - `extensions` (a Set of '.ext') restricts which files are read.
 *   - `excludeSuffixes` (e.g. ['_test.go']) drops files by name suffix.
 *   - `missing` reports scan roots that do not exist, so callers can fail closed
 *     instead of reporting "no findings" for a path they never looked at.
 */

import fs from 'node:fs';
import path from 'node:path';

// Build/Cache directories that never contain source worth scanning. This is the
// dependency-free equivalent of the ignore rules rg applied; it is an explicit
// list rather than .gitignore parsing because the scan roots are source trees.
const SKIP_DIRECTORIES = new Set([
  'node_modules',
  '.git',
  'dist',
  'build',
  'coverage',
  'test-results',
  'playwright-report',
]);

/**
 * scanTree walks every directory in `dirs` (relative to `root`) and returns the
 * lines matching `pattern`.
 *
 * @param {object} options
 * @param {string} options.root repository root
 * @param {string[]} options.dirs directories to scan, relative to root
 * @param {RegExp} options.pattern line pattern
 * @param {Set<string>|null} [options.extensions] allowed extensions (with dot)
 * @returns {{matches: {file: string, line: number, text: string}[], missing: string[]}}
 */
export function scanTree({ root, dirs, pattern, extensions = null, excludeSuffixes = null }) {
  const matches = [];
  const missing = [];

  for (const dir of dirs) {
    const absolute = path.resolve(root, dir);
    if (!fs.existsSync(absolute)) {
      missing.push(dir);
      continue;
    }
    walk(absolute);
  }

  return { matches, missing };

  function walk(dirPath) {
    let entries;
    try {
      entries = fs.readdirSync(dirPath, { withFileTypes: true });
    } catch {
      // An unreadable directory is a scan gap, not a clean result; record it the
      // same way as a missing root.
      missing.push(path.relative(root, dirPath).split(path.sep).join('/'));
      return;
    }

    for (const entry of entries) {
      const fullPath = path.join(dirPath, entry.name);
      if (entry.isDirectory()) {
        if (!SKIP_DIRECTORIES.has(entry.name)) {
          walk(fullPath);
        }
        continue;
      }
      if (!entry.isFile()) {
        continue;
      }
      if (extensions && !extensions.has(path.extname(entry.name))) {
        continue;
      }
      if (excludeSuffixes && excludeSuffixes.some((suffix) => entry.name.endsWith(suffix))) {
        continue;
      }

      let content;
      try {
        content = fs.readFileSync(fullPath, 'utf8');
      } catch {
        missing.push(path.relative(root, fullPath).split(path.sep).join('/'));
        continue;
      }

      const lines = content.split(/\r?\n/);
      for (let index = 0; index < lines.length; index += 1) {
        // Reset state for callers that pass a /g regex.
        pattern.lastIndex = 0;
        if (!pattern.test(lines[index])) {
          continue;
        }
        matches.push({
          file: path.relative(root, fullPath).split(path.sep).join('/'),
          line: index + 1,
          text: lines[index].trim(),
        });
      }
    }
  }
}
