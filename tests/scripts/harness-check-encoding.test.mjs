import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync, spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const TEST_DIR = path.dirname(fileURLToPath(import.meta.url));
const SCRIPT = path.resolve(TEST_DIR, '..', '..', 'scripts', 'harness', 'check-encoding.mjs');

function createFixture() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'encoding-check-'));
  execFileSync('git', ['-C', root, 'init', '-q']);
  fs.writeFileSync(path.join(root, 'clean.md'), '# 中文标题\n\n正常的 UTF-8 内容。\n');
  fs.writeFileSync(path.join(root, 'clean.go'), 'package main\n\n// 注释 comment\n');
  execFileSync('git', ['-C', root, 'add', '.']);
  return root;
}

function addCorruptedFile(root, name) {
  // Simulate mojibake: a valid UTF-8 Chinese sentence with one multi-byte
  // character truncated mid-sequence (the failure mode of a non-UTF-8
  // codepage write-back: bytes lost at character boundaries).
  const good = Buffer.from('本文定义工具无关协议。', 'utf8');
  const corrupted = Buffer.concat([good.subarray(0, 10), Buffer.from('?'), good.subarray(12)]);
  fs.writeFileSync(path.join(root, name), Buffer.concat([Buffer.from('# title\n\n'), corrupted, Buffer.from('\n')]));
  execFileSync('git', ['-C', root, 'add', name]);
}

test('check-encoding passes on clean UTF-8 tracked files', () => {
  const root = createFixture();
  const output = execFileSync(process.execPath, [SCRIPT, '--json', '--strict', '--root', root], { encoding: 'utf8' });
  const result = JSON.parse(output);
  assert.equal(result.findingCount, 0);
  assert.ok(result.scannedCount >= 2);
});

test('check-encoding fails in strict mode when a tracked file has invalid UTF-8', () => {
  const root = createFixture();
  addCorruptedFile(root, 'corrupted.md');
  const result = spawnSync(process.execPath, [SCRIPT, '--strict', '--root', root], { encoding: 'utf8' });
  assert.equal(result.status, 1);
  assert.match(result.stdout, /corrupted\.md/);
  assert.match(result.stdout, /invalid UTF-8/);
});

test('check-encoding reports line numbers of corruption', () => {
  const root = createFixture();
  addCorruptedFile(root, 'corrupted.md');
  const output = execFileSync(process.execPath, [SCRIPT, '--json', '--root', root], { encoding: 'utf8' });
  const result = JSON.parse(output);
  assert.equal(result.findingCount, 1);
  assert.match(result.findings[0].reason, /line\(s\) 3/);
});

test('check-encoding report-only mode exits 0 even with findings', () => {
  const root = createFixture();
  addCorruptedFile(root, 'corrupted.md');
  const result = spawnSync(process.execPath, [SCRIPT, '--root', root], { encoding: 'utf8' });
  assert.equal(result.status, 0);
  assert.match(result.stdout, /1 finding/);
});

test('check-encoding respects --ext filter', () => {
  const root = createFixture();
  addCorruptedFile(root, 'corrupted.txt');
  const output = execFileSync(process.execPath, [SCRIPT, '--json', '--strict', '--root', root, '--ext', '.md,.go'], { encoding: 'utf8' });
  const result = JSON.parse(output);
  assert.equal(result.findingCount, 0);
});

test('check-encoding tolerates UTF-8 BOM', () => {
  const root = createFixture();
  fs.writeFileSync(path.join(root, 'bom.md'), Buffer.concat([Buffer.from([0xef, 0xbb, 0xbf]), Buffer.from('# BOM 文件\n', 'utf8')]));
  execFileSync('git', ['-C', root, 'add', 'bom.md']);
  const output = execFileSync(process.execPath, [SCRIPT, '--json', '--strict', '--root', root], { encoding: 'utf8' });
  const result = JSON.parse(output);
  assert.equal(result.findingCount, 0);
});

test('check-encoding rejects Unicode replacement characters', () => {
  const root = createFixture();
  const replacementCharacter = String.fromCodePoint(0xfffd);
  fs.writeFileSync(path.join(root, 'replacement.md'), `# 标题\n\n损坏内容${replacementCharacter}\n`);
  execFileSync('git', ['-C', root, 'add', 'replacement.md']);
  const result = spawnSync(process.execPath, [SCRIPT, '--strict', '--root', root], { encoding: 'utf8' });
  assert.equal(result.status, 1);
  assert.match(result.stdout, /replacement\.md/);
  assert.match(result.stdout, /U\+FFFD/);
});

test('check-encoding skips untracked files', () => {
  const root = createFixture();
  // corrupted but never git-added
  const good = Buffer.from('中文', 'utf8');
  fs.writeFileSync(path.join(root, 'untracked.md'), Buffer.concat([good.subarray(0, 2), Buffer.from('?')]));
  const output = execFileSync(process.execPath, [SCRIPT, '--json', '--strict', '--root', root], { encoding: 'utf8' });
  const result = JSON.parse(output);
  assert.equal(result.findingCount, 0);
});
