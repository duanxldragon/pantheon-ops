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
