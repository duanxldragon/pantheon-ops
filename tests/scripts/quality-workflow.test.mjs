import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const workflowPath = path.resolve('.github/workflows/quality.yml');
const workflowSource = fs.readFileSync(workflowPath, 'utf8');

test('governance-only changes can skip runtime gates without failing Quality Gates', () => {
  assert.match(
    workflowSource,
    /uses:\s*dorny\/paths-filter@ceb8a2b8f2d89434be7ff52d3de7ec3738c5cc9d/i,
    'quality workflow should pin paths-filter to a revision that supports predicate-quantifier',
  );
  assert.match(
    workflowSource,
    /filters:\s*\|[\s\S]*all:\s*\n\s*-\s*'\*\*'/i,
    'quality workflow should count all changed files before classifying scopes',
  );
  assert.match(
    workflowSource,
    /docs_only:\s*\$\{\{[\s\S]*steps\.scope\.outputs\.all_count\s*==\s*steps\.scope\.outputs\.docs_only_count[\s\S]*\}\}/i,
    'change scope should classify docs-only by comparing matched docs count with all changed files',
  );
  assert.match(
    workflowSource,
    /governance_only:\s*\$\{\{[\s\S]*steps\.scope\.outputs\.all_count\s*==\s*steps\.scope\.outputs\.governance_only_count[\s\S]*\}\}/i,
    'change scope should classify governance-only by comparing matched governance count with all changed files',
  );
  assert.doesNotMatch(
    workflowSource,
    /predicate-quantifier:\s*every/i,
    'quality workflow should not require every whitelist glob to match the same file',
  );
  assert.match(
    workflowSource,
    /frontend-contract:[\s\S]*if:\s*\$\{\{\s*github\.event_name\s*==\s*'merge_group'\s*\|\|[\s\S]*needs\.change-scope\.outputs\.governance_only\s*!=\s*'true'/i,
    'frontend contract should be skipped for governance-only pull requests and pushes',
  );
  assert.match(
    workflowSource,
    /backend-tests:[\s\S]*if:\s*\$\{\{\s*github\.event_name\s*==\s*'merge_group'\s*\|\|[\s\S]*needs\.change-scope\.outputs\.governance_only\s*!=\s*'true'/i,
    'backend tests should be skipped for governance-only pull requests and pushes',
  );
  assert.match(
    workflowSource,
    /smoke-sanity:[\s\S]*if:\s*\$\{\{\s*github\.event_name\s*==\s*'merge_group'\s*\|\|[\s\S]*needs\.change-scope\.outputs\.governance_only\s*!=\s*'true'/i,
    'smoke sanity should be skipped for governance-only pull requests and pushes',
  );
  assert.match(
    workflowSource,
    /go-lint:[\s\S]*if:\s*\$\{\{\s*github\.event_name\s*==\s*'merge_group'\s*\|\|[\s\S]*needs\.change-scope\.outputs\.governance_only\s*!=\s*'true'/i,
    'go lint should be skipped for governance-only pull requests and pushes',
  );
  assert.match(
    workflowSource,
    /quality-gates:[\s\S]*needs:[\s\S]*- go-lint[\s\S]*if:\s*always\(\)/i,
    'quality gates should depend on go-lint results',
  );
  assert.match(
    workflowSource,
    /GO_LINT_RESULT.*!=.*"success"/i,
    'quality gates should enforce go-lint as a runtime gate',
  );
  assert.match(
    workflowSource,
    /Run golangci-lint[\s\S]*continue-on-error:\s*true/i,
    'go-lint step should use continue-on-error so findings are reported without failing the job',
  );
  assert.match(
    workflowSource,
    /Enforce go-lint gate[\s\S]*EVENT_NAME[\s\S]*pull_request[\s\S]*merge_group[\s\S]*report-only/i,
    'go-lint gate should be report-only for pull_request and merge_group events',
  );
  assert.match(
    workflowSource,
    /Quality Gates Summary[\s\S]*GITHUB_STEP_SUMMARY/i,
    'quality gates should write a job summary to GITHUB_STEP_SUMMARY',
  );
  assert.match(
    workflowSource,
    /if\s+\[\s*"\$\{GOVERNANCE_ONLY\}"\s*=\s*"true"\s*\];\s*then[\s\S]*RUNTIME_GATES_REQUIRED=false/i,
    'quality gates should only require runtime gates when the change is not governance-only',
  );
  for (const governancePath of [
    'package.json',
    'package-lock.json',
    'scripts/cleanup-github-branches.mjs',
    'scripts/collect-merge-targets.mjs',
    'scripts/execute-merge.mjs',
    'config/ui-quality-gate.json',
  ]) {
    assert.match(
      workflowSource,
      new RegExp(`- '${governancePath.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}'`),
      `${governancePath} should be classified as governance-only`,
    );
  }
});

test('docs governance blocks policy drift in the UI quality gate', () => {
  assert.match(
    workflowSource,
    /Enforce UI quality gate\s*\n\s*run:\s*npm run check:ui-quality-gate/i,
    'docs governance should run the strict UI quality gate without advisory continue-on-error handling',
  );
  assert.doesNotMatch(
    workflowSource,
    /Enforce UI quality gate[\s\S]{0,120}continue-on-error:\s*true/i,
    'UI quality policy drift must block the workflow',
  );
});

test('go lint scopes pull requests and merge groups to new code', () => {
  const goLintSource = workflowSource.slice(
    workflowSource.indexOf('  go-lint:'),
    workflowSource.indexOf('  duplication:'),
  );

  assert.match(goLintSource, /PR_BASE_SHA:\s*\$\{\{ github\.event\.pull_request\.base\.sha \}\}/);
  assert.match(goLintSource, /MERGE_GROUP_BASE_SHA:\s*\$\{\{ github\.event\.merge_group\.base_sha \}\}/);
  assert.match(
    goLintSource,
    /if \[ "\$\{EVENT_NAME\}" = "pull_request" \] && \[ -n "\$\{PR_BASE_SHA\}" \]; then[\s\S]*?args=--new-from-rev=\$\{PR_BASE_SHA\} \.\/\.\.\.[\s\S]*?elif \[ "\$\{EVENT_NAME\}" = "merge_group" \] && \[ -n "\$\{MERGE_GROUP_BASE_SHA\}" \]; then[\s\S]*?args=--new-from-rev=\$\{MERGE_GROUP_BASE_SHA\} \.\/\.\.\./,
    'pull requests and merge groups must both lint only newly introduced code',
  );
  assert.match(
    goLintSource,
    /if \[ "\$\{EVENT_NAME\}" = "push" \]; then[\s\S]*?Full-repo lint is report-only on push/,
    'only push events may retain the advisory full-repository lint path',
  );
});

test('docs governance validates the PR governance template and pull request body', () => {
  assert.match(
    workflowSource,
    /pull_request:\s*\n\s*types:\s*\n[\s\S]*-\s*edited/i,
    'quality workflow should rerun when a pull request body is edited',
  );
  assert.match(
    workflowSource,
    /Check PR governance template[\s\S]*npm run check:pr-governance/i,
    'docs governance should check the PR governance template',
  );
  assert.match(
    workflowSource,
    /Validate PR governance body[\s\S]*if:\s*github\.event_name\s*==\s*'pull_request'[\s\S]*node scripts\/check-pr-governance\.mjs --event "\$GITHUB_EVENT_PATH"/i,
    'docs governance should validate the pull request body on pull request events',
  );
});

test('docs governance does not checkout an unused pantheon-base foundation copy', () => {
  assert.doesNotMatch(
    workflowSource,
    /Checkout pantheon-base foundation/i,
    'quality workflow should not spend CI time on an unused secondary checkout',
  );
  assert.doesNotMatch(
    workflowSource,
    /path:\s*pantheon-base-foundation/i,
    'quality workflow should not materialize an unused pantheon-base-foundation path',
  );
});
