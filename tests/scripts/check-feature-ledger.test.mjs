import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(testDir, '../..');
const scriptPath = path.join(repoRoot, 'scripts', 'harness', 'check-feature-ledger.mjs');

function runChecker(root, args = ['--json', '--strict']) {
  const result = spawnSync(process.execPath, [scriptPath, ...args, '--root', root], {
    cwd: root,
    encoding: 'utf8',
    shell: false,
  });
  return {
    code: result.status ?? 0,
    stdout: result.stdout,
    stderr: result.stderr,
  };
}

function createFixture() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'feature-ledger-'));
  fs.mkdirSync(path.join(root, 'schema', 'generated', 'business'), { recursive: true });
  writeJson(path.join(root, 'schema', 'generated', 'business', 'ticket.json'), {
    name: 'ticket',
    scope: 'business',
    displayName: 'Ticket',
    metadata: {
      owner: 'ops',
      boundedContext: 'support',
      sourceMode: 'manual',
      autoRecycle: false,
    },
    model: {
      tableName: 'biz_ticket',
    },
  });
  writeLedger(root, [
    {
      moduleKey: 'business.ticket',
      name: 'ticket',
      scope: 'business',
      displayName: 'Ticket',
      owner: 'ops',
      boundedContext: 'support',
      sourceMode: 'manual',
      source: 'manual',
      maturity: 'experimental',
      status: 'pending_activation',
      tableName: 'biz_ticket',
      schemaPath: 'business/ticket.json',
      builtIn: false,
      autoRecycle: false,
    },
  ]);
  return root;
}

function writeLedger(root, entries, issues = []) {
  writeJson(path.join(root, 'schema', 'generated', 'feature-ledger.json'), {
    version: 1,
    sourceOfTruth: {
      registrations: 'system_module_registration',
      schemas: 'schema/generated',
      snapshot: 'schema/generated/feature-ledger.json',
    },
    entries,
    ...(issues.length > 0 ? { issues } : {}),
  });
}

function writeJson(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, `${JSON.stringify(value, null, 2)}\n`, 'utf8');
}

test('check-feature-ledger accepts a complete generated schema ledger snapshot', () => {
  const root = createFixture();
  try {
    const result = runChecker(root);

    assert.equal(result.code, 0, `${result.stderr}\n${result.stdout}`);
    const payload = JSON.parse(result.stdout);
    assert.equal(payload.findingCount, 0);
    assert.equal(payload.schemaCount, 1);
    assert.equal(payload.entryCount, 1);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('check-feature-ledger fails when the snapshot is missing', () => {
  const root = createFixture();
  try {
    fs.rmSync(path.join(root, 'schema', 'generated', 'feature-ledger.json'), { force: true });

    const result = runChecker(root);

    assert.equal(result.code, 1);
    const payload = JSON.parse(result.stdout);
    assert.equal(payload.findingCount, 1);
    assert.equal(payload.findings[0].code, 'ledger_snapshot_missing');
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('check-feature-ledger treats an empty generated schema tree without a snapshot as a no-op', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'feature-ledger-empty-'));
  try {
    fs.mkdirSync(path.join(root, 'schema', 'generated'), { recursive: true });

    const result = runChecker(root);

    assert.equal(result.code, 0, `${result.stderr}\n${result.stdout}`);
    const payload = JSON.parse(result.stdout);
    assert.equal(payload.findingCount, 0);
    assert.equal(payload.schemaCount, 0);
    assert.equal(payload.entryCount, 0);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('check-feature-ledger fails when a generated schema is missing from the ledger', () => {
  const root = createFixture();
  try {
    writeLedger(root, []);

    const result = runChecker(root);

    assert.equal(result.code, 1);
    const payload = JSON.parse(result.stdout);
    assert.equal(payload.findingCount, 1);
    assert.equal(payload.findings[0].code, 'ledger_entry_missing');
    assert.equal(payload.findings[0].moduleKey, 'business.ticket');
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('check-feature-ledger fails on snapshot issues and required field gaps', () => {
  const root = createFixture();
  try {
    writeLedger(
      root,
      [
        {
          moduleKey: 'business.ticket',
          name: 'ticket',
          scope: 'business',
          displayName: 'Ticket',
          owner: '',
          boundedContext: 'support',
          sourceMode: 'manual',
          source: 'manual',
          maturity: 'experimental',
          status: 'pending_activation',
          tableName: 'biz_ticket',
          schemaPath: 'business/ticket.json',
          builtIn: false,
          autoRecycle: false,
        },
      ],
      [
        {
          moduleKey: 'business.ticket',
          severity: 'warn',
          code: 'owner_missing',
          field: 'owner',
          detail: 'owner is required for feature ledger completeness',
        },
      ],
    );

    const result = runChecker(root);

    assert.equal(result.code, 1);
    const payload = JSON.parse(result.stdout);
    assert.equal(payload.findingCount, 2);
    assert.deepEqual(
      payload.findings.map((finding) => finding.code).sort(),
      ['ledger_issue_open', 'ledger_required_field_missing'],
    );
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});
