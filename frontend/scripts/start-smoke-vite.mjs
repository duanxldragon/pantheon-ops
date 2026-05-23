import path from 'node:path';
import process from 'node:process';
import { createServer } from 'vite';

function readArg(flag, fallback = '') {
  const index = process.argv.indexOf(flag);
  if (index < 0 || index + 1 >= process.argv.length) {
    return fallback;
  }
  return process.argv[index + 1];
}

const host = readArg('--host', '127.0.0.1');
const port = Number(readArg('--port', '5173'));
const proxyTarget = readArg('--proxy-target');
const readyToken = process.env.PANTHEON_SMOKE_READY_TOKEN ?? '';

process.env.PANTHEON_SMOKE = '1';
if (proxyTarget) {
  process.env.PANTHEON_API_PROXY_TARGET = proxyTarget;
}

const server = await createServer({
  clearScreen: false,
  configLoader: 'runner',
  server: {
    host,
    port,
    strictPort: true,
  },
});

let closing = false;
const closeServer = async (exitCode = 0) => {
  if (closing) {
    return;
  }
  closing = true;
  try {
    await server.close();
  } finally {
    process.exit(exitCode);
  }
};

for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP', 'SIGBREAK']) {
  process.on(signal, () => {
    void closeServer(0);
  });
}

process.on('uncaughtException', (error) => {
  console.error(error);
  void closeServer(1);
});

process.on('unhandledRejection', (error) => {
  console.error(error);
  void closeServer(1);
});

await server.listen();
const urls = server.resolvedUrls?.local ?? [];
if (urls.length > 0) {
  if (readyToken) {
    console.log(`${readyToken} ${urls[0]}`);
  }
  console.log(`Smoke Vite ready at ${urls[0]}`);
}
