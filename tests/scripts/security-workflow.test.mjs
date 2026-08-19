import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const workflowPath = path.resolve('.github/workflows/security.yml');
const workflowSource = fs.readFileSync(workflowPath, 'utf8');

test('security gates aggregate dependency vulnerability results without blocking skipped PR runs', () => {
  assert.match(
    workflowSource,
    /dependency-vulnerabilities:\s*\n[\s\S]*name:\s*Dependency Vulnerabilities/i,
    'security workflow should keep the dependency vulnerability job',
  );
  assert.match(
    workflowSource,
    /security-gates:[\s\S]*needs:[\s\S]*-\s*dependency-vulnerabilities/i,
    'security gates should aggregate dependency vulnerability results',
  );
  assert.match(
    workflowSource,
    /DEPENDENCY_VULNERABILITIES_RESULT:\s*\$\{\{\s*needs\.dependency-vulnerabilities\.result\s*\}\}/i,
    'security gates should read the dependency vulnerability job result',
  );
  assert.match(
    workflowSource,
    /if\s+\[\s*"\$\{DEPENDENCY_VULNERABILITIES_RESULT\}"\s*!=\s*"success"\s*\]\s*&&\s*\\[\s\S]*\[\s*"\$\{DEPENDENCY_VULNERABILITIES_RESULT\}"\s*!=\s*"skipped"\s*\]/i,
    'security gates should allow skipped dependency scans on PR and merge-group events',
  );
});
