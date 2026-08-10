import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawn } from 'node:child_process';
import { after, test } from 'node:test';
import assert from 'node:assert/strict';
import net from 'node:net';
import { fileURLToPath } from 'node:url';

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const runnerScript = path.join(frontendRoot, 'scripts', 'run-smoke-suite.mjs');
const fixtureServerScript = path.join(frontendRoot, 'scripts', 'test-fixtures', 'bind-ready-server.mjs');
const fakePlaywrightCli = path.join(frontendRoot, 'scripts', 'test-fixtures', 'fake-playwright-cli.mjs');
const cleanupRecorderScript = path.join(frontendRoot, 'scripts', 'test-fixtures', 'record-cleanup.mjs');
const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-smoke-runner-'));

after(() => {
  fs.rmSync(tmpRoot, { recursive: true, force: true });
});

function getFreePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.listen(0, '127.0.0.1', () => {
      const address = server.address();
      if (!address || typeof address === 'string') {
        reject(new Error('Failed to allocate free port'));
        return;
      }
      const { port } = address;
      server.close((error) => {
        if (error) {
          reject(error);
          return;
        }
        resolve(port);
      });
    });
    server.once('error', reject);
  });
}

function spawnCommand(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      cwd: frontendRoot,
      stdio: ['ignore', 'pipe', 'pipe'],
      shell: false,
      ...options,
    });
    let stdout = '';
    let stderr = '';
    child.stdout?.setEncoding('utf8');
    child.stderr?.setEncoding('utf8');
    child.stdout?.on('data', (chunk) => {
      stdout += chunk;
    });
    child.stderr?.on('data', (chunk) => {
      stderr += chunk;
    });
    child.once('error', reject);
    child.once('exit', (code, signal) => {
      resolve({ code: code ?? 0, signal: signal ?? null, stdout, stderr });
    });
  });
}

function startOccupantServer(port) {
  return new Promise((resolve, reject) => {
    const server = net.createServer((socket) => {
      socket.end('HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok');
    });
    server.listen(port, '127.0.0.1', () => resolve(server));
    server.once('error', reject);
  });
}

test('run-smoke-suite waits for its own ready signal before invoking playwright', async () => {
  const port = await getFreePort();
  const markerPath = path.join(tmpRoot, `marker-${port}.json`);
  const configPath = path.join(tmpRoot, `config-${port}.txt`);
  fs.writeFileSync(configPath, 'placeholder');

  const result = await spawnCommand(process.execPath, [
    runnerScript,
    '--host',
    '127.0.0.1',
    '--port',
    String(port),
    '--timeout',
    '10000',
    '--server-script',
    fixtureServerScript,
    '--playwright-cli',
    fakePlaywrightCli,
    '--config',
    configPath,
    '--',
    'tests/smoke/fake.spec.ts',
  ], {
    env: {
      ...process.env,
      PANTHEON_FAKE_PLAYWRIGHT_MARKER: markerPath,
    },
  });

  assert.equal(result.code, 0, `${result.stderr}\n${result.stdout}`);
  assert.equal(fs.existsSync(markerPath), true);
  const payload = JSON.parse(fs.readFileSync(markerPath, 'utf8'));
  assert.equal(payload.baseUrl, `http://127.0.0.1:${port}`);
  assert.match(result.stdout, /Fixture ready at http:\/\/127\.0\.0\.1:/);
  assert.doesNotMatch(result.stdout, /__PANTHEON_SMOKE_VITE_READY__/);
});

test('run-smoke-suite fails fast when the target port is already occupied', async () => {
  const port = await getFreePort();
  const configPath = path.join(tmpRoot, `config-conflict-${port}.txt`);
  const markerPath = path.join(tmpRoot, `marker-conflict-${port}.json`);
  fs.writeFileSync(configPath, 'placeholder');
  const occupant = await startOccupantServer(port);

  try {
    const result = await spawnCommand(process.execPath, [
      runnerScript,
      '--host',
      '127.0.0.1',
      '--port',
      String(port),
      '--timeout',
      '5000',
      '--server-script',
      fixtureServerScript,
      '--playwright-cli',
      fakePlaywrightCli,
      '--config',
      configPath,
      '--',
      'tests/smoke/fake.spec.ts',
    ], {
      env: {
        ...process.env,
        PANTHEON_FAKE_PLAYWRIGHT_MARKER: markerPath,
      },
    });

    assert.notEqual(result.code, 0);
    assert.equal(fs.existsSync(markerPath), false);
    assert.match(result.stderr, /EADDRINUSE|exited before ready signal|listen EADDRINUSE/);
  } finally {
    await new Promise((resolve, reject) => {
      occupant.close((error) => {
        if (error) {
          reject(error);
          return;
        }
        resolve();
      });
    });
  }
});

test('run-smoke-suite cleans generated modules before and after execution and fixtures around opted-in runs', async () => {
  const port = await getFreePort();
  const markerPath = path.join(tmpRoot, `cleanup-${port}.jsonl`);
  const configPath = path.join(tmpRoot, `config-cleanup-${port}.txt`);
  fs.writeFileSync(configPath, 'placeholder');

  const result = await spawnCommand(process.execPath, [
    runnerScript,
    '--host',
    '127.0.0.1',
    '--port',
    String(port),
    '--timeout',
    '10000',
    '--server-script',
    fixtureServerScript,
    '--playwright-cli',
    fakePlaywrightCli,
    '--generated-cleanup-script',
    cleanupRecorderScript,
    '--fixture-cleanup-script',
    cleanupRecorderScript,
    '--cleanup-fixtures',
    'all',
    '--config',
    configPath,
    '--',
    'tests/smoke/fake.spec.ts',
  ], {
    env: {
      ...process.env,
      PANTHEON_CLEANUP_MARKER: markerPath,
    },
  });

  assert.equal(result.code, 0, `${result.stderr}\n${result.stdout}`);
  const entries = fs
    .readFileSync(markerPath, 'utf8')
    .trim()
    .split('\n')
    .filter(Boolean)
    .map((line) => JSON.parse(line));
  assert.deepEqual(
    entries.map((entry) => entry.args),
    [
      ['--kind', 'generated', '--phase', 'pre'],
      ['--kind', 'fixture', '--phase', 'pre', '--scope', 'all'],
      ['--kind', 'fixture', '--phase', 'post', '--scope', 'all'],
      ['--kind', 'generated', '--phase', 'post'],
    ],
  );
});

test('run-smoke-suite skips fixture cleanup in preserve mode but still cleans generated modules', async () => {
  const port = await getFreePort();
  const markerPath = path.join(tmpRoot, `cleanup-preserve-${port}.jsonl`);
  const configPath = path.join(tmpRoot, `config-cleanup-preserve-${port}.txt`);
  fs.writeFileSync(configPath, 'placeholder');

  const result = await spawnCommand(process.execPath, [
    runnerScript,
    '--host',
    '127.0.0.1',
    '--port',
    String(port),
    '--timeout',
    '10000',
    '--server-script',
    fixtureServerScript,
    '--playwright-cli',
    fakePlaywrightCli,
    '--generated-cleanup-script',
    cleanupRecorderScript,
    '--fixture-cleanup-script',
    cleanupRecorderScript,
    '--cleanup-fixtures',
    'all',
    '--config',
    configPath,
    '--',
    'tests/smoke/fake.spec.ts',
  ], {
    env: {
      ...process.env,
      PANTHEON_CLEANUP_MARKER: markerPath,
      PANTHEON_SMOKE_PRESERVE_FIXTURES: '1',
    },
  });

  assert.equal(result.code, 0, `${result.stderr}\n${result.stdout}`);
  const entries = fs
    .readFileSync(markerPath, 'utf8')
    .trim()
    .split('\n')
    .filter(Boolean)
    .map((line) => JSON.parse(line));
  assert.deepEqual(
    entries.map((entry) => entry.args),
    [
      ['--kind', 'generated', '--phase', 'pre'],
      ['--kind', 'generated', '--phase', 'post'],
    ],
  );
});

test('run-smoke-suite propagates a pre-cleanup failure and does not invoke playwright', async () => {
  const port = await getFreePort();
  const cleanupMarkerPath = path.join(tmpRoot, `cleanup-failure-${port}.jsonl`);
  const playwrightMarkerPath = path.join(tmpRoot, `playwright-failure-${port}.json`);
  const configPath = path.join(tmpRoot, `config-cleanup-failure-${port}.txt`);
  fs.writeFileSync(configPath, 'placeholder');

  const result = await spawnCommand(process.execPath, [
    runnerScript,
    '--host',
    '127.0.0.1',
    '--port',
    String(port),
    '--timeout',
    '10000',
    '--server-script',
    fixtureServerScript,
    '--playwright-cli',
    fakePlaywrightCli,
    '--generated-cleanup-script',
    cleanupRecorderScript,
    '--config',
    configPath,
    '--',
    'tests/smoke/fake.spec.ts',
  ], {
    env: {
      ...process.env,
      PANTHEON_CLEANUP_MARKER: cleanupMarkerPath,
      PANTHEON_CLEANUP_FAIL_ON: 'generated:pre',
      PANTHEON_FAKE_PLAYWRIGHT_MARKER: playwrightMarkerPath,
    },
  });

  assert.equal(result.code, 7, `${result.stderr}\n${result.stdout}`);
  assert.equal(fs.existsSync(playwrightMarkerPath), false);
  assert.match(result.stderr, /generated cleanup exited with code 7/);
});
