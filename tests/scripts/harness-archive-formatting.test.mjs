import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../../', import.meta.url));
const prettier = fileURLToPath(import.meta.resolve('prettier/bin/prettier.cjs'));
const lintStaged = fileURLToPath(import.meta.resolve('lint-staged/bin'));
const archivedMarkdown = [
  '.harness/archive/2026-08/tasks/sample/plan.md',
  '.harness/archive/2026-11/evidence/sample/review.md',
  '.harness/archive/2026-11/preserved-live/sample/evidence/sample/summary.md',
  '.harness/archive/2026-11/STATUS-before-closeout.md',
];
const liveMarkdown = ['README.md', '.harness/tasks/active/plan.md', '.harness/evidence/active/summary.md'];
const originalMarkdown = Buffer.from('# Sample\r\n\r\n*   item\r\n\r\n\r\n');
const formattedMarkdown = Buffer.from('# Sample\n\n- item\n');

function fixture(t) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-archive-formatting-'));
  t.after(() => {
    assert.equal(path.dirname(directory), fs.realpathSync(os.tmpdir()));
    assert.ok(path.basename(directory).startsWith('pantheon-archive-formatting-'));
    fs.rmSync(directory, { recursive: true, force: true });
  });
  fs.copyFileSync(path.join(root, 'package.json'), path.join(directory, 'package.json'));
  fs.symlinkSync(path.join(root, 'node_modules'), path.join(directory, 'node_modules'),
    process.platform === 'win32' ? 'junction' : 'dir');
  return directory;
}

function run(directory, executable, args, binary = false) {
  const env = { ...process.env };
  for (const key of ['GIT_DIR', 'GIT_WORK_TREE', 'GIT_INDEX_FILE', 'GIT_COMMON_DIR']) delete env[key];
  const result = spawnSync(executable, args, {
    cwd: directory, env, windowsHide: true, encoding: binary ? undefined : 'utf8',
    timeout: 30_000, maxBuffer: 4 * 1024 * 1024,
  });
  assert.ifError(result.error);
  assert.equal(result.status, 0, `${executable}: ${result.stdout}\n${result.stderr}`);
  return result.stdout;
}

function git(directory, args, binary = false) {
  return run(directory, 'git', ['-c', 'core.autocrlf=false', '-c', 'core.hooksPath=.no-hooks', ...args], binary);
}

function writeMarkdown(directory) {
  for (const file of [...archivedMarkdown, ...liveMarkdown]) {
    const target = path.join(directory, file);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.writeFileSync(target, originalMarkdown);
  }
}

function enableArchiveProtection(directory) {
  fs.copyFileSync(path.join(root, '.prettierignore'), path.join(directory, '.prettierignore'));
}

test('Prettier explicit writes preserve archived Markdown and JSON while formatting live documents', (t) => {
  const directory = fixture(t);
  writeMarkdown(directory);
  enableArchiveProtection(directory);
  const archiveJson = '.harness/archive/2026-11/evidence/sample/commands.json';
  const originalJson = Buffer.from('{"commands": [ ]}\r\n\r\n');
  fs.writeFileSync(path.join(directory, archiveJson), originalJson);
  run(directory, process.execPath, [prettier, '--write', ...archivedMarkdown, archiveJson, ...liveMarkdown]);
  for (const file of archivedMarkdown) assert.deepEqual(fs.readFileSync(path.join(directory, file)), originalMarkdown, file);
  assert.deepEqual(fs.readFileSync(path.join(directory, archiveJson)), originalJson);
  for (const file of liveMarkdown) assert.deepEqual(fs.readFileSync(path.join(directory, file)), formattedMarkdown, file);
});

test('normal lint-staged configuration preserves archive worktree/index bytes and still formats live Markdown', (t) => {
  const directory = fixture(t);
  git(directory, ['init', '--quiet']);
  git(directory, ['-c', 'user.name=Archive Fixture', '-c', 'user.email=archive-fixture@example.invalid',
    'commit', '--quiet', '--allow-empty', '-m', 'fixture']);
  writeMarkdown(directory);
  git(directory, ['add', '--', ...archivedMarkdown, ...liveMarkdown]);
  // Witness the original defect using the unchanged root package configuration.
  run(directory, process.execPath, [lintStaged, '--quiet']);
  for (const file of archivedMarkdown) assert.notDeepEqual(fs.readFileSync(path.join(directory, file)), originalMarkdown, file);

  writeMarkdown(directory);
  enableArchiveProtection(directory);
  git(directory, ['add', '--', ...archivedMarkdown, ...liveMarkdown]);
  run(directory, process.execPath, [lintStaged, '--quiet']);
  for (const file of archivedMarkdown) {
    assert.deepEqual(fs.readFileSync(path.join(directory, file)), originalMarkdown, file);
    assert.deepEqual(git(directory, ['show', `:${file}`], true), originalMarkdown, `index: ${file}`);
  }
  for (const file of liveMarkdown) {
    assert.deepEqual(fs.readFileSync(path.join(directory, file)), formattedMarkdown, file);
    assert.deepEqual(git(directory, ['show', `:${file}`], true), formattedMarkdown, `index: ${file}`);
  }
});
