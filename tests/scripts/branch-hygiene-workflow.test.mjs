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
  // The push-to-main trigger was intentionally removed in #142 ("simplify
  // workflows for solo developer"); the scheduled fallback now carries the
  // job. This test previously still required the removed trigger because it
  // was never run in CI.
  assert.doesNotMatch(
    workflowSource,
    /^\s*push:/m,
    'branch hygiene no longer runs on push (intentionally removed in #142)',
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
    /uses:\s*actions\/checkout@[a-f0-9]{40}/i,
    'branch hygiene should pin checkout',
  );
  assert.match(
    workflowSource,
    /persist-credentials:\s*false/i,
    'branch hygiene should disable checkout credential persistence',
  );
  assert.match(
    workflowSource,
    /uses:\s*actions\/setup-node@[a-f0-9]{40}/i,
    'branch hygiene should pin setup-node',
  );
  assert.match(
    workflowSource,
    /run:\s*node scripts\/cleanup-github-branches\.mjs/i,
    'branch hygiene should delegate cleanup logic to the dedicated script',
  );
});
