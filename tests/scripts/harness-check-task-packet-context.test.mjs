import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const checker = path.resolve(testDir, '../../scripts/harness/check-task-packet.mjs');

function taskPacket(workspaceContext = '') {
  return `# Task Packet: cross-repo-fixture

## Goal

Validate workspace context.

## Primary Layer

inheritance-sync

${workspaceContext}
## Dependency Layers

- none

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-base
- Coverage Dimensions:
  - architecture-fitness

## Contract Anchors

- \`AGENTS.md\`

## Scope

### In

- validate context

### Out

- product code

## Expected Files

### Create

- none

### Modify

- none

### Do Not Touch

- product code

## Implementation Notes

- fixture only

## Verification Plan

- \`node --test\`

## Linkage

- Task ID: \`cross-repo-fixture\`
- OpenSpec Change: \`none\`
- Superpowers Plan: \`none\`
- Plan References: \`none\`
- Evidence Directory: \`.harness/evidence/cross-repo-fixture/\`
- Review File: \`none\`

## Evidence Required

- command output

## Human Gates

- none

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
`;
}

function runChecker(content) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-base-task-context-'));
  try {
    const taskDir = path.join(root, 'docs', 'harness', 'tasks');
    fs.mkdirSync(taskDir, { recursive: true });
    fs.writeFileSync(path.join(taskDir, 'cross-repo-fixture.task.md'), content, 'utf8');
    return spawnSync(process.execPath, [checker, '--root', root], {
      encoding: 'utf8',
    });
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
}

test('inheritance-sync task packets require Workspace Context', () => {
  const result = runChecker(taskPacket());

  assert.equal(result.status, 1);
  assert.match(result.stdout, /requires section "## Workspace Context"/);
});

test('inheritance-sync task packets accept complete Workspace Context', () => {
  const workspaceContext = `## Workspace Context

- Target Repository: pantheon-base
- Repository Role: foundation-source
- Upstream Dependencies: pantheon-harness
- Downstream Consumers: pantheon-ops
- Sync Expectation: required
- Release Requirement: foundation-release

`;
  const result = runChecker(taskPacket(workspaceContext));

  assert.equal(result.status, 0, result.stdout + result.stderr);
  assert.match(result.stdout, /\[PASS\]/);
});
