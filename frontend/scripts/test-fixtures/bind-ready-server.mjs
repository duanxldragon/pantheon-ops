import http from 'node:http';
import process from 'node:process';

function readArg(flag, fallback = '') {
  const index = process.argv.indexOf(flag);
  if (index < 0 || index + 1 >= process.argv.length) {
    return fallback;
  }
  return process.argv[index + 1];
}

const host = readArg('--host', '127.0.0.1');
const port = Number(readArg('--port', '5173'));
const readyToken = process.env.PANTHEON_SMOKE_READY_TOKEN ?? '';
const server = http.createServer((_request, response) => {
  response.writeHead(200, { 'content-type': 'text/plain' });
  response.end('ok');
});

let closing = false;
function closeServer(exitCode = 0) {
  if (closing) {
    return;
  }
  closing = true;
  server.close(() => {
    process.exit(exitCode);
  });
}

server.on('error', (error) => {
  console.error(error);
  process.exit(1);
});

for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP', 'SIGBREAK']) {
  process.on(signal, () => closeServer(0));
}

server.listen(port, host, () => {
  const url = `http://${host}:${port}`;
  if (readyToken) {
    console.log(`${readyToken} ${url}`);
  }
  console.log(`Fixture ready at ${url}`);
});
