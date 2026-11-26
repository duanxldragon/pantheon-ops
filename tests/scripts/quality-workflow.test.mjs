import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import {spawnSync, execFileSync} from 'node:child_process';
import assert from 'node:assert/strict';
import test from 'node:test';

const workflowPath = path.resolve('.github/workflows/quality.yml');
const workflowSource = fs.readFileSync(workflowPath, 'utf8');

// Mirrors security-audit-exit.test.mjs: extract the multiline `run: |` body of a
// named step so we can assert real behavior by executing the actual bash.
function shellFor(name) {
  const step = workflowSource.split(`      - name: ${name}`)[1];
  assert.ok(step, `missing workflow step: ${name}`);
  const lines = step.split(/\r?\n/);
  const start = lines.findIndex((l) => /^        run: \|\s*$/.test(l));
  assert.ok(start >= 0, `missing multiline shell in step: ${name}`);
  const body = [];
  for (const line of lines.slice(start + 1)) {
    if (line.trim() && !line.startsWith('          ')) break;
    body.push(line.slice(10));
  }
  return body.join('\n');
}

function resolveBash() {
  if (process.platform !== 'win32') return 'bash';
  const gitCmd = execFileSync('where.exe', ['git'], {encoding: 'utf8', windowsHide: true}).trim().split(/\r?\n/)[0];
  for (const rel of ['../bin/bash.exe', '../../bin/bash.exe', '../../usr/bin/bash.exe']) {
    const p = path.resolve(path.dirname(gitCmd), rel);
    if (fs.existsSync(p)) return p;
  }
  throw new Error(`git bash not found from ${gitCmd}`);
}
const bash = resolveBash();

const coreBusinessSmoke = [
  'test:smoke:business:cmdb',
  'test:smoke:business:deploy:api',
  'test:smoke:business:deploy',
  'test:smoke:business:service',
  'test:smoke:business:k8s',
];

function runBash(body, env) {
  return spawnSync(bash, ['--noprofile', '--norc', '-c', body], {
    cwd: path.dirname(workflowPath),
    env: {...process.env, ...env},
    encoding: 'utf8',
    windowsHide: true,
  });
}

test('smoke sanity requires all core Ops business suites after backend readiness', () => {
  const job = workflowSource.split('  smoke-sanity:')[1]?.split('\n  quality-gates:')[0];
  assert.ok(job, 'missing smoke-sanity job');
  const name = 'Run Ops core business smoke';
  const step = job.split(`      - name: ${name}`)[1]?.split('      - name:')[0];
  assert.ok(step, 'core business smoke must belong to the required smoke-sanity job');
  assert.ok(job.indexOf(`      - name: ${name}`) > job.indexOf('      - name: Wait for backend health'));
  assert.ok(job.indexOf(`      - name: ${name}`) > job.indexOf('      - name: Run smoke sanity suite'));
  assert.match(step, /working-directory: frontend/);
  assert.match(step, /shell: bash/);
  assert.doesNotMatch(step, /\bif:|continue-on-error:|\|\|\s*(?:true|echo)|exit\s+0/);
  assert.doesNotMatch(job, /continue-on-error:/);
  const scripts = JSON.parse(fs.readFileSync(path.resolve('frontend/package.json'), 'utf8')).scripts;
  const body = shellFor(name);
  for (const script of coreBusinessSmoke) {
    assert.ok(scripts[script], `missing frontend script: ${script}`);
    assert.ok(body.includes(`npm run ${script}`), `CI must invoke ${script}`);
  }
});

test('actual Ops business smoke shell propagates every failure and stops later suites', () => {
  const body = shellFor('Run Ops core business smoke');
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'ops-quality-smoke-'));
  const log = path.join(temp, 'npm-calls.log');
  // Replace npm only: exercise the real workflow body without running providers.
  const fakeNpm = 'npm() { printf "%s\\n" "$*" >> "$SMOKE_LOG"; if [ "$2" = "$FAIL_SMOKE" ]; then return 23; fi; };\n';
  try {
    for (const failed of ['', ...coreBusinessSmoke]) {
      fs.writeFileSync(log, '');
      const result = runBash(fakeNpm + body, {SMOKE_LOG: log.replaceAll('\\', '/'), FAIL_SMOKE: failed});
      assert.ifError(result.error);
      assert.equal(result.status, failed ? 23 : 0, `failed suite ${failed || '(none)'}: ${result.stderr}`);
      const count = failed ? coreBusinessSmoke.indexOf(failed) + 1 : coreBusinessSmoke.length;
      const actual = fs.readFileSync(log, 'utf8').trim().split(/\r?\n/);
      assert.deepEqual(actual, coreBusinessSmoke.slice(0, count).map((script) => `run ${script}`));
    }
  } finally {
    fs.rmSync(temp, {recursive: true, force: true});
  }
});

test('change scope classifies real gates instead of removed docs/governance cutoffs', () => {
  assert.match(
    workflowSource,
    /uses:\s*dorny\/paths-filter@ceb8a2b8f2d89434be7ff52d3de7ec3738c5cc9d/i,
    'quality workflow should pin paths-filter to the reviewed revision',
  );
  assert.match(
    workflowSource,
    /predicate-quantifier:\s*some/i,
    'change scope should use some (one matched file activates the gate), not every',
  );
  assert.match(
    workflowSource,
    /- name: Detect changed scope[\s\S]*?if:\s*\$\{\{\s*github\.event_name\s*==\s*'pull_request'\s*\}\}/i,
    'changed-scope detection should only run on pull requests',
  );
  // The scope outputs that really gate runtime jobs.
  for (const [output, globs] of [
    ['frontend_contract', ['frontend/**']],
    ['smoke_sanity', ['backend/**', 'frontend/**', 'go.mod', 'go.sum']],
    ['overlay_sensitive', ['business-overlay.json', 'scripts/business-overlay/**', 'backend/modules/business/**', 'frontend/src/modules/business/**']],
  ]) {
    assert.match(workflowSource, new RegExp(`${output}:`), `${output} should be a change-scope output`);
    for (const glob of globs) {
      const escaped = glob.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&');
      assert.match(workflowSource, new RegExp(`- '${escaped}'`), `${output} should match ${glob}`);
    }
  }
  for (const output of ['frontend_contract', 'smoke_sanity']) {
    assert.match(workflowSource, new RegExp(`${output}: \\$\\{\\{ steps\\.scope\\.outputs\\.${output} \\}\\}`));
    assert.doesNotMatch(workflowSource, new RegExp(`steps\\.scope\\.outputs\\.${output}\\s*\\|\\|`), 'missing PR scope must remain unknown');
  }
  // The old docs-only / governance-only classification was removed; asserting it
  // is gone keeps the test honest rather than silently entangled with a vocated flag.
  for (const removed of ['docs_only', 'governance_only', 'all_count']) {
    assert.doesNotMatch(workflowSource, new RegExp(removed), `${removed} classification no longer exists`);
  }
});

test('every checkout is pinned to the resolved candidate commit', () => {
  assert.match(
    workflowSource,
    /workflow_dispatch:\s*\n\s*inputs:\s*\n\s*sha:/i,
    'workflow_dispatch should expose a sha input for on-demand baseline runs',
  );
  const candidate = shellFor('Resolve and verify candidate commit');
  assert.ok(candidate.includes('=~ ^[[:xdigit:]]{6,40}$'), 'candidate sha must be 6-40 hex');
  assert.match(candidate, /rev-parse --verify "\$REQUESTED_SHA\^\{commit\}"/, 'candidate sha must resolve to a commit');
  assert.match(candidate, /\[ "\$ACTUAL" != "\$EXPECTED" \]; then[\s\S]*HEAD does not match requested commit/, 'HEAD must equal the requested commit');
  assert.match(candidate, /echo "sha=\$ACTUAL" >> "\$GITHUB_OUTPUT"/, 'candidate sha must be emitted as the checkout ref');
  assert.match(
    workflowSource,
    /candidate_sha:\s*\$\{\{\s*steps\.candidate\.outputs\.sha\s*\}\}/,
    'change-scope must expose candidate_sha to dependent jobs',
  );
  assert.match(
    workflowSource,
    /id: candidate[\s\S]*?echo "sha=\$ACTUAL" >> "\$GITHUB_OUTPUT"/i,
    'candidate verification step must be id=candidate wired to the output',
  );
  // Rejecting a non-hex candidate SHA is a real fail-closed branch.
  const bad = runBash(candidate, {REQUESTED_SHA: 'not-hex-!'});
  assert.ifError(bad.error);
  assert.equal(bad.status, 1, 'non-hex candidate sha must fail closed: ' + bad.stderr);
  // A candidate sha pointing at the live HEAD passes and emits candidate_sha.
  const head = execFileSync('git', ['rev-parse', 'HEAD'], {encoding: 'utf8', cwd: path.dirname(workflowPath), windowsHide: true}).trim();
  const out = path.join(os.tmpdir(), `pantheon-candidate-${process.pid}.out`);
  const ok = runBash(candidate, {REQUESTED_SHA: head, GITHUB_OUTPUT: out});
  assert.ifError(ok.error);
  assert.equal(ok.status, 0, 'valid candidate sha must pass: ' + ok.stderr);
  assert.match(fs.readFileSync(out, 'utf8'), /sha=/, 'candidate sha should be emitted to GITHUB_OUTPUT');
  fs.rmSync(out, {force: true});
  // Every consumer job checks out candidate_sha so all jobs inspect the same commit.
  const consumers = ['frontend-contract', 'backend-tests', 'go-lint', 'duplication', 'smoke-sanity'];
  for (const job of consumers) {
    const block = workflowSource.split(new RegExp(`^  ${job}:$`, 'm'))[1];
    assert.match(
      block.split(/^  [a-z][a-z-]*:$/m)[0],
      /ref:\s*\$\{\{\s*needs\.change-scope\.outputs\.candidate_sha\s*\}\}/,
      `${job} should checkout candidate_sha`,
    );
    assert.doesNotMatch(block.split(/^  [a-z][a-z-]*:$/m)[0], /ref:\s*\$\{\{\s*github\.sha\s*\}\}/, `${job} must not checkout an unchecked summary ref`);
  }
});

test('frontend contract and smoke sanity gate on pull-request scope only', () => {
  assert.match(
    workflowSource,
    /frontend-contract:[\s\S]*?if:\s*\$\{\{\s*github\.event_name !=\s*'pull_request'\s*\|\|\s*needs\.change-scope\.outputs\.frontend_contract ==\s*'true'\s*\}\}/,
    'frontend contract should run on every non-PR event and on scope-relevant pull requests',
  );
  assert.match(
    workflowSource,
    /smoke-sanity:[\s\S]*?if:\s*\$\{\{\s*github\.event_name !=\s*'pull_request'\s*\|\|\s*needs\.change-scope\.outputs\.smoke_sanity ==\s*'true'\s*\}\}/,
    'smoke sanity should run on every non-PR event and on scope-relevant pull requests',
  );
});

test('backend tests always run and fail closed', () => {
  const block = workflowSource.split(/^  backend-tests:$/m)[1].split(/^  [a-z][a-z-]*:$/m)[0];
  assert.match(block, /needs:\s*\n\s*- change-scope/, 'backend tests should wait for change-scope');
  assert.doesNotMatch(block, /^\s*if:\s*/, 'backend tests must not be skippable by scope');
  assert.match(block, /run:\s*go test -race \.\/\.\.\./, 'backend tests should run the full race suite');
});

function assertBackendDatabaseServices(block) {
  const mysqlMatch = block.match(/^ {6}mysql:\s*\n([\s\S]*?)(?=^ {6}[a-z][a-z-]*:\s*$|^ {4}[a-z][a-z-]*:\s*$)/m);
  assert.ok(mysqlMatch, 'backend tests need a MySQL service');
  assert.match(mysqlMatch[1], /^ {8}image: mysql:8\.0$/m, 'backend tests need MySQL 8');
  const mysqlEnvMatch = mysqlMatch[1].match(/^ {8}env:\n((?: {10}[^\n]*\n)*)/m);
  assert.ok(mysqlEnvMatch, 'MySQL service needs an environment block');
  assert.match(mysqlEnvMatch[1], /^ {10}TZ: UTC$/m, 'the MySQL service clock must be UTC');
  assert.match(block, /PANTHEON_TEST_DSN:\s*[^\n]*loc=UTC/, 'the test DSN must use UTC');
  assert.match(block, /PANTHEON_TEST_DSN_REQUIRED:\s*"true"/, 'a missing DSN must fail, not skip');
  const jobEnvMatch = block.match(/^ {4}env:\n((?: {6}[^\n]*\n| {6}#[^\n]*\n)*)/m);
  assert.ok(jobEnvMatch, 'backend tests need a job environment block');
  assert.match(jobEnvMatch[1], /^ {6}PANTHEON_TEST_REDIS_ADDR:\s*127\.0\.0\.1:6379$/m, 'backend tests need the generic Redis address');
  const redisMatch = block.match(/^ {6}redis:\s*\n([\s\S]*?)(?=^ {6}[a-z][a-z-]*:\s*$|^ {4}[a-z][a-z-]*:\s*$)/m);
  assert.ok(redisMatch, 'backend tests need a generic Redis service');
  assert.match(redisMatch[1], /^ {8}image: redis:7-alpine$/m, 'backend tests need Redis 7');
  assert.match(block, /PANTHEON_TEST_REDIS_REQUIRED:\s*"true"/, 'a missing Redis must fail, not skip');
}

test('backend tests provide MySQL and Redis and refuse to skip database tests', () => {
  const block = workflowSource.split(/^  backend-tests:$/m)[1].split(/^  [a-z][a-z-]*:$/m)[0];
  assertBackendDatabaseServices(block);
});

test('backend database service assertions reject misplaced or incorrect Redis configuration', () => {
  const block = workflowSource.split(/^  backend-tests:$/m)[1].split(/^  [a-z][a-z-]*:$/m)[0];
  const redisAddress = /^      PANTHEON_TEST_REDIS_ADDR: 127\.0\.0\.1:6379\n/m;
  assert.match(block, redisAddress, 'test fixture must include the expected job-level Redis address');

  const misplaced = block.replace(redisAddress, '').replace(
    /^    env:$/m,
    '      PANTHEON_TEST_REDIS_ADDR: 127.0.0.1:6379\n    env:',
  );
  assert.throws(() => assertBackendDatabaseServices(misplaced), /generic Redis address/);

  const incorrectEndpoint = block.replace(redisAddress, '      PANTHEON_TEST_REDIS_ADDR: 127.0.0.1:26316\n');
  assert.throws(() => assertBackendDatabaseServices(incorrectEndpoint), /generic Redis address/);
});

test('go lint scopes pull requests and merge groups to new code and enforces in real bash', () => {
  const goLintSource = workflowSource.slice(
    workflowSource.indexOf('  go-lint:'),
    workflowSource.indexOf('  duplication:'),
  );
  assert.match(goLintSource, /PR_BASE_SHA:\s*\$\{\{ github\.event\.pull_request\.base\.sha \}\}/);
  assert.match(goLintSource, /MERGE_GROUP_BASE_SHA:\s*\$\{\{ github\.event\.merge_group\.base_sha \}\}/);
  assert.match(
    goLintSource,
    /if \[\s*"\$\{EVENT_NAME\}" = "pull_request" \]\s*&&\s*\[ -n "\$\{PR_BASE_SHA\}" \]; then[\s\S]*?args=--new-from-rev=\$\{PR_BASE_SHA\} \.\/\.\.\.[\s\S]*?elif \[\s*"\$\{EVENT_NAME\}" = "merge_group" \]\s*&&\s*\[ -n "\$\{MERGE_GROUP_BASE_SHA\}" \]; then[\s\S]*?args=--new-from-rev=\$\{MERGE_GROUP_BASE_SHA\} \.\/\.\.\./,
    'pull requests and merge groups must lint only newly introduced code',
  );
  assert.match(
    goLintSource,
    /- name: Run golangci-lint[\s\S]*?continue-on-error:\s*true/i,
    'go-lint findings should be surfaced without failing the step',
  );
  const gate = shellFor('Enforce Go lint gate');
  assert.match(gate, /EVENT_NAME/);
  assert.match(gate, /GOLANGCI_LINT_RESULT/);
  assert.match(
    gate,
    /if \[\s*"\$\{EVENT_NAME\}" = "push" \] \|\| \[\s*"\$\{EVENT_NAME\}" = "workflow_dispatch" \]; then[\s\S]*?report-only/i,
    'push and manual baseline runs must keep the advisory report-only path',
  );
  assert.match(
    gate,
    /if \[\s*"\$\{GOLANGCI_LINT_RESULT\}" !=\s*"success" \]; then[\s\S]*?exit 1/,
    'any non-push lint failure must fail closed',
  );
  // Real behavior: PR/merge_group lint failure rejects, push/manual stays report-only.
  for (const event of ['pull_request', 'merge_group']) {
    const r = runBash(gate, {EVENT_NAME: event, GOLANGCI_LINT_RESULT: 'failure'});
    assert.ifError(r.error);
    assert.equal(r.status, 1, `go-lint must fail closed on ${event}: ${r.stderr}`);
  }
  for (const event of ['push', 'workflow_dispatch']) {
    const r = runBash(gate, {EVENT_NAME: event, GOLANGCI_LINT_RESULT: 'failure'});
    assert.ifError(r.error);
    assert.equal(r.status, 0, `go-lint must stay report-only on ${event}: ${r.stderr}`);
  }
  const pass = runBash(gate, {EVENT_NAME: 'merge_group', GOLANGCI_LINT_RESULT: 'success'});
  assert.ifError(pass.error);
  assert.equal(pass.status, 0, 'go-lint success must pass: ' + pass.stderr);
});

test('duplication gate is report-only on pull request and merge group in real bash', () => {
  const dupSource = workflowSource.slice(
    workflowSource.indexOf('  duplication:'),
    workflowSource.indexOf('  smoke-sanity:'),
  );
  assert.match(dupSource, /continue-on-error:\s*true/i, 'duplication scan should report without failing');
  assert.match(dupSource, /upload-artifact@[\w]+\n[\s\S]*?name: duplication-report/i, 'duplication report should be uploaded as an artifact');
  const gate = shellFor('Enforce duplication gate');
  for (const event of ['pull_request', 'merge_group']) {
    const r = runBash(gate, {EVENT_NAME: event, DUPLICATION_RESULT: 'failure'});
    assert.ifError(r.error);
    assert.equal(r.status, 0, `duplication must stay report-only on ${event}: ${r.stderr}`);
  }
  for (const event of ['push', 'workflow_dispatch']) {
    const r = runBash(gate, {EVENT_NAME: event, DUPLICATION_RESULT: 'failure'});
    assert.ifError(r.error);
    assert.equal(r.status, 1, `duplication must fail closed on ${event}: ${r.stderr}`);
  }
  const pass = runBash(gate, {EVENT_NAME: 'push', DUPLICATION_RESULT: 'success'});
  assert.ifError(pass.error);
  assert.equal(pass.status, 0, 'duplication success must pass: ' + pass.stderr);
});

test('quality gates aggregate every runtime job and enforce them in real bash', () => {
  const gates = workflowSource.slice(workflowSource.indexOf('  quality-gates:'));
  for (const dep of ['change-scope', 'docs-governance', 'frontend-contract', 'backend-tests', 'go-lint', 'duplication', 'smoke-sanity']) {
    assert.match(gates, new RegExp(`- ${dep}`), `quality gates should depend on ${dep}`);
  }
  assert.match(gates, /if:\s*always\(\)/, 'quality gates must run even when a dependency fails');
  const body = shellFor('Enforce quality results');
  assert.match(gates, /EXPECTED_RESULT:\s*success/i, 'quality gates should declare the passing outcome once');
  assert.match(body, /DOCS_GOVERNANCE_RESULT/);
  assert.match(body, /BACKEND_TESTS_RESULT/);
  assert.match(body, /FRONTEND_CONTRACT_RESULT/);
  assert.match(body, /GO_LINT_RESULT/);
  assert.match(body, /DUPLICATION_RESULT/);
  assert.match(body, /SMOKE_SANITY_RESULT/);

  const base = {
    EVENT_NAME: 'push',
    CHANGE_SCOPE_RESULT: 'success',
    EXPECTED_RESULT: 'success',
    FRONTEND_CONTRACT_SCOPE: 'true',
    SMOKE_SANITY_SCOPE: 'true',
    DOCS_GOVERNANCE_RESULT: 'success',
    FRONTEND_CONTRACT_RESULT: 'success',
    BACKEND_TESTS_RESULT: 'success',
    GO_LINT_RESULT: 'success',
    DUPLICATION_RESULT: 'success',
    SMOKE_SANITY_RESULT: 'success',
  };
  const allOk = runBash(body, base);
  assert.ifError(allOk.error);
  assert.equal(allOk.status, 0, 'all-success delegation must pass: ' + allOk.stderr);

  // Hard gate runtime jobs that are never scope-skippable.
  for (const [key, label] of [['DOCS_GOVERNANCE_RESULT', 'docs governance'], ['BACKEND_TESTS_RESULT', 'backend tests'], ['DUPLICATION_RESULT', 'duplication']]) {
    const r = runBash(body, {...base, [key]: 'failure'});
    assert.ifError(r.error);
    assert.equal(r.status, 1, `${label} failure must fail closed regardless of scope: ${r.stderr}`);
  }

  // Frontend contract is required on non-PR and on scope-relevant PRs, optional otherwise.
  const feNeeded = runBash(body, {...base, EVENT_NAME: 'pull_request', FRONTEND_CONTRACT_SCOPE: 'true', FRONTEND_CONTRACT_RESULT: 'failure'});
  assert.ifError(feNeeded.error);
  assert.equal(feNeeded.status, 1, 'frontend contract failure must fail closed when scope requires it: ' + feNeeded.stderr);
  const feSkipped = runBash(body, {...base, EVENT_NAME: 'pull_request', FRONTEND_CONTRACT_SCOPE: 'false', FRONTEND_CONTRACT_RESULT: 'skipped'});
  assert.ifError(feSkipped.error);
  assert.equal(feSkipped.status, 0, 'an out-of-scope pull request may skip frontend contract: ' + feSkipped.stderr);

  // Go-lint is enforced off push; push keeps it report-only (the job already exits 0).
  const goPr = runBash(body, {...base, EVENT_NAME: 'pull_request', GO_LINT_RESULT: 'failure'});
  assert.ifError(goPr.error);
  assert.equal(goPr.status, 1, 'go-lint failure must fail closed on a pull request: ' + goPr.stderr);
  const goPush = runBash(body, {...base, EVENT_NAME: 'push', GO_LINT_RESULT: 'failure'});
  assert.ifError(goPush.error);
  assert.equal(goPush.status, 0, 'go-lint failure must stay report-only on push: ' + goPush.stderr);

  // Smoke sanity follows the scope-required toggle.
  const smokeNeeded = runBash(body, {...base, EVENT_NAME: 'pull_request', SMOKE_SANITY_SCOPE: 'true', SMOKE_SANITY_RESULT: 'failure'});
  assert.ifError(smokeNeeded.error);
  assert.equal(smokeNeeded.status, 1, 'smoke sanity failure must fail closed when scope requires it: ' + smokeNeeded.stderr);
  const smokeSkipped = runBash(body, {...base, EVENT_NAME: 'pull_request', SMOKE_SANITY_SCOPE: 'false', SMOKE_SANITY_RESULT: 'skipped'});
  assert.ifError(smokeSkipped.error);
  assert.equal(smokeSkipped.status, 0, 'an out-of-scope pull request may skip smoke sanity: ' + smokeSkipped.stderr);
  for (const event of ['pull_request', 'push', 'workflow_dispatch', 'merge_group']) {
    for (const result of ['failure', 'cancelled', 'skipped', '']) {
      const failedScope = runBash(body, {...base, EVENT_NAME: event, CHANGE_SCOPE_RESULT: result});
      assert.ifError(failedScope.error);
      assert.equal(failedScope.status, 1, `${event} must reject a ${result || 'missing'} scope result`);
    }
    for (const [scope, result] of [['FRONTEND_CONTRACT_SCOPE', 'FRONTEND_CONTRACT_RESULT'], ['SMOKE_SANITY_SCOPE', 'SMOKE_SANITY_RESULT']]) {
      for (const failed of ['failure', 'cancelled', '']) {
        const unexpectedFailure = runBash(body, {...base, EVENT_NAME: event, [scope]: 'false', [result]: failed});
        assert.ifError(unexpectedFailure.error);
        assert.equal(unexpectedFailure.status, 1, `${event} must reject ${result}=${failed || 'missing'} even outside scope`);
      }
      for (const scopeValue of ['true', '']) {
        const missingRequired = runBash(body, {...base, EVENT_NAME: event, [scope]: scopeValue, [result]: 'skipped'});
        assert.ifError(missingRequired.error);
        assert.equal(missingRequired.status, 1, `${event} must reject a skipped result with required or missing scope`);
      }
      if (event !== 'pull_request') {
        const nonPrSkip = runBash(body, {...base, EVENT_NAME: event, [scope]: 'false', [result]: 'skipped'});
        assert.ifError(nonPrSkip.error);
        assert.equal(nonPrSkip.status, 1, `${event} must run ${result} regardless of PR scope`);
      }
    }
  }
});

test('docs governance blocks UI quality drift as a strict non-advisory step', () => {
  assert.match(
    workflowSource,
    /- name: Check UI quality gate\s*\n\s*run:\s*npm run check:ui-quality-gate/i,
    'docs governance should run the strict UI quality gate',
  );
  assert.doesNotMatch(
    workflowSource,
    /Check UI quality gate[\s\S]{0,120}continue-on-error:\s*true/i,
    'UI quality policy drift must block the workflow',
  );
  assert.match(
    workflowSource,
    /- name: Run tests\/scripts catch-all[\s\S]*?run:\s*npm run test:scripts/i,
    'docs governance should run the tests/scripts catch-all so no governance test silently goes unrun',
  );
});

test('docs governance validates the PR governance template and body, rerunning on edits', () => {
  assert.match(
    workflowSource,
    /pull_request:\s*\n\s*types:\s*\[opened, synchronize, reopened, edited, ready_for_review\]/,
    'quality workflow should rerun when a pull request body is edited',
  );
  assert.match(
    workflowSource,
    /- name: Check PR governance template\s*\n\s*run:\s*npm run check:pr-governance/i,
    'docs governance should check the PR governance template',
  );
  assert.match(
    workflowSource,
    /- name: Validate PR governance body[\s\S]*?if:\s*github\.event_name\s*==\s*'pull_request' && github\.event\.pull_request\.user\.login !=\s*'dependabot\[bot\]'[\s\S]*?node scripts\/check-pr-governance\.mjs --event "\$GITHUB_EVENT_PATH"/,
    'docs governance should validate the pull request body on pull request events, excluding dependabot',
  );
});

test('docs governance does not checkout an unused pantheon-base foundation copy', () => {
  assert.doesNotMatch(workflowSource, /Checkout pantheon-base foundation/i);
  assert.doesNotMatch(workflowSource, /path:\s*pantheon-base-foundation/i);
});
