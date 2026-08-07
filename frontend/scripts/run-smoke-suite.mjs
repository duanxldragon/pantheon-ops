import { spawn } from 'node:child_process';
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
  let normalized = String(url || '');
  while (normalized.endsWith('/')) {
    normalized = normalized.slice(0, -1);
  }
  return normalized;
}

function isTruthy(value) {
  return /^(1|true|yes|on)$/i.test(String(value ?? '').trim());
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
    stdio: options.captureOutput ? ['ignore', 'pipe', 'pipe'] : 'inherit',
    shell: false,
    ...options,
  });
  return new Promise((resolve, reject) => {
    let stdout = '';
    let stderr = '';
    if (options.captureOutput) {
      child.stdout?.setEncoding('utf8');
      child.stderr?.setEncoding('utf8');
      child.stdout?.on('data', (chunk) => {
        stdout += chunk;
        process.stdout.write(chunk);
      });
      child.stderr?.on('data', (chunk) => {
        stderr += chunk;
        process.stderr.write(chunk);
      });
    }
    child.once('error', reject);
    child.once('exit', (code, signal) => {
      resolve({ code: code ?? 0, signal: signal ?? null, stdout, stderr });
    });
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
const generatedCleanupScript = readArg('--generated-cleanup-script', 'scripts/cleanup-generated-modules.mjs');
const fixtureCleanupScript = readArg('--fixture-cleanup-script', 'scripts/cleanup-smoke-fixtures.mjs');
const fixtureCleanupScope = readArg('--cleanup-fixtures', '');
const separatorIndex = process.argv.indexOf('--');
const testArgs = separatorIndex >= 0 ? process.argv.slice(separatorIndex + 1) : [];
const webBaseUrl = `http://${host}:${port}`;
const preserveFixtures = isTruthy(process.env.PANTHEON_SMOKE_PRESERVE_FIXTURES);

let server = null;
let shuttingDown = false;

if (!config) {
  throw new Error('--config is required');
}

async function stopServer() {
  if (shuttingDown) {
    return;
  }
  shuttingDown = true;
  if (!server || server.exitCode !== null) {
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

function cleanupArgs(kind, phase, scope = '') {
  const args = ['--kind', kind, '--phase', phase];
  if (scope) {
    args.push('--scope', scope);
  }
  return args;
}

async function runCleanup(scriptPath, args, label) {
  const result = await spawnChild(process.execPath, [scriptPath, ...args], {
    cwd: process.cwd(),
    env: process.env,
    captureOutput: true,
  });
  const exitCode = result.code ?? (result.signal ? 1 : 0);
  if (exitCode !== 0) {
    console.error(`[smoke-suite] ${label} cleanup exited with code ${exitCode}`);
  }
  return {
    exitCode,
    stdout: result.stdout ?? '',
    stderr: result.stderr ?? '',
  };
}

for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP', 'SIGBREAK']) {
  process.on(signal, async () => {
    await stopServer();
    process.exit(130);
  });
}

async function main() {
  let finalExitCode = 0;

  try {
    const preGeneratedCleanup = await runCleanup(
      generatedCleanupScript,
      cleanupArgs('generated', 'pre'),
      'generated',
    );
    if (preGeneratedCleanup.exitCode !== 0) {
      finalExitCode = preGeneratedCleanup.exitCode;
      return;
    }

    const serverArgs = [serverScript, '--host', host, '--port', port];
    if (setupScript) {
      serverArgs.push('--setup', setupScript);
    }
    if (proxyTarget) {
      serverArgs.push('--proxy-target', proxyTarget);
    }

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

    if (fixtureCleanupScope && !preserveFixtures) {
      const preFixtureCleanup = await runCleanup(
        fixtureCleanupScript,
        cleanupArgs('fixture', 'pre', fixtureCleanupScope),
        'fixture',
      );
      if (preFixtureCleanup.exitCode !== 0) {
        finalExitCode = preFixtureCleanup.exitCode;
        return;
      }
    }

    const result = await spawnChild(
      process.execPath,
      [playwrightCli, playwrightSubcommand, ...testArgs, '--config', config],
      {
        cwd: process.cwd(),
        env: {
          ...process.env,
          PANTHEON_EXTERNAL_WEB_SERVER: '1',
          PANTHEON_WEB_BASE_URL: webBaseUrl,
        },
      },
    );
    finalExitCode = result.code ?? (result.signal ? 1 : 0);
  } catch (error) {
    console.error(error);
    finalExitCode = 1;
  } finally {
    await stopServer();

    if (fixtureCleanupScope && !preserveFixtures) {
      const postFixtureCleanup = await runCleanup(
        fixtureCleanupScript,
        cleanupArgs('fixture', 'post', fixtureCleanupScope),
        'fixture',
      );
      if (finalExitCode === 0 && postFixtureCleanup.exitCode !== 0) {
        finalExitCode = postFixtureCleanup.exitCode;
      }
    }

    const postGeneratedCleanup = await runCleanup(
      generatedCleanupScript,
      cleanupArgs('generated', 'post'),
      'generated',
    );
    if (finalExitCode === 0 && postGeneratedCleanup.exitCode !== 0) {
      finalExitCode = postGeneratedCleanup.exitCode;
    }

    process.exitCode = finalExitCode;
  }
}

await main();
