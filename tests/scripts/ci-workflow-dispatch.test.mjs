import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const workflowPath = path.resolve('.github/workflows/ci.yml');
const workflowSource = fs.readFileSync(workflowPath, 'utf8');

test('ci supports manual recovery for an exact commit', () => {
  assert.match(
    workflowSource,
    /on:\s*\n\s*workflow_dispatch:\s*\n\s*push:/i,
    'CI should support workflow_dispatch when an exact commit needs recovery validation',
  );
});

// Keep the catch-all wired: without it every test file that has no dedicated
// npm script (13 of them on 2026-09-23) silently never runs in CI, which is
// how a strict-mode change in check-boundaries.mjs broke its own test unnoticed.
test('ci runs the tests/scripts catch-all so no governance test is left unrun', () => {
  assert.match(
    workflowSource,
    /npm run test:scripts\b/,
    'CI must run the tests/scripts catch-all',
  );
});
