import { spawn } from 'node:child_process';
import os from 'node:os';
import path from 'node:path';
import process from 'node:process';

const readyToken = '__PANTHEON_SMOKE_VITE_READY__';

function readArg(flag, fallback = '') {
  const index = process.argv.indexOf(flag);
  if (index < 0 || index + 1 >= process.argv.length) {
    return fallback;
  }
  return process.argv[index + 1];
}

function normalizeUrl(url) {
  return String(url || '').replace(/\/+$/, '');
}

async function waitForHttp(url, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  let lastError = null;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        return;
      }
      lastError = new Error(`HTTP ${response.status}`);
    } catch (error) {
      lastError = error;
    }
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  throw lastError ?? new Error(`Timed out waiting for ${url}`);
}

function wireServerOutput(server) {
  let stdoutBuffer = '';
  let stderrBuffer = '';
  let readyUrl = null;

  const captureLines = (chunk, buffer, output) => {
    let nextBuffer = buffer + chunk;
    const visibleLines = [];
    let newlineIndex = nextBuffer.indexOf('\n');
    while (newlineIndex >= 0) {
      const line = nextBuffer.slice(0, newlineIndex).replace(/\r$/, '');
      if (line.startsWith(readyToken)) {
        readyUrl = line.slice(readyToken.length).trim();
      } else {
        visibleLines.push(line);
      }
      nextBuffer = nextBuffer.slice(newlineIndex + 1);
      newlineIndex = nextBuffer.indexOf('\n');
    }
    if (visibleLines.length > 0) {
      output.write(`${visibleLines.join('\n')}\n`);
    }
    return nextBuffer;
  };

  server.stdout?.setEncoding('utf8');
  server.stderr?.setEncoding('utf8');

  server.stdout?.on('data', (chunk) => {
    stdoutBuffer = captureLines(chunk, stdoutBuffer, process.stdout);
  });
  server.stderr?.on('data', (chunk) => {
    stderrBuffer = captureLines(chunk, stderrBuffer, process.stderr);
  });

  return () => {
    for (const line of [stdoutBuffer, stderrBuffer]) {
      const trimmed = line.replace(/\r$/, '').trim();
      if (trimmed.startsWith(readyToken)) {
        readyUrl = trimmed.slice(readyToken.length).trim();
      }
    }
    return readyUrl;
  };
}

function waitForServerReady(server, timeoutMs, expectedUrl) {
  return new Promise((resolve, reject) => {
    let settled = false;
    const getReadyUrl = wireServerOutput(server);

    const timer = setTimeout(() => {
      if (settled) {
        return;
      }
      settled = true;
      reject(new Error(`Timed out waiting for smoke server ready signal on ${expectedUrl}`));
    }, timeoutMs);

    const finalize = (callback) => {
      if (settled) {
        return;
      }
      settled = true;
      clearTimeout(timer);
      callback();
    };

    const pollReady = () => {
      const url = getReadyUrl();
      if (!url) {
        return;
      }
      finalize(() => resolve(url));
    };

    server.stdout?.on('data', pollReady);
    server.stderr?.on('data', pollReady);
    server.once('error', (error) => {
      finalize(() => reject(error));
    });
    server.once('exit', (code, signal) => {
      const url = getReadyUrl();
      if (url) {
        finalize(() => resolve(url));
        return;
      }
      const reason = signal ? `signal ${signal}` : `code ${code ?? 0}`;
      finalize(() =>
        reject(new Error(`Smoke server exited before ready signal for ${expectedUrl} (${reason})`)),
      );
    });
  });
}

function spawnChild(command, args, options = {}) {
  const child = spawn(command, args, {
    stdio: 'inherit',
    shell: false,
    ...options,
  });
  return new Promise((resolve, reject) => {
    child.once('error', reject);
    child.once('exit', (code, signal) => {
      resolve({ code: code ?? 0, signal: signal ?? null });
    });
  });
}

function runSetupScript(setupScript, action) {
  if (!setupScript) {
    return Promise.resolve();
  }
  return spawnChild(process.execPath, [setupScript, action], {
    cwd: process.cwd(),
    env: {
      ...process.env,
      PANTHEON_EXTERNAL_WEB_SERVER: '1',
      PANTHEON_WEB_BASE_URL: webBaseUrl,
    },
  }).then((result) => {
    if (result.code !== 0) {
      throw new Error(`setup script ${action} exited with code ${result.code ?? 'unknown'}`);
    }
  });
}

const port = readArg('--port', '5173');
const host = readArg('--host', '127.0.0.1');
const config = readArg('--config');
const timeoutMs = Number(readArg('--timeout', '60000'));
const proxyTarget = readArg('--proxy-target');
const setupScript = readArg('--setup');
const serverScript = readArg('--server-script', 'scripts/start-smoke-vite.mjs');
const playwrightCli = readArg('--playwright-cli', './node_modules/playwright/cli.js');
const playwrightSubcommand = readArg('--playwright-subcommand', 'test');
const separatorIndex = process.argv.indexOf('--');
const testArgs = separatorIndex >= 0 ? process.argv.slice(separatorIndex + 1) : [];
const webBaseUrl = `http://${host}:${port}`;
const playwrightOutputDir = path.join(
  os.tmpdir(),
  'pantheon-playwright',
  `smoke-run-${port}-${Date.now()}-${process.pid}`,
);

if (!config) {
  throw new Error('--config is required');
}

const serverArgs = [serverScript, '--host', host, '--port', port];
if (proxyTarget) {
  serverArgs.push('--proxy-target', proxyTarget);
}
let server = null;

let shuttingDown = false;
async function stopServer() {
  if (shuttingDown) {
    return;
  }
  shuttingDown = true;
  if (server.exitCode !== null) {
    return;
  }
  server.kill('SIGTERM');
  await new Promise((resolve) => {
    const timer = setTimeout(() => {
      if (server.exitCode === null) {
        server.kill('SIGKILL');
      }
    }, 5000);
    server.once('exit', () => {
      clearTimeout(timer);
      resolve();
    });
  });
}

for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP', 'SIGBREAK']) {
  process.on(signal, async () => {
    await stopServer();
    process.exit(130);
  });
}

try {
  await runSetupScript(setupScript, 'up');
  server = spawn(process.execPath, serverArgs, {
    cwd: process.cwd(),
    env: {
      ...process.env,
      PANTHEON_EXTERNAL_WEB_SERVER: '1',
      PANTHEON_SMOKE_READY_TOKEN: readyToken,
      PANTHEON_WEB_BASE_URL: webBaseUrl,
    },
    stdio: ['ignore', 'pipe', 'pipe'],
    shell: false,
  });
  const readyUrl = await waitForServerReady(server, timeoutMs, webBaseUrl);
  if (normalizeUrl(readyUrl) !== normalizeUrl(webBaseUrl)) {
    throw new Error(`Smoke server announced unexpected ready URL ${readyUrl}; expected ${webBaseUrl}`);
  }
  await waitForHttp(webBaseUrl, timeoutMs);
  const result = await spawnChild(process.execPath, [playwrightCli, playwrightSubcommand, ...testArgs, '--config', config], {
    cwd: process.cwd(),
    env: {
      ...process.env,
      PANTHEON_EXTERNAL_WEB_SERVER: '1',
      PANTHEON_WEB_BASE_URL: webBaseUrl,
      PANTHEON_PLAYWRIGHT_OUTPUT_DIR: playwrightOutputDir,
    },
  });
  await stopServer();
  try {
    await runSetupScript(setupScript, 'down');
  } catch (teardownError) {
    console.error(teardownError);
    process.exit(1);
  }
  process.exit(result.code ?? (result.signal ? 1 : 0));
} catch (error) {
  console.error(error);
  await stopServer();
  try {
    await runSetupScript(setupScript, 'down');
  } catch (teardownError) {
    console.error(teardownError);
  }
  process.exit(1);
}
