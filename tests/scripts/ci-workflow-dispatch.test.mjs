import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync, execFileSync } from 'node:child_process';
import test from 'node:test';

const source = fs.readFileSync('.github/workflows/quality.yml', 'utf8');
const preflightStep = source.split('      - name: Resolve requested candidate before checkout')[1]?.split('      - name: Checkout')[0];
assert.ok(preflightStep, 'abbreviated SHA must resolve before actions/checkout');
const preflightBody = preflightStep.split('        run: |\n')[1]?.split(/\r?\n/).filter((line) => line.startsWith('          ')).map((line) => line.slice(10)).join('\n');
assert.ok(preflightBody);
const resolverStep = source.split('      - name: Resolve and verify candidate commit')[1]?.split('      - name: Detect changed scope')[0];
assert.ok(resolverStep, 'candidate must be verified before any repository code runs');
const bashBody = resolverStep.split('        run: |\n')[1]?.split(/\r?\n/).filter((line) => line.startsWith('          ')).map((line) => line.slice(10)).join('\n');
assert.ok(bashBody);
const gitPath = process.platform === 'win32' ? execFileSync('where.exe', ['git'], { encoding: 'utf8', windowsHide: true }).trim().split(/\r?\n/)[0] : 'git';
// Git for Windows keeps bash in <root>/usr/bin or <root>/bin. `where git` may
// return a mingw64-internal path whose sibling bash does not exist, so walk up
// from that git until a known bash layout is found.
const winBash = () => {
  let dir = path.dirname(gitPath);
  for (let up = 0; up < 4; up += 1) {
    for (const candidate of ['usr/bin/bash.exe', 'bin/bash.exe']) {
      if (fs.existsSync(path.join(dir, candidate))) return path.join(dir, candidate);
    }
    dir = path.dirname(dir);
  }
  return 'bash';
};
const bash = process.platform === 'win32' ? winBash() : 'bash';

test('quality accepts SHA recovery and resolves it before changed-scope code', () => {
  assert.match(source, /workflow_dispatch:\s*\n\s*inputs:\s*\n\s*sha:/);
  assert.match(resolverStep, /REQUESTED_SHA:\s*\$\{\{ inputs\.sha \}\}/);
  assert.doesNotMatch(bashBody, /\$\{\{/, 'untrusted input enters Bash through env, never interpolation');
  assert.ok(source.indexOf('Resolve and verify candidate commit') < source.indexOf('Detect changed scope'));
});

test('all six downstream checkouts consume the verified full commit and require its job', () => {
  assert.equal((source.match(/ref: \$\{\{ steps\.requested\.outputs\.sha \}\}/g) ?? []).length, 1);
  assert.doesNotMatch(source, /ref: \$\{\{ inputs\.sha/);
  assert.equal((source.match(/ref: \$\{\{ needs\.change-scope\.outputs\.candidate_sha \}\}/g) ?? []).length, 6);
  assert.match(source, /candidate_sha: \$\{\{ steps\.candidate\.outputs\.sha \}\}/);
  for (const job of ['docs-governance', 'frontend-contract', 'backend-tests', 'go-lint', 'duplication', 'smoke-sanity']) {
    const section = source.split(`  ${job}:`)[1]?.split(/\n  [a-z][a-z-]+:/)[0];
    assert.ok(section, job);
    assert.match(section, /needs:\s*\n\s*- change-scope/, job);
    assert.match(section, /ref: \$\{\{ needs\.change-scope\.outputs\.candidate_sha \}\}/, job);
  }
});

test('pre-checkout resolver expands SHA prefixes and rejects ambiguous or failed API results', () => {
  assert.doesNotMatch(preflightBody, /\$\{\{/);
  const fixture = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-quality-preflight-'));
  const full = '123abc' + 'd'.repeat(34);
  const apiStub = `gh() { printf 'called\\n' >> api-calls; [ "$1" = api ] && [ "$2" = "repos/example/ops/commits/$REQUESTED_SHA" ] || return 99; printf '%s\\n' "$API_SHA"; return "$API_EXIT"; }\n`;
  const cases = [
    { requested: '', returned: '', expected: 0, called: false },
    { requested: full.slice(0, 6), returned: full, expected: 0 },
    { requested: full.toUpperCase().slice(0, 9), returned: full, expected: 0 },
    { requested: full, returned: full, expected: 0 },
    { requested: full.slice(0, 6), returned: full, apiExit: 42, expected: 42 },
    { requested: full.slice(0, 6), returned: 'f'.repeat(40), expected: 1 },
    { requested: full.slice(0, 6), returned: full.slice(0, 39), expected: 1 },
    { requested: full.slice(0, 6), returned: full + 'f', expected: 1 },
    { requested: full.slice(0, 6), returned: '', expected: 1 },
    ...['main', 'a'.repeat(5), 'a'.repeat(41), '$(touch injected)', '"; touch injected; #'].map(requested => ({requested, returned: full, expected: 1, called: false})),
  ];
  try {
    for (const { requested, returned, expected, apiExit = 0, called = true } of cases) {
      fs.writeFileSync(path.join(fixture, 'output.txt'), '');
      fs.writeFileSync(path.join(fixture, 'api-calls'), '');
      const result = spawnSync(bash, ['--noprofile', '--norc', '-e', '-o', 'pipefail', '-c', apiStub + preflightBody], {
        cwd: fixture, env: { ...process.env, REQUESTED_SHA: requested, DEFAULT_SHA: full, API_SHA: returned, API_EXIT: String(apiExit), GITHUB_OUTPUT: 'output.txt', GITHUB_REPOSITORY: 'example/ops' },
        encoding: 'utf8', windowsHide: true, shell: false,
      });
      assert.ifError(result.error);
      assert.equal(result.status, expected, `${requested}: ${result.stderr}`);
      assert.equal(fs.readFileSync(path.join(fixture, 'output.txt'), 'utf8'), expected === 0 ? `sha=${full}\n` : '');
      assert.equal(Boolean(fs.readFileSync(path.join(fixture, 'api-calls'), 'utf8')), called);
      assert.ok(!fs.existsSync(path.join(fixture, 'injected')));
    }
  } finally {
    assert.equal(path.dirname(path.resolve(fixture)), path.resolve(os.tmpdir()));
    fs.rmSync(fixture, { recursive: true, force: true });
  }
});

test('actual resolver rejects wrong HEAD, invalid refs and shell input; returns full verified SHA', () => {
  const fixture = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-quality-candidate-'));
  try {
    const git = (...args) => {
      const result = spawnSync(gitPath, args, { cwd: fixture, encoding: 'utf8', windowsHide: true, shell: false });
      assert.ifError(result.error);
      assert.equal(result.status, 0, result.stderr);
      return result.stdout.trim();
    };
    git('init', '-q');
    const commit = (message) => git('-c', 'user.name=Candidate Test', '-c', 'user.email=candidate@test.invalid', '-c', 'commit.gpgsign=false', 'commit', '--allow-empty', '-m', message);
    commit('first');
    const first = git('rev-parse', 'HEAD');
    commit('second');
    const second = git('rev-parse', 'HEAD');
    const output = path.join(fixture, 'output.txt');
    const cases = [
      ['', 0], [second, 0], [second.slice(0, 7), 0], [first, 1],
      ['main', 1], ['a'.repeat(5), 1], ['b'.repeat(41), 1], ['f'.repeat(40), 128],
      ['$(touch injected)', 1], ['"; touch injected; #', 1], ['--help', 1],
    ];
    for (const [requested, expectedExit] of cases) {
      fs.writeFileSync(output, '');
      const result = spawnSync(bash, ['--noprofile', '--norc', '-e', '-o', 'pipefail', '-c', bashBody], {
        cwd: fixture, env: { ...process.env, REQUESTED_SHA: requested, GITHUB_OUTPUT: 'output.txt' },
        encoding: 'utf8', windowsHide: true, shell: false,
      });
      assert.ifError(result.error);
      assert.equal(result.status, expectedExit, `${requested}: ${result.stderr}`);
      assert.equal(fs.readFileSync(output, 'utf8'), expectedExit === 0 ? `sha=${second}\n` : '');
      assert.ok(!fs.existsSync(path.join(fixture, 'injected')), 'input must not execute shell commands');
    }
  } finally {
    assert.equal(path.dirname(path.resolve(fixture)), path.resolve(os.tmpdir()));
    fs.rmSync(fixture, { recursive: true, force: true });
  }
});

test('quality retains script catch-all, push-main, PR and merge-group gates', () => {
  assert.match(source, /npm run test:scripts\b/);
  assert.match(source, /push:\s*\n\s*branches:[\s\S]*- main/);
  assert.match(source, /pull_request:[\s\S]*merge_group:/);
});
