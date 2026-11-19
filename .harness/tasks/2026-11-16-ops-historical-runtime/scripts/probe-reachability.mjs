#!/usr/bin/env node
/**
 * Task-owned read-only reachability probe for pantheon-ops historical-runtime closeout.
 *
 * Purpose: TCP-connect to candidate endpoints and report OPEN / CLOSED / TIMEOUT /
 * REFUSED / DEREF. Does NOT authenticate, does NOT read or write any data, does NOT
 * echo credentials. This is the "enumerate container/process/endpoint availability"
 * step of the D1/D4 runtime verification, runnable with only `node` (no docker/curl/git
 * permission required).
 *
 * Usage:
 *   node probe-reachability.mjs [--config probes.json] [--timeout 2000]
 *   node probe-reachability.mjs --selfcheck     # TCP to the app port only
 *
 * Output is tab-separated: endpoint  target  status  detail
 * Status values: OPEN | CLOSED(ferr) | TIMEOUT | REFUSED | ECONNREFUSED | DEREF
 * A "OPEN" only means a TCP handshake succeeded on the port; it is NOT proof the
 * service is healthy or the correct service. Open/closed is a reachability fact only.
 *
 * This probe is intentionally credential-free and non-destructive (net.connect only).
 */
import net from 'node:net';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_CONFIG = path.join(__dirname, 'probes.json');

function parseArgs(argv) {
  const out = { config: DEFAULT_CONFIG, timeoutMs: 2500, targets: [] };
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--config') out.config = argv[++i];
    else if (a === '--timeout') out.timeoutMs = parseInt(argv[++i], 10);
    else if (a === '--selfcheck') out.cmd = 'selfcheck';
    else out.targets.push(a);
  }
  return out;
}

function parseTarget(t) {
  if (typeof t === 'string') {
    const idx = t.lastIndexOf(':');
    if (idx > 0 && !Number.isNaN(parseInt(t.slice(idx + 1), 10))) {
      return { host: t.slice(0, idx), port: parseInt(t.slice(idx + 1), 10) };
    }
    return null;
  }
  if (t && typeof t.host === 'string' && typeof t.port === 'number') {
    return { host: t.host, port: t.port };
  }
  return null;
}

function probe(host, port, timeoutMs) {
  return new Promise((resolve) => {
    if (!host || !port) return resolve({ status: 'DEREF', detail: 'no host:port' });
    const socket = new net.Socket();
    let settled = false;
    const done = (status, detail) => {
      if (settled) return;
      settled = true;
      socket.destroy();
      resolve({ status, detail });
    };
    socket.setTimeout(timeoutMs);
    socket.once('connect', () => done('OPEN', `tcp ${host}:${port}`));
    socket.once('timeout', () => done('TIMEOUT', `no response in ${timeoutMs}ms`));
    socket.once('error', (e) => {
      const code = (e && e.code) || e.message;
      done('CLOSED', `${code}`);
    });
    socket.connect({ host, port });
  });
}

async function run(targets, timeoutMs) {
  const rows = [];
  for (const t of targets) {
    const p = parseTarget(t);
    if (!p) { rows.push(['INVALID', JSON.stringify(t), '', 'bad target']); continue; }
    const r = await probe(p.host, p.port, timeoutMs);
    rows.push([`${p.host}:${p.port}`, `${p.host}`, r.status, r.detail]);
  }
  return rows;
}

async function main() {
  const args = parseArgs(process.argv);

  // Selfcheck: TCP to the app/health server port that the candidate documents.
  if (args.cmd === 'selfcheck') {
    const app = parseTarget('127.0.0.1:8080');
    const r = await probe(app.host, app.port, args.timeoutMs);
    console.log(`selfcheck\t127.0.0.1:8080\t${r.status}\t${r.detail}`);
    return;
  }

  let targets;
  if (args.targets.length) {
    targets = args.targets;
  } else if (fs.existsSync(args.config)) {
    const cfg = JSON.parse(fs.readFileSync(args.config, 'utf8'));
    targets = cfg.targets || [];
  } else {
    console.error(`probe-reachability.mjs: no --targets and config not found at ${args.config}`);
    process.exit(2);
  }

  const rows = await run(targets, args.timeoutMs);
  let maxHost = 10;
  for (const r of rows) if (r[0].length > maxHost) maxHost = r[0].length;
  for (const r of rows) {
    console.log(`${r[0].padEnd(maxHost)}\t${r[2].padEnd(8)}\t${r[3]}`);
  }
  const opens = rows.filter((r) => r[2] === 'OPEN').length;
  console.log(`---\n${rows.length} targets, ${opens} OPEN (TCP handshake only; not service-health proof)`);
}

main().catch((e) => { console.error(e); process.exit(1); });