import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const workflowPath = path.resolve('.github/workflows/branch-hygiene.yml');

test('branch hygiene workflow runs independently of pull_request.closed and invokes the cleanup script', () => {
  assert.ok(fs.existsSync(workflowPath), 'branch hygiene workflow should exist');
  const workflowSource = fs.readFileSync(workflowPath, 'utf8');

  assert.match(
    workflowSource,
    /name:\s*Branch Hygiene/i,
    'branch hygiene workflow should be clearly named',
  );
  assert.match(
    workflowSource,
    /on:\s*\n\s*push:\s*\n\s*branches:\s*\n\s*-\s*main/i,
    'branch hygiene should run after pushes to main',
  );
  assert.match(
    workflowSource,
    /\n\s*schedule:\s*\n\s*-\s*cron:\s*["'][^"']+["']/i,
    'branch hygiene should include a scheduled fallback trigger',
  );
  assert.match(
    workflowSource,
    /\n\s*workflow_dispatch:\s*\n/i,
    'branch hygiene should allow manual dispatch',
  );
  assert.doesNotMatch(
    workflowSource,
    /pull_request:\s*\n[\s\S]*-\s*closed/i,
    'branch hygiene must not depend on pull_request.closed',
  );
  assert.match(
    workflowSource,
    /permissions:\s*\n\s*contents:\s*write\s*\n\s*pull-requests:\s*read/i,
    'branch hygiene should request only the permissions needed for branch deletion and PR inspection',
  );
  assert.match(
    workflowSource,
    /uses:\s*actions\/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10/i,
    'branch hygiene should pin checkout',
  );
  assert.match(
    workflowSource,
    /persist-credentials:\s*false/i,
    'branch hygiene should disable checkout credential persistence',
  );
  assert.match(
    workflowSource,
    /uses:\s*actions\/setup-node@48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e/i,
    'branch hygiene should pin setup-node',
  );
  assert.match(
    workflowSource,
    /run:\s*node scripts\/cleanup-github-branches\.mjs/i,
    'branch hygiene should delegate cleanup logic to the dedicated script',
  );
});
